package analytics

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	analyticskit "github.com/mobiletoly/goldr/examples/navigation/app/mounts/analytics"
	"github.com/mobiletoly/goldr/examples/navigation/app/store"
	"github.com/mobiletoly/goldr/examples/navigation/app/urls"
)

var Route = goldr.KitRouteMount[analyticskit.Kit]{
	New:   newKit,
	Mount: "analytics", // mounted route app/mounts/analytics
	Routes: goldr.MountRoutes{
		{
			Path: "/",
			Destinations: goldr.Destinations{
				// The destination selects the breadcrumb shape the report page should use
				// when a user enters it from this HQ analytics workflow.
				"customer-report": goldr.To(urls.Main.Hq.Teams.ByTeamID.Analytics.Customers.ByCustomerID.Report).
					TrailKey("hq-analytics"),
			},
		},
		{
			Path: "/customers/{customer_id}/report",
		},
	},
}

func newKit(r *http.Request) analyticskit.Kit {
	team := store.Default.Team(r.PathValue("team_id"))
	return analyticskit.Kit{
		Store: store.Default,
		AnalyticsURL: func() string {
			return urls.Main.Hq.Teams.ByTeamID.Bind(team.ID).Analytics.Path()
		},
		CustomerURL: func(customerID string) string {
			return urls.Main.Hq.Teams.ByTeamID.Bind(team.ID).Customers.ByCustomerID.Bind(customerID).Path()
		},
		CustomerReportHref: func(nav goldr.Navigation, customerID string) string {
			return urls.Main.Hq.Teams.ByTeamID.Analytics.Destinations.CustomerReport.Bind(team.ID).Bind(customerID).NavigationHref(nav)
		},
	}
}
