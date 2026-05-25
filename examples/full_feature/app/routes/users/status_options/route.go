package status_options

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

const selectedQuery = "selected"

var Route = goldr.RouteDef{
	Fragments: goldr.FuncFragments{
		goldr.FuncFragmentIndex(Options),
	},
}

func Options(r *http.Request) goldr.RouteResponse {
	return goldr.NewFragment(OptionsView(r.URL.Query().Get(selectedQuery)))
}
