package wiring

import (
	"reflect"
	"testing"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
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

func TestExplainRouteMatchesActionWithLayouts(t *testing.T) {
	requireRouteExplanation(t, routing.Manifest{
		Layouts: []routing.ManifestLayout{
			{RoutePrefix: "/", Unit: routing.RenderUnit{GoFile: "layout.go"}},
			{RoutePrefix: "/users", Unit: routing.RenderUnit{GoFile: "users/layout.go"}},
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
			Layouts: []RouteExplanationLayout{
				{RoutePrefix: "/", Source: "layout.go"},
				{RoutePrefix: "/users", Source: "users/layout.go"},
			},
		},
	})
}

func TestExplainRouteIncludesLocalDeclarationMetadata(t *testing.T) {
	requireRouteExplanation(t, routing.Manifest{
		Layouts: []routing.ManifestLayout{
			{RoutePrefix: "/", Unit: routing.RenderUnit{GoFile: "layout.go"}},
			{RoutePrefix: "/users", Unit: routing.RenderUnit{GoFile: "users/layout.go"}},
		},
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/users/{id}",
				Params: []string{"id"},
				GoFile: "users/by_id/route.go",
				Kind:   "local",
				Name:   "users.show",
				Title:  "User Details",
				Meta: []routing.RouteMetaLabel{
					{Key: "zeta", Value: "last"},
					{Key: "alpha", Value: "first"},
				},
				Page: &routing.RouteHandlerDeclaration{Handler: "page"},
			},
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
			Source: "users/by_id/route.go",
			Layouts: []RouteExplanationLayout{
				{RoutePrefix: "/", Source: "layout.go"},
				{RoutePrefix: "/users", Source: "users/layout.go"},
			},
			Declaration: &RouteDeclarationInfo{
				Source: "users/by_id/route.go",
				Kind:   "local",
				Name:   "users.show",
				Title:  "User Details",
				Labels: []RouteDeclarationLabel{
					{Key: "alpha", Value: "first"},
					{Key: "zeta", Value: "last"},
				},
				Page: &RouteDeclarationPage{
					Handler: "page",
					Adapter: "GoldrRoutePage",
				},
			},
		},
	})
}

func TestExplainRouteIncludesFragmentDeclarationMetadata(t *testing.T) {
	requireRouteExplanation(t, routing.Manifest{
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/users",
				GoFile: "users/route.go",
				Kind:   "local",
				Fragments: []routing.RouteFragmentDeclaration{
					{Name: "table", Segment: "table", SymbolName: "Table", Handler: "table"},
				},
			},
		},
	}, "get", "/users/table", RouteExplanation{
		Status:         RouteExplainStatusMatched,
		Method:         "GET",
		Path:           "/users/table",
		AllowedMethods: []string{"GET", "HEAD"},
		Match: RouteExplanationMatch{
			Kind:    RouteSurfaceKindFragment,
			Methods: []string{"GET", "HEAD"},
			Route:   "/users/table",
			Params:  []RouteExplanationParam{},
			Source:  "users/route.go",
			Declaration: &RouteDeclarationInfo{
				Source: "users/route.go",
				Kind:   "local",
				Fragment: &RouteDeclarationFragment{
					Name:    "table",
					Segment: "table",
					Handler: "table",
					Adapter: "GoldrRouteFragTable",
				},
			},
		},
	})
}

func TestExplainRouteIncludesIndexFragmentDeclarationMetadata(t *testing.T) {
	requireRouteExplanation(t, routing.Manifest{
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/users/status-options",
				GoFile: "users/status_options/route.go",
				Kind:   "local",
				Fragments: []routing.RouteFragmentDeclaration{
					{Name: "index", SymbolName: "Index", Index: true, Handler: "options"},
				},
			},
		},
	}, "get", "/users/status-options", RouteExplanation{
		Status:         RouteExplainStatusMatched,
		Method:         "GET",
		Path:           "/users/status-options",
		AllowedMethods: []string{"GET", "HEAD"},
		Match: RouteExplanationMatch{
			Kind:    RouteSurfaceKindFragment,
			Methods: []string{"GET", "HEAD"},
			Route:   "/users/status-options",
			Params:  []RouteExplanationParam{},
			Source:  "users/status_options/route.go",
			Declaration: &RouteDeclarationInfo{
				Source: "users/status_options/route.go",
				Kind:   "local",
				Fragment: &RouteDeclarationFragment{
					Name:    "index",
					Index:   true,
					Handler: "options",
					Adapter: "GoldrRouteFragIndex",
				},
			},
		},
	})
}

