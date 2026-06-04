package admin

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/full_feature/app/routes/internal/shelllayout"
	"github.com/mobiletoly/goldr/examples/full_feature/app/security"
)

var Route = goldr.RouteDef{
	Page: Page,
}

var errDemoAdminLoad = errors.New("demo admin load failed")

func Page(r *http.Request) goldr.PageRouteResponse {
	if r.URL.Query().Get("demo_error") == "1" {
		return goldr.RouteError{Err: errDemoAdminLoad}
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
		return goldr.WithLayoutValue(
			goldr.NewPage(ForbiddenView(role), metadata).WithStatus(http.StatusForbidden),
			shelllayout.Key,
			shelllayout.State{ActiveNav: shelllayout.NavProtected},
		)
	}

	return goldr.WithLayoutValue(
		goldr.NewPage(PageView(role), metadata),
		shelllayout.Key,
		shelllayout.State{ActiveNav: shelllayout.NavProtected},
	)
}
