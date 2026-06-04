package settings

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/full_feature/app/routes/internal/shelllayout"
)

var Route = goldr.RouteDef{
	Page: Page,
}

func Page(_ *http.Request) goldr.PageRouteResponse {
	return goldr.WithLayoutValue(
		goldr.NewPage(
			PageView(),
			goldr.PageMetadata{
				Title:       "Settings - Goldr Example",
				Description: "Application preferences and account controls.",
			},
		),
		shelllayout.Key,
		shelllayout.State{ActiveNav: shelllayout.NavSettings},
	)
}
