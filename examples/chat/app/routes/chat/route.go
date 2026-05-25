package chat

import (
	"net/http"
	"strings"
	"time"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/bind"
	"github.com/mobiletoly/goldr/examples/chat/app/session"
	"github.com/mobiletoly/goldr/examples/chat/app/urls"
)

const maxMessageBody = 64 << 10

var sendDelay = 3 * time.Second

var Route = goldr.RouteDef{
	Page: goldr.FuncPage(Page),
	Actions: goldr.FuncActions{
		goldr.FuncPostHandler("message", PostMessage),
		goldr.FuncPostHandler("sign-out", PostSignOut),
	},
}

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

func PostMessage(w http.ResponseWriter, r *http.Request) {
	name := session.Name(r)
	if name == "" {
		http.Error(w, "join the chat first", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxMessageBody)
	form, err := bind.ParseForm(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	body := strings.TrimSpace(form.Value("body"))
	if body == "" {
		var errors bind.FieldErrors
		errors.Add("body", "Enter a message.")
		form = form.WithErrors(errors)
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
	if err := goldr.WriteComponent(w, r, http.StatusOK, ComposerView(bind.Form{})); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func PostSignOut(w http.ResponseWriter, r *http.Request) {
	session.ClearName(w)
	http.Redirect(w, r, urls.Root.Path(), http.StatusSeeOther)
}
