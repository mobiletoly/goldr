package customerreport

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/navigation/app/store"
)

type Kit struct {
	Store store.Store
}

func (kit Kit) Page(r *http.Request) goldr.PageRouteResponse {
	customer := kit.Customer(r)
	return goldr.NewPage(PageView(customer), goldr.PageMetadata{Title: "Customer Report"})
}

func (kit Kit) Customer(r *http.Request) store.Customer {
	customer := kit.Store.Customer(r.PathValue("customer_id"))
	nav := goldr.Nav(r)
	// This report kit is mounted under multiple owners, so it resolves only the
	// navigation keys present in the current owner's path.
	if officeID := r.PathValue("office_id"); officeID != "" {
		office := kit.Store.Office(officeID)
		nav.Resolve("office", office.Name)
	}
	if teamID := r.PathValue("team_id"); teamID != "" {
		team := kit.Store.Team(teamID)
		nav.Resolve("team", team.Name)
	}
	nav.Resolve("customer", customer.Name)
	return customer
}
