# Routes

This page is the route reference. For the mental model, read
[Concepts](concepts.md) first.

goldr applications use a filesystem route tree rooted at:

```text
app/routes/
```

Route names are Go-native. Do not use JavaScript-style filesystem route syntax.
Goldr ignores Go-special directories named `internal`, `testdata`, and `vendor`
so applications can keep route-adjacent private code, test fixtures, or vendored
code without exposing those names as URL segments.

## Route Declarations

`route.go` is the primary route surface for a route directory. It declares the
page, fragments, and actions for that directory in one static `Route` value:

```go
package users

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

var Route = goldr.RouteDef{
	Page: page,
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/table", table),
	},
	Actions: goldr.Actions{
		goldr.Action(http.MethodPost, "/create", postCreate),
	},
}

func page(r *http.Request) goldr.PageRouteResponse {
	return goldr.NewPage(PageView(), goldr.PageMetadata{Title: "Users"})
}

func table(r *http.Request) goldr.FragmentRouteResponse {
	return goldr.NewFragment(TableView())
}

func postCreate(r *http.Request) goldr.RouteResponse {
	return goldr.Redirect{Location: "/users", Status: http.StatusSeeOther}
}
```

The `Route` value must be a static `goldr.RouteDef`, `goldr.KitRouteDef[K]`,
or `goldr.KitRouteMount[K]` composite literal. Goldr parses the declaration
without evaluating Go code, so `Page`, `Fragments`, and `Actions` use named
handlers and the Goldr helper calls shown above.

Route declarations are tooling input, not executable configuration. Keep route
declaration expressions named and inspectable. In particular, Kit constructors
in `New` must be named functions or method selectors accepted for that route
kind; inline function literals are not supported in `New`.

`RouteDef.Name`, `RouteDef.Title`, `KitRouteDef.Name`,
`KitRouteDef.Title`, and `RouteMeta.Labels` are optional declaration metadata.
Goldr can show them in route inspection output, but it treats them as
display-only and opaque. Labels are app-owned strings. Goldr does not interpret
them as auth, navigation, tenant, portal, or policy configuration.

`page.go`, `frag_*.go`, and `actions.go` under `app/routes` are invalid route
surface files. Layouts, middleware, templates, helpers, tests, and other
ordinary Go files can still live beside `route.go`.

For real route packages, prefer this file layout:

```text
route.go       route declaration only
handlers.go    page, fragment, and action handlers for one route workflow
page.templ     page-owned HTML when the page renders HTML
frag_*.templ   fragment-owned HTML when fragment handlers use templates
```

Split handlers into files such as `page_handlers.go`, `action_handlers.go`, or
`fragment_handlers.go` only when the package is large enough that one
`handlers.go` is harder to scan. Documentation examples may keep tiny handlers
inside `route.go` for brevity.

Keep templates used by only one route directly in that route directory. Use
`internal` packages or shared packages only when implementation is genuinely
shared by multiple sibling routes or route trees.

If a handler lives in another package and that package's declared package name
does not match the final import path segment, add an explicit import alias in
`route.go`:

```go
import view "example.com/app/pages/handlers"
```

### Shared Kit Routes

Use `goldr.KitRouteDef[K]` when multiple filesystem-owned routes should
reuse the same page, fragment, or action implementation. The route directory
still owns the URL and exposed route surface through its own `route.go`; the
shared package owns ordinary Go methods and templ components.

For example, `/admin/reports` and `/user/reports` can both bind to
`reports.Kit.Page` and `reports.Kit.Table` while each route provides its own
request-scoped constructor:

```go
var Route = goldr.KitRouteDef[reports.Kit]{
	Title: "Admin Reports",
	New:   newReportKit,
	Page:  reports.Kit.Page,
	Fragments: goldr.KitFragments[reports.Kit]{
		goldr.KitFragmentRoute("/table", reports.Kit.Table),
	},
}

func newReportKit(r *http.Request) reports.Kit {
	return reports.New(reportData(r))
}
```

Goldr generates direct route-local adapters that call
`newReportKit(r)` and then call the selected kit method. The shared package
does not declare URLs or hidden route surface. See
`examples/kit_routes` for a focused runnable example.

### Mounted Kit Route Subtrees

