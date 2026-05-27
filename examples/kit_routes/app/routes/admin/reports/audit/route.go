package audit

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
			Title:       "Admin Report Tools",
			Description: "Owner-only report operations for admins.",
		},
	)
}
