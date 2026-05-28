package status_options

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

const selectedQuery = "selected"

var Route = goldr.RouteDef{
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/", Options),
	},
}

func Options(r *http.Request) goldr.FragmentRouteResponse {
	return goldr.NewFragment(OptionsView(r.URL.Query().Get(selectedQuery)))
}
