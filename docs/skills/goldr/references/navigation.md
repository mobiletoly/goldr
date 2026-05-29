# Navigation Trails And Destinations

Use navigation trails for breadcrumb-style presentation data and app-level Back
links in Goldr apps.

Rules:

- Use clean route `Path()` helpers for canonical links.
- Use generated destination `Href()` helpers when a source route intentionally
  selects a target route navigation context.
- Use generated destination `HrefWithQuery(url.Values)` helpers when that same
  workflow link also carries app-owned query state.
- Use `goldr.QueryValues(r, ...)` to copy named app-owned query keys from the
  current request before passing them to `HrefWithQuery`. It skips
  `_goldr_trail`.
- Build `goldr.NavTrail` explicitly in handlers or app-owned view models with
  `goldr.NavStep` and `goldr.CurrentNavStep`.
- Use `goldr.NavTrailSelected(r, key)` for one-off selected-trail checks.
- Read selected context with `goldr.NavTrailKey(r)` when switching across
  route-scoped generated constants such as
  `urls.Regional.Customers.ByID.Report.NavTrails.AttentionCenter`.
- Render normal links with `href` only. Add `hx-get` only when the anchor is
  intentionally an HTMX swap trigger; keep HTMX attributes visible in `.templ`.
- Derive app-level Back with `goldr.BackHref(trail)` and render a normal
  anchor only when it returns `ok == true`.
- Do not add cookies, sessions, `Referer` parsing, browser history APIs, or
  Goldr-owned breadcrumb renderers for semantic navigation.

Target route setup:

```go
var Route = goldr.RouteDef{
	Page: Page,
	NavTrails: goldr.NavTrails{
		Allowed: []string{"hq-customer", "regional-customer"},
	},
}

func Page(r *http.Request) goldr.PageRouteResponse {
	customer := loadCustomer(r.PathValue("customer_id"))

	trail := goldr.NavTrail{
		goldr.NavStep("Home", urls.Root.Path()),
		goldr.NavStep("Reports", urls.Main.Path()),
		goldr.CurrentNavStep(customer.Name),
	}
	switch goldr.NavTrailKey(r) {
	case urls.Main.Reports.ByCustomerID.NavTrails.HqCustomer:
		trail = hqCustomerTrail(customer)
	case urls.Main.Reports.ByCustomerID.NavTrails.RegionalCustomer:
		trail = regionalCustomerTrail(customer)
	}

	return goldr.NewPage(PageView(trail, customer), goldr.PageMetadata{Title: "Report"})
}
```

For one selected-trail branch, prefer a boolean helper:

```go
if goldr.NavTrailSelected(r, urls.Main.Reports.ByCustomerID.NavTrails.HqCustomer) {
	parentHref = hqParentHref(customer)
}
```

Source route setup:

```go
var Route = goldr.RouteDef{
	Page: Page,
	Destinations: goldr.Destinations{
		"shared-report": goldr.To(urls.Main.Reports.ByCustomerID).
			NavTrail("hq-customer"),
	},
}

func Page(r *http.Request) goldr.PageRouteResponse {
	customer := loadCustomer(r.PathValue("customer_id"))
	href := urls.Main.Hq.Customers.ByCustomerID.Destinations.SharedReport.
		Bind(customer.ID).
		Href()
	return goldr.NewPage(CustomerView(customer, href), goldr.PageMetadata{Title: customer.Name})
}
```

When the link also needs app-owned query state, compose that state with the
same destination helper:

```go
func Page(r *http.Request) goldr.PageRouteResponse {
	customer := loadCustomer(r.PathValue("customer_id"))
	query := goldr.QueryValues(r, "view", "sort", "page")

	href := urls.Main.Hq.Customers.ByCustomerID.Destinations.SharedReport.
		Bind(customer.ID).
		HrefWithQuery(query)
	return goldr.NewPage(CustomerView(customer, href), goldr.PageMetadata{Title: customer.Name})
}
```

Goldr owns only `_goldr_trail`. Do not pass `_goldr_trail` as app state;
generated destination helpers ignore app-supplied values for that key.
Use `url.Values` directly when the query values come from loaded app state
instead of the current request.

Template shape:

```templ
templ CustomerView(customer Customer, reportHref string) {
	<a href={ reportHref }>Open report</a>
}
```

If that link is intentionally an HTMX partial update, keep the same canonical
URL visible in `href` and use `hx-get` for the HTMX request behavior:

```templ
templ CustomerRow(customer Customer, reportHref string) {
	<a href={ reportHref } hx-get={ reportHref } hx-target="#main" hx-push-url="true">Open report</a>
}
```

Back link:

```go
backHref, ok := goldr.BackHref(trail)
```

Do not make fallback URLs part of Goldr navigation. If a product wants a Back
fallback, choose that URL explicitly in app rendering code.

Back links that need query state should put that state into the rendered trail
step explicitly:

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

Breadcrumb-style rendering:

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

Mounted routes:

- Mounted source routes under `app/mounts` cannot declare live `NavTrails` or
  `Destinations`.
- Live owners attach owner-specific nav trails to structured
  `goldr.MountRoute` entries.
- Generated mount helpers stay path-only.
- Pass owner-specific trail bases and URL callbacks through `K`; let the
  mounted implementation append local suffix steps.
- If mounted UI must preserve app query state, make the kit-owned URL callback
  accept `url.Values`. The mounted code selects keys with
  `goldr.QueryValues(r, ...)`; the live owner composes the final destination
  `HrefWithQuery`.

Owner route:

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
		"customer-report": goldr.To(urls.Main.Hq.Teams.ByID.Analytics.Customers.ByID.Report).
			NavTrail("hq-analytics"),
	},
}

func newKit(r *http.Request) analytics.Kit {
	team := loadTeam(r.PathValue("team_id"))
	return analytics.Kit{
		TrailBase: func(*http.Request) goldr.NavTrail {
			return goldr.NavTrail{
				goldr.NavStep("Home", urls.Root.Path()),
				goldr.NavStep("HQ", urls.Main.Hq.Path()),
				goldr.NavStep(team.Name, urls.Main.Hq.Teams.ByID.Bind(team.ID).Path()),
			}
		},
		AnalyticsURL: func() string {
			return urls.Main.Hq.Teams.ByID.Bind(team.ID).Analytics.Path()
		},
		CustomerURL: func(customerID string) string {
			return urls.Main.Hq.Teams.ByID.Bind(team.ID).Customers.ByID.Bind(customerID).Path()
		},
		CustomerReportHref: func(customerID string) string {
			return urls.Main.Hq.Teams.ByID.Analytics.Destinations.CustomerReport.
				Bind(team.ID).
				Bind(customerID).
				Href()
		},
	}
}
```

Mounted implementation:

```go
type Kit struct {
	TrailBase          func(*http.Request) goldr.NavTrail
	AnalyticsURL       func() string
	CustomerURL        func(string) string
	CustomerReportHref func(string) string
}

func (kit Kit) CustomerReport(r *http.Request) goldr.PageRouteResponse {
	customer := loadCustomer(r.PathValue("customer_id"))
	trail := append(kit.TrailBase(r),
		goldr.NavStep("Analytics", kit.AnalyticsURL()),
		goldr.NavStep(customer.Name, kit.CustomerURL(customer.ID)),
		goldr.CurrentNavStep("Report"),
	)
	return goldr.NewPage(ReportView(trail, customer), goldr.PageMetadata{Title: "Report"})
}
```

For a complete pattern, inspect `examples/navigation`.
