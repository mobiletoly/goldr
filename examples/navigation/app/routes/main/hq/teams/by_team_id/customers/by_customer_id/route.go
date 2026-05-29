package by_id

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/navigation/app/store"
	"github.com/mobiletoly/goldr/examples/navigation/app/ui"
	"github.com/mobiletoly/goldr/examples/navigation/app/urls"
)

var Route = goldr.RouteDef{
	Page: Page,
	Destinations: goldr.Destinations{
		"shared-report": goldr.To(urls.Main.Reports.ByCustomerID).
			NavTrail("hq-customer"),
	},
}

func Page(r *http.Request) goldr.PageRouteResponse {
	team := store.Default.Team(r.PathValue("team_id"))
	customer := store.Default.Customer(r.PathValue("customer_id"))
	trail := goldr.NavTrail{
		goldr.NavStep("Home", urls.Root.Path()),
		goldr.NavStep("HQ", urls.Main.Hq.Path()),
		goldr.NavStep(team.Name, urls.Main.Hq.Teams.ByTeamID.Bind(team.ID).Path()),
		goldr.CurrentNavStep(customer.Name),
	}
	return goldr.NewPage(ui.Page(customer.Name, trail, []ui.Link{
		{Label: "Report", Href: urls.Main.Hq.Teams.ByTeamID.Bind(team.ID).Customers.ByCustomerID.Bind(customer.ID).Report.Path()},
		{Label: "Shared report", Href: urls.Main.Hq.Teams.ByTeamID.Customers.ByCustomerID.Destinations.SharedReport.Bind(customer.ID).Href()},
	}, "Customer labels are loaded by the target page."), goldr.PageMetadata{Title: customer.Name})
}