func TestExplainRouteIncludesLocalActionDeclarationMetadata(t *testing.T) {
	requireRouteExplanation(t, routing.Manifest{
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/users",
				GoFile: "users/route.go",
				Kind:   "local",
				Actions: []routing.RouteActionDeclaration{
					{Method: "POST", Index: true, SymbolName: "Index", Handler: "postIndex"},
				},
			},
		},
	}, "post", "/users", RouteExplanation{
		Status:         RouteExplainStatusMatched,
		Method:         "POST",
		Path:           "/users",
		AllowedMethods: []string{"POST"},
		Match: RouteExplanationMatch{
			Kind:     RouteSurfaceKindAction,
			Methods:  []string{"POST"},
			Route:    "/users",
			Params:   []RouteExplanationParam{},
			Source:   "users/route.go",
			Function: "GoldrRoutePostIndex",
			Layouts:  []RouteExplanationLayout{},
			Declaration: &RouteDeclarationInfo{
				Source: "users/route.go",
				Kind:   "local",
				Action: &RouteDeclarationAction{
					Method:  "POST",
					Index:   true,
					Handler: "postIndex",
					Adapter: "GoldrRoutePostIndex",
				},
			},
		},
	})
}

func TestExplainRouteIncludesKitPageDeclarationMetadata(t *testing.T) {
	requireRouteExplanation(t, routing.Manifest{
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/reports",
				GoFile: "reports/route.go",
				Kind:   "kit",
				Page:   &routing.RouteHandlerDeclaration{Handler: "reports.Kit.Page"},
				Kit: &routing.RouteKitDeclaration{
					KitType: "reports.Kit",
					New:     "newKit",
				},
			},
		},
	}, "get", "/reports", RouteExplanation{
		Status:         RouteExplainStatusMatched,
		Method:         "GET",
		Path:           "/reports",
		AllowedMethods: []string{"GET", "HEAD"},
		Match: RouteExplanationMatch{
			Kind:    RouteSurfaceKindPage,
			Methods: []string{"GET", "HEAD"},
			Route:   "/reports",
			Params:  []RouteExplanationParam{},
			Source:  "reports/route.go",
			Layouts: []RouteExplanationLayout{},
			Declaration: &RouteDeclarationInfo{
				Source: "reports/route.go",
				Kind:   "kit",
				Kit: &RouteDeclarationKit{
					KitType: "reports.Kit",
					New:     "newKit",
				},
				Page: &RouteDeclarationPage{
					Handler: "reports.Kit.Page",
					Adapter: "GoldrRoutePage",
				},
			},
		},
	})
}

func TestExplainRouteIncludesKitActionDeclarationMetadata(t *testing.T) {
	requireRouteExplanation(t, routing.Manifest{
		Layouts: []routing.ManifestLayout{
			{RoutePrefix: "/", Unit: routing.RenderUnit{GoFile: "layout.go"}},
		},
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/reports",
				GoFile: "reports/route.go",
				Kind:   "kit",
				Actions: []routing.RouteActionDeclaration{
					{Method: "POST", Name: "export", Segment: "export", SymbolName: "Export", Handler: "reports.Kit.PostExport"},
				},
				Kit: &routing.RouteKitDeclaration{
					KitType: "reports.Kit",
					New:     "newKit",
				},
			},
		},
	}, "post", "/reports/export", RouteExplanation{
		Status:         RouteExplainStatusMatched,
		Method:         "POST",
		Path:           "/reports/export",
		AllowedMethods: []string{"POST"},
		Match: RouteExplanationMatch{
			Kind:     RouteSurfaceKindAction,
			Methods:  []string{"POST"},
			Route:    "/reports/export",
			Params:   []RouteExplanationParam{},
			Source:   "reports/route.go",
			Function: "GoldrRoutePostExport",
			Layouts: []RouteExplanationLayout{
				{RoutePrefix: "/", Source: "layout.go"},
			},
			Declaration: &RouteDeclarationInfo{
				Source: "reports/route.go",
				Kind:   "kit",
				Kit: &RouteDeclarationKit{
					KitType: "reports.Kit",
					New:     "newKit",
				},
				Action: &RouteDeclarationAction{
					Method:  "POST",
					Name:    "export",
					Segment: "export",
					Handler: "reports.Kit.PostExport",
					Adapter: "GoldrRoutePostExport",
				},
			},
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
