package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

var Route = goldr.RouteDef{
	Page: Page,
}

func Page(_ *http.Request) goldr.RouteResponse {
	return goldr.NewPage(
		PageView(),
		goldr.PageMetadata{
			Title:       "Goldr Kit Routes",
			Description: "A focused example for shared KitRouteDef page implementations.",
		},
	)
}
