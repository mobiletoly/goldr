# Forms, CSRF, And Dependencies

Use this reference when editing form parsing, validation redisplay, multipart
uploads, CSRF protection, or app dependency wiring.

## Forms

Goldr's `bind` package is a small helper. Validation rules and persistence are
application-owned.

Function names are ordinary Go names. `route.go` declares the route surface:

```go
var Route = goldr.RouteDef{
	Actions: goldr.FuncActions{
		goldr.FuncPost("create", postCreate),
	},
}
```

For URL-encoded forms:

```go
func postCreate(r *http.Request) goldr.RouteResponse {
	form, err := bind.ParseForm(r)
	if err != nil {
		return goldr.Text{Status: http.StatusBadRequest, Body: "bad request"}
	}

	var errors bind.FieldErrors
	if form.Value("name") == "" {
		errors.Add("name", "Name is required.")
	}

	form = form.WithErrors(errors)
	if form.HasErrors() {
		return goldr.NewFragment(UserForm(form)).
			WithStatus(http.StatusUnprocessableEntity).
			WithHeader(hx.HeaderRetarget, "#user-form").
			WithHeader(hx.HeaderReswap, "outerHTML")
	}

	// Application-owned persistence.
	return goldr.NoContent{}
}
```

For multipart forms:

```go
r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
form, err := bind.ParseMultipartForm(r, 1<<20)
```

`maxMemory` is the standard library memory threshold, not a hard request size
limit. Use `http.MaxBytesReader` from an explicit `...Handler` action when the
app needs a hard limit.

HTMX multipart forms need both normal HTML encoding and HTMX encoding:

```html
<form method="post" enctype="multipart/form-data" hx-encoding="multipart/form-data">
```

Goldr does not store uploads, validate file type, scan contents, or choose
file-size policy.

## Form Values And Files

Read values explicitly:

```go
name := form.Value("name")
tags := form.Values("tags")
```

For duplicate field names, `Value` returns the first value and `Values` returns
a copy of all values. For both URL-encoded and multipart parsing, body values
come before query-string values.

Read validation errors explicitly:

```go
if form.HasFieldError("name") {
	message := form.FieldError("name")
	_ = message
}

messages := form.FieldErrors("name")
_ = messages
```

`bind.FieldErrors` also has read helpers for form-independent validation code:

```go
var errors bind.FieldErrors
errors.Add("name", "Name is required.")
errors.Add("name", "Name is too short.")

_ = errors.Any()
_ = errors.Has("name")
_ = errors.First("name")
_ = errors.Values("name")
```

Read uploaded files with standard library types:

```go
file, header, err := form.File("avatar")
if err != nil {
	if errors.Is(err, http.ErrMissingFile) {
		return
	}
	http.Error(w, "bad request", http.StatusBadRequest)
	return
}
defer file.Close()
_ = header.Filename
```

Read multi-file fields with `Files`:

```go
for _, header := range form.Files("attachments") {
	file, err := header.Open()
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer file.Close()
}
```

## CSRF

Applications own middleware mounting, secrets, auth, sessions, request body
limits, and error responses. Goldr's `csrf` package provides signed-cookie
token issue and validation helpers.

Create a guard from an application secret:

```go
guard, err := csrf.New(csrf.Config{
	Secret: []byte(os.Getenv("CSRF_SECRET")),
})
```

The secret must be at least 32 bytes.

`TokenMiddleware` issues or reuses a signed token and stores it on the request
for templates. It does not reject unsafe requests. Actions still validate after
parsing.

Wrap generated routes in app server setup when the CSRF guard is directly
available there:

```go
mux.Handle("/", guard.TokenMiddleware(routes.Handler()))
```

For apps that store dependencies on the request first, wrap generated routes
with the app dependency middleware and use route-tree middleware to issue CSRF
tokens:

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

Render a hidden field explicitly:

```templ
<input type="hidden" name={ csrf.FieldName } value={ csrfToken }/>
```

Page handlers usually get the hidden-field token from the request-scoped guard:

```go
func page(r *http.Request) goldr.RouteResponse {
	appDeps := deps.From(r)
	return goldr.NewPage(
		PageView(appDeps.CSRF.Token(r)),
		goldr.PageMetadata{Title: "Users"},
	)
}
```

Validate after parsing:

```go
appDeps := deps.From(r)
if err := appDeps.CSRF.Validate(r, form.Value(csrf.FieldName)); err != nil {
	http.Error(w, "forbidden", http.StatusForbidden)
	return
}
```

For unsafe HTMX requests that do not submit a form field, send
`csrf.HeaderName` and validate with an empty form token:

```go
if err := guard.Validate(r, ""); err != nil {
	http.Error(w, "forbidden", http.StatusForbidden)
	return
}
```

`csrf.HeaderName` takes precedence over the submitted form token when both are
present.

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

Wrap generated routes with middleware that stores the dependencies on the
request context:

```go
mux.Handle("/", deps.Middleware(appDeps, routes.Handler()))
```

Route functions read dependencies through the app helper. The handler name is
ordinary Go; `route.go` decides whether this function is a page:

```go
func page(r *http.Request) goldr.RouteResponse {
	appDeps := deps.From(r)
	return goldr.NewPage(PageView(appDeps.BasePath), goldr.PageMetadata{Title: "Home"})
}
```

Use the dependency package for stable app environment such as stores, auth
managers, configuration, base paths, redirect policy, CSRF guards, mailers,
queues, or feature flag clients. Use request context for request-scoped facts
such as cancellation, deadlines, request IDs, current user, or tenant.
