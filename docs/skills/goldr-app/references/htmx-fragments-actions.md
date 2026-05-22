# HTMX, Fragments, And Actions

Use this reference when editing HTMX interactions, fragments, embedded
fragments, or action responses in a Goldr application.

## Keep HTMX Visible

Prefer ordinary `hx-*` attributes in `.templ` files:

```templ
<button
	hx-get={ urls.Users.FragTable.Path() }
	hx-target="#users-table-slot"
	hx-swap="innerHTML"
>
	Load users
</button>
```

Do not hide HTMX behind proprietary Go components or generic client-state
wrappers. URL helpers remove hard-coded paths; HTMX still stays visible in the
markup.

## Fragments

Fragments use `frag_<name>.go` and `frag_<name>.templ`.

```text
app/routes/users/frag_table.go -> /users/frag-table
```

A fragment route provides:

```go
func FragTable(r *http.Request) goldr.RouteResponse {
	return goldr.NewFragment(FragTableView(loadRows(r)))
}
```

Fragments render for `GET` and `HEAD`; they are not layout-wrapped. A fragment
route may return a fragment, redirect, text response, or server error.
Returning a page from a fragment route is invalid.

## Replacement Boundaries

When a control refreshes a fragment, prefer a page-owned slot as the HTMX
replacement boundary:

```templ
<button
	hx-get={ urls.Users.FragTable.Path() }
	hx-target="#users-table-slot"
	hx-swap="innerHTML"
>
	Load users
</button>

<div id="users-table-slot">
	@renderFragTable(FragTableView(rows))
</div>
```

The fragment root stays inside the slot. This is friendly to HTMX, styling,
fragment-local IDs, and optional template-inspection comments.

## Embedded Fragment Wrappers

When embedding a first-class fragment inside a page, use the generated
package-local wrapper such as `renderFragTable(...)` when the embedded fragment
needs a distinct inspection boundary.

```templ
@renderFragTable(FragTableView(rows))
```

Rendering the templ view directly is valid HTML output:

```templ
@FragTableView(rows)
```

But direct rendering does not add a separate embedded-fragment inspection
boundary.

## Action Responses

Actions are ordinary `net/http` handlers. Use them when route-local mutations
need to parse forms, set response headers, or redisplay HTML.

Set headers before writing the body:

```go
func PostCreate(w http.ResponseWriter, r *http.Request) {
	hx.Retarget(w, "#users-table-slot")
	hx.Reswap(w, "innerHTML")
	hx.Trigger(w, "user:created")
	if err := goldr.WriteComponent(w, r, http.StatusOK, UsersTable()); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
```

Use `goldr.WriteComponent` for action-owned component HTML. It buffers the
templ component, sets HTML content type, writes the status, and skips bodies
for `HEAD`.

Use `goldr.WriteRouteResponse` when an action must return a page through the
matched layout stack:

```go
err := goldr.WriteRouteResponse(
	w,
	r,
	goldr.NewPage(CreatedView(), goldr.PageMetadata{Title: "Created"}).
		WithStatus(http.StatusCreated),
)
```

## HTMX Header Helpers

Use `github.com/mobiletoly/goldr/hx` for request and response headers.

Request checks include:

```go
hx.IsRequest(r)
hx.IsBoosted(r)
hx.CurrentURL(r)
hx.Target(r)
hx.TriggerID(r)
```

Response helpers include:

```go
hx.Location(w, "/dashboard")
hx.PushURL(w, "/users")
hx.Redirect(w, "/login")
hx.Refresh(w)
hx.Retarget(w, "#form-errors")
hx.Reswap(w, "outerHTML")
hx.Trigger(w, "user:saved")
hx.TriggerAfterSwap(w, "swapped")
hx.TriggerAfterSettle(w, "settled")
```

HTMX response headers are for non-3xx responses. Do not use `http.Redirect`
when the response depends on HTMX processing `HX-*` headers.
