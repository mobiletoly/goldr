# CSRF

Goldr provides a small `csrf` package for unsafe form and HTMX requests.

Applications still own middleware mounting, secrets, auth, sessions, request
body limits, and error responses.

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
mux.Handle("/", guard.Middleware(routes.Handler()))
```

The middleware sets a signed HttpOnly CSRF cookie when the request does not
already carry a valid token. It does not reject requests and does not parse
request bodies.

## Forms

Pass the request token to the template:

```go
func Page(r *http.Request) goldr.Page {
    return goldr.Page{
        Component: PageView(guard.Token(r)),
    }
}
```

Render the hidden input explicitly:

```templ
package routes

import "github.com/mobiletoly/goldr/csrf"

templ PageView(csrfToken string) {
    <form method="post">
        <input type="hidden" name={ csrf.FieldName } value={ csrfToken }/>
        <button type="submit">Save</button>
    </form>
}
```

Validate after parsing form values:

```go
func PostSave(w http.ResponseWriter, r *http.Request) {
    form, err := bind.ParseForm(r)
    if err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }
    if err := guard.Validate(r, form.Value(csrf.FieldName)); err != nil {
        http.Error(w, "forbidden", http.StatusForbidden)
        return
    }

    // Perform the mutation.
}
```

This action-level validation keeps multipart request-size limits and memory
policy application-owned.

## HTMX Headers

For unsafe HTMX requests that do not submit a form field, send the same token
in `X-CSRF-Token`:

```html
<button
  hx-post="/users/save-preview"
  hx-headers='{"X-CSRF-Token": "..."}'>
  Save
</button>
```

Then validate with an empty form token:

```go
if err := guard.Validate(r, ""); err != nil {
    http.Error(w, "forbidden", http.StatusForbidden)
    return
}
```

`X-CSRF-Token` takes precedence over a submitted form token when both are
present.

## Cookie Policy

The default cookie name is `goldr_csrf`, the default path is `/`, the default
lifetime is 12 hours, and SameSite defaults to Lax. Cookies are always
HttpOnly. Set `Config.Secure` for HTTPS deployments.
