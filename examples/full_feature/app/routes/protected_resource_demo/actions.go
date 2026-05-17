package protected_resource_demo

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/bind"
	"github.com/mobiletoly/goldr/csrf"
	"github.com/mobiletoly/goldr/examples/full_feature/app/security"
)

func PostSignOut(w http.ResponseWriter, r *http.Request) {
	form, err := bind.ParseForm(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := security.CSRF.Validate(r, form.Value(csrf.FieldName)); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	security.ClearDemoRole(w)
	http.Redirect(w, r, "/protected-resource-demo", http.StatusSeeOther)
}

func PostRevealSecret(w http.ResponseWriter, r *http.Request) {
	form, err := bind.ParseForm(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := security.CSRF.Validate(r, form.Value(csrf.FieldName)); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	role := security.DemoRole(r)
	if role == "" {
		http.Redirect(w, r, "/sign-in?next=%2Fprotected-resource-demo", http.StatusSeeOther)
		return
	}

	err = goldr.WriteRouteResponse(
		w,
		r,
		goldr.NewPage(
			SecretRevealView(role),
			goldr.PageMetadata{
				Title:       "One-time secret - Goldr Example",
				Description: "A full page response rendered from an action.",
			},
		).WithStatus(http.StatusCreated),
	)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
