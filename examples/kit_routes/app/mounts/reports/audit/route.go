package audit

import (
	"github.com/mobiletoly/goldr"
	goldrmount_reports "github.com/mobiletoly/goldr/examples/kit_routes/app/mounts/reports"
)

var Route = goldr.KitRouteDef[goldrmount_reports.Kit]{
	Title: "Admin Report Tools",
	Page:  goldrmount_reports.Kit.Audit,
}
