package analytics

import "github.com/mobiletoly/goldr"

var Route = goldr.KitRouteDef[Kit]{
	Nav:  goldr.RouteNav{Label: "Analytics"},
	Page: Kit.Page,
}
