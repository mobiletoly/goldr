# Goldr Routes Reference For App Agents

Use this reference when editing pages, layouts, dynamic route directories, URL
helpers, generated handlers, or route-level error handling in a Goldr app.

## Filesystem Route Tree

Goldr routes live under `app/routes`.

```text
app/routes/
  layout.go
  layout.templ
  page.go
  page.templ
  users/
    layout.go
    layout.templ
    page.go
    page.templ
    frag_table.go
    frag_table.templ
    actions.go
    by_id/
      page.go
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

`page.go` defines a page route for its directory. `page.templ` is optional and
is used for page-owned HTML written in templ.

```go
func Page(r *http.Request) goldr.RouteResponse {
	return goldr.NewPage(
		PageView(),
		goldr.PageMetadata{
			Title:       "Users",
			Description: "Manage users.",
		},
	)
}
```

Pages can return:

- `goldr.NewPage(component, metadata)`
- `goldr.Redirect{Location: "...", Status: http.StatusSeeOther}`
- `goldr.Text{Status: http.StatusForbidden, Body: "forbidden"}`
- `goldr.ServerError{Err: err}`

Use explicit status pages for request-shaped failures and `goldr.ServerError`
for unexpected application errors.

## Layouts

`layout.go` defines a layout for pages in that directory and below. It has a
matching `layout.templ`.

```go
func Layout(r *http.Request, ctx goldr.LayoutContext) templ.Component {
	return LayoutView(ctx.Metadata, ctx.Child)
}
```

`ctx.Child` is the child page or nested layout component. `ctx.Metadata` comes
from the matched page. Fragments are not layout-wrapped. Actions are ordinary
handlers unless they explicitly call `goldr.WriteRouteResponse`.

## Dynamic Routes

Use `by_<param>/` directories.

```text
app/routes/users/by_id/page.go -> /users/{id}
```

Read decoded params with:

```go
id := r.PathValue("id")
```

Static routes win over dynamic routes when both could match.

## Actions

Actions live in one `actions.go` file per route directory. They are ordinary
exported `net/http` handlers.

```go
func PostCreate(w http.ResponseWriter, r *http.Request)
func PutProfile(w http.ResponseWriter, r *http.Request)
func PatchProfile(w http.ResponseWriter, r *http.Request)
func DeleteAvatar(w http.ResponseWriter, r *http.Request)
```

Supported method prefixes are `Post`, `Put`, `Patch`, and `Delete`. `Index`
maps to the current directory path:

```text
PostIndex -> POST /users
PostCreate -> POST /users/create
PostSavePreview -> POST /users/save-preview
```

Actions own status codes, headers, redirects, HTMX response headers, and body
writing.

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
urls.Users.FragTable.Path()
urls.Users.ByID(id).Path()
```

Dynamic params are explicit string arguments and are path-escaped by helpers.
Use helpers instead of hard-coded internal route paths when helpers exist.

## Generated Handler

Generated route packages expose:

```go
func Handler() http.Handler
func HandlerWithOptions(options HandlerOptions) http.Handler
```

`Handler()` is the normal generated route handler. `HandlerWithOptions` is for
custom error handlers and template inspection.

Error hooks are optional:

```go
type ErrorHandlers struct {
	NotFound            http.HandlerFunc
	MethodNotAllowed    http.HandlerFunc
	InternalServerError func(http.ResponseWriter, *http.Request, error)
}
```

Nil hooks keep Goldr defaults. Action error responses and static asset error
responses stay application-owned.
