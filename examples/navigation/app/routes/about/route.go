package about

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

var Route = goldr.RouteDef{
	Page: Page,
	Nav:  goldr.RouteNav{Label: "About"},
}

func Page(r *http.Request) goldr.PageRouteResponse {
	nav := goldr.Nav(r).Navigation()
	return goldr.NewPage(PageView(nav), goldr.PageMetadata{Title: "About"})
}
