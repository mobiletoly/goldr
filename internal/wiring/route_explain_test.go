package wiring

import (
	"reflect"
	"testing"

	"github.com/mobiletoly/goldr/internal/routing"
)

func TestExplainRouteMatchesDynamicPageWithLayoutsAndDecodedParam(t *testing.T) {
	requireRouteExplanation(t, routing.Manifest{
		Layouts: []routing.ManifestLayout{
			{RoutePrefix: "/", Unit: routing.RenderUnit{GoFile: "layout.go"}},
			{RoutePrefix: "/users", Unit: routing.RenderUnit{GoFile: "users/layout.go"}},
		},
		Pages: []routing.ManifestPage{
			{Route: "/users/{id}", Params: []string{"id"}, Unit: routing.RenderUnit{GoFile: "users/by_id/page.go"}},
		},
	}, "get", "/users/a%2Fb", RouteExplanation{
		Status:         RouteExplainStatusMatched,
		Method:         "GET",
		Path:           "/users/a%2Fb",
		AllowedMethods: []string{"GET", "HEAD"},
		Match: RouteExplanationMatch{
			Kind:    RouteSurfaceKindPage,
			Methods: []string{"GET", "HEAD"},
			Route:   "/users/{id}",
			Params: []RouteExplanationParam{
				{Name: "id", Value: "a/b"},
			},
			Source: "users/by_id/page.go",
			Layouts: []RouteExplanationLayout{
				{RoutePrefix: "/", Source: "layout.go"},
				{RoutePrefix: "/users", Source: "users/layout.go"},
			},
		},
	})
}

func TestExplainRouteMatchesActionWithoutLayouts(t *testing.T) {
	requireRouteExplanation(t, routing.Manifest{
		Layouts: []routing.ManifestLayout{
			{RoutePrefix: "/", Unit: routing.RenderUnit{GoFile: "layout.go"}},
		},
		Actions: []routing.ManifestAction{
			{Method: "POST", Route: "/users/create", GoFile: "users/actions.go", Function: "PostCreate"},
		},
	}, "post", "/users/create", RouteExplanation{
		Status:         RouteExplainStatusMatched,
		Method:         "POST",
		Path:           "/users/create",
		AllowedMethods: []string{"POST"},
		Match: RouteExplanationMatch{
			Kind:     RouteSurfaceKindAction,
			Methods:  []string{"POST"},
			Route:    "/users/create",
			Params:   []RouteExplanationParam{},
			Source:   "users/actions.go",
			Function: "PostCreate",
		},
	})
}

func TestExplainRouteMatchesDispatchTreeStaticChildPriority(t *testing.T) {
	requireRouteExplanation(t, routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/foo/{first}/{second}", Params: []string{"first", "second"}, Unit: routing.RenderUnit{GoFile: "foo/by_first/by_second/page.go"}},
		},
		Actions: []routing.ManifestAction{
			{Method: "POST", Route: "/{section}/static/static", Params: []string{"section"}, GoFile: "by_section/actions.go", Function: "PostStaticStatic"},
		},
	}, "GET", "/foo/static/static", RouteExplanation{
		Status:         RouteExplainStatusMatched,
		Method:         "GET",
		Path:           "/foo/static/static",
		AllowedMethods: []string{"GET", "HEAD"},
		Match: RouteExplanationMatch{
			Kind:    RouteSurfaceKindPage,
			Methods: []string{"GET", "HEAD"},
			Route:   "/foo/{first}/{second}",
			Params: []RouteExplanationParam{
				{Name: "first", Value: "static"},
				{Name: "second", Value: "static"},
			},
			Source:  "foo/by_first/by_second/page.go",
			Layouts: []RouteExplanationLayout{},
		},
	})
}

func TestExplainRouteReportsMethodNotAllowed(t *testing.T) {
	requireRouteExplanation(t, routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/users/{id}", Params: []string{"id"}, Unit: routing.RenderUnit{GoFile: "users/by_id/page.go"}},
		},
	}, "POST", "/users/7", RouteExplanation{
		Status:         RouteExplainStatusMethodNotAllowed,
		Method:         "POST",
		Path:           "/users/7",
		AllowedMethods: []string{"GET", "HEAD"},
	})
}

func TestExplainRouteReportsNoMatch(t *testing.T) {
	requireRouteExplanation(t, routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/users", Unit: routing.RenderUnit{GoFile: "users/page.go"}},
		},
	}, "GET", "/missing", RouteExplanation{
		Status: RouteExplainStatusNotFound,
		Method: "GET",
		Path:   "/missing",
	})
}

func requireRouteExplanation(t *testing.T, manifest routing.Manifest, method string, path string, want RouteExplanation) {
	t.Helper()

	explanation, err := ExplainRoute(manifest, method, path)
	if err != nil {
		t.Fatalf("ExplainRoute() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(explanation, want) {
		t.Fatalf("ExplainRoute() = %#v, want %#v", explanation, want)
	}
}
