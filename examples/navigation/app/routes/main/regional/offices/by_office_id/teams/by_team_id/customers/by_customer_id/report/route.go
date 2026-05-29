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
	office := store.Default.Office(r.PathValue("office_id"))
	team := store.Default.Team(r.PathValue("team_id"))
	customer := store.Default.Customer(r.PathValue("customer_id"))
	return reportkit.Kit{
		Store: store.Default,
		TrailBase: func(*http.Request) goldr.NavTrail {
			return goldr.NavTrail{
				goldr.NavStep("Home", urls.Root.Path()),
				goldr.NavStep("Regional", urls.Main.Regional.Path()),
				goldr.NavStep(office.Name, urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID).Path()),
				goldr.NavStep(team.Name, urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID).Teams.ByTeamID.Bind(team.ID).Path()),
				goldr.NavStep(customer.Name, urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID).Teams.ByTeamID.Bind(team.ID).Customers.ByCustomerID.Bind(customer.ID).Path()),
			}
		},
	}
}
