# Navigation Reference

Use this when working on route-derived navigation, destination trail keys, or
app-level Back links in Goldr apps.

Core model:

- `RouteDef.Nav` and `KitRouteDef.Nav` declare one canonical route navigation
  contribution.
- `goldr.RouteNav{Label: "..."}` declares a static step.
- `goldr.RouteNav{Key: "model"}` declares a dynamic step that the handler
  resolves from app data.
- `Destination.TrailKey("from-inventory")` declares workflow state on a live
  source-to-target edge; target accepted keys are derived from inbound
  destinations.
- `goldr.Nav(r)` returns request navigation state prepared by generated
  dispatch.
- `goldr.Nav(r).Navigation()` returns resolved `goldr.Navigation` data for
  templates.
- `goldr.Navigation.Trail` is the resolved app-owned trail rendered by
  templates.

Canonical page pattern:

```go
var Route = goldr.RouteDef{
	Page: Page,
	Nav:  goldr.RouteNav{Key: "customer"},
}

func Page(r *http.Request) goldr.PageRouteResponse {
	customer := loadCustomer(r.PathValue("customer_id"))

	nav := goldr.Nav(r)
	nav.Resolve("customer", customer.Name)

	return goldr.NewPage(PageView(nav.Navigation(), customer), goldr.PageMetadata{
		Title: customer.Name,
	})
}
```

Rules:

- Use `Label` for static labels and `Key` for labels loaded by handlers.
- `Label` and `Key` are mutually exclusive.
- `Key` names are semantic label keys, not necessarily route param names.
- Resolve every dynamic ancestor that should appear in the rendered trail.
- Use `ResolveHref` only when the app intentionally overrides a canonical href.
- Repeated resolves for the same key use the last non-empty label.
- Never resolve labels from raw path values in framework code.

Alternate workflow pattern:

```go
var Route = goldr.RouteDef{
	Page: Page,
	Nav:  goldr.RouteNav{Label: "Report"},
}
```

```go
var Route = goldr.RouteDef{
	Page: Page,
	Destinations: goldr.Destinations{
		"model-report": goldr.To(urls.Admin.Models.ByModelID.Report).
			TrailKey("from-inventory"),
	},
}
```

```go
nav := goldr.Nav(r)
switch nav.TrailKey() {
case urls.Admin.Models.ByModelID.Report.TrailKeys.FromInventory:
	return goldr.NewPage(ReportView(nav.NavigationWithTrail(inventoryTrail(r, model)), model), goldr.PageMetadata{Title: "Report"})
default:
	nav.Resolve("model", model.Name)
	return goldr.NewPage(ReportView(nav.Navigation(), model), goldr.PageMetadata{Title: "Report"})
}
```

Generated destination helpers:

- `Href()` appends `_goldr_nav_trail_key` only for destination-selected keys.
- `NavigationHref(nav)` is generated only for destinations with
  `TrailKey(...)`. It appends `_goldr_nav_trail_key` and preserves the current
  relative URL in `_goldr_return_to`.
- Destination helpers do not copy request query parameters onto the target URL
  or expose query forwarding helpers.
- Compose target-owned query strings explicitly in app code.
- Do not carry source-page filters or pagination into a detached target route.
- Use `NavigationHref(nav)` when link-based Back should return to the exact
  current source URL.
- Plain route `Path()` never emits `_goldr_nav_trail_key`.
- Destination map keys name generated source helpers; they do not become trail
  keys.

Mounted routes:

- Source routes under `app/mounts` may declare default `Nav` labels or keys,
  but cannot declare live `Destinations`.
- Live owners select mounted children, override nav only when needed, and
  declare destinations through selected `goldr.MountRoute` entries.
- `KitRouteMount` has no top-level `Nav` field.
- If a child-only selection omits `/`, Goldr must not synthesize a mount-root
  nav step.
- Reusable mounted handlers may call `goldr.Nav(r).Resolve(...)` for live owner
  keys, or receive app-owned trail builders and URL callbacks through `K`.

Rendering:

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

Use `nav.Back` for app-level Back. Do not add framework fallback URLs;
fallback policy belongs to app rendering code.
