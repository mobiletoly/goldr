package report

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	reportkit "github.com/mobiletoly/goldr/examples/navigation/app/mounts/customerreport"
	"github.com/mobiletoly/goldr/examples/navigation/app/store"
)

var Route = goldr.KitRouteMount[reportkit.Kit]{
	New:   newKit,
	Mount: "customerreport",
}

func newKit(_ *http.Request) reportkit.Kit {
	return reportkit.Kit{
		Store: store.Default,
	}
}
