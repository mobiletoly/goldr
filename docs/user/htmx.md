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
			hx-get={ urls.Users.Table.Path() }
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

Fragments can also render modal or dialog partials loaded on demand. Prefer a
stable page-owned slot for those interactions instead of appending directly to
`body`:

```templ
<button
	hx-get={ urls.Tenants.ByID(id).WebhookSettings.Edit.Path() }
	hx-target="#webhook-settings-dialog-slot"
	hx-swap="innerHTML"
>
	Change key
</button>

<div id="webhook-settings-dialog-slot"></div>
```

The fragment should render the dialog root. Replacing the slot keeps repeated
opens deterministic and avoids duplicate dialog IDs.

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

func PostCreate(r *http.Request) goldr.RouteResponse {
	return goldr.NewFragment(UsersTable()).
		WithHeader(hx.HeaderRetarget, "#users-table-slot").
		WithHeader(hx.HeaderReswap, "innerHTML").
		WithHeader(hx.HeaderTrigger, "user:created")
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

For templ HTML action responses, return `goldr.NewFragment(component)` with
the intended status and headers. Goldr buffers the component before committing
headers, sets `Content-Type: text/html; charset=utf-8`, writes the status, and
skips the body for `HEAD`.

When an HTMX action or fragment exists only for one page workflow, keep the
endpoint under that page route:

```text
users/
  route.go
  page.templ
  prepare/
    route.go
    action_handlers.go
    result.templ
  save/
    route.go
    action_handlers.go
```

Nested action or fragment directories do not need standalone pages. They can
exist solely to own the route segment, handler, local templates, middleware, or
helper name. Keep one-route templates in the route directory; move them to
`internal` or shared packages only when multiple sibling routes or route trees
actually reuse them.

Choose child route names for the generated helper shape:

```text
users/prepare -> urls.Users.Prepare.Path()
users/save    -> urls.Users.Save.Path()
```

Avoid names that repeat the parent route context.

## CSRF Headers

For unsafe HTMX requests that do not submit a form field, put Goldr's CSRF
header JSON on a shared layout element:

```templ
<body hx-headers={ csrf.Headers(csrfToken) }>
    @child
</body>
```

The action validates the header token with `guard.Validate(r, "")`. For normal
forms, prefer `@csrf.Input(csrfToken)`. Keep the signed CSRF cookie HttpOnly;
do not add readable CSRF cookies for HTMX convenience.

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
- fragment rendering for `/users/table`
- form redisplay from `/users/create`

Run it from a goldr checkout:

```bash
(cd examples/full_feature && go run .)
```

For server-sent event streams and named SSE swaps, read [SSE](sse.md).