Use `goldr.KitRouteMount[K]` with `app/mounts` when multiple live route owners
need the same child route subtree. `app/routes` remains the only live URL tree;
`app/mounts` contains reusable route surfaces that are not routable unless a
real route mounts them.

```text
app/routes/admin/reports/route.go
app/routes/user/reports/route.go

app/mounts/reports/route.go
app/mounts/reports/page.templ
app/mounts/reports/fragments.templ
```

A live owner mounts the subtree:

```go
var Route = goldr.KitRouteMount[reports.Kit]{
	New:   newReportKit,
	Mount: "reports",
	Routes: goldr.MountRoutes{
		{Path: "/"},
		{Path: "/audit"},
	},
}
```

The mounted files declare route surfaces with `KitRouteDef` and omit `New`.
The live `KitRouteMount` owner supplies the request-scoped kit constructor as
a local identifier. `Mount` is a clean relative slash path under `app/mounts`;
each component is a lowercase Go-safe route directory name, with underscores
still becoming hyphens in final URL paths.

`KitRouteMount.New` must be a local named function in the mount owner's route
package. Do not use an inline function literal there, even though the Go field
type is `func(*http.Request) K`.

`KitRouteMount.Routes` is optional. Omit it to expose the full mounted
subtree. Set it to a `goldr.MountRoutes` allowlist when one live owner should
expose only part of the mounted subtree. Entries are structured
`goldr.MountRoute` values with mount-relative browser route patterns such as
`/`, `/audit`, or `/{id}`. Excluded children are not routable for that owner
and do not receive live URL helpers. If a child is selected without `/`, the
live owner still gets a mount-base `Path()` helper so mounted helpers can be
bound from the owner route. That helper does not make the mount root
dispatchable.

Mounted owners may attach owner-specific navigation trail metadata to selected
children:

```go
Routes: goldr.MountRoutes{
	{
		Path: "/customers/{customer_id}/report",
		NavTrails: goldr.NavTrails{
			Allowed: []string{"team-analytics"},
		},
	},
}
```

See [Navigation Trails And Destinations](navigation.md) for destination
`Href()`, route-scoped `NavTrails` constants, and mounted owner trail patterns.

```go
var Route = goldr.KitRouteDef[reports.Kit]{
	Page: reports.Kit.Page,
	Fragments: goldr.KitFragments[reports.Kit]{
		goldr.KitFragmentRoute("/table", reports.Kit.Table),
	},
}
```

For each referenced mounted subtree, Goldr also writes mount-relative URL
helpers into the mounted package:

```text
app/mounts/reports/goldr_gen.go
```

Bind those helpers from the real owner route helper before passing them to
shared code:

```go
reportURLs := reports.NewGoldrMountURLs(urls.Admin.Reports)
```

The mounted package can use helpers such as `reportURLs.Path()` and
`reportURLs.Table.Path()` without knowing whether the live owner is
`/admin/reports`, `/user/reports`, or another mount point.

Mount-relative helpers include every route declaration from the mounted source
subtree, including owner-specific children. They are subtree path helpers, not
live route inventory. Excluded children are still absent from `app/urls`,
dispatch, and normal route lists for owners that did not select them.
When an owner selects only child routes, `app/urls` still includes the mount
base helper used to bind `NewGoldrMountURLs`; the mount root is not registered
unless `/` is selected.

The mounted package can own the reusable kit type and templ components for the
mounted subtree.

The final paths and helpers are still derived from the live mount owner:

```go
urls.Admin.Reports.Table.Path()
urls.User.Reports.Table.Path()
```

See [Mounted Kit Route Subtrees](mounted-routes.md) for rules, layout behavior,
inspection output, and collision checks.

## Route-Local Workflows

If an HTMX action or fragment exists only to support one page workflow, prefer
nesting it under that page route instead of making it a top-level sibling.

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

This is appropriate even when `prepare/` or `save/` has no standalone page. A
child route directory can declare only index actions, named actions, index
fragments, or named fragments. The directory owns the workflow segment,
templates, middleware, params, or generated helper namespace.

Prefer shallow route-owned files for one-route fragments:

```text
user_events/prepare/
  route.go
  action_handlers.go
  user_registered_tab.templ
  user_updated_tab.templ
```

Do not create packages such as `prepare/internal/prepareui` unless the UI is
actually reused by multiple sibling routes or route trees.

### Naming Nested Routes

Generated URL helpers mirror route directory names and action or fragment
segments. Choose child names relative to the parent route.

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

