package by_customer_id

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/navigation/app/store"
	"github.com/mobiletoly/goldr/examples/navigation/app/ui"
	"github.com/mobiletoly/goldr/examples/navigation/app/urls"
)

var Route = goldr.RouteDef{
	Page: Page,
	NavTrails: goldr.NavTrails{
		Allowed: []string{"hq-customer", "regional-customer"},
	},
}

func Page(r *http.Request) goldr.PageRouteResponse {
	customer := store.Default.Customer(r.PathValue("customer_id"))
	team := store.Default.Team(customer.TeamID)

	trail := goldr.NavTrail{
		goldr.NavStep("Home", urls.Root.Path()),
		goldr.NavStep("Reports", urls.Main.Path()),
		goldr.CurrentNavStep(customer.Name),
	}
	switch goldr.NavTrailKey(r) {
	case urls.Main.Reports.ByCustomerID.NavTrails.HqCustomer:
		trail = goldr.NavTrail{
			goldr.NavStep("Home", urls.Root.Path()),
			goldr.NavStep("HQ", urls.Main.Hq.Path()),
			goldr.NavStep(team.Name, urls.Main.Hq.Teams.ByTeamID.Bind(team.ID).Path()),
			goldr.NavStep(customer.Name, urls.Main.Hq.Teams.ByTeamID.Bind(team.ID).Customers.ByCustomerID.Bind(customer.ID).Path()),
			goldr.CurrentNavStep("Report"),
		}
	case urls.Main.Reports.ByCustomerID.NavTrails.RegionalCustomer:
		office := store.Default.Office(team.OfficeID)
		trail = goldr.NavTrail{
			goldr.NavStep("Home", urls.Root.Path()),
			goldr.NavStep("Regional", urls.Main.Regional.Path()),
			goldr.NavStep(office.Name, urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID).Path()),
			goldr.NavStep(team.Name, urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID).Teams.ByTeamID.Bind(team.ID).Path()),
			goldr.NavStep(customer.Name, urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID).Teams.ByTeamID.Bind(team.ID).Customers.ByCustomerID.Bind(customer.ID).Path()),
			goldr.CurrentNavStep("Report"),
		}
	}

	return goldr.NewPage(ui.Page("Shared Report", trail, nil, "Shared target selected by destination trail."), goldr.PageMetadata{Title: "Shared Report"})
}
