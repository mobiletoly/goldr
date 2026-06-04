# Shared Kit Routes

Use this reference when two or more filesystem-owned routes should reuse the
same page, fragment, action implementation, or route subtree.

Kit routes are an advanced reuse tool. They are not a second router, a component
registry, dependency injection, or a way to hide route ownership. Each URL is
still declared by the route directory's own `route.go`.

## When To Use Kit

Use `goldr.KitRouteDef[K]` when all of these are true:

- multiple route directories need the same route behavior
- the shared behavior can be represented as ordinary Go methods on one kit type
- each route can adapt its own request into a fresh request-scoped kit value
- route URLs, params, metadata, and generated URL helpers should remain owned
  by the route directories

Use `goldr.KitRouteMount[K]` and `app/mounts` when the repeated code is an
entire child route subtree, not one route declaration.

Prefer ordinary local `goldr.RouteDef` routes when the behavior belongs to one
route family, when only a small templ helper is shared, or when reuse would make
the route harder to inspect. A route-local named fragment can call a shared
function directly; do not introduce a Kit route only to reuse a simple fragment
handler.

## Route-Owned Binding

A Kit route declares a static `Route` value in `route.go`:

```go
package reports

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	sharedreports "myapp/app/reports"
	"myapp/app/urls"
)

var Route = goldr.KitRouteDef[sharedreports.Kit]{
	Title: "Admin Reports",
	New:   newReportKit,
	Page:  sharedreports.Kit.Page,
	Fragments: goldr.KitFragments[sharedreports.Kit]{
		goldr.KitFragmentRoute("/table", sharedreports.Kit.Table),
	},
	Actions: goldr.KitActions[sharedreports.Kit]{
		goldr.KitAction(http.MethodPost, "/refresh", sharedreports.Kit.PostRefresh),
	},
}

func newReportKit(r *http.Request) (sharedreports.Kit, error) {
	return sharedreports.New(sharedreports.ReportData{
		Audience:  "Admin",
		Title:     "Admin Reports",
		TablePath: urls.Admin.Reports.Table.Path(),
	}), nil
}
```

The route package owns `newReportKit`. That constructor reads the request,
generated URL helpers, app dependencies, path params, query params, session
facts, or other app-owned state and returns the kit value for this request.

A child route can expose a Kit-backed index fragment at its own path:

```go
var Route = goldr.KitRouteDef[sharedreports.Kit]{
	New: newReportKit,
	Fragments: goldr.KitFragments[sharedreports.Kit]{
		goldr.KitFragmentRoute("/", sharedreports.Kit.StatusOptions),
	},
}
```

`KitRouteDef[K]` has one type parameter. `New` is:

```go
func(*http.Request) (K, error)
```

The only supported Kit declaration shape is `goldr.KitRouteDef[K]`. Under
`app/routes`, it must declare `New func(*http.Request) (K, error)`. Under
`app/mounts`, it must omit `New` because the `KitRouteMount` owner supplies the
constructor. Do not add a second type parameter or a separate framework context
object.

For live `KitRouteDef` declarations under `app/routes`, `New` may be a named
local function or method selector with type `func(*http.Request) (K, error)`.
Inline function literals are not supported because Goldr parses `route.go`
statically for inspection and generated adapters.

## Mounted Route Subtrees

Use `app/mounts` for non-live reusable route subtrees:

```text
app/routes/admin/reports/route.go
app/routes/user/reports/route.go

app/mounts/reports/route.go
app/mounts/reports/page.templ
app/mounts/reports/fragments.templ
```

The live route owner mounts the subtree:

```go
var Route = goldr.KitRouteMount[sharedreports.Kit]{
	New:   newReportKit,
	Mount: "reports",
}
```

The mounted subtree uses `KitRouteDef` without `New`:

```go
var Route = goldr.KitRouteDef[Kit]{
	Page: Kit.Page,
	Fragments: goldr.KitFragments[Kit]{
		goldr.KitFragmentRoute("/table", Kit.Table),
	},
}
```

Rules:

- `app/mounts` never owns live URLs by itself.
- A mounted subtree can own the reusable kit type, handler methods, and templ
  components for that subtree.
- A mounted source route may declare default `Nav` label or key metadata for
  its reusable page.
- `Mount` is a clean relative path under `app/mounts`.
- The mount owner supplies the request-scoped
  `New func(*http.Request) (K, error)` as a local identifier in the owner route
  package.
- Do not use an inline function literal for `KitRouteMount.New`; it must be a
  local named function in the live owner package.