Use `go tool goldr routes list --app-root <app-root>` during route refactors to
inspect paths and helpers together. The helper should read like the filesystem
ownership you intended.

## Pages

In `route.go`, the `Page` field defines the page route for its directory:

```go
var Route = goldr.RouteDef{
	Page: page,
}

func page(r *http.Request) goldr.PageRouteResponse {
	return goldr.NewPage(PageView(), goldr.PageMetadata{Title: "Users"})
}
```

```text
app/routes/route.go                     -> /
app/routes/users/route.go               -> /users
app/routes/settings/build_info/route.go -> /settings/build-info
app/routes/users/by_id/route.go         -> /users/{id}
```

`page.templ` is optional. Use it for page-owned HTML written in templ. A page
that always redirects, returns text, or delegates to `goldr.RouteError` does
not need a marker-only template file.

Use `goldr.NewPage` for a normal rendered page:

```go
return goldr.NewPage(
	PageView(),
	goldr.PageMetadata{
		Title:       "Users",
		Description: "Manage users.",
	},
)
```

Use `WithStatus`, `WithHeader`, and `AddHeader` when the page response needs
explicit response details:

```go
return goldr.NewPage(
	PrivateView(),
	goldr.PageMetadata{Title: "Private"},
).WithHeader("Cache-Control", "no-store")
```

Supported metadata fields are `Title` and `Description`. goldr passes metadata
to layouts. Layouts decide how to render it.

goldr does not infer page titles, render canonical links, or choose active
navigation entries. Those are application layout decisions. Use request data,
generated URL helpers, or app-owned state when a layout needs them.

Page handlers can also return responses before normal rendering:

```go
return goldr.Redirect{Location: "/sign-in", Status: http.StatusSeeOther}
return goldr.NewPage(ForbiddenView(), goldr.PageMetadata{Title: "Forbidden"}).WithStatus(http.StatusForbidden)
return goldr.Text{Status: http.StatusForbidden, Body: "forbidden"}
return goldr.RouteError{Err: err}
```

Redirects and text status responses do not render layouts. Status responses
with a templ component render through the same layout chain as normal pages.
`goldr.RouteError` delegates to the generated `RouteError` hook; if that hook
returns a page, generated dispatch renders it through the matched layout stack.

`goldr.Redirect` accepts only redirect statuses that clients follow: `301`,
`302`, `303`, `307`, and `308`. Rendered page responses and `goldr.Text` accept
only final body-carrying statuses: `2xx` except `204 No Content` and `205 Reset
Content`, plus `4xx` and `5xx`.

`goldr.Page`, `goldr.Fragment`, `goldr.Redirect`, and `goldr.Text` support
`WithHeader` and `AddHeader`. `WithHeader` replaces existing values for that
header name, matching `http.Header.Set`. `AddHeader` appends a value, matching
`http.Header.Add`:

```go
return goldr.Redirect{
	Location: "/sign-in",
	Status:   http.StatusSeeOther,
}.WithHeader("Set-Cookie", sessionCookie.String()).
	AddHeader("Set-Cookie", csrfCookie.String())
```

### Page Error Handling

Use explicit status responses when the page owns the error response shape. Use
`goldr.RouteError{Err: err}` when the matched route should delegate error
classification and page-versus-fragment response shape to the generated
`RouteError` hook.

Generated dispatch resolves the returned route response internally. If
resolution returns an error, the page returned an invalid Goldr contract, such
as `goldr.Page{}`, `goldr.NewPage(nil, metadata)`,
`goldr.Redirect{Location: "", Status: http.StatusSeeOther}`,
`goldr.Redirect{Location: "/sign-in", Status: http.StatusNotModified}`,
`goldr.NewPage(view, metadata).WithStatus(http.StatusNoContent)`, or
`goldr.RouteError{Err: nil}`. Those validation errors are routed to generated
route error handling. `goldr.RouteError{Err: err}` is a valid route response:
its error is passed to the generated `RouteError` hook.

See [Error Handling](error-handling.md) for route handler error examples,
custom generated error hooks, and HTMX error fragments.

## Layouts

`layout.go` defines a layout for pages in that directory and below.

```text
app/routes/layout.go        -> wraps / and pages below /
app/routes/users/layout.go  -> wraps /users and pages below /users
```

