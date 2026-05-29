package by_customer_id

import (
	"net/http"
	"slices"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/navigation/app/store"
	"github.com/mobiletoly/goldr/examples/navigation/app/urls"
)

var Route = goldr.RouteDef{
	Page: Page,
	Nav:  goldr.RouteNav{Label: "Report"},
}

func Page(r *http.Request) goldr.PageRouteResponse {
	customer := store.Default.Customer(r.PathValue("customer_id"))
	team := store.Default.Team(customer.TeamID)

	nav := goldr.Nav(r)
	trail := nav.Trail()
	// A route may use the generated trail key constants instead of raw query
	// values when it needs workflow-specific navigation state.
	switch nav.TrailKey() {
	case urls.Main.Reports.ByCustomerID.TrailKeys.HqCustomer:
		if len(trail) > 0 {
			trail = slices.Insert(trail, len(trail)-1,
				goldr.NavStep("HQ", urls.Main.Hq.Path()),
				goldr.NavStep(team.Name, urls.Main.Hq.Teams.ByTeamID.Bind(team.ID).Path()),
				goldr.NavStep(customer.Name, urls.Main.Hq.Teams.ByTeamID.Bind(team.ID).Customers.ByCustomerID.Bind(customer.ID).Path()),
			)
		}
	case urls.Main.Reports.ByCustomerID.TrailKeys.RegionalCustomer:
		office := store.Default.Office(team.OfficeID)
		if len(trail) > 0 {
			trail = slices.Insert(trail, len(trail)-1,
				goldr.NavStep("Regional", urls.Main.Regional.Path()),
				goldr.NavStep(office.Name, urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID).Path()),
				goldr.NavStep(team.Name, urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID).Teams.ByTeamID.Bind(team.ID).Path()),
				goldr.NavStep(customer.Name, urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID).Teams.ByTeamID.Bind(team.ID).Customers.ByCustomerID.Bind(customer.ID).Path()),
			)
		}
	}

	navigation := nav.NavigationWithTrail(trail)
	return goldr.NewPage(PageView(navigation), goldr.PageMetadata{Title: "Shared Report"})
}
