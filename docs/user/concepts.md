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
  page.go            -> GET /
  page.templ         -> page HTML
  users/
    layout.go        -> layout logic for /users and below
    layout.templ     -> users layout HTML
    page.go          -> GET /users
    page.templ       -> users page HTML
    by_id/
      page.go        -> GET /users/{id}
      page.templ     -> user detail HTML
    frag_table.go    -> GET /users/frag-table
    frag_table.templ -> fragment HTML
    actions.go       -> POST /users/create via PostCreate
```

The route tree uses Go-safe names. Static directory underscores become hyphens
in browser URLs, and dynamic directories use `by_<param>/`, such as `by_id/`
for `{id}`.

## Render Units

Pages, layouts, and fragments are render units:

```text
page.go       page.templ
layout.go     layout.templ
frag_table.go frag_table.templ
```

The `.go` file owns request handling, data loading, and render state. The
`.templ` file owns HTML.

## Pages

A page is a route endpoint. `app/routes/users/page.go` maps to `/users`.

Page functions return `goldr.Page`:

```go
func Page(r *http.Request) goldr.Page
```

The page component renders the body. Page metadata is passed to matching
layouts.

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
app/routes/users/frag_table.go -> /users/frag-table
```

Fragment functions return a templ component:

```go
func FragTable(r *http.Request) templ.Component
```

Fragments render partial HTML and are not layout-wrapped.

## Actions

Actions are ordinary `net/http` handlers colocated with a route directory.
They live in `actions.go`:

```go
func PostCreate(w http.ResponseWriter, r *http.Request)
```

This maps to:

```text
POST /users/create
```

Actions own status codes, headers, bodies, redirects, HTMX response headers,
and form redisplay. They are not layout-wrapped.

## Generated Code

goldr generates:

```text
app/routes/goldr_gen.go
app/urls/goldr_gen.go
```

`app/routes/goldr_gen.go` provides the generated route handler:

```go
routes.Handler()
```

`app/urls/goldr_gen.go` provides route-shaped URL helpers:

```go
urls.Users.ByID(id).Path()
urls.Users.Create.Path()
urls.Users.FragTable.Path()
```

Use helpers in links, forms, and HTMX attributes when a path should track the
route tree.

## Application Ownership

goldr does not own the whole server. Applications still own:

- `net/http` server setup
- mux registration
- middleware
- auth and sessions
- CSRF
- static assets
- cache headers
- logging and recovery
- data access

Generated routes are ordinary `http.Handler` values, so they compose with
standard Go code.
