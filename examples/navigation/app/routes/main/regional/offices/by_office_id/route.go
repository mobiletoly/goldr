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
	team := store.Default.Team("regional-team")
	trail := goldr.NavTrail{
		goldr.NavStep("Home", urls.Root.Path()),
		goldr.NavStep("Regional", urls.Main.Regional.Path()),
		goldr.CurrentNavStep(office.Name),
	}
	return goldr.NewPage(ui.Page(office.Name, trail, []ui.Link{
		{Label: team.Name, Href: urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID).Teams.ByTeamID.Bind(team.ID).Path()},
	}, "Regional office context is loaded from the store."), goldr.PageMetadata{Title: office.Name})
}
