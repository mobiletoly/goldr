# Forms

`bind` owns minimal form parsing and field-error carrier helpers.

It does not own:
- application validation rules
- persistence
- CSRF policy
- multipart upload behavior
- client-side state

`bind.ParseForm` delegates parsing to `http.Request.ParseForm` and copies parsed
values before returning a `Form`.

`bind.FieldErrors` is a small redisplay carrier. It supports zero-value use and
multiple messages per field, but it is not a validation framework.

The forms boundary matches the HTMX helper boundary: application code uses
ordinary `net/http` handlers when it needs `http.ResponseWriter`.

Generated action routes provide the route-local place for form mutation and
redisplay handlers. `bind` remains a small parsing and error-carrier package;
it does not validate forms, persist data, choose status codes, or render
responses.
