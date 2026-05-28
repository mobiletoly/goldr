# Error Handling

Goldr page handlers return `goldr.PageRouteResponse`, fragment handlers return
`goldr.FragmentRouteResponse`, and action handlers return broad
`goldr.RouteResponse`. Generated route dispatch uses the broad response model
for custom error hooks.

## Route Handler Errors

Use explicit status responses for expected request-shaped failures:

```go
func page(r *http.Request) goldr.PageRouteResponse {
	if !allowed(r) {
		return goldr.NewPage(
			ForbiddenView(),
			goldr.PageMetadata{Title: "Forbidden"},
		).WithStatus(http.StatusForbidden)
	}

	return goldr.NewPage(PageView(), goldr.PageMetadata{Title: "Users"})
}
```

Use `goldr.ServerError` for unexpected application errors that should flow to
generated internal-server-error handling:

```go
func page(r *http.Request) goldr.PageRouteResponse {
	users, err := loadUsers(r.Context())
	if err != nil {
		return goldr.ServerError{Err: err}
	}

	return goldr.NewPage(UsersView(users), goldr.PageMetadata{Title: "Users"})
}
```

`goldr.ServerError{Err: nil}` is invalid. Nil components and invalid route
responses also flow to internal-server-error handling.

## Generated Error Hooks

Use `HandlerWithOptions` when generated route dispatch should render custom
error responses:

```go
mux.Handle("/", routes.HandlerWithOptions(routes.HandlerOptions{
	ErrorHandlers: routes.ErrorHandlers{
		NotFound:            routes.NotFound,
		MethodNotAllowed:    routes.MethodNotAllowed,
		InternalServerError: routes.InternalServerError,
	},
}))
```

Each hook is optional:

```go
type ErrorHandlers struct {
	NotFound            func(*http.Request) goldr.RouteResponse
	MethodNotAllowed    func(*http.Request) goldr.RouteResponse
	InternalServerError func(*http.Request, error) goldr.RouteResponse
}
```

Nil hooks keep Goldr defaults:

- unmatched generated routes return `404`
- matched generated paths with unsupported methods return `405`
- nil components, invalid route responses, and render failures return `500`

## Full-Page Errors

Return pages when the response should replace the whole document. Set the
status explicitly:

```go
func NotFound(r *http.Request) goldr.RouteResponse {
	return goldr.NewPage(
		NotFoundView(r.URL.EscapedPath()),
		goldr.PageMetadata{Title: "Page not found"},
	).WithStatus(http.StatusNotFound)
}

func InternalServerError(r *http.Request, err error) goldr.RouteResponse {
	return goldr.NewPage(
		ErrorPage(),
		goldr.PageMetadata{Title: "Something went wrong"},
	).WithStatus(http.StatusInternalServerError)
}
```

Goldr does not coerce custom hook statuses. This keeps HTMX and non-HTML
responses app-owned, but it means full-page error hooks should call
`.WithStatus(...)`.

Full 404 and 405 pages use the root layout when available. Full internal-error
pages use the matched route layout stack when the error comes from a matched
generated route.

## HTMX Error Fragments

Goldr passes the original request to error hooks. Apps can use `hx.IsRequest`
to choose a fragment response for HTMX requests:

```go
func InternalServerError(r *http.Request, err error) goldr.RouteResponse {
	if hx.IsRequest(r) {
		return goldr.NewFragment(ErrorToast()).
			WithHeader(hx.HeaderRetarget, "#toast")
	}

	return goldr.NewPage(
		ErrorPage(),
		goldr.PageMetadata{Title: "Something went wrong"},
	).WithStatus(http.StatusInternalServerError)
}
```

Goldr does not choose app components such as `ErrorToast`. The app decides
which component to return from the hook. Fragment, text, redirect, and
no-content responses are written as returned by the hook.

## Method Not Allowed

Generated `405` responses set the `Allow` header before calling a custom
`MethodNotAllowed` hook:

```go
func MethodNotAllowed(r *http.Request) goldr.RouteResponse {
	return goldr.NewPage(
		MethodNotAllowedView(),
		goldr.PageMetadata{Title: "Method not allowed"},
	).WithStatus(http.StatusMethodNotAllowed)
}
```

## App-Owned Error Surfaces

Generated error hooks apply only to generated route dispatch. Direct writer
actions, static asset handlers, SSE endpoints, streaming handlers, recovery
middleware, logging, and panic policy remain application-owned `net/http`
behavior.

If a custom error hook returns an invalid response, returns `goldr.ServerError`,
or fails while rendering, generated dispatch writes a plain `500` and does not
call custom error hooks recursively.
