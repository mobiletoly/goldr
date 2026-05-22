# Forms, CSRF, And Dependencies

Use this reference when editing form parsing, validation redisplay, multipart
uploads, CSRF protection, or app dependency wiring.

## Forms

Goldr's `bind` package is a small helper. Validation rules and persistence are
application-owned.

For URL-encoded forms:

```go
func PostCreate(w http.ResponseWriter, r *http.Request) {
	form, err := bind.ParseForm(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var errors bind.FieldErrors
	if form.Value("name") == "" {
		errors.Add("name", "Name is required.")
	}

	form = form.WithErrors(errors)
	if form.HasErrors() {
		hx.Retarget(w, "#user-form")
		hx.Reswap(w, "outerHTML")
		_ = goldr.WriteComponent(w, r, http.StatusUnprocessableEntity, UserForm(form))
		return
	}

	// Application-owned persistence.
}
```

For multipart forms:

```go
r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
form, err := bind.ParseMultipartForm(r, 1<<20)
```

`maxMemory` is the standard library memory threshold, not a hard request size
limit. Use `http.MaxBytesReader` when the app needs a hard limit.

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

Wrap generated routes in app server setup:

```go
mux.Handle("/", guard.Middleware(routes.Handler()))
```

Render a hidden field explicitly:

```templ
<input type="hidden" name={ csrf.FieldName } value={ csrfToken }/>
```

Validate after parsing:

```go
if err := guard.Validate(r, form.Value(csrf.FieldName)); err != nil {
	http.Error(w, "forbidden", http.StatusForbidden)
	return
}
```

For unsafe HTMX requests that do not submit a form field, send
`X-CSRF-Token` and validate with an empty form token:

```go
if err := guard.Validate(r, ""); err != nil {
	http.Error(w, "forbidden", http.StatusForbidden)
	return
}
```

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

Route functions read dependencies through the app helper:

```go
func Page(r *http.Request) goldr.RouteResponse {
	appDeps := deps.From(r)
	return goldr.NewPage(PageView(appDeps.BasePath), goldr.PageMetadata{Title: "Home"})
}
```

Use the dependency package for stable app environment such as stores, auth
managers, configuration, base paths, redirect policy, CSRF guards, mailers,
queues, or feature flag clients. Use request context for request-scoped facts
such as cancellation, deadlines, request IDs, current user, or tenant.
