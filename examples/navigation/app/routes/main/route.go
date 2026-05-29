package mainroute

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
	trail := goldr.NavTrail{
		goldr.NavStep("Home", urls.Root.Path()),
		goldr.CurrentNavStep("Main"),
	}
	return goldr.NewPage(ui.Page("Main", trail, []ui.Link{
		{Label: "HQ", Href: urls.Main.Hq.Path()},
		{Label: "Regional", Href: urls.Main.Regional.Path()},
	}, "Top-level sections use different trail defaults."), goldr.PageMetadata{Title: "Main"})
}
