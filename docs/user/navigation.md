# Navigation Trails And Destinations

Breadcrumbs look simple until the same page can be reached from more than one
workflow.

A report page might be opened from HQ:

```text
Home > HQ > HQ Team > Contoso Retail > Report
```

The same report page might also be opened from a regional office:

```text
Home > Regional > Seattle > Regional Team > Northwind Supply > Report
```

The target route cannot guess which trail the user intended. Browser history is
not enough because it is session-local, not semantic, and not reliable for
server-rendered app links. `Referer`, cookies, and hidden server state make the
trail harder to inspect and easier to get wrong across tabs, direct links, and
HTMX requests.

Goldr solves this by keeping route truth and navigation presentation separate:

- generated destination helpers build workflow navigation links with `Href()`
  or `HrefWithQuery(url.Values)`;
- generated route helpers build canonical URLs with `Path()` when no workflow
  context should be selected;
- the source route explicitly chooses a target navigation context;
- the target route declares which context keys it accepts;
- the target handler builds the rendered `goldr.NavTrail` from app data it
  already loads.

Goldr does not store a route stack, infer arbitrary visit history, or render a
built-in breadcrumb component. The app owns labels, HTML, styling, and the
final trail.

## The Core Rule

For navigation links that should preserve breadcrumb context, use generated
destination `Href()`:

```go
href := urls.Main.Hq.Teams.ByTeamID.Customers.ByCustomerID.
	Destinations.SharedReport.
	Bind(customer.ID).
	Href()
```

Use `Path()` only when you intentionally want a clean canonical URL without
context:

```go
urls.Main.Hq.Teams.ByTeamID.Bind(team.ID).Path()
```

That distinction is important. `Href()` is the normal API for workflow links
that should affect breadcrumbs or app-level Back. `Path()` is still useful for
canonical links and the hrefs you place inside a rendered `NavTrail`.

The contextual URL is still ordinary HTTP. Goldr appends a small query value
only when the destination declared a trail:

```text
/main/reports/contoso?_goldr_trail=hq-customer
```

Generated dispatch validates `_goldr_trail` after the route matches. Missing,
unknown, or malformed values are ignored.

When a workflow link also needs app-owned query state, keep that state as
ordinary app query parameters and use `HrefWithQuery(url.Values)`:

```go
query := goldr.QueryValues(r, "view", "sort", "page")

href := urls.Main.Hq.Teams.ByTeamID.Customers.ByCustomerID.
	Destinations.SharedReport.
	Bind(customer.ID).
	HrefWithQuery(query)
```

Goldr owns only `_goldr_trail`. App query values are copied into the generated
destination URL, and app-supplied `_goldr_trail` values are ignored so the
route-declared destination remains the navigation source of truth.
`goldr.QueryValues` is a small helper for carrying named app-owned query keys
from the current request. Use `url.Values` directly when the values come from
loaded app state instead of the current request.

## Target Route

The target route declares the navigation trail keys it knows how to render:

```go
var Route = goldr.RouteDef{
	Page: Page,
	NavTrails: goldr.NavTrails{
		Allowed: []string{"hq-customer", "regional-customer"},
	},
}
```

Goldr generates route-scoped constants for those keys:

```go
urls.Main.Reports.ByCustomerID.NavTrails.HqCustomer
urls.Main.Reports.ByCustomerID.NavTrails.RegionalCustomer
```

Use `goldr.NavTrailSelected` for one-off boolean checks:

```go
if goldr.NavTrailSelected(r, urls.Main.Reports.ByCustomerID.NavTrails.HqCustomer) {
	// Choose page chrome or a parent href for this trail.
}
```

Use `goldr.NavTrailKey` when a handler needs to switch across multiple trail
keys.

In the handler, read the validated key and build the trail explicitly:

```go
func Page(r *http.Request) goldr.PageRouteResponse {
	customer := store.Default.Customer(r.PathValue("customer_id"))
	team := store.Default.Team(customer.TeamID)

	trail := goldr.NavTrail{
		goldr.NavStep("Home", urls.Root.Path()),
		goldr.NavStep("Reports", urls.Main.Path()),
		goldr.CurrentNavStep(customer.Name),
	}

	switch goldr.NavTrailKey(r) {
	case urls.Main.Reports.ByCustomerID.NavTrails.HqCustomer:
		hqTeam := urls.Main.Hq.Teams.ByTeamID.Bind(team.ID)
		trail = goldr.NavTrail{
			goldr.NavStep("Home", urls.Root.Path()),
			goldr.NavStep("HQ", urls.Main.Hq.Path()),
			goldr.NavStep(team.Name, hqTeam.Path()),
			goldr.NavStep(customer.Name, hqTeam.Customers.ByCustomerID.Bind(customer.ID).Path()),
			goldr.CurrentNavStep("Report"),
		}
	case urls.Main.Reports.ByCustomerID.NavTrails.RegionalCustomer:
		office := store.Default.Office(team.OfficeID)
		regionalOffice := urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID)
		regionalTeam := regionalOffice.Teams.ByTeamID.Bind(team.ID)
		trail = goldr.NavTrail{
			goldr.NavStep("Home", urls.Root.Path()),
			goldr.NavStep("Regional", urls.Main.Regional.Path()),
			goldr.NavStep(office.Name, regionalOffice.Path()),
			goldr.NavStep(team.Name, regionalTeam.Path()),
			goldr.NavStep(customer.Name, regionalTeam.Customers.ByCustomerID.Bind(customer.ID).Path()),
			goldr.CurrentNavStep("Report"),
		}
	}

	return goldr.NewPage(ReportPage(trail, customer), goldr.PageMetadata{Title: "Report"})
}
```