- `Mount` is a clean relative slash path under `app/mounts`; each component is
  a lowercase Go-safe route directory name, with underscores still becoming
  hyphens in final URL paths.
- Final URL helpers remain path-derived from the live `app/routes` owner.
- Referenced mount roots also get `app/mounts/<mount>/goldr_gen.go` with
  `NewGoldrMountURLs(route interface{ Path() string })` for links within the
  mounted subtree.
- Use `KitRouteMount.Routes` when one live owner exposes only part of a
  mounted subtree. Excluded children are absent from that owner's dispatch,
  route list, URL helpers, and middleware composition. Child-only selections
  still get the owner mount-base URL helper for binding `NewGoldrMountURLs`;
  the mount root does not dispatch unless `/` is selected.
- Mount-relative helpers include mounted source routes, including children
  exposed by only some owners. They are subtree path helpers, not selected live
  route inventory.
- Mounted subtrees may define layouts, but middleware in `app/mounts` is
  rejected.
- `RouteDef` is invalid in `app/mounts`.
- `KitRouteDef.New` is invalid in `app/mounts`; the mount owner supplies it.
- `KitRouteDef.Destinations` is invalid in `app/mounts`; live owners declare
  destinations on selected `MountRoute` entries.
- `KitRouteDef.New` is required in `app/routes`.

Run `go tool goldr routes list --json` when inspecting mounted routes. JSON
declaration output includes the mounted source and the live owner:

```json
"mount": {
  "path": "reports",
  "owner": "admin/reports/route.go"
}
```

Bind mounted helpers from the live owner before passing them into shared kit
data:

```go
reportURLs := reports.NewGoldrMountURLs(urls.Admin.Reports)
```

Mounted templates can then use `reportURLs.Path()` for the mount root and
`reportURLs.Table.Path()` for child routes without hard-coding the owner path.
Use app-owned state before rendering links to owner-specific mounted children.
A child-only owner still has the mount-base helper needed for this binding, but
that helper does not make the root URL live.

## Mounted Fragment And Action URLs

When a mounted page links to a fragment or action inside the same mounted
subtree, pass the bound `GoldrMountURLs` value into the kit and derive the
child path inside the mounted implementation. Do not pass one raw URL string
per mounted child route.

Good:

```go
func newAccountEditKit(r *http.Request) (accountedit.Kit, error) {
	appURLs := urls.App()
	editRoute := appURLs.Admin.Users.ByID.Bind(r.PathValue("id")).Edit
	editURLs := accountedit.NewGoldrMountURLs(editRoute)

	return accountedit.New(accountedit.Config{
		URLs: editURLs,
	}), nil
}
```

```templ
templ Page(kit Kit) {
	<button
		hx-get={ kit.URLs.ResetPassword.Path() }
		hx-target="#reset-password-dialog-slot"
		hx-swap="innerHTML"
	>
		Reset password
	</button>
	<div id="reset-password-dialog-slot"></div>
}

templ ResetPasswordDialog(kit Kit) {
	<form
		hx-post={ kit.URLs.ResetPassword.Path() }
		hx-target="#reset-password-dialog-slot"
		hx-swap="innerHTML"
	>
		<!-- fields -->
	</form>
}
```

Avoid:

```go
return accountedit.New(accountedit.Config{
	ResetPasswordURL: appURLs.Admin.Users.ByID.Bind(id).Edit.ResetPassword.Path(),
})
```

Owner-provided callbacks or raw URL fields are still appropriate for
destinations outside the mounted subtree, owner-specific external links, or
routes that cannot be represented by generated helpers.

## Shared Package Shape

The shared package owns ordinary Go data, methods, and templ components. It
does not declare URLs.

```go
package reports

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

type ReportData struct {
	Audience  string
	Title     string
	TablePath string
	Rows      []Row
}

type Row struct {
	Name  string
	Value string
}

type Kit struct {
	data ReportData
}

func New(data ReportData) Kit {
	return Kit{data: data}
}

func (kit Kit) Page(_ *http.Request) goldr.PageRouteResponse {
	return goldr.NewPage(
		PageView(kit.data),
		goldr.PageMetadata{Title: kit.data.Title},
	)
}

func (kit Kit) Table(_ *http.Request) goldr.FragmentRouteResponse {
	return goldr.NewFragment(TableView(kit.data))
}

func (kit Kit) PostRefresh(r *http.Request) goldr.RouteResponse {
	return goldr.NewFragment(TableView(kit.data))
}
```

The templ files can live in the shared package when the HTML is genuinely
shared:

