package sign_in

import (
	"net/http"
	"net/url"

	"github.com/mobiletoly/goldr/bind"
	"github.com/mobiletoly/goldr/csrf"
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

func parseSignInForm(w http.ResponseWriter, r *http.Request) (bind.Form, bool) {
	form, err := bind.ParseForm(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return bind.Form{}, false
	}
	if err := security.CSRF.Validate(r, form.Value(csrf.FieldName)); err != nil {
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
