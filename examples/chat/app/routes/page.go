package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/bind"
)

func Page(_ *http.Request) goldr.Page {
	return goldr.Page{
		Component: PageView(bind.Form{}),
		Metadata: goldr.PageMetadata{
			Title:       "Join Chat - Goldr Chat",
			Description: "Enter a display name for the goldr SSE chat example.",
		},
	}
}
