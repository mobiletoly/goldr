package reports

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	sharedreports "github.com/mobiletoly/goldr/examples/kit_routes/app/mounts/reports"
	"github.com/mobiletoly/goldr/examples/kit_routes/app/urls"
)

var Route = goldr.KitRouteMount[sharedreports.Kit]{
	New:   newReportKit,
	Mount: "reports",
	Routes: goldr.MountRoutes{
		"/",
		"/audit",
	},
}

func newReportKit(r *http.Request) sharedreports.Kit {
	return sharedreports.New(reportData(r))
}

func reportData(_ *http.Request) sharedreports.ReportData {
	return sharedreports.ReportData{
		Audience:    "Admin",
		Heading:     "Admin Reports",
		Description: "Operational view across all teams.",
		URLs:        sharedreports.NewGoldrMountURLs(urls.Admin.Reports),
		ShowAudit:   true,
		Periods: []sharedreports.PeriodOption{
			{Value: "7d", Label: "Last 7 days"},
			{Value: "30d", Label: "Last 30 days"},
			{Value: "90d", Label: "Last 90 days"},
		},
		Rows: []sharedreports.Row{
			{Metric: "Revenue", Value: "$128,400", Note: "All teams"},
			{Metric: "Churn risk", Value: "4 accounts", Note: "Needs follow-up"},
		},
	}
}
