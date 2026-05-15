# Forms

goldr provides small form helpers in the `bind` package. Validation rules and
persistence stay application-owned.

Use form helpers from an action route or another ordinary `net/http` handler.

## Parse And Validate

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
		response, err := goldr.Render(r, UserForm(form))
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		hx.Retarget(w, "#user-form")
		hx.Reswap(w, "outerHTML")
		_ = response.Write(w, r)
		return
	}

	// Application-owned persistence happens here.
	response, err := goldr.Render(r, UsersTable())
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	_ = response.Write(w, r)
}
```

`bind.ParseForm` uses `http.Request.ParseForm` and copies parsed values.

## Read Values

Read values explicitly:

```go
name := form.Value("name")
tags := form.Values("tags")
```

`Value` returns the first value for a field. `Values` returns all values for a
field.

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
	_ = message
}

messages := form.FieldErrors("name")
```

`FieldErrors` supports multiple messages per field.

## Redisplay With HTMX

Page, layout, and fragment render functions do not receive
`http.ResponseWriter`. Use `actions.go` when a route-local mutation needs to
parse a form, set headers, or redisplay partial HTML.

For HTMX redisplay, combine `bind` with `hx` response headers:

```go
response, err := goldr.Render(r, UserForm(form))
if err != nil {
	http.Error(w, "internal server error", http.StatusInternalServerError)
	return
}
hx.Retarget(w, "#user-form")
hx.Reswap(w, "outerHTML")
_ = response.Write(w, r)
```

goldr does not validate required fields, allowed values, CSRF tokens, or
business rules. Applications own those decisions. `goldr.Render` only provides
the default buffered templ HTML response.

## Runnable Example

`examples/full_feature/` demonstrates form parsing, field-error redisplay, and
successful HTMX replacement from `PostCreate` in:

```text
app/routes/users/actions.go
```
