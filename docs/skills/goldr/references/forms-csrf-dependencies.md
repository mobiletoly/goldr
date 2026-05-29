# Request Parsing, CSRF, And Dependencies

Use this reference when editing route actions that parse requests, redisplay
HTML after validation, handle multipart uploads, validate CSRF tokens, or wire
app dependencies.

## Request Parsing Boundary

Goldr does not own form parsing or validation state. Use Go's `net/http`
request APIs or an application-selected decoder such as `gorilla/schema`.
Keep validation types, error messages, file policy, and persistence
application-owned.

Function names are ordinary Go names. `route.go` declares the route surface:

```go
var Route = goldr.RouteDef{
	Actions: goldr.Actions{
		goldr.Action(http.MethodPost, "/create", postCreate),
	},
}
```

For a small URL-encoded form, plain `net/http` is enough:

```go
type userForm struct {
	Name      string
	NameError string
}

func postCreate(r *http.Request) goldr.RouteResponse {
	if err := r.ParseForm(); err != nil {
		return goldr.Text{Status: http.StatusBadRequest, Body: "bad request"}
	}

	form := userForm{Name: r.PostFormValue("name")}
	if form.Name == "" {
		form.NameError = "Name is required."
	}

	if form.NameError != "" {
		return goldr.NewFragment(UserForm(form)).
			WithStatus(http.StatusUnprocessableEntity).
			WithHeader(hx.HeaderRetarget, "#user-form").
			WithHeader(hx.HeaderReswap, "outerHTML")
	}

	// Application-owned persistence.
	return goldr.NoContent{}
}
```

If a rendered HTMX response uses a non-2xx status such as `422`, the app must
configure HTMX response handling or use an HTMX extension that swaps that
response. Goldr does not install a global browser policy.

For multipart forms, use the standard library:

```go
func postUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return
		}
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer file.Close()
	_ = header.Filename
}
```

`maxMemory` is the standard library memory threshold, not a hard request size
limit. Use `http.MaxBytesReader` from an explicit `HTTPAction` when the app
needs a hard limit.

HTMX multipart forms need both normal HTML encoding and HTMX encoding:

```html
<form method="post" enctype="multipart/form-data" hx-encoding="multipart/form-data">
```

Goldr does not store uploads, validate file type, scan contents, or choose
file-size policy.

## CSRF

Applications own middleware mounting, secrets, auth, sessions, request body
limits, request parsing, and error responses. Goldr's optional `csrf` package
provides signed-cookie token issue and validation helpers. Applications can
also use another CSRF library, framework-specific middleware, or browser
cookie policy such as SameSite when that is enough for the app's threat model.

Create a guard from an application secret:

```go
guard, err := csrf.New(csrf.Config{
	Secret: []byte(os.Getenv("CSRF_SECRET")),
})
```

The secret must be at least 32 bytes.

`TokenMiddleware` issues or reuses a signed token and stores it on the request
for templates. It does not reject unsafe requests. Actions still validate
after parsing.

Wrap generated routes in app server setup when the CSRF guard is directly
available there:

```go
mux.Handle("/", guard.TokenMiddleware(routes.Handler()))
```

For apps that store dependencies on the request first, wrap generated routes
with the app dependency middleware and use route-tree endpoint middleware to
issue CSRF tokens for matched pages, actions, and fragments:

```go
mux.Handle("/", deps.Middleware(appDeps, routes.Handler()))
```

```go
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deps.From(r).CSRF.TokenMiddleware(next).ServeHTTP(w, r)
	})
}
```

If token issuance must also run on generated 404 and 405 responses, issue the
token from mux-level middleware instead.

### Recommended CSRF Shape

In `app/routes/layout.go`, read the request token that `TokenMiddleware`
stored:

```go
func Layout(r *http.Request, ctx goldr.LayoutContext) templ.Component {
	return LayoutView(csrf.Token(r), ctx.Child)
}
```

In `app/routes/layout.templ`, expose that token through server-rendered HTML:

```templ
package routes

import "github.com/mobiletoly/goldr/csrf"

templ LayoutView(csrfToken string, child templ.Component) {
	<head>
		@csrf.Meta(csrfToken)
	</head>
	<body hx-headers={ csrf.Headers(csrfToken) }>
		@child
	</body>
}
```

Use `csrf.Input` inside ordinary forms:

```templ
package users

import "github.com/mobiletoly/goldr/csrf"

templ PageView(csrfToken string) {
	<form hx-post={ urls.Users.Create.Path() }>
		@csrf.Input(csrfToken)
		<button type="submit">Save</button>
	</form>
}
```

Page handlers pass the request token to templates:

```go
func Page(r *http.Request) goldr.PageRouteResponse {
	return goldr.NewPage(
		PageView(csrf.Token(r)),
		goldr.PageMetadata{Title: "Users"},
	)
}
```

Validate form submissions after parsing:

```go
appDeps := deps.From(r)
if err := r.ParseForm(); err != nil {
	http.Error(w, "bad request", http.StatusBadRequest)
	return
}
if err := appDeps.CSRF.Validate(r, r.PostFormValue(csrf.FieldName)); err != nil {
	http.Error(w, "forbidden", http.StatusForbidden)
	return
}
```

For unsafe HTMX controls that do not submit a form field, rely on the inherited
layout `hx-headers` and validate with an empty form token:

```go
appDeps := deps.From(r)
if err := appDeps.CSRF.Validate(r, ""); err != nil {
	http.Error(w, "forbidden", http.StatusForbidden)
	return
}
```

`csrf.HeaderName` is `X-CSRF-Token`. Header matching is case-insensitive, and
the header token takes precedence over the submitted form token when both are
present.

Keep the signed CSRF cookie HttpOnly. Do not read CSRF tokens from cookies in
browser JavaScript. App-owned JavaScript fetch helpers can read
`meta[name="csrf-token"]`, rendered by `csrf.Meta(csrfToken)`, and send that
value in `X-CSRF-Token`.

## App Dependencies

Goldr does not generate an app dependency container. Use a small app-owned
package outside `app/routes` when route packages need stable dependencies.

Example shape:

```text
app/deps/
  deps.go
```

```go
type Dependencies struct {
	CSRF     *csrf.Guard
	Store    *store.Store
	BasePath string
}
```

The CSRF field can hold Goldr's optional guard, another library's guard, or an
app-owned adapter.

Wrap generated routes with middleware that stores the dependencies on the
request context:

```go
mux.Handle("/", deps.Middleware(appDeps, routes.Handler()))
```

Route functions read dependencies through the app helper. The handler name is
ordinary Go; `route.go` decides whether this function is a page:

```go
func page(r *http.Request) goldr.PageRouteResponse {
	appDeps := deps.From(r)
	return goldr.NewPage(PageView(appDeps.BasePath), goldr.PageMetadata{Title: "Home"})
}
```

Use the dependency package for stable app environment such as stores, auth
managers, configuration, base paths, redirect policy, CSRF guards, mailers,
queues, or feature flag clients. Use request context for request-scoped facts
such as cancellation, deadlines, request IDs, current user, or tenant.
