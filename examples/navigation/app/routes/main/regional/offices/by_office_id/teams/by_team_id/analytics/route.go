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
	Mount: "analytics",
	Routes: goldr.MountRoutes{
		{Path: "/"},
		{
			Path: "/customers/{customer_id}/report",
			NavTrails: goldr.NavTrails{
				Allowed: []string{"regional-analytics"},
			},
		},
	},
	Destinations: goldr.Destinations{
		"customer-report": goldr.To(urls.Main.Regional.Offices.ByOfficeID.Teams.ByTeamID.Analytics.Customers.ByCustomerID.Report).
			NavTrail("regional-analytics"),
	},
}

func newKit(r *http.Request) analyticskit.Kit {
	office := store.Default.Office(r.PathValue("office_id"))
	team := store.Default.Team(r.PathValue("team_id"))
	return analyticskit.Kit{
		Store: store.Default,
		TrailBase: func(*http.Request) goldr.NavTrail {
			return goldr.NavTrail{
				goldr.NavStep("Home", urls.Root.Path()),
				goldr.NavStep("Regional", urls.Main.Regional.Path()),
				goldr.NavStep(office.Name, urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID).Path()),
				goldr.NavStep(team.Name, urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID).Teams.ByTeamID.Bind(team.ID).Path()),
			}
		},
		AnalyticsURL: func() string {
			return urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID).Teams.ByTeamID.Bind(team.ID).Analytics.Path()
		},
		CustomerURL: func(customerID string) string {
			return urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID).Teams.ByTeamID.Bind(team.ID).Customers.ByCustomerID.Bind(customerID).Path()
		},
		CustomerReportHref: func(customerID string) string {
			return urls.Main.Regional.Offices.ByOfficeID.Teams.ByTeamID.Analytics.Destinations.CustomerReport.Bind(office.ID).Bind(team.ID).Bind(customerID).Href()
		},
	}
}
