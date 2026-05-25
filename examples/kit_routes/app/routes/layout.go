package routes

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

const defaultTitle = "Goldr Kit Routes"

func Layout(_ *http.Request, context goldr.LayoutContext) templ.Component {
	return LayoutView(context.Metadata, context.Child)
}

func pageTitle(metadata goldr.PageMetadata) string {
	if metadata.Title == "" {
		return defaultTitle
	}
	return metadata.Title + " - " + defaultTitle
}
