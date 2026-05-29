package routes

import (
	"net/http"
	"strings"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/chat/app/session"
	"github.com/mobiletoly/goldr/examples/chat/app/urls"
)

type joinForm struct {
	Name      string
	NameError string
}

var Route = goldr.RouteDef{
	Page: Page,
	Actions: goldr.Actions{
		goldr.HTTPAction(http.MethodPost, "/join", PostJoin),
	},
}

func Page(_ *http.Request) goldr.PageRouteResponse {
	return goldr.NewPage(
		PageView(joinForm{}),
		goldr.PageMetadata{
			Title:       "Join Chat - Goldr Chat",
			Description: "Enter a display name for the goldr SSE chat example.",
		},
	)
}

func PostJoin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	form := joinForm{Name: r.PostFormValue("name")}
	name := strings.TrimSpace(form.Name)
	if name == "" {
		form.NameError = "Enter your name."
		if err := goldr.WriteComponent(w, r, http.StatusUnprocessableEntity, JoinForm(form)); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	session.SetName(w, name)
	http.Redirect(w, r, urls.Chat.Path(), http.StatusSeeOther)
}