Each layout must have a matching `.templ` file and must provide:

```go
func Layout(r *http.Request, ctx goldr.LayoutContext) templ.Component
```

`ctx.Child` is the child page or nested layout component. `ctx.Metadata` is the
page metadata returned by the matched page.

Fragments are not layout-wrapped. Actions return route responses; page
responses from actions are written through the matched route layout stack.

## Dynamic Routes

Dynamic route directories use `by_<param>/`.

```text
app/routes/users/by_id/route.go
```

maps to:

```text
/users/{id}
```

Nested dynamic routes work the same way:

```text
app/routes/orgs/by_org_id/users/by_user_id/route.go
```

maps to:

```text
/orgs/{org_id}/users/{user_id}
```

Generated runtime dispatch attaches decoded params to the request:

```go
id := r.PathValue("id")
```

Dynamic segments must be non-empty. Static routes win when a static and dynamic
route could both match.

## Fragments

Fragments render standalone partial HTML. In `route.go`, fragment segments map
to `<segment>` browser routes:

```go
var Route = goldr.RouteDef{
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/table", table),
	},
}
```

```text
goldr.FragmentRoute("/table", table) in app/routes/users/route.go -> /users/table
```

Fragments use route params from their directory prefix:

```text
goldr.FragmentRoute("/row", row) in app/routes/users/by_id/route.go -> /users/{id}/row
```

Fragments render for `GET` and `HEAD`. They are not layout-wrapped.
Fragment responses created with `goldr.NewFragment` default to
`Cache-Control: no-store`. Set `Cache-Control` with `WithHeader` or `AddHeader`
when a fragment is intentionally cacheable.

Use an index fragment when an HTMX partial owns the route directory path itself:

```go
var Route = goldr.RouteDef{
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/", statusOptions),
	},
}
```

```text
goldr.FragmentRoute("/", statusOptions) in app/routes/users/status_options/route.go -> /users/status-options
```

Index fragments are still fragments. They are not layout-wrapped, they must not
be declared beside `Page` in the same route directory, and they may share the
path with index actions when the HTTP methods differ:

```text
GET  /users/status-options -> goldr.FragmentRoute("/", statusOptions)
HEAD /users/status-options -> goldr.FragmentRoute("/", statusOptions)
POST /users/status-options -> goldr.Action(http.MethodPost, "/", postIndex)
```

Kit routes use the same shape. When a Kit page already owns the parent path,
put the index fragment in a child route directory:

```go
var Route = goldr.KitRouteDef[reports.Kit]{
	New: newReportKit,
	Fragments: goldr.KitFragments[reports.Kit]{
		goldr.KitFragmentRoute("/", reports.Kit.StatusOptions),
	},
}
```

Use `goldr.NewFragment` for normal fragment HTML:

```go
func FragTable(r *http.Request) goldr.FragmentRouteResponse {
	return goldr.NewFragment(FragTableView(loadRows(r))).
		WithHeader("Hx-Trigger", "table-loaded")
}
```

Fragments may also return `goldr.Redirect`, `goldr.Text`, and
`goldr.RouteError`. Returning `goldr.Page` from a fragment route is an invalid
route-response contract because fragments do not render through layouts.

## Actions

Actions return `goldr.RouteResponse` by default. In `route.go`, action
declarations name the HTTP method and route segment:

```go
var Route = goldr.RouteDef{
	Actions: goldr.Actions{
		goldr.Action(http.MethodPost, "/create", postCreate),
		goldr.Action(http.MethodPost, "/", postIndex),
	},
}
```

`Get<Name>` is not an action route. Pages and fragments own generated `GET`
and `HEAD` behavior.

Use `"/"` to map an action to the current route directory path:

```text
goldr.Action(http.MethodPost, "/", postIndex) in app/routes/users/route.go -> POST /users
```

The action path maps to `"/"` or one route-local child segment. Child segments
must use lowercase ASCII letters, digits, underscores, or hyphens, and must
start with a lowercase ASCII letter. Underscores are normalized to hyphens in
browser paths:

```text
goldr.Action(http.MethodPost, "/create", postCreate) in app/routes/users/route.go -> POST /users/create
goldr.Action(http.MethodPost, "/save-preview", postSavePreview) in app/routes/users/route.go -> POST /users/save-preview
goldr.Action(http.MethodPatch, "/profile", patchProfile) in app/routes/users/by_id/route.go -> PATCH /users/{id}/profile
```

