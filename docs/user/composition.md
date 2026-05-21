# Composition

goldr generated routes are ordinary `http.Handler` values.

Applications own the HTTP server, mux, middleware, static assets, cache
headers, auth, sessions, logging, recovery, and non-route handlers.

## Mux Shape

Use the standard library mux. Register more specific application handlers
before generated routes:

```go
mux := http.NewServeMux()
mux.Handle("/assets/", staticHandler())
mux.Handle("/", routes.Handler())
```

`/assets/` is application-owned. It is not part of generated route dispatch and
does not appear in generated URL helpers.

## Goldr Browser Helpers

Goldr browser helpers are also mounted explicitly by the application. For named
SSE event swaps, mount the `browser` helper before generated routes:

```go
import (
	"net/http"

	"myapp/app/routes"

	"github.com/mobiletoly/goldr/browser"
)

mux := http.NewServeMux()
mux.Handle("/goldr/", http.StripPrefix("/goldr/", browser.Handler()))
mux.Handle("/", routes.Handler())
```

Then load the helper from the layout that uses it:

```html
<script src="/goldr/goldr-sse-event.js" defer></script>
```

The helper URL is stable and served with `Cache-Control: no-cache` plus ETag
revalidation. Do not apply immutable asset cache headers to this stable path.

## Middleware

Wrap generated routes like any other `http.Handler`:

```go
handler := appHeaders(routes.HandlerWithErrors(routes.ErrorHandlers{
	NotFound: routes.NotFound,
}))

mux.Handle("/", handler)
```

Middleware can handle authentication, sessions, CSRF, logging, recovery,
security headers, request IDs, or other application concerns.

goldr does not provide a framework middleware stack. Keeping middleware as
plain `net/http` keeps behavior explicit and lets applications use ordinary Go
libraries.

Goldr's `csrf` package provides a small signed-cookie token guard. Applications
still choose where to mount it:

```go
guard, err := csrf.New(csrf.Config{
	Secret: []byte(os.Getenv("CSRF_SECRET")),
})
if err != nil {
	return err
}

mux.Handle("/", guard.Middleware(routes.Handler()))
```

Read [CSRF](csrf.md) for form and HTMX validation patterns.

## Application Dependencies

When several route packages need stable app dependencies such as stores, auth
managers, configuration, base paths, redirect policy, or CSRF guards, keep the
dependency shape app-owned and typed. A small package such as `app/deps` can
attach one `*deps.Dependencies` value at the generated-route boundary:

```go
appDeps := &deps.Dependencies{
	Auth:     authManager,
	CSRF:     csrfGuard,
	BasePath: cfg.BasePath,
}

routesHandler := routes.HandlerWithErrors(routes.ErrorHandlers{
	NotFound: routes.NotFound,
})

mux.Handle("/", deps.Middleware(appDeps, routesHandler))
```

Route packages can then read the typed value with the app-owned helper while
still passing `r.Context()` into stores and services for per-request work.

Read [Application Dependencies](dependencies.md) for the recommended helper
shape and the boundary between stable app dependencies and request-scoped
context values.

## Static Assets

Keep assets outside `app/routes` and serve them through an application handler:

```go
import "myapp/assets"

func staticHandler() http.Handler {
	return http.StripPrefix("/assets/", http.FileServer(http.FS(assets.FS())))
}
```

Register the asset handler before generated routes:

```go
mux.Handle("/assets/", staticHandler())
mux.Handle("/", routes.Handler())
```

Static asset errors are application-owned. Generated error hooks apply only to
generated route dispatch.

When using `go tool goldr assets dist`, templates can reference the generated
path explicitly:

```templ
<link rel="stylesheet" href={ assets.Path("app.css") }/>
```

Read [Assets](assets.md) for the full fingerprinting workflow and asset-tool
integration.

## Cache Headers

Cache policy is application-owned:

```go
func staticCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=60")
		next.ServeHTTP(w, r)
	})
}
```

Applications with fingerprinted assets can choose stronger caching policies:

```go
w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
```

Apply long immutable cache headers only to fingerprinted assets, not dynamic
pages, private fragments, or action responses.

## Custom Error Pages

Generated route dispatch supports optional error hooks:

```go
mux.Handle("/", routes.HandlerWithErrors(routes.ErrorHandlers{
	NotFound: routes.NotFound,
}))
```

Use these hooks for generated route dispatch errors such as unmatched routes or
method mismatches. Action handlers and static asset handlers own their own
errors.
