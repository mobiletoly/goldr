package sign_in

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/full_feature/app/security"
)

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
		PageView(security.CSRF.Token(r), security.DemoRole(r), next, notice, errorMessage),
		goldr.PageMetadata{
			Title:       "Sign in - Goldr Example",
			Description: "Demo sign-in page for page-level redirect and status responses.",
		},
	)
}

func signInReturnPath(r *http.Request) string {
	if r == nil || r.URL == nil {
		return defaultReturnPath
	}
	return cleanReturnPath(r.URL.Query().Get(signInReturnPathField))
}
