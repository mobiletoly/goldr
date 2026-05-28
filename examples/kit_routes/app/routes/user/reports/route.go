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
	},
}

func newReportKit(r *http.Request) sharedreports.Kit {
	return sharedreports.New(reportData(r))
}

func reportData(_ *http.Request) sharedreports.ReportData {
	return sharedreports.ReportData{
		Audience:    "User",
		Heading:     "User Reports",
		Description: "Personal report view for the signed-in user.",
		URLs:        sharedreports.NewGoldrMountURLs(urls.User.Reports),
		Periods: []sharedreports.PeriodOption{
			{Value: "7d", Label: "Last 7 days"},
			{Value: "30d", Label: "Last 30 days"},
			{Value: "90d", Label: "Last 90 days"},
		},
		Rows: []sharedreports.Row{
			{Metric: "My tasks", Value: "7 open", Note: "Due this week"},
			{Metric: "My usage", Value: "42 reports", Note: "Last 30 days"},
		},
	}
}
