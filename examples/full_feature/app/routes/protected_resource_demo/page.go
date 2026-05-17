package protected_resource_demo

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/full_feature/app/security"
)

func Page(r *http.Request) goldr.Page {
	return goldr.RenderPage(
		PageView(security.CSRF.Token(r), security.DemoRole(r)),
		goldr.PageMetadata{
			Title:       "Protected Resource Demo - Goldr Example",
			Description: "Sign in as different demo users before opening a protected page.",
		},
	)
}
