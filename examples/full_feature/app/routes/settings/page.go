package settings

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(_ *http.Request) goldr.Page {
	return goldr.Page{
		Component: PageView(),
		Metadata: goldr.PageMetadata{
			Title:       "Settings - Goldr Example",
			Description: "Application preferences and account controls.",
		},
	}
}
