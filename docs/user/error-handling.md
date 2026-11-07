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

An optional `HandlerOptions.AdditionalPageSource` runs before final not-found
handling on an ordinary router miss. It is not an error hook. A declined source
reaches `RouteNotFound` once. A handled `goldr.RouteError`, nil response,
invalid response, or rendering failure reaches `RouteError`. Additional-page
error pages use only the live `app/routes` root layout when present, excluding
mounted-root layouts. A handled explicit 404 remains terminal and does not call
`RouteNotFound`.

Matched routes retain ownership: a matched handler's 404 or error and a matched
path's 405 never invoke the source. Explicit invalid-path rejection also skips
the source. If a custom `RouteError` response itself fails, Goldr writes the
existing generic 500 without retrying the source or calling another hook.

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

Additional-source success pages use the eligible static layout ancestry
selected from the original escaped request path. A decline runs this final 404
inside the selected static middleware chain while preserving the existing root
404 renderer.

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

Generated error hooks apply only to generated route dispatch. Applications own
direct writer actions, static asset handlers, SSE endpoints, streaming handlers,
logging, and panic policy. They can explicitly install Goldr's whole-handler
recovery boundary for the shared HTTP mechanics described below.

If a custom error hook returns an invalid response, returns `goldr.RouteError`,
or fails while rendering, generated dispatch writes a plain `500` and does not
call custom error hooks recursively.

## Whole-Handler Panic Recovery

Install `goldr.Recover` around the application handler that needs protection:

```go
handler := goldr.Recover(mux, func(r *http.Request, info goldr.PanicInfo) {
	logger.ErrorContext(r.Context(), "Request handler panicked",
		"frames", info.Frames, "frames_truncated", info.FramesTruncated,
		"committed", info.Committed, "status", info.Status,
		"hijacked", info.Hijacked)
})
```

Both the handler and reporter are required. Recovery is opt-in and does not
change generated error hooks or ordinary responses. A wrapper around the final
application mux covers generated routes, direct writers, static handlers,
middleware inside that boundary, and SSE streams. It catches unexpected panics
only on the serving goroutine, not detached goroutines or fatal runtime errors.

The synchronous reporter receives no panic value or type. `PanicInfo.Frames`
contains function symbols, file basenames, and positive source lines, with at
most 16 frames and 4 KiB of JSON-encoded frame-array data. Symbols and basenames
are limited to 256 and 128 printable ASCII characters respectively. Truncation
and sanitization set `FramesTruncated`. Reports exclude absolute source paths,
addresses, arguments, local values, and recovery/runtime-panic plumbing frames.
Applications own logging, correlation, access and retention policy; do not log
the callback request indiscriminately.

The callback receives the request passed to the wrapper. A downstream
`r.WithContext` copy does not change that original request. Establish shared
request correlation outside the wrapper when the callback needs it. A reporter
panic is contained without recursive reporting or raw error output, but can
prevent receipt emission. Keep reporters short and local.

Before final response commitment, recovery clears pending headers, including
cookies, redirects and HTMX instructions, and writes a plain-text 500 with body
`Internal server error.` followed by a newline. HEAD has no body. The fallback
sets `Content-Type: text/plain; charset=utf-8`, `X-Content-Type-Options: nosniff`,
`Cache-Control: no-store`, and `Referrer-Policy: no-referrer`.

After final headers, a body write, or a successful flush, recovery reports and
aborts the exchange through `http.ErrAbortHandler`. It never appends an error
page or replaces the committed status. Informational 1xx responses except 101
do not commit a final response; 101 does. After successful hijacking, an
unexpected panic closes the connection without writing fallback bytes. An exact
`http.ErrAbortHandler` panic passes through without reporting or fallback.
An attempted flush that fails with an error other than `http.ErrNotSupported`
is conservatively treated as committed: its implicit status may already have
been written before the transport failure.

The writer forwards output immediately. Supported flush, hijack, push and
`io.ReaderFrom` capabilities remain available; unsupported flushing still makes
`sse.Start` decline. `http.ResponseController` deadline and full-duplex operations
delegate through `Unwrap`. Recovery does not add custom writer interfaces or
deprecated `CloseNotifier` support. HTTP-server logs outside the handler, such
as TLS and accept failures, remain the application's `http.Server.ErrorLog`
responsibility.
