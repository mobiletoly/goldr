package settings

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

var Route = goldr.RouteDef{
	Page: goldr.FuncPage(Page),
}

func Page(_ *http.Request) goldr.RouteResponse {
	return goldr.NewPage(
		PageView(),
		goldr.PageMetadata{
			Title:       "Settings - Goldr Example",
			Description: "Application preferences and account controls.",
		},
	)
}
