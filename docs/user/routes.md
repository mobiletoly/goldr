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

## Pages

`page.go` defines a page route for its directory.

```text
app/routes/page.go                     -> /
app/routes/users/page.go               -> /users
app/routes/settings/build_info/page.go -> /settings/build-info
app/routes/users/by_id/page.go         -> /users/{id}
```

Each page must have a matching `.templ` file and must provide:

```go
func Page(r *http.Request) goldr.RouteResponse
```

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
return goldr.ServerError{Err: err}
```

Redirects, text status responses, and errors do not render layouts. Status
responses with a templ component render through the same layout chain as normal
pages.

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

Use explicit status responses for request-shaped failures:

```go
if !validID(r.PathValue("project_id")) {
	return goldr.NewPage(BadRequestView(), goldr.PageMetadata{Title: "Bad request"}).WithStatus(http.StatusBadRequest)
}

project, err := store.Project(r.Context(), r.PathValue("project_id"))
if errors.Is(err, store.ErrNotFound) {
	return goldr.NewPage(NotFoundView(), goldr.PageMetadata{Title: "Not found"}).WithStatus(http.StatusNotFound)
}
if err != nil {
	return goldr.ServerError{Err: err}
}

return goldr.NewPage(ProjectView(project), goldr.PageMetadata{Title: project.Name})
```

Use `goldr.ServerError{Err: err}` only for unexpected application or runtime
failures that should use Goldr's internal server error handling:

```go
project, err := store.Project(r.Context(), r.PathValue("project_id"))
if err != nil {
	return goldr.ServerError{Err: err}
}
```

Generated dispatch resolves the returned route response internally. If
resolution returns an error, the page returned an invalid Goldr contract, such
as `goldr.Page{}`, `goldr.NewPage(nil, metadata)`,
`goldr.Redirect{Location: "", Status: http.StatusSeeOther}`,
`goldr.Redirect{Location: "/sign-in", Status: http.StatusNotModified}`,
`goldr.NewPage(view, metadata).WithStatus(http.StatusNoContent)`, or
`goldr.ServerError{Err: nil}`. Those validation errors are routed to internal
server error handling. `goldr.ServerError{Err: err}` is a valid route response:
its error is the application error passed to the generated internal server
error handler.

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

Fragments are not layout-wrapped. Actions are ordinary handlers and are not
automatically layout-wrapped, but an action can explicitly call
`goldr.WriteRouteResponse` to write a full page through the matched route
layout stack.

## Dynamic Routes

Dynamic route directories use `by_<param>/`.

```text
app/routes/users/by_id/page.go
```

maps to:

```text
/users/{id}
```

Nested dynamic routes work the same way:

```text
app/routes/orgs/by_org_id/users/by_user_id/page.go
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

Fragments use the `frag_` prefix and render standalone partial HTML.

```text
app/routes/users/frag_table.go -> /users/frag-table
```

Each fragment must have a matching `.templ` file and must provide:

```go
func FragTable(r *http.Request) goldr.RouteResponse
```

Fragments use route params from their directory prefix:

```text
app/routes/users/by_id/frag_row.go -> /users/{id}/frag-row
```

Fragments render for `GET` and `HEAD`. They are not layout-wrapped.

Use `goldr.NewFragment` for normal fragment HTML:

```go
func FragTable(r *http.Request) goldr.RouteResponse {
	return goldr.NewFragment(FragTableView(loadRows(r))).
		WithHeader("Hx-Trigger", "table-loaded")
}
```

Fragments may also return `goldr.Redirect`, `goldr.Text`, and
`goldr.ServerError`. Returning `goldr.Page` from a fragment route is an invalid
route-response contract because fragments do not render through layouts.

## Actions

Actions live in one `actions.go` file per route directory.

```text
app/routes/users/actions.go
app/routes/users/by_id/actions.go
```

Action functions are exported top-level `net/http` handlers with a supported
method prefix:

```go
func PostCreate(w http.ResponseWriter, r *http.Request)
func PutProfile(w http.ResponseWriter, r *http.Request)
func PatchProfile(w http.ResponseWriter, r *http.Request)
func DeleteAvatar(w http.ResponseWriter, r *http.Request)
```

Supported prefixes are:

```text
Post   -> POST
Put    -> PUT
Patch  -> PATCH
Delete -> DELETE
```

`Get<Name>` is not an action route. Pages and fragments own generated `GET`
and `HEAD` behavior.

`Index` maps to the current route directory path:

```text
PostIndex -> POST /users
```

Other suffixes map to one lowercase kebab-case child segment:

```text
PostCreate      -> POST /users/create
PostSavePreview -> POST /users/save-preview
PatchProfile    -> PATCH /users/{id}/profile
```

Action handlers are called directly. They own status codes, headers, response
bodies, redirects, HTMX response headers, and form redisplay.

Use `goldr.WriteComponent` for fragment-style rendered action responses:

```go
hx.Retarget(w, "#user-form")
hx.Reswap(w, "outerHTML")
if err := goldr.WriteComponent(w, r, http.StatusUnprocessableEntity, UserForm(form)); err != nil {
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
```

Use `goldr.WriteRouteResponse` when an action needs to return a full page
through the matched layout stack:

```go
err := goldr.WriteRouteResponse(
	w,
	r,
	goldr.NewPage(CreatedView(key), goldr.PageMetadata{Title: "Created"}).
		WithStatus(http.StatusCreated).
		WithHeader("Cache-Control", "no-store"),
)
if err != nil {
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
```

This is explicit. Actions are not automatically layout-wrapped, so fragments,
redirects, validation snippets, and custom responses stay ordinary
`net/http` handler behavior.

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
urls.Users.FragTable.Path()
urls.Users.ByID(id).Path()
urls.Users.ByID(id).Profile.Path()
```

Pages, fragments, and actions contribute helper paths. Same-path routes with
different HTTP methods share one helper. The method stays visible at the call
site:

```templ
<a href={ urls.Users.ByID(contact.ID).Path() }>{ contact.Name }</a>
<button hx-get={ urls.Users.FragTable.Path() } hx-target="#users-table">
<form method="post" hx-post={ urls.Users.Create.Path() }>
```

Dynamic params are explicit string arguments. Helpers escape each dynamic
segment with `url.PathEscape`:

```go
urls.Users.ByID("a/b").Path() // /users/a%2Fb
```

Generated dispatch matches escaped request paths and exposes decoded values
through `r.PathValue`.

Static assets are application-owned and are not included in URL helpers.

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

## Custom Error Responses

Use `HandlerWithErrors` when generated route dispatch should render custom
error responses:

```go
mux.Handle("/", routes.HandlerWithErrors(routes.ErrorHandlers{
	NotFound: routes.NotFound,
}))
```

Each hook is optional:

```go
type ErrorHandlers struct {
	NotFound            http.HandlerFunc
	MethodNotAllowed    http.HandlerFunc
	InternalServerError func(http.ResponseWriter, *http.Request, error)
}
```

Nil hooks keep goldr defaults:

- unmatched generated routes return `404`
- matched generated paths with unsupported methods return `405`
- nil components and templ render failures return `500`

Generated `405` responses set the `Allow` header before calling a custom
`MethodNotAllowed` hook.

Custom internal-server-error hooks receive `goldr.ErrNilComponent` for nil
render units or the underlying templ render error.

Action error responses and static asset error responses are application-owned.

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
the scanner. Only `actions.go` has action-routing meaning.

Go test files and templ-generated `*_templ.go` files are ignored by the
scanner.
