package users

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/csrf"
	"github.com/mobiletoly/goldr/examples/full_feature/app/deps"
	"github.com/mobiletoly/goldr/examples/full_feature/app/routes/internal/shelllayout"
	"github.com/mobiletoly/goldr/hx"
)

const (
	statusActive   = "Active"
	statusInactive = "Inactive"

	statusFilterQuery    = "status"
	statusFilterActive   = "active"
	statusFilterInactive = "inactive"

	maxContactFormBody   = 2 << 20
	maxContactFormMemory = 1 << 20
)

var Route = goldr.RouteDef{
	Page: Page,
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/table", FragTable),
	},
	Actions: goldr.Actions{
		goldr.HTTPAction(http.MethodPost, "/create", PostCreate),
		goldr.Action(http.MethodPost, "/save-preview", PostSavePreview),
	},
}

func Page(r *http.Request) goldr.PageRouteResponse {
	return goldr.WithLayoutValue(
		goldr.NewPage(
			PageView(contactForm{}, ListContacts(), csrf.Token(r)),
			goldr.PageMetadata{
				Title:       "Users - Goldr Example",
				Description: "Browse and manage example contacts.",
			},
		),
		shelllayout.Key,
		shelllayout.State{ActiveNav: shelllayout.NavUsers},
	)
}

func FragTable(r *http.Request) goldr.FragmentRouteResponse {
	return goldr.NewFragment(FragTableView(filteredContacts(r.URL.Query().Get(statusFilterQuery))))
}

func PostCreate(w http.ResponseWriter, r *http.Request) {
	appDeps := deps.From(r)
	r.Body = http.MaxBytesReader(w, r.Body, maxContactFormBody)
	if err := r.ParseMultipartForm(maxContactFormMemory); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := appDeps.CSRF.Validate(r, r.PostFormValue(csrf.FieldName)); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	avatarFilename, err := optionalUploadFilename(r, "avatar")
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	form := validateContactForm(contactForm{
		Name:   r.PostFormValue("name"),
		Status: r.PostFormValue("status"),
	})
	created := !form.hasErrors()
	if created {
		AddContact(form.Name, form.Status, avatarFilename)
		form = contactForm{}
	}

	if created {
		hx.Trigger(w, "user:created")
	}
	hx.Retarget(w, "#users-directory")
	hx.Reswap(w, "outerHTML")
	if form.hasErrors() {
		if err := goldr.WriteComponent(w, r, http.StatusUnprocessableEntity, DirectoryView(form, ListContacts(), csrf.Token(r))); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}
	if err := goldr.WriteComponent(w, r, http.StatusOK, DirectoryView(form, ListContacts(), csrf.Token(r))); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func PostSavePreview(r *http.Request) goldr.RouteResponse {
	appDeps := deps.From(r)
	if err := appDeps.CSRF.Validate(r, ""); err != nil {
		return goldr.Text{Status: http.StatusForbidden, Body: "forbidden"}
	}

	return goldr.NewFragment(FragTableView(ListContacts())).
		WithHeader(hx.HeaderTrigger, "user:saved").
		WithHeader(hx.HeaderRetarget, "#users-table-slot").
		WithHeader(hx.HeaderReswap, "innerHTML")
}

func filteredContacts(filter string) []Contact {
	contacts := ListContacts()
	switch filter {
	case statusFilterActive:
		return contactsWithStatus(contacts, statusActive)
	case statusFilterInactive:
		return contactsWithStatus(contacts, statusInactive)
	default:
		return contacts
	}
}

func contactsWithStatus(contacts []Contact, status string) []Contact {
	var filtered []Contact
	for _, contact := range contacts {
		if contact.Status == status {
			filtered = append(filtered, contact)
		}
	}
	return filtered
}

func optionalUploadFilename(r *http.Request, field string) (string, error) {
	if r.MultipartForm == nil || len(r.MultipartForm.File[field]) == 0 {
		return "", nil
	}
	header := r.MultipartForm.File[field][0]
	file, err := header.Open()
	if err != nil {
		return "", err
	}
	defer func() {
		_ = file.Close()
	}()
	return header.Filename, nil
}
