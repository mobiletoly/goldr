package customerreport

import (
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, ctx goldr.LayoutContext) templ.Component {
	return LayoutView(goldr.Nav(r).Navigation(), reportURLs(r), ctx.Child)
}

func reportURLs(r *http.Request) GoldrMountURLs {
	if r == nil || r.URL == nil {
		return newGoldrMountURLs("")
	}
	path := strings.TrimSuffix(r.URL.Path, "/")
	path = strings.TrimSuffix(path, "/brief")
	path = strings.TrimSuffix(path, "/detailed")
	return newGoldrMountURLs(path)
}
