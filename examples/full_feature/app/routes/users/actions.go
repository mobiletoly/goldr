package users

import (
	"net/http"

	"github.com/mobiletoly/goldr"
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
	created := !form.HasErrors()
	if created {
		AddContact(form.Value("name"), form.Value("status"))
		form = bind.Form{}
	}

	response, err := goldr.Render(r, DirectoryView(form, ListContacts()))
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if created {
		hx.Trigger(w, "user:created")
	}
	hx.Retarget(w, "#users-directory")
	hx.Reswap(w, "outerHTML")
	_ = response.Write(w, r)
}

func PostSavePreview(w http.ResponseWriter, r *http.Request) {
	response, err := goldr.Render(r, FragTableView(ListContacts()))
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	hx.Trigger(w, "user:saved")
	hx.Retarget(w, "#users-table")
	hx.Reswap(w, "outerHTML")
	_ = response.Write(w, r)
}
