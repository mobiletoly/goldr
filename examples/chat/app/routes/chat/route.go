package chat

import (
	"net/http"
	"strings"
	"time"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/chat/app/session"
	"github.com/mobiletoly/goldr/examples/chat/app/urls"
)

const maxMessageBody = 64 << 10

var sendDelay = 3 * time.Second

type messageForm struct {
	Body      string
	BodyError string
}

var Route = goldr.RouteDef{
	Page: Page,
	Actions: goldr.Actions{
		goldr.HTTPAction(http.MethodPost, "/message", PostMessage),
		goldr.HTTPAction(http.MethodPost, "/sign-out", PostSignOut),
	},
}

func Page(r *http.Request) goldr.PageRouteResponse {
	name := session.Name(r)
	return goldr.NewPage(
		PageView(name, messageForm{}, listMessages()),
		goldr.PageMetadata{
			Title:       "Chat - Goldr Chat",
			Description: "A small server-sent events chat example for goldr.",
		},
	)
}

func PostMessage(w http.ResponseWriter, r *http.Request) {
	name := session.Name(r)
	if name == "" {
		http.Error(w, "join the chat first", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxMessageBody)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	form := messageForm{Body: r.PostFormValue("body")}
	body := strings.TrimSpace(form.Body)
	if body == "" {
		form.BodyError = "Enter a message."
		if err := goldr.WriteComponent(w, r, http.StatusUnprocessableEntity, ComposerView(form)); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	select {
	case <-time.After(sendDelay):
	case <-r.Context().Done():
		return
	}

	addMessage(name, body)
	if err := goldr.WriteComponent(w, r, http.StatusOK, ComposerView(messageForm{})); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func PostSignOut(w http.ResponseWriter, r *http.Request) {
	session.ClearName(w)
	http.Redirect(w, r, urls.Root.Path(), http.StatusSeeOther)
}
