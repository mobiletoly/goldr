package analytics

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/navigation/app/store"
	"github.com/mobiletoly/goldr/examples/navigation/app/ui"
)

type Kit struct {
	Store              store.Store
	TrailBase          func(*http.Request) goldr.NavTrail
	AnalyticsURL       func() string
	CustomerURL        func(string) string
	CustomerReportHref func(string) string
}

func (kit Kit) Page(r *http.Request) goldr.PageRouteResponse {
	trail := append(kit.TrailBase(r), goldr.CurrentNavStep("Analytics"))
	var links []ui.Link
	for _, customer := range kit.Store.TeamCustomers(r.PathValue("team_id")) {
		links = append(links, ui.Link{
			Label: customer.Name + " report",
			Href:  kit.CustomerReportHref(customer.ID),
		})
	}
	return goldr.NewPage(ui.Page("Analytics", trail, links, "Shared analytics root uses owner-provided links."), goldr.PageMetadata{Title: "Analytics"})
}

func (kit Kit) CustomerReport(r *http.Request) goldr.PageRouteResponse {
	customer := kit.Store.Customer(r.PathValue("customer_id"))
	trail := append(kit.TrailBase(r),
		goldr.NavStep("Analytics", kit.AnalyticsURL()),
		goldr.NavStep(customer.Name, kit.CustomerURL(customer.ID)),
		goldr.CurrentNavStep("Report"),
	)
	return goldr.NewPage(ui.Page("Analytics Report", trail, nil, "Risk: "+customer.Risk), goldr.PageMetadata{Title: "Analytics Report"})
}
