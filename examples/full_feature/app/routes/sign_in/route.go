package sign_in

import (
	"net/http"
	"net/url"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/bind"
	"github.com/mobiletoly/goldr/csrf"
	"github.com/mobiletoly/goldr/examples/full_feature/app/deps"
	"github.com/mobiletoly/goldr/examples/full_feature/app/security"
)

const (
	defaultReturnPath       = "/protected-resource-demo"
	adminReturnPath         = "/admin"
	signInErrorCredentials  = "credentials"
	signInErrorQueryKey     = "error"
	signInCredentialField   = "credential"
	signInReturnPathField   = "next"
	signInCredentialsNotice = "Unknown credentials."
)

var Route = goldr.RouteDef{
	Page: Page,
	Actions: goldr.Actions{
		goldr.HTTPAction(http.MethodPost, "/", PostIndex),
	},
}

func Page(r *http.Request) goldr.RouteResponse {
	next := signInReturnPath(r)
	notice := ""
	if next == adminReturnPath {
		notice = "Sign in to open the protected admin page."
	}
	errorMessage := ""
	if r.URL.Query().Get(signInErrorQueryKey) == signInErrorCredentials {
		errorMessage = signInCredentialsNotice
	}

	return goldr.NewPage(
		PageView(csrf.Token(r), security.DemoRole(r), next, notice, errorMessage),
		goldr.PageMetadata{
			Title:       "Sign in - Goldr Example",
			Description: "Demo sign-in page for page-level redirect and status responses.",
		},
	)
}

func PostIndex(w http.ResponseWriter, r *http.Request) {
	form, ok := parseSignInForm(w, r)
	if !ok {
		return
	}

	next := cleanReturnPath(form.Value(signInReturnPathField))
	switch form.Value(signInCredentialField) {
	case security.RoleAdmin, security.RoleMember:
		security.SetDemoRole(w, form.Value(signInCredentialField))
		http.Redirect(w, r, next, http.StatusSeeOther)
	default:
		http.Redirect(w, r, signInErrorLocation(next), http.StatusSeeOther)
	}
}

func signInReturnPath(r *http.Request) string {
	if r == nil || r.URL == nil {
		return defaultReturnPath
	}
	return cleanReturnPath(r.URL.Query().Get(signInReturnPathField))
}

func parseSignInForm(w http.ResponseWriter, r *http.Request) (bind.Form, bool) {
	appDeps := deps.From(r)
	form, err := bind.ParseForm(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return bind.Form{}, false
	}
	if err := appDeps.CSRF.Validate(r, form.Value(csrf.FieldName)); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return bind.Form{}, false
	}
	return form, true
}

func cleanReturnPath(path string) string {
	switch path {
	case adminReturnPath, defaultReturnPath:
		return path
	default:
		return defaultReturnPath
	}
}

func signInErrorLocation(next string) string {
	return "/sign-in?next=" + url.QueryEscape(cleanReturnPath(next)) + "&" + signInErrorQueryKey + "=" + signInErrorCredentials
}
