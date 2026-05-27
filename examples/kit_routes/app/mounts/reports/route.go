package reports

import (
	"github.com/mobiletoly/goldr"
)

var Route = goldr.KitRouteDef[Kit]{
	Title: "Reports",
	Page:  Kit.Page,
	Fragments: goldr.KitFragments[Kit]{
		goldr.KitFragmentRoute("/table", Kit.Table),
	},
}
