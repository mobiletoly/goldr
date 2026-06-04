package routes

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/csrf"
	"github.com/mobiletoly/goldr/examples/full_feature/app/routes/internal/shelllayout"
)

const defaultTitle = "goldr full-feature example"

func Layout(r *http.Request, ctx goldr.LayoutContext) templ.Component {
	state, _ := goldr.LayoutValue(ctx, shelllayout.Key)
	return LayoutView(ctx.Metadata, state.ActiveNav, csrf.Token(r), ctx.Child)
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
