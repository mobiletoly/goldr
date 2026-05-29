package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/examples/navigation/app/ui"
	"github.com/mobiletoly/goldr/examples/navigation/app/urls"
)

var Route = goldr.RouteDef{
	Page: Page,
}

func Page(_ *http.Request) goldr.PageRouteResponse {
	trail := goldr.NavTrail{goldr.CurrentNavStep("Home")}
	return goldr.NewPage(ui.Page("Home", trail, []ui.Link{
		{Label: "HQ", Href: urls.Main.Hq.Path()},
		{Label: "Regional", Href: urls.Main.Regional.Path()},
	}, "Choose a route owner."), goldr.PageMetadata{Title: "Home"})
}
