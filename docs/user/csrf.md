# CSRF

Goldr provides a small `csrf` package for unsafe form and HTMX requests.

Applications still own middleware mounting, secrets, auth, sessions, request
body limits, request parsing, and error responses. The package is optional: an
application can use a different CSRF library, a framework-specific middleware,
or browser cookie policy such as SameSite when that is enough for its threat
model.

## Guard

Create one guard from an application secret:

```go
guard, err := csrf.New(csrf.Config{
    Secret: []byte(os.Getenv("CSRF_SECRET")),
})
if err != nil {
    return err
}
```

The secret must be at least 32 bytes.

Wrap generated routes with the guard middleware:

```go
mux := http.NewServeMux()
mux.Handle("/", guard.TokenMiddleware(routes.Handler()))
```

The middleware sets a signed HttpOnly CSRF cookie when the request does not
already carry a valid token. It does not reject requests and does not parse
request bodies.

## Forms

Pass the request token to the template:

```go
func Page(r *http.Request) goldr.PageRouteResponse {
    return goldr.NewPage(
        PageView(csrf.Token(r)),
        goldr.PageMetadata{},
    )
}
```

Render the hidden input with `csrf.Input`:

```templ
package routes

import "github.com/mobiletoly/goldr/csrf"

templ PageView(csrfToken string) {
    <form method="post">
        @csrf.Input(csrfToken)
        <button type="submit">Save</button>
    </form>
}
```

Validate after parsing form values:

```go
func PostSave(r *http.Request) goldr.RouteResponse {
    if err := r.ParseForm(); err != nil {
        return goldr.Text{Status: http.StatusBadRequest, Body: "bad request"}
    }
    if err := guard.Validate(r, r.PostFormValue(csrf.FieldName)); err != nil {
        return goldr.Text{Status: http.StatusForbidden, Body: "forbidden"}
    }

    // Perform the mutation.
    return goldr.NoContent{}
}
```

This action-level validation keeps parsing, multipart request-size limits, and
memory policy application-owned.

## HTMX Headers

For unsafe HTMX requests that do not submit a form field, render inherited
`hx-headers` from the current request token:

```templ
templ LayoutView(csrfToken string, child templ.Component) {
    <body hx-headers={ csrf.Headers(csrfToken) }>
        @child
    </body>
}
```

Then validate with an empty form token:

```go
if err := guard.Validate(r, ""); err != nil {
    http.Error(w, "forbidden", http.StatusForbidden)
    return
}
```

`csrf.HeaderName` is `X-CSRF-Token`. HTTP header names are case-insensitive, so
normal Go request header lookup also accepts equivalent spellings such as
`x-csrf-token`. `X-CSRF-Token` takes precedence over a submitted form token
when both are present.

## App JavaScript

Do not create a readable CSRF cookie for app JavaScript. Keep the signed token
cookie HttpOnly and render the current request token into the page when a
non-HTMX `fetch` helper needs it:

```templ
<head>
    @csrf.Meta(csrfToken)
</head>
```

App-owned JavaScript can read `meta[name="csrf-token"]` and send the value in
`X-CSRF-Token`.

## Cookie Policy

The default cookie name is `goldr_csrf`, the default path is `/`, the default
lifetime is 12 hours, and SameSite defaults to Lax. Cookies are always
HttpOnly. Set `Config.Secure` for HTTPS deployments.
