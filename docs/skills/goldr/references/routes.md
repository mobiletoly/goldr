# Goldr Routes Reference For App Agents

Use this reference when editing pages, layouts, dynamic route directories, URL
helpers, generated handlers, or route-level error handling in a Goldr app.

## Filesystem Route Tree

Goldr routes live under `app/routes`.

```text
app/routes/
  layout.go
  layout.templ
  route.go
  page.templ
  users/
    layout.go
    layout.templ
    route.go
    page.templ
    frag_table.templ
    by_id/
      route.go
      page.templ
```

Do not introduce JavaScript-style route syntax. Goldr uses Go-safe names.
Static directory underscores become hyphens in browser paths:

```text
admin_v1/ -> /admin-v1
by_id/    -> {id}
```

Invalid route names include uppercase names, `blog-posts/`, `by_/`,
dot-prefixed names, and underscore-prefixed private names. Goldr ignores
Go-special directories such as `internal`, `testdata`, and `vendor`.

## Pages

`route.go` declares the page, fragments, and actions for its directory.
`page.templ` is optional and is used for page-owned HTML written in templ.

```go
var Route = goldr.RouteDef{
	Page: page,
}

func page(r *http.Request) goldr.PageRouteResponse {
	return goldr.NewPage(
		PageView(),
		goldr.PageMetadata{
			Title:       "Users",
			Description: "Manage users.",
		},
	)
}
```

The `Route` value must be the package's only `Route` declaration. It must use
a static, keyed `goldr.RouteDef`, `goldr.KitRouteDef[K]`, or
`goldr.KitRouteMount[K]` composite literal.
Blank and dot imports are unsupported in `route.go`, and app code must not
declare reserved `GoldrRoute*` symbols.
Keep helper Go files ordinary and let `route.go` declare the route endpoints.
Kit constructors in `New` must be named functions or method selectors accepted
for that route kind. Do not use inline function literals in `New`; Goldr parses
`route.go` statically for inspection and generated adapters.

Recommended route package layout:

- `route.go`: route declaration only.
- `handlers.go`: page, fragment, and action handlers when they belong to one
  route workflow.
- `page.templ` and `frag_*.templ`: route-owned templ views.

Split handlers into files such as `page_handlers.go`, `action_handlers.go`, or
`fragment_handlers.go` only when the route package is large enough that one
`handlers.go` is harder to scan. Small examples may keep handlers in `route.go`
for brevity.

Keep templates used by only one route directly in that route directory. Do not
create route-local packages such as `prepare/internal/prepareui` just to hold
templates for one route. Use `internal` packages, shared packages, or
`KitRouteDef` only when implementation is genuinely shared by multiple sibling
routes or route trees.

Pages can return:

- `goldr.NewPage(component, metadata)`
- `goldr.NewPage(component, metadata).WithStatus(status)`
- `goldr.Redirect{Location: "...", Status: http.StatusSeeOther}`
- `goldr.Text{Status: http.StatusForbidden, Body: "forbidden"}`
- `goldr.RouteError{Err: err}`

Use explicit status pages when the page owns the error response shape and
`goldr.RouteError` when generated route error handling should classify the
error and choose the response shape.

`goldr.Page`, `goldr.Fragment`, `goldr.Redirect`, and `goldr.Text` can carry
headers with `WithHeader` and `AddHeader`.

## Shared Kit Routes

Use `goldr.KitRouteDef[K]` when multiple filesystem-owned routes should reuse
the same page, fragment, or action implementation. Each route directory still
owns its URL and route surface through its own `route.go`.

```go
import sharedreports "myapp/app/reports"

var Route = goldr.KitRouteDef[sharedreports.Kit]{
	Title: "Admin Reports",
	New:   newReportKit,
	Page:  sharedreports.Kit.Page,
	Fragments: goldr.KitFragments[sharedreports.Kit]{
		goldr.KitFragmentRoute("/table", sharedreports.Kit.Table),
	},
}

func newReportKit(r *http.Request) sharedreports.Kit {
	return sharedreports.New(reportData(r))
}
```

Generated adapters call `newReportKit(r)` for each request and then call the
selected kit method directly. There is no runtime registry, hidden router, or
shared URL owner. Read `shared-kit-routes.md` before introducing or changing
Kit-backed routes.

Use `goldr.KitRouteMount[K]` with `app/mounts` when the same Kit-backed route
subtree should appear under multiple live route owners:

```text
app/routes/admin/reports/route.go
app/routes/user/reports/route.go
app/mounts/reports/route.go
app/mounts/reports/page.templ
app/mounts/reports/fragments.templ
```

The live owner declares:

```go
var Route = goldr.KitRouteMount[sharedreports.Kit]{
	New:   newReportKit,
	Mount: "reports",
	Routes: goldr.MountRoutes{
		{Path: "/"},
		{Path: "/audit"},
	},
}
```

