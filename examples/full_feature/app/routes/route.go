package routes

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
			Title:       "Goldr Example",
			Description: "Server-rendered pages, nested layouts, HTMX fragments, actions, and custom error views.",
		},
	)
}
