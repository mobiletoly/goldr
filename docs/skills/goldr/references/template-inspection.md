# Template Inspection For App Debugging

Use this reference when a developer or agent needs to understand which Goldr
layout, page, or fragment rendered a visible region in a local browser.

Template inspection is a development debugging aid. Normal generated handler
output is unchanged by default.

## Modes

Generated handlers accept:

```go
routes.HandlerWithOptions(routes.HandlerOptions{
	TemplateInspection: goldr.TemplateInspectionComments,
})
```

Modes:

- `goldr.TemplateInspectionOff`: normal output, no markers.
- `goldr.TemplateInspectionComments`: paired HTML comments around render-unit
  boundaries.
- `goldr.TemplateInspectionOverlay`: comment markers plus a visible browser
  overlay when the app renders `goldr.TemplateInspector()`.

Do not make application behavior, tests, or production output depend on
inspector comments or overlay elements.

## Comments Mode

Comments mode helps source inspection:

```html
<!--goldr:start id=g_pageusers_page_templ kind=page route=/users source=app/routes/users/page.templ go=app/routes/users/route.go-->
...
<!--goldr:end id=g_pageusers_page_templ-->
```

Paths are app-relative, not absolute machine paths. `source` is the file shown
and copied by the overlay; it prefers a colocated template file when Goldr can
identify one and otherwise falls back to the Go source file. `go` is the route
declaration or handler file.

Fragment markers can also carry `handler` when Goldr knows the callable that
handles the fragment route. For convention fragments this is the generated
fragment function such as `FragTable`; for route declarations this is the
handler expression from `route.go`, such as `FragTable` or `Kit.Table`.

## Overlay Mode

Overlay mode requires three app-side pieces:

1. Generated handler option:

   ```go
   routes.HandlerWithOptions(routes.HandlerOptions{
   	TemplateInspection: goldr.TemplateInspectionOverlay,
   })
   ```

2. Browser helper mounted explicitly:

   ```go
   mux.Handle("/goldr/", http.StripPrefix("/goldr/", browser.Handler()))
   ```

3. Layout helper rendered explicitly, usually near the end of the root layout:

   ```templ
   @goldr.TemplateInspector()
   ```

`goldr.TemplateInspector()` renders nothing unless overlay mode is active for
the request.

## Env Vars Are App-Owned

Some apps may map an env var to inspection mode:

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

Goldr does not define a universal inspection env var. Check the app server
setup before assuming `GOLDR_TEMPLATE_INSPECTION` or any other switch exists.

## Labeled Components

Use `goldr.LabeledComponent` when an ordinary templ component is important
enough to deserve its own inspection boundary:

```templ
@goldr.LabeledComponent("User directory", DirectoryView(form, contacts, csrfToken))
```

The second argument can be any `templ.Component`, regardless of how many
parameters the template function accepts. With inspection off, the wrapper
renders the component without comments. With comments or overlay mode active,
it emits `kind=component` markers. The collapsed overlay badge combines the
component kind token, label, and nearest source context, such as
`component User directory: app/routes/users/page.templ`. Badge text is split
into styled kind, label, and path/context parts so route kinds and paths are
easy to distinguish.

Click the badge arrow to expand details. Expanded component details show the
component label, source context, rendered context, and a vertical render chain.
`render chain` lists the enclosing route-owned render units and includes
labeled component parents as `component <label>` entries. `source context` is
the enclosing Goldr render-unit source path, not necessarily the templ function
definition for the labeled component.

Prefer it for essential regions such as a page shell section, a complex form,
a data table, a reusable panel, or another meaningful render boundary that an
agent or developer would naturally want to isolate in the browser inspector.
Do not wrap every templ function through `LabeledComponent`: too many component
markers make the inspector noisy and obscure the route, layout, page, and
fragment boundaries that Goldr already marks automatically.

Labels must not be empty. Duplicate labels are allowed. This helper does not
add DOM wrappers, discover source paths, or create a Goldr component system.

## Embedded Fragments

Generated dispatch marks direct page, layout, and fragment route responses
automatically. When a page embeds a first-class fragment and the developer
wants a separate local inspection boundary, use the generated package-local
wrapper:

```templ
@renderFragTable(FragTableView(rows))
```

For an index fragment, the wrapper is `renderFragIndex`.

Hyphenated fragment paths are normalized to valid Go identifiers, so
`/daytempo-chart` uses `renderFragDaytempoChart`. If several mounted fragments
in the same route package would collide on the same wrapper, Goldr generates a
route-qualified wrapper such as `renderFragMountCustomerChartIndex`.
Mounted wrapper markers use the live mounted route path and the mounted
implementation source context, so `/admin/reports/table` can point at
`app/mounts/reports/route.go` while still being generated in the live owner
route package.

Expanded route-owned details keep route patterns and source paths separate.
The `unit` row names the render unit kind, the `route pattern` row shows the
URL-facing pattern such as `/users/{id}/table`, and the `source` row shows the
filesystem path such as `app/routes/users/by_id/route.go`.

Expanded fragment details show the handler when available. The handler names
the route function Goldr calls for the fragment response; it is not a source
locator for every templ declaration rendered by that function.

Direct templ rendering is valid, but it is just normal HTML output:

```templ
@FragTableView(rows)
```

Use inspection to understand local render ownership, not as a production
contract.
