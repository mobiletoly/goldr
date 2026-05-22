# Application Dependencies

goldr generated routes are ordinary `http.Handler` values, and route functions
keep simple request-facing signatures:

```go
func Page(r *http.Request) goldr.RouteResponse
func Layout(r *http.Request, ctx goldr.LayoutContext) templ.Component
func FragTable(r *http.Request) goldr.RouteResponse
func PostCreate(w http.ResponseWriter, r *http.Request)
```

Applications still need stable dependencies such as stores, auth managers,
configuration, redirect policy, base paths, feature flags, and CSRF guards.
Use one app-owned typed dependency package instead of scattered context helpers
or package globals.

goldr does not generate this package. The application owns its fields, helper
names, tests, and lifecycle.

## Recommended Shape

Create a small package outside `app/routes`, for example:

```text
app/deps/
  deps.go
```

Keep the package outside `app/routes` so route packages can import it without
creating route-generation ownership confusion.

Example:

```go
package deps

import (
	"context"
	"net/http"

	"myapp/internal/auth"

	"github.com/mobiletoly/goldr/csrf"
)

type Dependencies struct {
	Auth     *auth.Manager
	CSRF     *csrf.Guard
	BasePath string
}

type contextKey struct{}

func Middleware(value *Dependencies, next http.Handler) http.Handler {
	if value == nil {
		panic("deps: nil Dependencies")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, WithRequest(r, value))
	})
}

func WithRequest(r *http.Request, value *Dependencies) *http.Request {
	if value == nil {
		panic("deps: nil Dependencies")
	}
	ctx := context.WithValue(r.Context(), contextKey{}, value)
	return r.WithContext(ctx)
}

func From(r *http.Request) *Dependencies {
	value, ok := r.Context().Value(contextKey{}).(*Dependencies)
	if !ok || value == nil {
		panic("deps: missing Dependencies; wrap generated routes with deps.Middleware")
	}
	return value
}
```

The helper panics for missing dependencies because that is an application
setup error. Tests should wrap requests with `WithRequest` when they exercise
route functions directly.

The full-feature example uses this shape in `examples/full_feature/app/deps`.

## Server Setup

Construct dependencies at the application boundary and wrap generated routes:

```go
appDeps := &deps.Dependencies{
	Auth:     authManager,
	CSRF:     csrfGuard,
	BasePath: cfg.BasePath,
}

routesHandler := routes.HandlerWithOptions(routes.HandlerOptions{
	ErrorHandlers: routes.ErrorHandlers{
		NotFound: routes.NotFound,
	},
})

mux := http.NewServeMux()
mux.Handle("/assets/", staticHandler())
mux.Handle("/", deps.Middleware(appDeps, routesHandler))
```

Middleware ordering remains application-owned. Mount auth, session, CSRF,
logging, recovery, and other middleware in the order your app needs.

## Route Usage

Read stable dependencies through the app-owned helper:

```go
func Page(r *http.Request) goldr.RouteResponse {
	appDeps := deps.From(r)

	return goldr.NewPage(
		PageView(appDeps.CSRF.Token(r), appDeps.BasePath),
		goldr.PageMetadata{Title: "Sign in"},
	)
}
```

Pass `r.Context()` to lower-level operations that execute work for the current
request:

```go
func PostIndex(w http.ResponseWriter, r *http.Request) {
	appDeps := deps.From(r)

	user, err := appDeps.Auth.SignIn(r.Context(), r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	_ = user
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
```

## Boundary

Use the dependency package for stable app environment:

- stores
- auth managers
- configuration
- base paths
- redirect policy
- CSRF guards
- feature flag clients
- mailers
- queues

Use request context for request-scoped facts:

- cancellation
- deadlines
- request IDs
- current user or session selected for this request
- tenant selected for this request
- auth claims derived from this request

Do not turn the dependency struct into a catch-all global object. Keep fields
boring, explicit, and application-owned. If a dependency belongs to only one
small helper package, it can stay in that package instead of being added to
the shared dependency struct.
