# Forms

goldr provides small form helpers in the `bind` package. Validation rules and
persistence stay application-owned.

Use form helpers from a response-returning action route or another ordinary
`net/http` handler.

## Parse And Validate

For ordinary URL-encoded forms, use `bind.ParseForm`:

```go
func PostCreate(r *http.Request) goldr.RouteResponse {
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

    // Application-owned persistence happens here.
    return goldr.NewFragment(UsersTable())
}
```

`bind.ParseForm` uses `http.Request.ParseForm` and copies parsed values.

For multipart forms, use `bind.ParseMultipartForm`:

```go
func PostCreate(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, 2<<20)

    form, err := bind.ParseMultipartForm(r, 1<<20)
    if err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }

    // Application-owned validation and persistence happen here.
    _ = form
}
```

`bind.ParseMultipartForm` uses `http.Request.ParseMultipartForm(maxMemory)`
and returns the same `bind.Form` type as `ParseForm`. `maxMemory` is the
standard library memory threshold for multipart parsing. It is not a hard
request-size limit. Use `http.MaxBytesReader` before parsing when the
application needs a total request-size limit.
Declare actions that need `http.MaxBytesReader` with `goldr.FuncPostHandler`
or the matching low-level handler helper.

For HTMX multipart submissions, set both ordinary HTML form encoding and HTMX
request encoding:

```html
<form method="post" enctype="multipart/form-data" hx-encoding="multipart/form-data" hx-post="/users/create">
```

## Read Values

Read values explicitly:

```go
name := form.Value("name")
tags := form.Values("tags")
```

`Value` returns the first value for a field. `Values` returns all values for a
field.

## Read Files

Multipart files use Go standard library types:

```go
file, header, err := form.File("avatar")
if err != nil {
    if errors.Is(err, http.ErrMissingFile) {
        // The upload field was absent.
        return
    }
    http.Error(w, "bad request", http.StatusBadRequest)
    return
}
defer file.Close()

filename := header.Filename
```

`Form.File` returns the first uploaded file for a field as a
`multipart.File` and `*multipart.FileHeader`. The caller closes the returned
file. `Form.File` does not parse the request; it reads file headers captured
by `bind.ParseMultipartForm`.

Use `Form.Files` for multiple-file fields:

```go
for _, header := range form.Files("attachments") {
    file, err := header.Open()
    if err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }
    defer file.Close()

    // Application-owned inspection, validation, copying, or storage.
}
```

goldr does not store uploads, validate file type, scan file contents, or choose
file-size policy. Applications own those decisions.

## Attach Errors

Attach application-owned validation errors:

```go
var errors bind.FieldErrors
errors.Add("name", "Name is required.")
errors.Add("status", "Choose a valid status.")

form = form.WithErrors(errors)
```

Read errors during redisplay:

```go
if form.HasFieldError("name") {
    message := form.FieldError("name")
}

messages := form.FieldErrors("name")
```

`FieldErrors` supports multiple messages per field.

## Redisplay With HTMX

Page, layout, and fragment render functions do not receive
`http.ResponseWriter`. Use a route action when a route-local mutation needs to
parse a form, set headers, or redisplay partial HTML.

For HTMX redisplay, combine `bind` with `hx` response headers:

```go
return goldr.NewFragment(UserForm(form)).
    WithStatus(http.StatusUnprocessableEntity).
    WithHeader(hx.HeaderRetarget, "#user-form").
    WithHeader(hx.HeaderReswap, "outerHTML")
```

goldr does not validate required fields, allowed values, or business rules.
Applications own those decisions. For CSRF, use the `csrf` package and validate
the submitted token after parsing:

```go
if err := guard.Validate(r, form.Value(csrf.FieldName)); err != nil {
    return goldr.Text{Status: http.StatusForbidden, Body: "forbidden"}
}
```

`goldr.NewFragment` provides the default buffered templ HTML response for
partial redisplay. If redisplayed HTML should use a non-200 status such as
`422`, attach it with `WithStatus`.

## Runnable Example

`examples/full_feature/` demonstrates multipart form parsing, field-error
redisplay with `422`, optional file-header access, and successful HTMX
replacement from `PostCreate` in:

```text
app/routes/users/route.go
```
