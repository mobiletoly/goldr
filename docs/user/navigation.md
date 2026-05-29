# Route Navigation Trails

Goldr can prepare route-derived navigation data for breadcrumb-style UI,
app-level Back links, sidebars, and similar server-rendered navigation.

Goldr owns only the route-derived skeleton and one optional selected trail key.
The app still owns labels that come from data, rendered HTML, styling, product
fallbacks, and alternate workflow decisions.

Goldr does not store browser history, read `Referer`, use cookies or sessions
for navigation, render a built-in trail component, or infer labels from path
values.

## Canonical Route Nav

Declare each route's canonical contribution with `RouteDef.Nav`:

```go
var Route = goldr.RouteDef{
	Page: Page,
	Nav: goldr.RouteNav{
		Label: "Firmware",
	},
}
```

Use `Label` for static route labels and `Key` for labels the handler must
resolve from loaded app data:

```go
var Route = goldr.RouteDef{
	Page: Page,
	Nav: goldr.RouteNav{
		Key: "model",
	},
}
```

`Label` and `Key` are mutually exclusive. `Key` is a semantic label key, not a
route parameter name. If one canonical trail needs two model labels, use
different keys such as `device_model` and `firmware_model`.

Handlers read the generated navigation state from the request:

```go
func Page(r *http.Request) goldr.PageRouteResponse {
	model := loadModel(r.PathValue("model_id"))

	nav := goldr.Nav(r)
	nav.Resolve("model", model.Name)

	return goldr.NewPage(
		PageView(nav.Navigation(), model),
		goldr.PageMetadata{Title: model.Name + " Firmware"},
	)
}
```

`goldr.Nav(r).Navigation()` returns resolved `goldr.Navigation` data for
templates. Its `Trail` field is a resolved `goldr.NavTrail`. Static steps are
ready immediately. Dynamic steps appear only after the handler resolves their
keys. Ancestor steps need a label and href; the current step needs only a
label.

Goldr binds canonical parent hrefs from the matched request path values when
the values are available. It never resolves labels from raw path values.

## Href Overrides

Use `ResolveHref` when a dynamic step should carry app-owned query state or a
workflow-specific URL:

```go
nav := goldr.Nav(r)
nav.ResolveHref("customer", customer.Name, customerHref)
```

`ResolveHref` keeps the canonical order but replaces both the dynamic label and
the href for that step. An empty href behaves like `Resolve`.

Repeated `Resolve` or `ResolveHref` calls for the same key use the last
non-empty label.

## Base Paths

Generated route handlers accept a base path for canonical nav hrefs:

```go
handler := routes.HandlerWithOptions(routes.HandlerOptions{
	BasePath: "/webapp",
})
```

`BasePath` affects only hrefs prepared for `goldr.Nav(r).Navigation()`. It
does not change route matching. Mounting or stripping the URL prefix remains
app-owned `net/http` policy.

Use the same base path for URL helpers when the app is served below a prefix:

```go
appURLs := urls.WithBasePath("/webapp")
handler := routes.HandlerWithOptions(routes.HandlerOptions{BasePath: "/webapp"})
```

App-provided hrefs passed to `ResolveHref` are used as-is and are not prefixed.

## Alternate Trail Keys

Canonical route nav is the default. Use alternate trail keys only when the
same target route is intentionally reached through different workflows.

The target route declares only its own canonical nav step:

```go
var Route = goldr.RouteDef{
	Page: Page,
	Nav:  goldr.RouteNav{Label: "Report"},
}
```

The source route selects one key through a destination. The destination map key
names the generated source-side helper; `TrailKey` names the workflow state the
target route can read:

```go
var Route = goldr.RouteDef{
		Page: Page,
		Destinations: goldr.Destinations{
			"model-report": goldr.To(urls.Admin.Models.ByModelID.Report).
				TrailKey("from-inventory"),
		},
	}
```

Goldr derives the target route's accepted keys from live inbound destinations
and generates route-scoped constants:

```go
urls.Admin.Models.ByModelID.Report.TrailKeys.FromInventory
```

The source handler uses the generated destination helper:

```go
href := urls.Admin.Inventory.Destinations.ModelReport.
	Bind(model.ID).
	Href()
```

Destination `Href()` appends `_goldr_nav_trail_key` only when the destination
declares a key. Plain route `Path()` helpers never append it.

