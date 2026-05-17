package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(_ *http.Request) goldr.Page {
	return goldr.RenderPage(
		PageView(),
		goldr.PageMetadata{
			Title:       "Goldr Example",
			Description: "Server-rendered pages, nested layouts, HTMX fragments, actions, and custom error views.",
		},
	)
}
