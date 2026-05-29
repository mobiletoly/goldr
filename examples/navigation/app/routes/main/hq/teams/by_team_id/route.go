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
	team := store.Default.Team(r.PathValue("team_id"))
	trail := goldr.NavTrail{
		goldr.NavStep("Home", urls.Root.Path()),
		goldr.NavStep("HQ", urls.Main.Hq.Path()),
		goldr.CurrentNavStep(team.Name),
	}
	return goldr.NewPage(ui.Page(team.Name, trail, []ui.Link{
		{Label: "Analytics", Href: urls.Main.Hq.Teams.ByTeamID.Bind(team.ID).Analytics.Path()},
		{Label: "Customer", Href: urls.Main.Hq.Teams.ByTeamID.Bind(team.ID).Customers.ByCustomerID.Bind("contoso").Path()},
	}, "HQ team context is loaded from the store."), goldr.PageMetadata{Title: team.Name})
}
