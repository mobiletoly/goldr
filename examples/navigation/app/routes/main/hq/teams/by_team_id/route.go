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
	team := store.Default.Team(r.PathValue("team_id"))
	nav := goldr.Nav(r)
	nav.Resolve("team", team.Name)
	navigation := nav.Navigation()
	return goldr.NewPage(PageView(navigation, team), goldr.PageMetadata{Title: team.Name})
}
