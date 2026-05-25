package protected_resource_demo

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/bind"
	"github.com/mobiletoly/goldr/csrf"
	"github.com/mobiletoly/goldr/examples/full_feature/app/deps"
	"github.com/mobiletoly/goldr/examples/full_feature/app/security"
)

var Route = goldr.RouteDef{
	Page: goldr.FuncPage(Page),
	Actions: goldr.FuncActions{
		goldr.FuncPostHandler("sign-out", PostSignOut),
		goldr.FuncPost("reveal-secret", PostRevealSecret),
	},
}

func Page(r *http.Request) goldr.RouteResponse {
	appDeps := deps.From(r)
	return goldr.NewPage(
		PageView(appDeps.CSRF.Token(r), security.DemoRole(r)),
		goldr.PageMetadata{
			Title:       "Protected Resource Demo - Goldr Example",
			Description: "Sign in as different demo users before opening a protected page.",
		},
	)
}

func PostSignOut(w http.ResponseWriter, r *http.Request) {
	appDeps := deps.From(r)
	form, err := bind.ParseForm(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := appDeps.CSRF.Validate(r, form.Value(csrf.FieldName)); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	security.ClearDemoRole(w)
	http.Redirect(w, r, "/protected-resource-demo", http.StatusSeeOther)
}

func PostRevealSecret(r *http.Request) goldr.RouteResponse {
	appDeps := deps.From(r)
	form, err := bind.ParseForm(r)
	if err != nil {
		return goldr.Text{Status: http.StatusBadRequest, Body: "bad request"}
	}
	if err := appDeps.CSRF.Validate(r, form.Value(csrf.FieldName)); err != nil {
		return goldr.Text{Status: http.StatusForbidden, Body: "forbidden"}
	}

	role := security.DemoRole(r)
	if role == "" {
		return goldr.Redirect{Location: "/sign-in?next=%2Fprotected-resource-demo", Status: http.StatusSeeOther}
	}

	return goldr.NewPage(
		SecretRevealView(role),
		goldr.PageMetadata{
			Title:       "One-time secret - Goldr Example",
			Description: "A full page response rendered from an action.",
		},
	).WithStatus(http.StatusCreated)
}
