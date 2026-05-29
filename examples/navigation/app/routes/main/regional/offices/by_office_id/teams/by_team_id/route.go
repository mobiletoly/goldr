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
}

func Page(r *http.Request) goldr.PageRouteResponse {
	office := store.Default.Office(r.PathValue("office_id"))
	team := store.Default.Team(r.PathValue("team_id"))
	customer := store.Default.Customer("northwind")
	trail := goldr.NavTrail{
		goldr.NavStep("Home", urls.Root.Path()),
		goldr.NavStep("Regional", urls.Main.Regional.Path()),
		goldr.NavStep(office.Name, urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID).Path()),
		goldr.CurrentNavStep(team.Name),
	}
	return goldr.NewPage(ui.Page(team.Name, trail, []ui.Link{
		{Label: "Analytics", Href: urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID).Teams.ByTeamID.Bind(team.ID).Analytics.Path()},
		{Label: customer.Name, Href: urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID).Teams.ByTeamID.Bind(team.ID).Customers.ByCustomerID.Bind(customer.ID).Path()},
	}, "Regional team context includes office and team labels."), goldr.PageMetadata{Title: team.Name})
}
