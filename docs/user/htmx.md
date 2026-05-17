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
	hx.Retarget(w, "#users-table")
	hx.Reswap(w, "outerHTML")
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

## Server-Sent Events

Goldr provides a small `sse` package for event-stream response mechanics. For
realtime HTML updates, keep the stream app-owned: register the SSE handler in
the application mux, own subscriber state and replay policy, render HTML
fragments with templ, and keep HTMX SSE attributes visible in the template.

```go
package chat

import (
	"net/http"
	"strconv"

	"github.com/mobiletoly/goldr/sse"
)

func Events(w http.ResponseWriter, r *http.Request) {
	stream, ok := sse.Start(w)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	stream.Comment("connected")
	stream.Flush()

	message := Message{ID: 1, Body: "Hello"}
	_ = stream.Component(r, sse.ComponentEvent{
		ID:        strconv.FormatInt(message.ID, 10),
		Component: MessageView(message),
	})
	stream.Flush()
}
```

Templates still show the HTMX SSE connection explicitly:

```templ
<div
	id="messages"
	hx-sse:connect="/chat/events"
	hx-swap="beforeend scroll:bottom"
>
	<!-- streamed message HTML lands here -->
</div>
```

`examples/chat/` demonstrates this pattern with htmx 4 `hx-sse:connect`,
ordinary generated actions for posted messages, and in-memory server-side
persistence:

```bash
go run ./examples/chat
```
