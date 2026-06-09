# Template Inspection

Template inspection is a development-only way to see which Goldr render unit
produced a page region.

It is off by default. Normal `routes.Handler()` output does not include
inspector comments, overlay scripts, or debug elements.

![inspect.png](scr/inspect.png)

## Inspection Modes

Generated route packages expose:

```go
type HandlerOptions struct {
	ErrorHandlers       ErrorHandlers
	TemplateInspection goldr.TemplateInspectionMode
}
```

The supported modes are:

- `goldr.TemplateInspectionOff`: no template inspection.
- `goldr.TemplateInspectionComments`: HTML comment markers only.
- `goldr.TemplateInspectionOverlay`: HTML comment markers plus Goldr's browser
  overlay helper when the layout renders `goldr.TemplateInspector()`.

Goldr does not read environment variables by itself. If you want an env var for
local development, keep that mapping in application code.

## Comment Inspection

Use comments mode when you want DevTools or View Source to show render
boundaries without adding a visible overlay:

```go
import "github.com/mobiletoly/goldr"

mux.Handle("/", routes.HandlerWithOptions(routes.HandlerOptions{
	TemplateInspection: goldr.TemplateInspectionComments,
}))
```

Generated dispatch emits paired HTML comments around page, layout, and
fragment render boundaries:

```html
<!--goldr:start id=g_pageusers_page_templ kind=page route=/users source=app/routes/users/page.templ go=app/routes/users/route.go-->
...
<!--goldr:end id=g_pageusers_page_templ-->
```

The paths are app-relative, never absolute machine paths. `source` is the file
shown and copied by the browser overlay. It prefers the colocated template file
when Goldr can identify one, such as `page.templ` or `layout.templ`; otherwise
it falls back to the Go source file. `go` is the route declaration or handler
file that Goldr calls. Redirect, text, error, and `HEAD` response bodies do not
emit inspector markers.

Fragment markers derived from `route.go` may also include a `handler` field.
That field is the route declaration handler expression, such as `FragTable` or
`Kit.Table`.

## Overlay Inspection

Use overlay mode when you want visible outlines and labels in the browser.

Mount Goldr's browser helper and set the generated handler option:

```go
package main

import (
	"net/http"

	"myapp/app/routes"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/browser"
)

func handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/goldr/", http.StripPrefix("/goldr/", browser.Handler()))
	mux.Handle("/", routes.HandlerWithOptions(routes.HandlerOptions{
		TemplateInspection: goldr.TemplateInspectionOverlay,
	}))
	return mux
}
```

Render the layout helper explicitly, usually near the end of the root layout:

```templ
<body>
	@child
	@goldr.TemplateInspector()
</body>
```

`goldr.TemplateInspector()` renders nothing in off or comments mode. In overlay
mode, it renders a script tag for `/goldr/goldr-template-inspector.js`, the
helper file named by `browser.TemplateInspectorHelperPath`.

The browser helper reads the inspector comments and draws colored outlines and
labels over layout, page, fragment, and labeled component regions. It appends
debug overlay nodes outside application render regions and does not wrap app
content.

The helper also adds a small floating control for the current page:

- `All` shows every visible render-unit boundary.
- `Next` starts one-at-a-time inspection from the outermost visible render
  unit, then advances through visible render units and wraps when it reaches
  the end.
- `Off` hides overlay boxes and labels while keeping the control visible.

The helper stores only `All` and `Off` in `localStorage`. One-at-a-time
inspection is temporary: if you use `Next`, Goldr stores that as `All`, so a
reload returns to the all-boundaries view. Pages rendered with
`TemplateInspectionOff` or `TemplateInspectionComments` do not load the overlay
helper, so there is no runtime control to show.

## Env Var Example

An application can choose inspection mode from an env var during local
development:

```go
func templateInspectionMode() goldr.TemplateInspectionMode {
	switch os.Getenv("GOLDR_TEMPLATE_INSPECTION") {
	case "comments":
		return goldr.TemplateInspectionComments
	case "overlay":
		return goldr.TemplateInspectionOverlay
	default:
		return goldr.TemplateInspectionOff
	}
}
```

Then pass the mode to the generated handler:

```go
mux.Handle("/", routes.HandlerWithOptions(routes.HandlerOptions{
	TemplateInspection: templateInspectionMode(),
}))
```

Run locally with the mode you want:

```bash
GOLDR_TEMPLATE_INSPECTION=comments go run .
GOLDR_TEMPLATE_INSPECTION=overlay go run .
```

`GOLDR_TEMPLATE_INSPECTION` is only the env var used by these examples. You can
choose another name, use flags, or wire the mode directly.

## Labeled Components

