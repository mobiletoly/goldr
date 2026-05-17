package chat

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/bind"
	"github.com/mobiletoly/goldr/examples/chat/app/session"
)

func Page(r *http.Request) goldr.RouteResponse {
	name := session.Name(r)
	return goldr.NewPage(
		PageView(name, bind.Form{}, listMessages()),
		goldr.PageMetadata{
			Title:       "Chat - Goldr Chat",
			Description: "A small server-sent events chat example for goldr.",
		},
	)
}
