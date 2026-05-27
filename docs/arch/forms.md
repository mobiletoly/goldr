# Forms

`bind` owns minimal form parsing and field-error carrier helpers.

It does not own:
- application validation rules
- persistence
- upload storage, validation, scanning, or size policy
- client-side state

CSRF is intentionally separate from `bind`. The `csrf` package owns signed
token issue, template exposure helpers, and validation helpers, while
applications own where the guard is mounted, which secret is used, how failed
validation is rendered, and when unsafe actions call validation.

`bind.ParseForm` delegates parsing to `http.Request.ParseForm` and copies parsed
values before returning a `Form`.

`bind.ParseMultipartForm` delegates parsing to
`http.Request.ParseMultipartForm`, copies parsed text values, and carries
standard library multipart file headers in the same `Form` value. File access
returns `multipart.File` and `*multipart.FileHeader`; Goldr does not wrap
uploads in a custom object type.

The multipart boundary is deliberately narrow. `bind` owns enough parsing and
carrier behavior for multipart submissions to use the same value/error
redisplay flow as URL-encoded forms. Applications still own
`http.MaxBytesReader` limits, allowed file types, content inspection, copying,
storage, cleanup policy, and all business validation.

`bind.FieldErrors` is a small redisplay carrier. It supports zero-value use and
multiple messages per field, but it is not a validation framework.

The forms boundary matches the HTMX helper boundary: application code uses
ordinary `net/http` handlers when it needs `http.ResponseWriter`.

Generated action routes provide the route-local place for form mutation and
redisplay handlers. `bind` remains a small parsing and error-carrier package;
it does not validate forms, persist data, choose status codes, or choose
redirect behavior.

`goldr.WriteComponent` is the default templ HTML response helper for actions
that need to redisplay partial HTML. It buffers a `templ.Component` before
committing headers, sets `Content-Type: text/html; charset=utf-8`, writes the
requested status, and returns render or write errors to the action. Actions set
HTMX headers explicitly before calling it. The helper does not set HTMX
headers, inspect form state, or replace action-owned response control. htmx 4
swaps validation statuses such as `422` by default, so applications do not need
custom client response handling for ordinary form redisplay.
