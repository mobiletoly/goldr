package by_id

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/navigation/app/store"
	"github.com/mobiletoly/goldr/examples/navigation/app/urls"
)

var Route = goldr.RouteDef{
	Page: Page,
	Nav:  goldr.RouteNav{Key: "customer"},
	Destinations: goldr.Destinations{
		// The shared report route has no parent path context, so the source route
		// chooses the destination trail key that lets it rebuild the regional breadcrumb.
		"shared-report": goldr.To(urls.Main.Reports.ByCustomerID).
			TrailKey("regional-customer"),
	},
}

func Page(r *http.Request) goldr.PageRouteResponse {
	office := store.Default.Office(r.PathValue("office_id"))
	team := store.Default.Team(r.PathValue("team_id"))
	customer := store.Default.Customer(r.PathValue("customer_id"))
	nav := goldr.Nav(r)
	nav.Resolve("office", office.Name)
	nav.Resolve("team", team.Name)
	nav.Resolve("customer", customer.Name)
	navigation := nav.Navigation()
	return goldr.NewPage(PageView(navigation, office, team, customer), goldr.PageMetadata{Title: customer.Name})
}
