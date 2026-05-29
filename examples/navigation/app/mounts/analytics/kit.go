package analytics

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/navigation/app/store"
)

type Kit struct {
	Store              store.Store
	AnalyticsURL       func() string
	CustomerURL        func(string) string
	CustomerReportHref func(goldr.Navigation, string) string
}

func (kit Kit) Page(r *http.Request) goldr.PageRouteResponse {
	nav := goldr.Nav(r)
	// The route tree provides key placeholders such as "office" and "team";
	// the mounted kit resolves those keys after loading real labels.
	kit.ResolveNav(r, nav)
	navigation := nav.Navigation()
	risk := r.URL.Query().Get("risk")
	customers := filterCustomersByRisk(kit.Store.TeamCustomers(r.PathValue("team_id")), risk)
	customerReportHref := func(customerID string) string {
		return kit.CustomerReportHref(navigation, customerID)
	}
	return goldr.NewPage(PageView(navigation, kit.AnalyticsURL(), risk, customers, customerReportHref), goldr.PageMetadata{Title: "Analytics"})
}

func (kit Kit) ResolveNav(r *http.Request, nav goldr.RequestNav) {
	if officeID := r.PathValue("office_id"); officeID != "" {
		office := kit.Store.Office(officeID)
		nav.Resolve("office", office.Name)
	}
	if teamID := r.PathValue("team_id"); teamID != "" {
		team := kit.Store.Team(teamID)
		nav.Resolve("team", team.Name)
	}
}

func filterCustomersByRisk(customers []store.Customer, risk string) []store.Customer {
	if risk == "" {
		return customers
	}
	var filtered []store.Customer
	for _, customer := range customers {
		if customer.Risk == risk {
			filtered = append(filtered, customer)
		}
	}
	return filtered
}