Action handlers may return pages, fragments, redirects, text, server errors,
or no-content responses. Goldr writes the response with the same writer used
for pages and fragments.

Use `goldr.NewFragment` for fragment-style rendered action responses:

```go
return goldr.NewFragment(UserForm(view)).
	WithStatus(http.StatusUnprocessableEntity).
	WithHeader(hx.HeaderRetarget, "#user-form").
	WithHeader(hx.HeaderReswap, "outerHTML")
```

The handler owns request parsing and validation state. If a rendered HTMX
response uses a non-2xx status such as `422`, configure app-owned HTMX
response handling as described in [HTMX](htmx.md).

Return a page when an action needs to render through the matched layout stack:

```go
return goldr.NewPage(CreatedView(key), goldr.PageMetadata{Title: "Created"}).
	WithStatus(http.StatusCreated).
	WithHeader("Cache-Control", "no-store")
```

Use `goldr.NoContent{}` for header-only action responses. It defaults to
`204 No Content` and also accepts `205 Reset Content` and `304 Not Modified`.

Use `goldr.HTTPAction` when an action needs direct `http.ResponseWriter`
control, such as streaming, installing `http.MaxBytesReader`, or calling an
API that requires the writer. Use `"/"` to map the raw HTTP action to the
route directory path.

## URL Helpers

goldr generates app-specific URL helpers in:

```text
app/urls/goldr_gen.go
```

Import the generated package from templates or handlers:

```go
import "myapp/app/urls"
```

Helpers are route-shaped namespaces ending in `.Path()`:

```go
urls.Root.Path()
urls.Users.Path()
urls.Users.Create.Path()
urls.Users.Table.Path()
urls.Users.ByID.Bind(id).Path()
urls.Users.ByID.Bind(id).Profile.Path()
```

Pages, fragments, and actions contribute helper paths. Same-path routes with
different HTTP methods share one helper. The method stays visible at the call
site:

```templ
<a href={ urls.Users.ByID.Bind(contact.ID).Path() }>{ contact.Name }</a>
<button hx-get={ urls.Users.Table.Path() } hx-target="#users-table-slot" hx-swap="innerHTML">
<select hx-get={ urls.Users.StatusOptions.Path() } hx-target="#status-options" hx-swap="innerHTML">
<form method="post" hx-post={ urls.Users.Create.Path() }>
```

Dynamic params are bound at the dynamic route node. Helpers escape each dynamic
segment with `url.PathEscape` when `.Bind(value)` is called:

```go
urls.Users.ByID.Bind("a/b").Path() // /users/a%2Fb
```

When the current request already matched a route with that path value, use
`goldr.BindFromRequest` to bind one dynamic node from the request:

```go
userURL, ok := goldr.BindFromRequest(r, urls.Users.ByID)
if !ok {
	// The current request does not carry id.
}
```

`goldr.BindFromRequest` uses the generated node's route params and binds the
last param with `r.PathValue("<param>")`. It is a checked shortcut for one
dynamic node, not a whole-route binder. Nested dynamic routes still bind one
node at a time:

```go
orgURL, ok := goldr.BindFromRequest(r, urls.Orgs.ByOrgID)
userURL, ok := goldr.BindFromRequest(r, orgURL.Users.ByUserID)
```

When the generated handler is mounted below a URL prefix, bind the generated
helpers once instead of writing route-specific string helpers:

```go
appURLs := urls.WithBasePath("/webapp")
appURLs.Users.ByID.Bind("42").Path() // /webapp/users/42
```

`WithBasePath` returns `urls.MountedRoutes`, so applications can pass the
mounted helper set into local functions or templates when that keeps URL
construction explicit. It normalizes a missing leading slash and trims trailing
slashes. `""` and `"/"` mean no mount prefix. The generated handler still
receives the unmounted path after the application's mux or middleware strips
the prefix.

Referenced `app/mounts` Kit subtrees also get mount-relative helpers in their
own package, such as `app/mounts/reports/goldr_gen.go`. A mount owner should
bind them from the final live helper:

```go
reportURLs := reports.NewGoldrMountURLs(urls.Admin.Reports)
```

