package routes

import (
	"net/http"
	"strings"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/bind"
	"github.com/mobiletoly/goldr/examples/chat/app/session"
	"github.com/mobiletoly/goldr/examples/chat/app/urls"
)

func PostJoin(w http.ResponseWriter, r *http.Request) {
	form, err := bind.ParseForm(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(form.Value("name"))
	if name == "" {
		var errors bind.FieldErrors
		errors.Add("name", "Enter your name.")
		form = form.WithErrors(errors)
		response, err := goldr.Render(r, JoinForm(form))
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		_ = response.WriteStatus(w, r, http.StatusUnprocessableEntity)
		return
	}

	session.SetName(w, name)
	http.Redirect(w, r, urls.Chat.Path(), http.StatusSeeOther)
}
