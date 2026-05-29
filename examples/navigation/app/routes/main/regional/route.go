package regional

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/navigation/app/store"
)

var Route = goldr.RouteDef{
	Page: Page,
	Nav:  goldr.RouteNav{Label: "Regional"},
}

func Page(r *http.Request) goldr.PageRouteResponse {
	office := store.Default.Office("sea")
	nav := goldr.Nav(r).Navigation()
	return goldr.NewPage(PageView(nav, office), goldr.PageMetadata{Title: "Regional"})
}
