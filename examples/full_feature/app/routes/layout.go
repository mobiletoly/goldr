package routes

import (
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

const defaultTitle = "goldr full-feature example"

const (
	navUsers     = "users"
	navSettings  = "settings"
	navProtected = "protected"
	navSignIn    = "sign-in"
)

func Layout(r *http.Request, ctx goldr.LayoutContext) templ.Component {
	return LayoutView(ctx.Metadata, activeNav(r), ctx.Child)
}

func pageTitle(metadata goldr.PageMetadata) string {
	if metadata.Title != "" {
		return metadata.Title
	}
	return defaultTitle
}

func currentPageAttr(active bool) templ.Attributes {
	return templ.Attributes{
		"aria-current": templ.KV("page", active),
	}
}

func activeNav(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	switch {
	case r.URL.Path == "/users" || strings.HasPrefix(r.URL.Path, "/users/"):
		return navUsers
	case r.URL.Path == "/settings" || strings.HasPrefix(r.URL.Path, "/settings/"):
		return navSettings
	case r.URL.Path == "/protected-resource-demo" || strings.HasPrefix(r.URL.Path, "/protected-resource-demo/") ||
		r.URL.Path == "/admin" || strings.HasPrefix(r.URL.Path, "/admin/"):
		return navProtected
	case r.URL.Path == "/sign-in" || strings.HasPrefix(r.URL.Path, "/sign-in/"):
		return navSignIn
	default:
		return ""
	}
}
