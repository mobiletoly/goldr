package routes

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

const defaultTitle = "Goldr Chat"

func Layout(_ *http.Request, ctx goldr.LayoutContext) templ.Component {
	return LayoutView(ctx.Metadata, ctx.Child)
}

func pageTitle(metadata goldr.PageMetadata) string {
	if metadata.Title != "" {
		return metadata.Title
	}
	return defaultTitle
}
