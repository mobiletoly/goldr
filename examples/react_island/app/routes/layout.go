package routes

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/csrf"
)

func Layout(r *http.Request, ctx goldr.LayoutContext) templ.Component {
	return LayoutView(ctx.Metadata, csrf.Token(r), ctx.Child)
}
