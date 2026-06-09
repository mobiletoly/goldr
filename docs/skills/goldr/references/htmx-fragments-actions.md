# HTMX, Fragments, And Actions

Use this reference when editing HTMX interactions, fragments, embedded
fragments, or action responses in a Goldr application.

## Keep HTMX Visible

Prefer ordinary `hx-*` attributes in `.templ` files:

```templ
<button
	hx-get={ urls.Users.Table.Path() }
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

Fragments are declared in `route.go`. Fragment templates can stay in
`frag_<name>.templ` when handlers use them.

```text
goldr.FragmentRoute("/table", table) in app/routes/users/route.go -> /users/table
goldr.FragmentRoute("/", statusOptions) in app/routes/users/status_options/route.go -> /users/status-options
```

A fragment route provides:

```go
func table(r *http.Request) goldr.FragmentRouteResponse {
	return goldr.NewFragment(FragTableView(loadRows(r)))
}
```

Fragments render for `GET` and `HEAD`; they are not layout-wrapped. A fragment
route may return a fragment, redirect, text response, or `goldr.RouteError`.
Returning a page from a fragment route is invalid. Use an index fragment when an
HTMX partial owns the route directory path itself; do not model that endpoint as
a page that returns a fragment. `goldr.NewFragment` defaults to
`Cache-Control: no-store`; set `Cache-Control` explicitly for intentionally
cacheable fragments.

Prefer a named fragment on the page route for simple child endpoints:

```go
var Route = goldr.RouteDef{
	Page: page,
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/filters", filterui.Fragment),
	},
}
```

This keeps `/users/filters` owned by `app/routes/users/route.go` while letting
`filterui` provide reusable implementation code. Do not add a child
`filters/route.go` with an index fragment only to create the `/filters`
segment. Use a child index fragment when that child directory has independent
route ownership, such as its own middleware, params, actions, page, or
route-local behavior.

When a fragment exists only to support one page workflow but has its own route
ownership, nest it under the page route. The child route does not need a page:

```text
users/
  route.go
  page.templ
  prepare/
    route.go
    result.templ
```

`prepare/route.go` can declare an index fragment for `GET /users/prepare` or a
named fragment for a child segment. Keep templates used only by that prepare
route directly in `prepare/` instead of adding a route-local UI package.

Kit-backed fragments use the same browser path shape:

```text
goldr.KitFragmentRoute("/table", reports.Kit.Table) -> /reports/table
goldr.KitFragmentRoute("/summary", reports.Kit.Summary) -> /reports/summary
```

The shared kit method receives the request-scoped kit value and the request:

```go
func (kit Kit) Table(r *http.Request) goldr.FragmentRouteResponse
```

For complex reusable partials, keep the same route-owned shape and make the
handler an adapter into a shared app package. This is useful when two pages
need the same chart, picker, or preview behavior but each page owns different
query params, defaults, empty states, surrounding UX, and URL helpers.

```go
var Route = goldr.RouteDef{
	Page: page,
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/chart", chart),
	},
}

func chart(r *http.Request) goldr.FragmentRouteResponse {
	input := chartui.Input{
		EndDate: selectedEndDate(r),
		Focus:   r.FormValue("focus"),
		Metric:  r.FormValue("metric"),
	}
	model, err := chartui.Load(r.Context(), appDeps(r), input)
	if err != nil {
		return chartui.ErrorFragment(err)
	}
	return goldr.NewFragment(chartui.View(model))
}
```

The shared package may own typed input, model, loading, formatting, and templ
views. It should not declare live URLs, infer page-owned query state, hide
HTMX, or store request-scoped data in package globals. The page template still
uses the route-owned helper in visible HTMX markup, such as
`hx-get={ urls.Reports.Chart.Path() }`.

## Replacement Boundaries

When a control refreshes a fragment, prefer a page-owned slot as the HTMX
replacement boundary:

```templ
<button
	hx-get={ urls.Users.Table.Path() }
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

Fragments can render modal or dialog partials loaded on demand. Prefer a stable
page-owned slot over `hx-target="body"` with `hx-swap="beforeend"`:

```templ
<button
	hx-get={ urls.Tenants.ByID.Bind(id).WebhookSettings.Edit.Path() }
	hx-target="#webhook-settings-dialog-slot"
	hx-swap="innerHTML"
>
	Change key
</button>

<div id="webhook-settings-dialog-slot"></div>
```

The fragment renders the dialog root. Replacing the slot avoids repeated opens
creating duplicate dialog IDs.

For modal fragments inside a mounted route subtree, build `hx-get` and
matching modal `hx-post` URLs from the bound mount helper, for example
`kit.URLs.ResetPassword.Path()`. Do not pass separate raw URLs for same-mount
fragment/action routes when a generated mount helper exists.

## Embedded Fragment Wrappers