Generated dispatch validates the selected key after route matching. Missing,
unknown, malformed, or undeclared keys are ignored:

```go
nav := goldr.Nav(r)
switch nav.TrailKey() {
case urls.Admin.Models.ByModelID.Report.TrailKeys.FromInventory:
	trail := inventoryReportTrail(r, model)
	return goldr.NewPage(ReportView(nav.NavigationWithTrail(trail), model), goldr.PageMetadata{Title: "Report"})
default:
	nav.Resolve("model", model.Name)
	return goldr.NewPage(ReportView(nav.Navigation(), model), goldr.PageMetadata{Title: "Report"})
}
```

Selected trail keys do not automatically change `Trail()`. If an alternate
workflow needs a different number or order of steps, build an explicit
`goldr.NavTrail` in that branch.

Goldr destination helpers carry only Goldr-owned navigation state. `Href()`
does not copy request query parameters or expose a generic query-forwarding
helper. When the target route owns query inputs, compose that target URL
explicitly in app code:

```go
href, err := url.Parse(urls.Admin.Inventory.Destinations.ModelReport.
	Bind(model.ID).
	Href())
if err != nil {
	return err
}
query := href.Query()
query.Set("view", reportView)
query.Set("sort", reportSort)
href.RawQuery = query.Encode()
```

Do not forward source-page filters or pagination into a detached target route's
query string. Those keys belong to the source page, not the target route
contract. If link-based Back should return to the exact current URL, use
`NavigationHref(nav)` on a destination that declares `TrailKey(...)`:

```go
href := urls.Admin.Inventory.Destinations.ModelReport.
	Bind(model.ID).
	NavigationHref(nav.Navigation())
```

`NavigationHref` preserves the current relative URL in `_goldr_return_to`.
Goldr honors that value only when the target request also has a valid selected
trail key. Existing `_goldr_return_to` values are stripped before capture so
Goldr does not create a return stack.

## Rendering

`goldr.NavTrail` is data:

```go
type NavTrail []NavTrailStep

type NavTrailStep struct {
	Label   string
	Href    string
	Current bool
}
```

`goldr.Navigation` carries `Trail`, `Back`, and `Current` fields for normal
server-rendered navigation UI:

```templ
templ TrailNav(nav goldr.Navigation) {
	<nav aria-label="Breadcrumb">
		<ol>
			for _, step := range nav.Trail {
				<li>
					if step.Current {
						<span aria-current="page">{ step.Label }</span>
					} else {
						<a href={ step.Href }>{ step.Label }</a>
					}
				</li>
			}
		</ol>
	</nav>
	if nav.Back.OK {
		<a href={ nav.Back.Href }>Back</a>
	}
}
```

HTMX stays visible in templates. Goldr provides route-safe URLs; the app
decides whether an anchor is a full-page link or an HTMX request.

## App-Level Back

Semantic Back should come from `goldr.Navigation`, not browser history.
Render Back only when `nav.Back.OK` is true. If a product needs a fallback
Back URL, choose that URL in app code.

## Mounted Routes

Mounted source routes under `app/mounts` are reusable implementation, not live
route owners. They may declare default `Nav` labels or keys for their reusable
pages, but they cannot declare live `Destinations`.

Live owners select mounted children, attach destinations, and override nav only
when the live app wants a different label or key:

```go
var Route = goldr.KitRouteMount[analytics.Kit]{
	New:   newKit,
	Mount: "analytics",
	Routes: goldr.MountRoutes{
		{
			Path: "/",
			Destinations: goldr.Destinations{
				"customer-report": goldr.To(urls.Admin.Analytics.Customers.ByCustomerID.Report).
					TrailKey("hq-analytics"),
			},
		},
		{
			Path: "/customers/{customer_id}/report",
		},
	},
}
```

If `Routes` omits `/`, Goldr does not synthesize a root nav step. If `Routes`
is omitted entirely and the full mounted subtree is exposed, included mounted
source nav defaults participate in canonical route navigation.

Mounted handlers can resolve mounted source or live owner keys with
`goldr.Nav(r)` like any other handler. When reusable mounted code needs
owner-specific alternate trail shapes, pass trail builders or URL callbacks
through `K`.

See `examples/navigation` for a complete HQ and Regional example using
canonical route nav, explicit alternate trail keys, mounted owners, and
app-owned trail rendering.
