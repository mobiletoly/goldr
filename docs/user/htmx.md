# HTMX

goldr keeps HTMX visible in templates and uses ordinary Go handlers for
response control.

## Template Attributes Stay Visible

Use ordinary `hx-*` attributes in templ files:

```templ
package users

import "example.com/hello-goldr/app/urls"

templ DirectoryView() {
	<button
		hx-get={ urls.Users.FragTable.Path() }
		hx-target="#users-table"
		hx-swap="outerHTML"
	>
		Load users
	</button>
	<form
		method="post"
		hx-post={ urls.Users.Create.Path() }
		hx-target="#users-table"
		hx-swap="outerHTML"
	>
		<button type="submit">Add user</button>
	</form>
}
```

URL helpers remove hard-coded paths. HTMX still owns the interaction through
visible attributes such as `hx-get`, `hx-post`, `hx-target`, and `hx-swap`.

## Response Headers

Use the `hx` package from action handlers or other ordinary `net/http`
handlers when code needs HTMX request or response headers:

```go
package users

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/hx"
)

func PostCreate(w http.ResponseWriter, r *http.Request) {
	response, err := goldr.Render(r, UsersTable())
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	hx.Retarget(w, "#users-table")
	hx.Reswap(w, "outerHTML")
	hx.Trigger(w, "user:created")
	_ = response.Write(w, r)
}
```

Page, layout, and fragment render functions do not receive
`http.ResponseWriter`:

```go
func Page(r *http.Request) goldr.Page
func Layout(r *http.Request, ctx goldr.LayoutContext) templ.Component
func FragTable(r *http.Request) templ.Component
```

Use actions when a route-local mutation needs to set headers, parse forms, or
redisplay partial HTML.

For default templ HTML action responses, `goldr.Render` buffers the component
and returns an error if rendering fails. After a successful render, set any
headers, then call `response.Write(w, r)`. Use
`response.WriteStatus(w, r, status)` when the HTML response needs a non-200
status. Both write methods set `Content-Type: text/html; charset=utf-8` before
writing status or body, and both skip the body for `HEAD`. `goldr.Render` does
not set HTMX headers, parse forms, redirect, or choose application status
codes.

## Request Helpers

Request helpers read HTMX request headers:

```go
hx.IsRequest(r)
hx.IsBoosted(r)
hx.IsHistoryRestoreRequest(r)
hx.CurrentURL(r)
hx.Prompt(r)
hx.Target(r)
hx.TriggerID(r)
hx.TriggerName(r)
```

The boolean helpers return true only when the request header value is exactly
`"true"`.

## Response Helpers

Response helpers set HTMX response headers:

```go
hx.Location(w, "/dashboard")
hx.PushURL(w, "/users")
hx.PreventPushURL(w)
hx.Redirect(w, "/login")
hx.Refresh(w)
hx.ReplaceURL(w, "/settings")
hx.PreventReplaceURL(w)
hx.Reselect(w, "#dialog")
hx.Retarget(w, "#form-errors")
hx.Reswap(w, "outerHTML")
hx.Trigger(w, "user:saved")
hx.Trigger(w, "a", "b")
hx.TriggerAfterSettle(w, "settled")
hx.TriggerAfterSwap(w, "swapped")
```

These calls set `HX-*` response headers. HTMX response headers are for non-3xx
responses. Do not use `http.Redirect` when the response depends on HTMX
processing `HX-*` headers, because browsers handle HTTP redirects before HTMX
can process those headers.

## Header Constants

The `hx` package exposes constants for HTMX request and response header names:

```go
hx.HeaderRequest
hx.HeaderTarget
hx.HeaderTrigger
hx.HeaderLocation
hx.HeaderRedirect
hx.HeaderRetarget
hx.HeaderReswap
hx.HeaderTriggerAfterSwap
```

See package documentation or completion for the full list.

## Runnable Example

`examples/full_feature/` demonstrates:

- `hx-get` and `hx-post` in templates
- `HX-Trigger`, `HX-Retarget`, and `HX-Reswap` in action handlers
- `goldr.Render` for action-owned templ HTML responses
- fragment rendering for `/users/frag-table`
- form redisplay from `/users/create`

Run it from the repository root:

```bash
go run ./examples/full_feature
```
