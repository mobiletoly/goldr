package hq

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/navigation/app/store"
)

var Route = goldr.RouteDef{
	Page: Page,
	Nav:  goldr.RouteNav{Label: "HQ"},
}

func Page(r *http.Request) goldr.PageRouteResponse {
	team := store.Default.Team("hq-team")
	nav := goldr.Nav(r).Navigation()
	return goldr.NewPage(PageView(nav, team), goldr.PageMetadata{Title: "HQ"})
}
