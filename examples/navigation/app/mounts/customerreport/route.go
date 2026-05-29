package customerreport

import "github.com/mobiletoly/goldr"

var Route = goldr.KitRouteDef[Kit]{
	Nav:  goldr.RouteNav{Label: "Report"},
	Page: Kit.Page,
}