Generated dispatch marks route-owned layouts, pages, and fragments
automatically. For an ordinary templ component that is important while
debugging a page, wrap the render call with `goldr.LabeledComponent`:

```templ
@goldr.LabeledComponent("User directory", DirectoryView(form, contacts, csrfToken))
```

`LabeledComponent` accepts any `templ.Component`, so the wrapped template
function can have any parameter list. The wrapper does not add DOM elements and
does not change the component's rendered HTML when template inspection is off.

Use labeled components selectively. They are best for essential regions such as
page shell sections, complex forms, data tables, reusable panels, or other
meaningful render boundaries that you would naturally isolate while debugging.
Do not wrap every templ function with `LabeledComponent`: too many component
markers make the inspector noisy and make route, layout, page, and fragment
boundaries harder to see.

When comments or overlay mode is active, Goldr emits paired comments around the
wrapped component:

```html
<!--goldr:start id=g_component_user_directory kind=component label=User%20directory-->
...
<!--goldr:end id=g_component_user_directory-->
```

The collapsed overlay badge shows an explicit component kind token, the
component label, and the nearest source context. For the example above, a
component rendered inside the `/users` page shows:

```text
component User directory: app/routes/users/page.templ
```

The badge renders kind, label, and path context as separate styled parts, so
source paths are visually distinct from words such as `layout`, `page`,
`fragment`, or `component`.

Click the badge arrow to expand details. Expanded details show the component
label, the enclosing render context, the source context, and the render chain:

```text
component: User directory
source context: app/routes/users/page.templ
rendered in: page /users
render chain:
  layout /
    layout /users
      page /users
```

`source context` is the enclosing Goldr render-unit source, not necessarily the
templ function definition for the labeled component. Component labels are for
inspection only. `render chain` shows the enclosing route-owned render units
and includes labeled component parents with an explicit component kind token.
For example, a fragment rendered inside the `User directory` labeled component
can show:

```text
render chain:
  layout /
    layout /users
      page /users
        component User directory
          fragment /users/table
```

When source context is available, the overlay copy button copies that source
context path.

Labels must not be empty or whitespace-only. Duplicate labels are allowed, so
repeated components in lists can share a label when that is the clearest
description.

## Embedded Fragments

Generated dispatch marks direct page, layout, and fragment route responses
automatically. When a page embeds a first-class fragment, use the generated
package-local wrapper for that fragment:

```templ
<div id="users-table-slot">
	@renderFragTable(FragTableView(contacts))
</div>
```

For HTMX refreshes, target the slot with `innerHTML`:

```templ
<button
	hx-get={ urls.Users.Table.Path() }
	hx-target="#users-table-slot"
	hx-swap="innerHTML"
>
	Load users
</button>
```

The inspector boundary comments are siblings of the rendered fragment root. A
slot keeps those comments inside the HTMX replacement boundary, so repeated
swaps do not leave stale inspector comments in the DOM.

For comparison, this still renders the same HTML, but does not add an
inspector boundary around the embedded fragment:

```templ
@FragTableView(contacts)
```

The wrapper name follows the fragment path in `route.go`. For example,
`goldr.FragmentRoute("/table", table)` uses `renderFragTable`. An index
fragment uses `renderFragIndex`. Hyphenated fragment paths are normalized to
valid Go identifiers, so `/daytempo-chart` uses `renderFragDaytempoChart`.
When several mounted fragments in the same route package would otherwise share
the same wrapper name, Goldr generates route-qualified wrapper names such as
`renderFragMountCustomerChartIndex`. The helper takes the component you
already render, so the application still chooses the templ function and
arguments.

For mounted route fragments, the wrapper marker uses the live mounted route
path, such as `/admin/reports/table`, while `source context` points at the
mounted implementation under `app/mounts`. That keeps the browser inspector
oriented around the URL the user is viewing without hiding where the reusable
fragment implementation lives.

Click a fragment badge arrow to expand details. When Goldr knows the handler
expression from `route.go`, expanded details include it:

```text
unit: fragment
route pattern: /users/table
handler: FragTable
source: app/routes/users/route.go
render chain:
  layout /
    layout /users
      page /users
        fragment /users/table
```

Route patterns use URL-facing parameter names such as `/{id}`. Source rows use
filesystem paths such as `app/routes/users/by_id/route.go`. Keep those two
concepts separate when reading the inspector: route patterns explain what the
browser can request, while source paths explain where the route-owned code
lives.

For mounted fragments the handler can be a mounted kit selector such as
`Kit.Table`. The handler is the function selected by `goldr.FragmentRoute` or
`goldr.KitFragmentRoute`, not a separate source locator for every templ
declaration called by that function.

Multiple templ declarations inside one fragment template are internal to that
fragment render unit. Declare separate fragments in `route.go` when they need
separately inspectable fragment identities.
