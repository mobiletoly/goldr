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
<!--goldr:start id=g_pageusers_page_templ kind=page route=/users source=app/routes/users/page.templ go=app/routes/users/page.go-->
...
<!--goldr:end id=g_pageusers_page_templ-->
```

Paths are app-relative, not absolute machine paths.

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

## Embedded Fragments

Generated dispatch marks direct page, layout, and fragment route responses
automatically. When a page embeds a first-class fragment and the developer
wants a separate local inspection boundary, use the generated package-local
wrapper:

```templ
@renderFragTable(FragTableView(rows))
```

Direct templ rendering is valid, but it is just normal HTML output:

```templ
@FragTableView(rows)
```

Use inspection to understand local render ownership, not as a production
contract.
