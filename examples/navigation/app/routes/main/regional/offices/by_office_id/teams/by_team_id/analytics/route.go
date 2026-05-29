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
				// This is the same shared analytics kit as HQ, but this owner selects
				// the regional breadcrumb shape for its report destination.
				"customer-report": goldr.To(urls.Main.Regional.Offices.ByOfficeID.Teams.ByTeamID.Analytics.Customers.ByCustomerID.Report).
					TrailKey("regional-analytics"),
			},
		},
		{
			Path: "/customers/{customer_id}/report",
		},
	},
}

func newKit(r *http.Request) analyticskit.Kit {
	office := store.Default.Office(r.PathValue("office_id"))
	team := store.Default.Team(r.PathValue("team_id"))
	return analyticskit.Kit{
		Store: store.Default,
		AnalyticsURL: func() string {
			return urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID).Teams.ByTeamID.Bind(team.ID).Analytics.Path()
		},
		CustomerURL: func(customerID string) string {
			return urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID).Teams.ByTeamID.Bind(team.ID).Customers.ByCustomerID.Bind(customerID).Path()
		},
		CustomerReportHref: func(nav goldr.Navigation, customerID string) string {
			return urls.Main.Regional.Offices.ByOfficeID.Teams.ByTeamID.Analytics.Destinations.CustomerReport.Bind(office.ID).Bind(team.ID).Bind(customerID).NavigationHref(nav)
		},
	}
}
