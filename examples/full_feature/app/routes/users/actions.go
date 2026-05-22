package users

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/bind"
	"github.com/mobiletoly/goldr/csrf"
	"github.com/mobiletoly/goldr/examples/full_feature/app/deps"
	"github.com/mobiletoly/goldr/hx"
)

const (
	statusActive   = "Active"
	statusInactive = "Inactive"

	maxContactFormBody   = 2 << 20
	maxContactFormMemory = 1 << 20
)

func PostCreate(w http.ResponseWriter, r *http.Request) {
	appDeps := deps.From(r)
	r.Body = http.MaxBytesReader(w, r.Body, maxContactFormBody)
	form, err := bind.ParseMultipartForm(r, maxContactFormMemory)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := appDeps.CSRF.Validate(r, form.Value(csrf.FieldName)); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
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

	if created {
		hx.Trigger(w, "user:created")
	}
	hx.Retarget(w, "#users-directory")
	hx.Reswap(w, "outerHTML")
	if form.HasErrors() {
		if err := goldr.WriteComponent(w, r, http.StatusUnprocessableEntity, DirectoryView(form, ListContacts(), appDeps.CSRF.Token(r))); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}
	if err := goldr.WriteComponent(w, r, http.StatusOK, DirectoryView(form, ListContacts(), appDeps.CSRF.Token(r))); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func PostSavePreview(w http.ResponseWriter, r *http.Request) {
	appDeps := deps.From(r)
	if err := appDeps.CSRF.Validate(r, ""); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	hx.Trigger(w, "user:saved")
	hx.Retarget(w, "#users-table-slot")
	hx.Reswap(w, "innerHTML")
	if err := goldr.WriteComponent(w, r, http.StatusOK, FragTableView(ListContacts())); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
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
