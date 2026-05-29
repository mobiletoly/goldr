package report

import (
	"net/http"
	"slices"

	"github.com/mobiletoly/goldr"
	analytics "github.com/mobiletoly/goldr/examples/navigation/app/mounts/analytics"
)

var Route = goldr.KitRouteDef[analytics.Kit]{
	Nav:  goldr.RouteNav{Label: "Report"},
	Page: Page,
}

func Page(kit analytics.Kit, r *http.Request) goldr.PageRouteResponse {
	customer := kit.Store.Customer(r.PathValue("customer_id"))
	nav := goldr.Nav(r)
	kit.ResolveNav(r, nav)
	trail := nav.Trail()
	// Goldr already supplies the owner route stack. This route inserts the
	// customer crumb because there is no separate customer page in this mounted flow.
	if len(trail) > 0 {
		trail = slices.Insert(trail, len(trail)-1, goldr.NavStep(customer.Name, kit.CustomerURL(customer.ID)))
	}
	navigation := nav.NavigationWithTrail(trail)
	return goldr.NewPage(PageView(navigation, customer), goldr.PageMetadata{Title: "Analytics Report"})
}