The default trail handles direct links and clean URLs. Contextual trails
handle links that intentionally came from a workflow route. Binding generated
route helpers into local variables keeps trail code readable while remaining
route-safe.

When a trail step points at a route whose next param is already on the current
request, use `goldr.BindFromRequest` instead of spelling out `r.PathValue(...)`:

```go
teamURL, ok := goldr.BindFromRequest(r, urls.Main.Hq.Teams.ByTeamID)
if !ok {
	teamURL = urls.Main.Hq.Teams.ByTeamID.Bind(team.ID)
}

trail := goldr.NavTrail{
	goldr.NavStep("Home", urls.Root.Path()),
	goldr.NavStep("HQ", urls.Main.Hq.Path()),
	goldr.NavStep(team.Name, teamURL.Path()),
	goldr.CurrentNavStep(customer.Name),
}
```

`goldr.BindFromRequest` binds only one dynamic node. For nested dynamic routes,
bind the parent node first, then bind the child node from the bound parent. It
does not infer breadcrumb meaning or bind unrelated parent and child helpers at
once.

## Source Route

The source route declares the navigation edge with `Destinations`:

```go
var Route = goldr.RouteDef{
	Page: Page,
	Destinations: goldr.Destinations{
		"shared-report": goldr.To(urls.Main.Reports.ByCustomerID).
			NavTrail("hq-customer"),
	},
}
```

The generator checks that:

- the target is a live generated route;
- the selected trail key is allowed by that target;
- the generated destination helper name is unique for the source route.

The source handler uses the generated helper:

```go
func Page(r *http.Request) goldr.PageRouteResponse {
	customer := store.Default.Customer(r.PathValue("customer_id"))
	reportHref := urls.Main.Hq.Teams.ByTeamID.Customers.ByCustomerID.
		Destinations.SharedReport.
		Bind(customer.ID).
		Href()

	return goldr.NewPage(CustomerPage(customer, reportHref), goldr.PageMetadata{Title: customer.Name})
}
```

Destination `Bind` calls bind the target route parameters in target route
order. Do not type `_goldr_trail` by hand; use the generated destination
`Href()` helper, or `HrefWithQuery(url.Values)` when the same link also carries
app-owned query state.

## Rendering Links

For normal page navigation, render a normal anchor:

```templ
templ CustomerPage(customer Customer, reportHref string) {
	<a href={ reportHref }>Open report</a>
}
```

Use `hx-get` only when the anchor is intentionally an HTMX partial request:

```templ
templ CustomerRow(customer Customer, reportHref string) {
	<a href={ reportHref } hx-get={ reportHref } hx-target="#main" hx-push-url="true">
		Open report
	</a>
}
```

HTMX remains visible in templates. Goldr only provides the route-safe URL.

## Rendering The Trail

`goldr.NavTrail` is just data. Render it as normal server HTML:

```templ
templ TrailNav(trail goldr.NavTrail) {
	<nav aria-label="Breadcrumb">
		<ol>
			for _, step := range trail {
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
}
```

Labels are app data. If a customer name changes, the page can load the current
name before building the trail. No previous page has to store or replay labels.

## App-Level Back

Semantic Back is not the same thing as `history.back()`. Use the nearest
previous linked step in the rendered trail. If there is no previous linked
step, do not render Back:

```go
backHref, ok := goldr.BackHref(trail)
```

```templ
templ BackLink(trail goldr.NavTrail) {
	if backHref, ok := goldr.BackHref(trail); ok {
		<a href={ backHref }>Back</a>
	}
}
```

This works for direct links, HTMX links, and mounted routes because it is based
on the same explicit trail the page already rendered. A fallback Back URL is
product policy, so choose it in app code only when the product explicitly wants
that behavior.

When Back should preserve workflow state, put that state into the parent step's
href before rendering the trail:

```go
query := goldr.QueryValues(r, "view", "sort", "page")
parentHref := urls.Admin.Analytics.Destinations.CustomerProfile.
	Bind(customer.ID).
	HrefWithQuery(query)

trail := goldr.NavTrail{
	goldr.NavStep("Customers", parentHref),
	goldr.CurrentNavStep(customer.Name),
}
```

Goldr does not infer which app query keys belong on a parent link. Keep that
policy explicit at the route or kit boundary.

## Mounted Routes

Mounted routes are reusable implementation. They do not own live URLs. A route
under `app/mounts` cannot declare live `NavTrails` or `Destinations`.

The live owner declares trail metadata on the selected mounted children:

```go
var Route = goldr.KitRouteMount[analytics.Kit]{
	New:   newKit,
	Mount: "analytics",
	Routes: goldr.MountRoutes{
		{Path: "/"},
		{
			Path: "/customers/{customer_id}/report",
			NavTrails: goldr.NavTrails{
				Allowed: []string{"hq-analytics"},
			},
		},
	},
	Destinations: goldr.Destinations{
		"customer-report": goldr.To(
			urls.Main.Hq.Teams.ByTeamID.Analytics.Customers.ByCustomerID.Report,
		).NavTrail("hq-analytics"),
	},
}
```

The live owner passes owner-specific trail behavior through the kit value:

```go
func newKit(r *http.Request) analytics.Kit {
	team := store.Default.Team(r.PathValue("team_id"))
	teamURL, ok := goldr.BindFromRequest(r, urls.Main.Hq.Teams.ByTeamID)
	if !ok {
		teamURL = urls.Main.Hq.Teams.ByTeamID.Bind(team.ID)
	}
	return analytics.Kit{
		TrailBase: func(*http.Request) goldr.NavTrail {
			return goldr.NavTrail{
				goldr.NavStep("Home", urls.Root.Path()),
				goldr.NavStep("HQ", urls.Main.Hq.Path()),
				goldr.NavStep(team.Name, teamURL.Path()),
			}
		},
		AnalyticsURL: func() string {
			return teamURL.Analytics.Path()
		},
		CustomerURL: func(customerID string) string {
			return teamURL.Customers.ByCustomerID.Bind(customerID).Path()
		},
		CustomerReportHref: func(customerID string) string {
			return urls.Main.Hq.Teams.ByTeamID.Analytics.Destinations.CustomerReport.
				Bind(team.ID).
				Bind(customerID).
				Href()
		},
	}
}
```

The mounted implementation appends its local suffix:

```go
func (kit Kit) CustomerReport(r *http.Request) goldr.PageRouteResponse {
	customer := kit.Store.Customer(r.PathValue("customer_id"))
	trail := append(kit.TrailBase(r),
		goldr.NavStep("Analytics", kit.AnalyticsURL()),
		goldr.NavStep(customer.Name, kit.CustomerURL(customer.ID)),
		goldr.CurrentNavStep("Report"),
	)
	return goldr.NewPage(ReportPage(trail, customer), goldr.PageMetadata{Title: "Report"})
}
```

This lets the same mounted implementation work under different owners with
different trail lengths.

For shared mounted UI that must preserve app-owned query values, keep the same
owner-callback pattern and pass `url.Values` through the kit-owned function
type:

```go
type Kit struct {
	CustomerReportHref func(customerID string, query url.Values) string
}

func newKit(r *http.Request) analytics.Kit {
	team := store.Default.Team(r.PathValue("team_id"))
	return analytics.Kit{
		CustomerReportHref: func(customerID string, query url.Values) string {
			return urls.Main.Hq.Teams.ByTeamID.Analytics.Destinations.CustomerReport.
				Bind(team.ID).
				Bind(customerID).
				HrefWithQuery(query)
		},
	}
}
```

The mounted implementation decides which app query keys to carry and asks the
owner callback to compose the final URL:

```go
query := goldr.QueryValues(r, "view", "sort", "page")
href := kit.CustomerReportHref(customer.ID, query)
```

## Checklist

When adding contextual navigation:

- declare `NavTrails.Allowed` on the target route;
- declare `Destinations` on the source route;
- use generated destination `Href()` only when the link should select context;
- use generated destination `HrefWithQuery(url.Values)` when the link should
  select context and carry app-owned query state;
- use `goldr.QueryValues(r, ...)` when that query state should be copied from
  the current request;
- use `goldr.NavTrailSelected(r, key)` for one-off selected-trail checks and
  `goldr.NavTrailKey(r)` when switching across multiple trail keys;
- use `Path()` only for canonical links and rendered trail steps;
- build the concrete `NavTrail` from app data in the target handler;
- render breadcrumbs and Back as normal anchors, and hide Back when
  `goldr.BackHref` returns `ok == false`;
- keep mounted route navigation owned by the live route owner and passed
  through the kit value.

See `examples/navigation` for a complete in-memory example with HQ and
Regional owners mounting the same analytics and report implementations.