When embedding a first-class fragment inside a page, use the generated
package-local wrapper such as `renderFragTable(...)` when the embedded fragment
needs a distinct inspection boundary.

```templ
@renderFragTable(FragTableView(rows))
```

Hyphenated fragment paths are normalized in wrapper names, and mounted
fragments may get route-qualified wrapper names when simple names would collide
inside one live owner package. Read `template-inspection.md` before relying on
a generated wrapper name.

Rendering the templ view directly is valid HTML output:

```templ
@FragTableView(rows)
```

But direct rendering does not add a separate embedded-fragment inspection
boundary.

## Action Responses

Actions are route-local mutation endpoints. Use them when a route needs to
parse forms, set response headers, redirect, or redisplay HTML.

### Same-Page Component Actions

A page that renders multiple interactive components should not default every
mutation to the page index action. If a component owns a distinct workflow,
form, validation path, or mutation, give it a route-local action endpoint even
when the component is rendered inside the parent page.

Avoid making the page index action a multiplexer based on hidden `intent`
fields, query modes, or component-specific branches. The visible browser URL
may remain the parent page URL, while forms post to component-owned action URLs
and preserve page state through explicit query params or form fields.

Use fragments only when the response boundary is partial HTML. Do not introduce
fragment routes merely to split Go or templ files.

Function names are ordinary Go names. `route.go` declares the route surface:

```go
var Route = goldr.RouteDef{
	Actions: goldr.Actions{
		goldr.Action(http.MethodPost, "/create", postCreate),
	},
}
```

Set headers before writing the body:

```go
func postCreate(r *http.Request) goldr.RouteResponse {
	return goldr.NewFragment(UsersTable()).
		WithHeader(hx.HeaderRetarget, "#users-table-slot").
		WithHeader(hx.HeaderReswap, "innerHTML").
		WithHeader(hx.HeaderTrigger, "user:created")
}
```

Return `goldr.NewFragment`, `goldr.NewPage`, `goldr.Redirect`, `goldr.Text`,
`goldr.NoContent`, or `goldr.RouteError` from ordinary action helpers.
`goldr.NoContent{}` defaults to `204 No Content`.

For an action that only supports one page workflow, prefer a nested route
directory when the child segment has meaningful ownership:

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

This gives helpers such as `urls.Users.Prepare.Path()` and
`urls.Users.Save.Path()`. Avoid redundant child names such as
`prepare_user`, `save_user`, or `send_user_updated_event` when the parent
route already supplies that context.

Return `goldr.NewPage` when an action must render a page through the matched
layout stack:

```go
return goldr.NewPage(CreatedView(), goldr.PageMetadata{Title: "Created"}).
	WithStatus(http.StatusCreated)
```

Kit actions use the same method and segment model as local actions, with the
kit value as the first argument:

```go
Actions: goldr.KitActions[reports.Kit]{
	goldr.KitAction(http.MethodPost, "/refresh", reports.Kit.PostRefresh),
}

func (kit Kit) PostRefresh(r *http.Request) goldr.RouteResponse {
	return goldr.NewFragment(TableView(kit.data))
}
```

Use `HTTPAction` or `KitHTTPAction` only when direct `http.ResponseWriter`
control is required, such as streaming, installing `http.MaxBytesReader`, or
calling an API that requires the writer.

For `KitHTTPAction`, the generated adapter still constructs the request-scoped
kit first. If `KitRouteDef.New` or `KitRouteMount.New` returns an error, Goldr
routes that error through generated route error handling before the writer
action runs. Read `shared-kit-routes.md` when changing Kit-backed actions.

## HTMX Header Helpers

Use `github.com/mobiletoly/goldr/hx` for request and response headers.

Request checks include:

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

Response helpers include:

```go
hx.Location(w, urls.Dashboard.Path())
hx.PushURL(w, urls.Users.Path())
hx.PreventPushURL(w)
hx.Redirect(w, urls.Login.Path())
hx.Refresh(w)
hx.ReplaceURL(w, urls.Settings.Path())
hx.PreventReplaceURL(w)
hx.Reselect(w, "#dialog")
hx.Retarget(w, "#form-errors")
hx.Reswap(w, "outerHTML")
hx.Trigger(w, "user:saved")
hx.Trigger(w, "user:saved", "analytics:refresh")
hx.TriggerAfterSwap(w, "swapped")
hx.TriggerAfterSettle(w, "settled")
```

These helpers accept strings, but app-internal route targets should usually
come from generated URL helpers instead of hard-coded paths.

The `hx` package also exports canonical request and response header constants,
such as `hx.HeaderRequest`, `hx.HeaderTarget`, `hx.HeaderLocation`,
`hx.HeaderTrigger`, and `hx.HeaderTriggerAfterSwap`, for app code that needs
direct header access.

HTMX response headers are for non-3xx responses. Do not use `http.Redirect`
when the response depends on HTMX processing `HX-*` headers.
