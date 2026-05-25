# Concepts

goldr is the Go Layout-Driven Router.

The filesystem is the route map. Go files handle request-facing behavior and
templ files render HTML. Layouts compose by route directory, and generated
code turns the route tree into ordinary `net/http` handlers.

## Route Tree

goldr applications use:

```text
app/routes/
```

A small tree maps directly to URLs:

```text
app/routes/
  layout.go          -> layout logic for / and below
  layout.templ       -> layout HTML
  route.go           -> GET /
  page.templ         -> page HTML
  users/
    layout.go        -> layout logic for /users and below
    layout.templ     -> users layout HTML
    route.go         -> GET /users, GET /users/table, POST /users/create
    page.templ       -> users page HTML
    by_id/
      route.go       -> GET /users/{id}
      page.templ     -> user detail HTML
    frag_table.templ -> fragment HTML
```

The route tree uses Go-safe names. Static directory underscores become hyphens
in browser URLs, and dynamic directories use `by_<param>/`, such as `by_id/`
for `{id}`.

## Render Units

Pages, layouts, and fragments keep request logic beside templates:

```text
route.go          page.templ optional for pages
layout.go         layout.templ required
route.go          fragment templates when the handler uses them
```

The `.go` file owns request handling, data loading, and render state. The
`.templ` file owns HTML. Pages that only redirect, return text, or return
server errors can omit `page.templ`.

## Pages

A page is a route endpoint. `app/routes/users/route.go` maps to `/users` when
it declares a page:

Page functions return `goldr.RouteResponse`:

```go
func page(r *http.Request) goldr.RouteResponse
```

The page component renders the body. Page metadata is passed to matching
layouts.

Normal pages use `goldr.NewPage`. Page handlers may also return `goldr.Redirect`,
`goldr.Text`, or `goldr.ServerError` when the route needs to respond before
normal rendering. Use `page.WithStatus(status)` for rendered page responses
such as `403` or `404`.

## Page Metadata

Page metadata is a small page-owned value:

```go
type PageMetadata struct {
	Title       string
	Description string
}
```

goldr passes this value to every matching layout through
`goldr.LayoutContext`. goldr does not render head tags by itself, choose active
navigation, or define canonical URL policy. Layouts decide how to use title and
description, and applications own any extra shell behavior.

## Layouts

A layout wraps pages in its route directory and below.

```text
app/routes/layout.go       -> wraps / and pages below /
app/routes/users/layout.go -> wraps /users and pages below /users
```

Layouts do not wrap fragments or actions.

Layouts receive the rendered child component and page metadata:

```go
func Layout(r *http.Request, ctx goldr.LayoutContext) templ.Component
```

For nested pages, generated runtime wiring applies matching layouts from the
page directory back to the root, so the root layout is outermost.

## Fragments

Fragments are standalone partials for HTMX swaps.

```text
goldr.FuncFragment("table", table) in app/routes/users/route.go -> /users/table
goldr.FuncFragmentIndex(statusOptions) in app/routes/users/status_options/route.go -> /users/status-options
```

Fragment functions return `goldr.RouteResponse`:

```go
func FragTable(r *http.Request) goldr.RouteResponse
```

Use `goldr.NewFragment` for normal fragment HTML. Fragments may also return
redirect, text, and server-error route responses. They are not layout-wrapped.
An index fragment uses the route directory path itself and cannot coexist with
a page in the same directory.
Fragment responses default to `Cache-Control: no-store`; set `Cache-Control`
explicitly when a fragment should be cacheable.

## Actions

Actions are mutation endpoints colocated with a route directory. They are
declared in `route.go`:

```go
var Route = goldr.RouteDef{
	Actions: goldr.FuncActions{
		goldr.FuncPost("create", postCreate),
	},
}

func postCreate(r *http.Request) goldr.RouteResponse
```

This maps to:

```text
POST /users/create
```

Actions may return pages, fragments, redirects, text, server errors, or
`goldr.NoContent{}`. Page responses from actions are written through the
matched layout stack.

## Generated Code

goldr generates:

```text
app/routes/goldr_gen.go
app/routes/**/goldr_gen.go when route packages need generated helpers
app/internal/goldrinspect/goldr_gen.go
app/urls/goldr_gen.go
app/mounts/<mount>/goldr_gen.go for referenced Kit mount subtrees
```

`app/routes/goldr_gen.go` provides the generated route handler:

```go
routes.Handler()
```

`app/urls/goldr_gen.go` provides route-shaped URL helpers:

```go
urls.Users.ByID(id).Path()
urls.Users.Create.Path()
urls.Users.Table.Path()
```

Use helpers in links, forms, and HTMX attributes when a path should track the
route tree.

Referenced Kit mount subtrees also get mount-relative helpers in their own
`app/mounts/<mount>/goldr_gen.go` file. Bind those helpers from the live route
helper and keep owner-only child links in app-owned kit or page data.

## Application Ownership

goldr does not own the whole server. Applications still own:

- `net/http` server setup
- mux registration
- middleware
- auth and sessions
- CSRF policy and secrets
- static assets
- cache headers
- logging and recovery
- data access

Generated routes are ordinary `http.Handler` values, so they compose with
standard Go code.
