package chat

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/bind"
	"github.com/mobiletoly/goldr/examples/chat/app/session"
)

func Page(r *http.Request) goldr.Page {
	name := session.Name(r)
	return goldr.Page{
		Component: PageView(name, bind.Form{}, listMessages()),
		Metadata: goldr.PageMetadata{
			Title:       "Chat - Goldr Chat",
			Description: "A small server-sent events chat example for goldr.",
		},
	}
}
