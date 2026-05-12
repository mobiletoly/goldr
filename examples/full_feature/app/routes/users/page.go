package users

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/bind"
)

func Page(_ *http.Request) goldr.Page {
	return goldr.Page{
		Component: PageView(bind.Form{}, ListContacts()),
		Metadata: goldr.PageMetadata{
			Title:       "Users - Goldr Example",
			Description: "Browse and manage example contacts.",
		},
	}
}
