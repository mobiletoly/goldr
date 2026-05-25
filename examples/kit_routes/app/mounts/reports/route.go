package reports

import (
	"github.com/mobiletoly/goldr"
)

var Route = goldr.KitRouteDef[Kit]{
	Title: "Reports",
	Page:  goldr.KitPage(Kit.Page),
	Fragments: goldr.KitFragments[Kit]{
		goldr.KitFragment("table", Kit.Table),
	},
}
