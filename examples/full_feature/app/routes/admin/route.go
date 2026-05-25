package admin

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/full_feature/app/security"
)

var Route = goldr.RouteDef{
	Page: goldr.FuncPage(Page),
}

var errDemoAdminLoad = errors.New("demo admin load failed")

func Page(r *http.Request) goldr.RouteResponse {
	if r.URL.Query().Get("demo_error") == "1" {
		return goldr.ServerError{Err: errDemoAdminLoad}
	}

	role := security.DemoRole(r)
	if role == "" {
		return goldr.Redirect{Location: "/sign-in?next=" + url.QueryEscape("/admin"), Status: http.StatusSeeOther}
	}

	metadata := goldr.PageMetadata{
		Title:       "Protected admin - Goldr Example",
		Description: "A page-level redirect and forbidden status example.",
	}
	if role != security.RoleAdmin {
		return goldr.NewPage(ForbiddenView(role), metadata).WithStatus(http.StatusForbidden)
	}

	return goldr.NewPage(PageView(role), metadata)
}
