package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func NotFound(r *http.Request) goldr.RouteResponse {
	return goldr.NewPage(
		NotFoundView(r.URL.EscapedPath()),
		goldr.PageMetadata{
			Title:       "Page not found - Goldr Example",
			Description: "No goldr route matches this path.",
		},
	).WithStatus(http.StatusNotFound)
}
