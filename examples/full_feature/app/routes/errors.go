package routes

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func NotFound(w http.ResponseWriter, r *http.Request) {
	renderError(w, r, http.StatusNotFound, NotFoundPage(r))
}

func NotFoundPage(r *http.Request) templ.Component {
	return Layout(r, goldr.LayoutContext{
		Child: NotFoundView(r.URL.EscapedPath()),
		Metadata: goldr.PageMetadata{
			Title:       "Page not found - Goldr Example",
			Description: "No goldr route matches this path.",
		},
	})
}

func renderError(w http.ResponseWriter, r *http.Request, status int, component templ.Component) {
	if err := goldr.WriteComponent(w, r, status, component); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
