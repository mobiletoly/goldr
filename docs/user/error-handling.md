# Error Handling

Goldr page handlers return `goldr.PageRouteResponse`, fragment handlers return
`goldr.FragmentRouteResponse`, and action handlers return broad
`goldr.RouteResponse`. Generated route dispatch uses the broad response model
for custom error hooks.

## Route Handler Errors

Use explicit status responses when the route owns the response shape:

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

Use `goldr.RouteError` when a matched route should delegate error
classification and response shape to the generated route error hook:

```go
func page(r *http.Request) goldr.PageRouteResponse {
	product, err := loadProduct(r.PathValue("id"))
	if err != nil {
		return goldr.RouteError{Err: err}
	}

	return goldr.NewPage(
		ProductView(product),
		goldr.PageMetadata{Title: product.Name},
	)
}
```

`goldr.RouteError{Err: nil}` is invalid. Nil components and invalid route
responses also flow to route error handling.

## Generated Error Hooks

Use `HandlerWithOptions` when generated route dispatch should render custom
error responses:

```go
mux.Handle("/", routes.HandlerWithOptions(routes.HandlerOptions{
	ErrorHandlers: routes.ErrorHandlers{
		RouteNotFound:         routes.RouteNotFound,
		RouteMethodNotAllowed: routes.RouteMethodNotAllowed,
		RouteError:            routes.RouteError,
	},
}))
```

Each hook is optional:

```go
type ErrorHandlers struct {
	RouteNotFound         func(*http.Request) goldr.RouteResponse
	RouteMethodNotAllowed func(*http.Request) goldr.RouteResponse
	RouteError            func(*http.Request, error) goldr.RouteResponse
}
```

Nil hooks keep Goldr defaults:

- unmatched generated routes return `404`
- matched generated paths with unsupported methods return `405`
- delegated route errors, nil components, invalid route responses, and render
  failures return `500`

## Router Errors

`RouteNotFound` handles generated router misses. It is not used for business
not-found errors from matched routes:

```go
func RouteNotFound(r *http.Request) goldr.RouteResponse {
	return goldr.NewPage(
		RouteNotFoundView(r.URL.EscapedPath()),
		goldr.PageMetadata{Title: "Page not found"},
	).WithStatus(http.StatusNotFound)
}
```

`RouteMethodNotAllowed` handles generated router method mismatches. Generated
dispatch sets the `Allow` header before calling the hook:

```go
func RouteMethodNotAllowed(r *http.Request) goldr.RouteResponse {
	return goldr.NewPage(
		RouteMethodNotAllowedView(),
		goldr.PageMetadata{Title: "Method not allowed"},
	).WithStatus(http.StatusMethodNotAllowed)
}
```

Full 404 and 405 pages use the root layout when available.

## Matched Route Errors

`RouteError` handles errors delegated by matched page, fragment, and action
routes. The app owns classification, status, public message, logging, and
response shape:

```go
func RouteError(r *http.Request, err error) goldr.RouteResponse {
	status, message := classifyRouteError(err)
	if hx.IsRequest(r) {
		return goldr.NewFragment(ErrorToast(message)).
			WithStatus(status).
			WithHeader(hx.HeaderRetarget, "#toast")
	}

	return goldr.NewPage(
		ErrorPage(message),
		goldr.PageMetadata{Title: http.StatusText(status)},
	).WithStatus(status)
}
```

Matched route error pages use the matched route layout stack. HTMX requests can
return fragments from the same hook.

Goldr does not coerce custom hook statuses. Fragment, page, text, redirect, and
no-content responses are written as returned by the hook.

## App-Owned Error Surfaces

Generated error hooks apply only to generated route dispatch. Direct writer
actions, static asset handlers, SSE endpoints, streaming handlers, recovery
middleware, logging, and panic policy remain application-owned `net/http`
behavior.

If a custom error hook returns an invalid response, returns `goldr.RouteError`,
or fails while rendering, generated dispatch writes a plain `500` and does not
call custom error hooks recursively.