The mounted subtree declares `goldr.KitRouteDef[K]` without `New`. It is not
routable on its own, and it does not generate standalone URL helpers. It can
own the reusable kit type, handlers, and templ components. The live owner
supplies `New` as a local identifier, and `Mount` is a clean relative slash
path under `app/mounts` using lowercase Go-safe route directory components.
Underscores still become hyphens in final URL paths. Route helpers
remain owned by the final mounted path, such as
`urls.Admin.Reports.Table.Path()`.

`KitRouteMount.New` must be a local named function in the live owner package.
Do not use an inline function literal there, even though the Go field type is
`func(*http.Request) K`.

Use `KitRouteMount.Routes` when a live owner exposes only part of a mounted
subtree. Omit it to expose the full mounted subtree. Entries are
structured `goldr.MountRoute` values with mount-relative browser route
patterns such as `/`, `/audit`, or `/{id}`. Excluded children are not live
endpoints for that owner. If an owner selects a child without `/`, the owner
still gets a mount-base `Path()` helper for binding `NewGoldrMountURLs`; the
mount root still does not dispatch.

`RouteDef.Name`, `RouteDef.Title`, `KitRouteDef.Name`, `KitRouteDef.Title`,
and `RouteMeta.Labels` are optional display metadata for route inspection.
They do not change URL paths, URL helper names, dispatch, auth, navigation, or
other application policy.

## Route-Local Workflows

When an HTMX action or fragment exists only to support one page, prefer nesting
it under the page route instead of creating a flat sibling route.

Recommended shape:

```text
users/
  route.go
  page.templ
  prepare/
    route.go
    action_handlers.go
    result.templ
  save/
    route.go
    action_handlers.go
```

This is valid even when `prepare/` and `save/` have no standalone page. Their
`route.go` files can declare only index actions, named actions, index
fragments, or named fragments. The directory exists because it owns a child
route workflow, middleware, params, templates, or helper name.

Prefer this over flat routes such as:

```text
users/prepare_user
users/save_user
```

The nested shape keeps URL helpers and filesystem ownership aligned:

```go
urls.Users.Prepare.Path()
urls.Users.Save.Path()
```

## Route Directory Names And Helpers

Generated URL helper namespaces mirror route directory names and action or
fragment segments. Choose child directory names relative to their parent.

Good:

```text
pending_events/send      -> PendingEvents.Send
user_events/prepare      -> UserEvents.Prepare
user_events/send_updated -> UserEvents.SendUpdated
```

Avoid repeating parent context:

```text
user_events/send_user_updated_event
notifications/send_pending_events
notifications/prepare_user_event
```

Use `go tool goldr routes list --app-root <app-root>` before and after route
refactors to inspect the path and helper together. The `HELPER` column should
read like the route ownership you intended.

## Layouts

`layout.go` defines a layout for pages in that directory and below. It has a
matching `layout.templ`.

```go
func Layout(r *http.Request, ctx goldr.LayoutContext) templ.Component {
	return LayoutView(ctx.Metadata, ctx.Child)
}
```

`ctx.Child` is the child page or nested layout component. `ctx.Metadata` comes
from the matched page. Fragments are not layout-wrapped. Actions return route
responses; page responses from actions are written through the matched layout
stack.

## Dynamic Routes

Use `by_<param>/` directories.

```text
app/routes/users/by_id/route.go -> /users/{id}
```

Read decoded params with:

```go
id := r.PathValue("id")
```

Static routes win over dynamic routes when both could match.

## Index Fragments

Use an index fragment for a body-only HTMX endpoint at the route directory path:

```go
var Route = goldr.RouteDef{
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/", statusOptions),
	},
}
```

```text
goldr.FragmentRoute("/", statusOptions) -> GET,HEAD /users/status-options
```

Index fragments are fragments, not pages. They are not layout-wrapped, cannot be
declared beside `Page`, and can share the path with index actions when methods
differ.

## Actions

Actions are declared in `route.go`. Ordinary action handlers return
`goldr.RouteResponse`.

```go
var Route = goldr.RouteDef{
	Actions: goldr.Actions{
		goldr.Action(http.MethodPost, "/create", postCreate),
		goldr.Action(http.MethodPost, "/", postIndex),
	},
}
```

Supported method prefixes are `Post`, `Put`, `Patch`, and `Delete`. `Index`
maps to the current directory path:

```text
goldr.Action(http.MethodPost, "/", postIndex) -> POST /users
goldr.Action(http.MethodPost, "/create", postCreate) -> POST /users/create
goldr.Action(http.MethodPost, "/save-preview", postSavePreview) -> POST /users/save-preview
```

Actions may return pages, fragments, redirects, text, route errors, or
`goldr.NoContent{}`. Use `goldr.HTTPAction` only when an action needs direct
`http.ResponseWriter` control.

## Route-Tree Middleware

Use `middleware.go` when ordinary `net/http` endpoint middleware belongs to a
route subtree.

```go
func Middleware(next http.Handler) http.Handler
```

Use the exact `http.Handler` spelling with an unaliased `net/http` import.