Those helpers are for shared mounted code to link within its own subtree. They
include every route declaration from the mounted source subtree, including
children selected by only some live owners. They do not replace the final route
inventory in `app/urls`; an excluded child is absent from that owner's app URL
helper surface and dispatch. Render owner-specific mounted links only when
app-owned state says the current owner selected that child.
If the owner selected a child without `/`, the mount-base helper remains in
`app/urls` for `NewGoldrMountURLs`, but the root URL still does not dispatch.

Generated dispatch matches escaped request paths and exposes decoded values
through `r.PathValue`.

Static assets are application-owned and are not included in URL helpers.
Route declaration names, titles, and labels do not change generated helper
names. Helper namespaces are derived from the route path.

## Generated Handler

Generated route dispatch provides:

```go
func Handler() http.Handler
```

It renders generated page and fragment routes for `GET` and `HEAD`, and
dispatches generated action routes for `POST`, `PUT`, `PATCH`, and `DELETE`.

Pages, fragments, and actions may share a path when their methods differ:

```text
GET  /users -> Page
HEAD /users -> Page
POST /users -> PostIndex
```

For matched paths with unsupported methods, generated dispatch returns `405`
and sets `Allow` to the supported methods for that path.

## Route-Tree Middleware

`middleware.go` defines ordinary `net/http` endpoint middleware for matched
pages, actions, and fragments in its route directory and child route
directories.

```text
app/routes/middleware.go                  -> / and below
app/routes/main/admin/middleware.go       -> /main/admin and below
app/routes/main/admin/tenants/middleware.go -> /main/admin/tenants and below
```

Each middleware file must provide:

```go
func Middleware(next http.Handler) http.Handler
```

Use the exact `http.Handler` spelling with an unaliased `net/http` import.

Goldr discovers middleware by route directory and composes it into generated
endpoint dispatch. Runtime execution is root-to-leaf:

```text
app/routes/Middleware
app/routes/main/Middleware
app/routes/main/admin/Middleware
endpoint
```

Route-tree middleware wraps matched route endpoints:

- pages
- actions
- fragments

Layouts are not standalone middleware targets. Page layout rendering happens
inside the already wrapped endpoint request. Page responses returned from
actions are rendered inside the action middleware request.

Middleware inheritance follows the route package tree, not runtime URL prefix
matching. A middleware in `app/routes/users/create/middleware.go` does not wrap
`postCreate` from `app/routes/users/route.go` just because that action maps
to `/users/create`.

Goldr does not own CSRF, auth, roles, rate limits, sessions, or adapter policy.
Middleware remains application code. Use mux-level middleware for concerns that
must also run on generated 404 and 405 responses. Common route-tree patterns
are:

- `app/routes/middleware.go` issuing CSRF tokens for a cookie/session HTML app
- `app/routes/main/admin/middleware.go` authenticating, checking an admin role,
  and attaching a principal to request context for `/main/admin/**`

Generated 404 and 405 responses do not run route-tree middleware.

## Custom Error Responses

Generated route packages expose optional error hooks:

```go
type ErrorHandlers struct {
	RouteNotFound         func(*http.Request) goldr.RouteResponse
	RouteMethodNotAllowed func(*http.Request) goldr.RouteResponse
	RouteError            func(*http.Request, error) goldr.RouteResponse
}
```

Hooks return normal `goldr.RouteResponse` values. See
[Error Handling](error-handling.md) for wiring, status policy, layout behavior,
HTMX fragments, and app-owned error surfaces.

## Template Inspection

Generated handlers can emit development-only render-unit markers for pages,
layouts, and fragments. The same markers can also power a visible browser
overlay for local debugging.

See [Template Inspection](template-inspection.md) for comments mode, overlay
mode, embedded fragment wrappers, and app-owned env-var wiring.

## Valid Names

Valid route directories are lowercase Go-safe names. Static directory
underscores become hyphens in browser URLs:

```text
users/       -> /users
admin_v1/    -> /admin-v1
blog_posts/  -> /blog-posts
by_id/       -> {id}
by_user_id/  -> {user_id}
```

Invalid names include:

```text
Users/
blog-posts/
by_/
.hidden/
_private/
testdata/
```

Non-convention Go files such as `helpers.go` and `post_save.go` are ignored by
the scanner. `route.go` declares route endpoints. `.templ` files, tests,
generated templ files, and ordinary helper files do not declare routes.

Go test files and templ-generated `*_templ.go` files are ignored by the
scanner.
