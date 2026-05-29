package brief

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	customerreport "github.com/mobiletoly/goldr/examples/navigation/app/mounts/customerreport"
)

var Route = goldr.KitRouteDef[customerreport.Kit]{
	Nav:  goldr.RouteNav{Label: "Brief"},
	Page: Page,
}

func Page(kit customerreport.Kit, r *http.Request) goldr.PageRouteResponse {
	customer := kit.Customer(r)
	return goldr.NewPage(PageView(customer), goldr.PageMetadata{Title: "Brief Customer Report"})
}
