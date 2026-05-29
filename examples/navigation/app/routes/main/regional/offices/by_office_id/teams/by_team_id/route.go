package by_id

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/navigation/app/store"
)

var Route = goldr.RouteDef{
	Page: Page,
	Nav:  goldr.RouteNav{Key: "team"},
}

func Page(r *http.Request) goldr.PageRouteResponse {
	office := store.Default.Office(r.PathValue("office_id"))
	team := store.Default.Team(r.PathValue("team_id"))
	customer := store.Default.Customer("northwind")
	nav := goldr.Nav(r)
	nav.Resolve("office", office.Name)
	nav.Resolve("team", team.Name)
	navigation := nav.Navigation()
	return goldr.NewPage(PageView(navigation, office, team, customer), goldr.PageMetadata{Title: team.Name})
}
