# Routes

This page is the route reference. For the mental model, read
[Concepts](concepts.md) first.

goldr applications use a filesystem route tree rooted at:

```text
app/routes/
```

Route names are Go-native. Do not use JavaScript-style filesystem route syntax.

## Pages

`page.go` defines a page route for its directory.

```text
app/routes/page.go              -> /
app/routes/users/page.go        -> /users
app/routes/users/by_id/page.go  -> /users/{id}
```

Each page must have a matching `.templ` file and must provide:

```go
func Page(r *http.Request) goldr.Page
```

`goldr.Page` contains the component and optional page metadata:

```go
return goldr.Page{
	Component: PageView(),
	Metadata: goldr.PageMetadata{
		Title:       "Users",
		Description: "Manage users.",
	},
}
```

Supported metadata fields are `Title` and `Description`. goldr passes metadata
to layouts. Layouts decide how to render it.

goldr does not infer page titles, render canonical links, or choose active
navigation entries. Those are application layout decisions. Use request data,
generated URL helpers, or app-owned state when a layout needs them.

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

Fragments and actions are not layout-wrapped.

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
app/routes/users/frag_table.go -> /users/frag_table
```

Each fragment must have a matching `.templ` file and must provide:

```go
func FragTable(r *http.Request) templ.Component
```

Fragments use route params from their directory prefix:

```text
app/routes/users/by_id/frag_row.go -> /users/{id}/frag_row
```

Fragments render for `GET` and `HEAD`. They are not layout-wrapped.

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
	InternalServerError http.HandlerFunc
}
```

Nil hooks keep goldr defaults:

- unmatched generated routes return `404`
- matched generated paths with unsupported methods return `405`
- nil components and templ render failures return `500`

Generated `405` responses set the `Allow` header before calling a custom
`MethodNotAllowed` hook.

Action error responses and static asset error responses are application-owned.

## Valid Names

Valid route directories are lowercase Go-safe names:

```text
users/
admin_v1/
blog_posts/
by_id/
by_user_id/
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
