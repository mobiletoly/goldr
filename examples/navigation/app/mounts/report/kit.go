package report

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/navigation/app/store"
	"github.com/mobiletoly/goldr/examples/navigation/app/ui"
)

type Kit struct {
	Store     store.Store
	TrailBase func(*http.Request) goldr.NavTrail
}

func (kit Kit) Page(r *http.Request) goldr.PageRouteResponse {
	customer := kit.Store.Customer(r.PathValue("customer_id"))
	trail := append(kit.TrailBase(r), goldr.CurrentNavStep("Report"))
	return goldr.NewPage(ui.Page("Customer Report", trail, nil, "Customer: "+customer.Name), goldr.PageMetadata{Title: "Customer Report"})
}
