package regional

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/navigation/app/store"
	"github.com/mobiletoly/goldr/examples/navigation/app/ui"
	"github.com/mobiletoly/goldr/examples/navigation/app/urls"
)

var Route = goldr.RouteDef{
	Page: Page,
}

func Page(_ *http.Request) goldr.PageRouteResponse {
	office := store.Default.Office("sea")
	trail := goldr.NavTrail{
		goldr.NavStep("Home", urls.Root.Path()),
		goldr.CurrentNavStep("Regional"),
	}
	return goldr.NewPage(ui.Page("Regional", trail, []ui.Link{
		{Label: office.Name, Href: urls.Main.Regional.Offices.ByOfficeID.Bind(office.ID).Path()},
	}, "Regional owns an office-aware trail prefix."), goldr.PageMetadata{Title: "Regional"})
}
