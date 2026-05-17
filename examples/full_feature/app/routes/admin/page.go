package admin

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/full_feature/app/security"
)

var errDemoAdminLoad = errors.New("demo admin load failed")

func Page(r *http.Request) goldr.Page {
	if r.URL.Query().Get("demo_error") == "1" {
		return goldr.Error(errDemoAdminLoad)
	}

	role := security.DemoRole(r)
	if role == "" {
		return goldr.Redirect("/sign-in?next="+url.QueryEscape("/admin"), http.StatusSeeOther)
	}

	metadata := goldr.PageMetadata{
		Title:       "Protected admin - Goldr Example",
		Description: "A page-level redirect and forbidden status example.",
	}
	if role != security.RoleAdmin {
		return goldr.Status(http.StatusForbidden, ForbiddenView(role), metadata)
	}

	return goldr.RenderPage(PageView(role), metadata)
}
