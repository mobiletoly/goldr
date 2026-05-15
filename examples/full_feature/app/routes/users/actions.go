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

	maxContactFormBody   = 2 << 20
	maxContactFormMemory = 1 << 20
)

func PostCreate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxContactFormBody)
	form, err := bind.ParseMultipartForm(r, maxContactFormMemory)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	avatarFilename, err := optionalUploadFilename(form, "avatar")
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	form = form.WithErrors(validateContactForm(form))
	created := !form.HasErrors()
	if created {
		AddContact(form.Value("name"), form.Value("status"), avatarFilename)
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
	if form.HasErrors() {
		_ = response.WriteStatus(w, r, http.StatusUnprocessableEntity)
		return
	}
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

func optionalUploadFilename(form bind.Form, field string) (string, error) {
	if len(form.Files(field)) == 0 {
		return "", nil
	}
	file, header, err := form.File(field)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = file.Close()
	}()
	return header.Filename, nil
}
