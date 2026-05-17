package settings

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(_ *http.Request) goldr.RouteResponse {
	return goldr.NewPage(
		PageView(),
		goldr.PageMetadata{
			Title:       "Settings - Goldr Example",
			Description: "Application preferences and account controls.",
		},
	)
}
