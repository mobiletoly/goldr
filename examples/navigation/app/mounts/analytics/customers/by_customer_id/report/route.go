package report

import (
	"github.com/mobiletoly/goldr"
	analytics "github.com/mobiletoly/goldr/examples/navigation/app/mounts/analytics"
)

var Route = goldr.KitRouteDef[analytics.Kit]{
	Page: analytics.Kit.CustomerReport,
}
