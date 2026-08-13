package about

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

var Route = goldr.RouteDef{Page: Page}

func Page(_ *http.Request) goldr.PageRouteResponse {
	return goldr.NewPage(PageView(), goldr.PageMetadata{Title: "About - React Island"})
}
