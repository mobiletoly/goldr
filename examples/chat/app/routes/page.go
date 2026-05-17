package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/bind"
)

func Page(_ *http.Request) goldr.RouteResponse {
	return goldr.NewPage(
		PageView(bind.Form{}),
		goldr.PageMetadata{
			Title:       "Join Chat - Goldr Chat",
			Description: "Enter a display name for the goldr SSE chat example.",
		},
	)
}
