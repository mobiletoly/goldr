package users

import (
	"bytes"
	"net/http"

	"github.com/mobiletoly/goldr/bind"
)

func validateContactForm(form bind.Form) bind.FieldErrors {
	var errors bind.FieldErrors
	if form.Value("name") == "" {
		errors.Add("name", "Name is required.")
	}
	switch form.Value("status") {
	case statusActive, statusInactive:
	default:
		errors.Add("status", "Choose a valid status.")
	}
	return errors
}

func renderDirectory(w http.ResponseWriter, r *http.Request, form bind.Form) {
	component := DirectoryView(form, ListContacts())
	var body bytes.Buffer
	if err := component.Render(r.Context(), &body); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = body.WriteTo(w)
}
