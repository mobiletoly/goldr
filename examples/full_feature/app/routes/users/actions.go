package users

import (
	"net/http"

	"github.com/mobiletoly/goldr/bind"
	"github.com/mobiletoly/goldr/hx"
)

const (
	statusActive   = "Active"
	statusInactive = "Inactive"
)

func PostCreate(w http.ResponseWriter, r *http.Request) {
	form, err := bind.ParseForm(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	form = form.WithErrors(validateContactForm(form))
	if !form.HasErrors() {
		AddContact(form.Value("name"), form.Value("status"))
		form = bind.Form{}
		hx.Trigger(w, "user:created")
	}

	hx.Retarget(w, "#users-directory")
	hx.Reswap(w, "outerHTML")
	renderDirectory(w, r, form)
}

func PostSavePreview(w http.ResponseWriter, r *http.Request) {
	renderSavePreview(w, r)
}
