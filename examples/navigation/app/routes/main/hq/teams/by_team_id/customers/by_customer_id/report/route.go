package report

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	reportkit "github.com/mobiletoly/goldr/examples/navigation/app/mounts/report"
	"github.com/mobiletoly/goldr/examples/navigation/app/store"
	"github.com/mobiletoly/goldr/examples/navigation/app/urls"
)

var Route = goldr.KitRouteMount[reportkit.Kit]{
	New:   newKit,
	Mount: "report",
}

func newKit(r *http.Request) reportkit.Kit {
	team := store.Default.Team(r.PathValue("team_id"))
	customer := store.Default.Customer(r.PathValue("customer_id"))
	return reportkit.Kit{
		Store: store.Default,
		TrailBase: func(*http.Request) goldr.NavTrail {
			return goldr.NavTrail{
				goldr.NavStep("Home", urls.Root.Path()),
				goldr.NavStep("HQ", urls.Main.Hq.Path()),
				goldr.NavStep(customer.Name, urls.Main.Hq.Teams.ByTeamID.Bind(team.ID).Customers.ByCustomerID.Bind(customer.ID).Path()),
			}
		},
	}
}
