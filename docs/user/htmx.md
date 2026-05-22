# HTMX

goldr keeps HTMX visible in templates and uses ordinary Go handlers for
response control.

## Template Attributes Stay Visible

Use ordinary `hx-*` attributes in templ files:

```templ
package users

import "example.com/hello-goldr/app/urls"

templ DirectoryView() {
	<div>
		<button
			hx-get={ urls.Users.FragTable.Path() }
			hx-target="#users-table-slot"
			hx-swap="innerHTML"
		>
			Load users
		</button>
	</div>
	<div id="users-table-slot">
		@renderFragTable(FragTableView(contacts))
	</div>
	<form
		method="post"
		hx-post={ urls.Users.Create.Path() }
		hx-target="#users-table-slot"
		hx-swap="innerHTML"
	>
		<button type="submit">Add user</button>
	</form>
}
```

URL helpers remove hard-coded paths. HTMX still owns the interaction through
visible attributes such as `hx-get`, `hx-post`, `hx-target`, and `hx-swap`.

When a control refreshes a fragment, prefer a page-owned slot as the HTMX
replacement boundary and put `hx-target` / `hx-swap` on the triggering element.
The slot uses `innerHTML`; the fragment root remains inside the slot for
semantic markup, styling, and fragment-local IDs. This shape stays correct when
the template inspector emits comment markers around embedded fragments.

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
	hx.Retarget(w, "#users-table-slot")
	hx.Reswap(w, "innerHTML")
	hx.Trigger(w, "user:created")
	if err := goldr.WriteComponent(w, r, http.StatusOK, UsersTable()); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
```

Page, layout, and fragment render functions do not receive
`http.ResponseWriter`:

```go
func Page(r *http.Request) goldr.RouteResponse
func Layout(r *http.Request, ctx goldr.LayoutContext) templ.Component
func FragTable(r *http.Request) goldr.RouteResponse
```

Use actions when a route-local mutation needs to set headers, parse forms, or
redisplay partial HTML.

For templ HTML action responses, set any headers first, then call
`goldr.WriteComponent(w, r, status, component)`. It buffers the component before
committing headers, sets `Content-Type: text/html; charset=utf-8`, writes the
status, and skips the body for `HEAD`. `goldr.WriteComponent` does not set HTMX
headers, parse forms, redirect, or choose application status codes.

## CSRF Headers

For unsafe HTMX requests that do not submit a form field, send the token from
Goldr's `csrf` guard with `X-CSRF-Token`:

```html
<button
  hx-post="/users/save-preview"
  hx-headers='{"X-CSRF-Token": "..."}'>
  Save
</button>
```

The action validates the header token with `guard.Validate(r, "")`. For normal
forms, prefer a visible hidden input named `csrf.FieldName`.

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
- CSRF validation for unsafe HTMX requests
- `goldr.WriteComponent` for action-owned templ HTML responses
- fragment rendering for `/users/frag-table`
- form redisplay from `/users/create`

Run it from the repository root:

```bash
go run ./examples/full_feature
```

For server-sent event streams and named SSE swaps, read [SSE](sse.md).