Goldr discovers middleware by route directory and wraps matched pages, actions,
and fragments in generated endpoint dispatch. Inherited middleware runs
root-to-leaf. Layouts are not standalone middleware targets; they render inside
the already wrapped page or action request.

Middleware inheritance follows source directory ancestry, not runtime URL
prefix matching. For example, `app/routes/users/create/middleware.go` does not
wrap `POST /users/create` when that action is declared in
`app/routes/users/route.go`.

Examples:

- `app/routes/middleware.go` can issue CSRF tokens for a cookie/session HTML
  app.
- `app/routes/main/admin/middleware.go` can authenticate, check an admin role,
  and attach a principal to request context.

Goldr does not own CSRF validation policy, auth, roles, rate limits, sessions,
or adapters through this convention. Keep those rules in app-owned middleware.
Use mux-level middleware for concerns that must also run on generated 404 and
405 responses.
Generated 404 and 405 responses do not run route-tree middleware.

## URL Helpers

Goldr generates app-specific URL helpers in `app/urls/goldr_gen.go`.

Use them in templates and handlers:

```go
import "myapp/app/urls"
```

Examples:

```go
urls.Root.Path()
urls.Users.Path()
urls.Users.Create.Path()
urls.Users.Table.Path()
urls.Users.ByID.Bind(id).Path()
```

Dynamic params are bound with `.Bind(value)` and path-escaped by helpers.
Use helpers instead of hard-coded internal route paths when helpers exist.

For apps mounted below a URL prefix, bind helpers once with the app base path:

```go
appURLs := urls.WithBasePath(appDeps.BasePath)
appURLs.Users.ByID.Bind(id).Path()
```

`WithBasePath` returns `urls.MountedRoutes`, so app-owned helpers may accept the
mounted route set when they need to add query strings or make semantic choices.
Do not create route-specific string helpers such as `TenantURL` when generated
helpers already cover the route. Keep mux mounting and prefix stripping
application-owned.

Referenced Kit mount subtrees also get mount-relative helpers in
`app/mounts/<mount>/goldr_gen.go`. Bind them from the final live route helper
and pass them into the shared kit data:

```go
reportURLs := reports.NewGoldrMountURLs(urls.Admin.Reports)
```

Mounted code can use `reportURLs.Path()` and child helpers such as
`reportURLs.Table.Path()` without knowing the live owner path.

Do not pass `urls.Admin.Reports.Path()` to `NewGoldrMountURLs`; the constructor
expects a route helper object so child helper paths stay relative to the same
live owner.

Generated mounted helpers include every route declaration from the mounted
source subtree, including owner-specific children. They are subtree path
helpers, not live route inventory. Use app-owned state before rendering links
to children that only some owners select:

```go
reportURLs := reports.NewGoldrMountURLs(urls.Admin.Reports)
if report.ShowAudit {
	link := reportURLs.Audit.Path()
}
```

A child-only owner still exposes the mount-base helper in `app/urls` for the
`NewGoldrMountURLs` binding pattern. That helper does not make the mount root a
live route.

## Generated Handler

Generated route packages expose:

```go
func Handler() http.Handler
func HandlerWithOptions(options HandlerOptions) http.Handler
```

`Handler()` is the normal generated route handler. `HandlerWithOptions` is for
custom error responses and template inspection.

Error hooks are optional:

```go
type ErrorHandlers struct {
	RouteNotFound         func(*http.Request) goldr.RouteResponse
	RouteMethodNotAllowed func(*http.Request) goldr.RouteResponse
	RouteError            func(*http.Request, error) goldr.RouteResponse
}
```

Hooks return normal `goldr.RouteResponse` values. Use explicit status on full
error pages:

```go
func RouteNotFound(r *http.Request) goldr.RouteResponse {
	return goldr.NewPage(NotFoundView(), goldr.PageMetadata{
		Title: "Page not found",
	}).WithStatus(http.StatusNotFound)
}
```

Use `goldr.RouteError{Err: err}` from page, fragment, or action handlers when
the app should classify the error in one generated hook. This is for matched
route errors, including public request-shaped errors such as validation,
authorization, not-found, and conflict cases. Router misses use
`RouteNotFound`, not `RouteError`:

```go
func RouteError(r *http.Request, err error) goldr.RouteResponse {
	status, message := classifyRouteError(err)
	if hx.IsRequest(r) {
		return goldr.NewFragment(ErrorToast(message)).
			WithStatus(status).
			WithHeader(hx.HeaderRetarget, "#toast")
	}
	return goldr.NewPage(ErrorPage(message), goldr.PageMetadata{
		Title: http.StatusText(status),
	}).WithStatus(status)
}
```

Apps choose HTMX behavior inside the hook using the original request, for
example by returning a toast fragment when `hx.IsRequest(r)` is true. Goldr
does not choose app components.

Nil hooks keep Goldr defaults. Full 404 and 405 pages use the root layout when
available; full route-error pages use the matched route layout stack.
Fragment, text, redirect, and no-content responses are written as returned.
Direct writer action responses and static asset error responses stay
application-owned.
