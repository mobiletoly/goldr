package hq

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
	team := store.Default.Team("hq-team")
	trail := goldr.NavTrail{
		goldr.NavStep("Home", urls.Root.Path()),
		goldr.CurrentNavStep("HQ"),
	}
	return goldr.NewPage(ui.Page("HQ", trail, []ui.Link{
		{Label: team.Name, Href: urls.Main.Hq.Teams.ByTeamID.Bind(team.ID).Path()},
	}, "HQ owns a shorter analytics trail prefix."), goldr.PageMetadata{Title: "HQ"})
}
