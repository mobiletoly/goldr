package users

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/bind"
	"github.com/mobiletoly/goldr/examples/full_feature/app/deps"
)

func Page(r *http.Request) goldr.RouteResponse {
	appDeps := deps.From(r)
	return goldr.NewPage(
		PageView(bind.Form{}, ListContacts(), appDeps.CSRF.Token(r)),
		goldr.PageMetadata{
			Title:       "Users - Goldr Example",
			Description: "Browse and manage example contacts.",
		},
	)
}
