package by_id

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/navigation/app/store"
)

var Route = goldr.RouteDef{
	Page: Page,
	Nav:  goldr.RouteNav{Key: "office"},
}

func Page(r *http.Request) goldr.PageRouteResponse {
	office := store.Default.Office(r.PathValue("office_id"))
	team := store.Default.Team("regional-team")
	nav := goldr.Nav(r)
	nav.Resolve("office", office.Name)
	navigation := nav.Navigation()
	return goldr.NewPage(PageView(navigation, office, team), goldr.PageMetadata{Title: office.Name})
}