```templ
package reports

templ PageView(report ReportData) {
	<section>
		<h1>{ report.Title }</h1>
		<button
			hx-get={ report.TablePath }
			hx-target="#report-table-slot"
			hx-swap="innerHTML"
		>
			Refresh
		</button>
		<div id="report-table-slot">
			@TableView(report)
		</div>
	</section>
}
```

Keep `hx-*` attributes visible in templ. Kit should share server-rendered
behavior, not hide HTMX behind proprietary components.

## Handler Signatures

Kit handlers use the kit value as the first argument:

```go
type KitPageHandler[K any] func(K, *http.Request) goldr.PageRouteResponse

func KitFragmentRoute[K any](path string, fn func(K, *http.Request) goldr.FragmentRouteResponse) goldr.KitFragmentRouteDef[K]
func KitAction[K any](method string, path string, fn func(K, *http.Request) goldr.RouteResponse) goldr.KitActionDef[K]
func KitHTTPAction[K any](method string, path string, fn func(K, http.ResponseWriter, *http.Request)) goldr.KitActionDef[K]
```

Kit pages return `goldr.PageRouteResponse`, fragments return
`goldr.FragmentRouteResponse`, and normal actions return `goldr.RouteResponse`.
`KitHTTPAction` handlers are ordinary HTTP handlers with the kit argument
added.

## Generated Behavior

Goldr generates route-local adapters. For a page, fragment, or action, the
adapter constructs a fresh kit value for the request, returns a route error if
construction fails, and then calls the selected method directly:

```go
func GoldrRoutePage(r *http.Request) goldr.PageRouteResponse {
	goldrKit, err := newReportKit(r)
	if err != nil {
		return goldr.RouteError{Err: err}
	}
	return sharedreports.Kit.Page(goldrKit, r)
}

func GoldrRouteFragTable(r *http.Request) goldr.FragmentRouteResponse {
	goldrKit, err := newReportKit(r)
	if err != nil {
		return goldr.RouteError{Err: err}
	}
	return sharedreports.Kit.Table(goldrKit, r)
}

func GoldrRoutePostRefresh(r *http.Request) goldr.RouteResponse {
	goldrKit, err := newReportKit(r)
	if err != nil {
		return goldr.RouteError{Err: err}
	}
	return sharedreports.Kit.PostRefresh(goldrKit, r)
}
```

For a `KitHTTPAction`, Goldr generates a route-local writer adapter that
returns constructor errors to the dispatcher. The dispatcher then routes that
error through the same route error handling path before returning:

```go
func GoldrRoutePostExport(w http.ResponseWriter, r *http.Request) error {
	goldrKit, err := newReportKit(r)
	if err != nil {
		return err
	}
	sharedreports.Kit.PostExport(goldrKit, w, r)
	return nil
}
```

Goldr does not cache kit values globally or across requests. Do not put
request-scoped facts into package globals.

## URL Helpers And Metadata

Kit routes use the same path-derived URL helpers as local routes:

```go
urls.Admin.Reports.Path()
urls.Admin.Reports.Table.Path()
urls.Admin.Reports.Refresh.Path()
```

Referenced mounted subtrees also generate mount-relative helper sets under
`app/mounts/<mount>/goldr_gen.go`. Bind that helper set from the live owner
route helper object:

```go
reportURLs := reports.NewGoldrMountURLs(urls.Admin.Reports)
reportURLs.Table.Path()
```

Do not bind mounted helpers from a raw path string. Mount helpers include only
mounted source routes; `app/urls` remains the selected live route helper
surface for each owner.

For mounted apps, bind helpers once:

```go
appURLs := urls.WithBasePath("/webapp")
appURLs.Admin.Reports.Table.Path()
```

`Name`, `Title`, and `Meta.Labels` are optional declaration metadata. Goldr can
display them in route inspection output, but they do not change paths, helper
names, auth, navigation, tenant handling, roles, or runtime behavior.

## Inspection And Validation

After changing Kit routes, regenerate and inspect the route surface:

```bash
go tool goldr generate
go tool goldr check
go tool goldr routes list
go tool goldr routes list --json
go tool goldr routes explain /admin/reports
go tool goldr routes explain /admin/reports/table
go test ./...
```

Use `--app-root <path>` on these commands when the Goldr app is nested inside a
larger repository.

If generated code or route inspection shows that a shared package owns the URL
surface, redesign the route binding. The route directory should own the URL;
the shared package should own only reusable behavior.

In `routes list --json`, current Kit declaration rows include
`declaration.kit.kit_type` and `declaration.kit.new`. No separate framework
context fields are part of the current Kit JSON model.
