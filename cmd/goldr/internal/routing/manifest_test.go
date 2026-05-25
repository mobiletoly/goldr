package routing

import (
	"reflect"
	"testing"
)

func TestBuildManifestMapsScannerTree(t *testing.T) {
	tree := Tree{
		Root: "/repo/app/routes",
		Pages: []Page{
			{Route: "/users/{id}", Params: []string{"id"}, GoFile: "users/by_id/page.go", TemplFile: "users/by_id/page.templ", HasTempl: true},
		},
		Layouts: []Layout{
			{RoutePrefix: "/users", GoFile: "users/layout.go"},
		},
		Fragments: []Fragment{
			{Name: "row", RoutePrefix: "/users", Params: []string{"id"}, GoFile: "users/by_id/frag_row.go", TemplFile: "users/by_id/frag_row.templ", HasTempl: true},
		},
		Actions: []Action{
			{Method: "POST", Route: "/users/{id}/save", Params: []string{"id"}, GoFile: "users/by_id/actions.go", Function: "PostSave", Suffix: "Save", Segment: "save"},
		},
		Middlewares: []Middleware{
			{RoutePrefix: "/", GoFile: "middleware.go"},
			{RoutePrefix: "/users/{id}", Params: []string{"id"}, GoFile: "users/by_id/middleware.go"},
		},
		Routes: []RouteDeclaration{
			{
				Route:  "/users/{id}",
				Params: []string{"id"},
				GoFile: "users/by_id/route.go",
				Kind:   routeDeclarationKindLocal,
				Name:   "users.show",
				Title:  "User",
				Meta:   []RouteMetaLabel{{Key: "nav", Value: "users"}},
				Page:   &RouteHandlerDeclaration{Handler: "page"},
				Fragments: []RouteFragmentDeclaration{
					{Name: "summary", Segment: "summary", SymbolName: "Summary", Handler: "summary"},
				},
				Actions: []RouteActionDeclaration{
					{Method: "POST", Name: "save", Segment: "save", SymbolName: "Save", Handler: "postSave"},
				},
			},
		},
	}

	got := BuildManifest(tree)
	want := Manifest{
		Root: "/repo/app/routes",
		Pages: []ManifestPage{
			{
				Route:  "/users/{id}",
				Params: []string{"id"},
				Unit:   RenderUnit{GoFile: "users/by_id/page.go", TemplFile: "users/by_id/page.templ", HasTempl: true},
			},
		},
		Layouts: []ManifestLayout{
			{
				RoutePrefix: "/users",
				Unit:        RenderUnit{GoFile: "users/layout.go"},
			},
		},
		Fragments: []ManifestFragment{
			{
				Name:        "row",
				RoutePrefix: "/users",
				Params:      []string{"id"},
				Unit:        RenderUnit{GoFile: "users/by_id/frag_row.go", TemplFile: "users/by_id/frag_row.templ", HasTempl: true},
			},
		},
		Actions: []ManifestAction{
			{
				Method:   "POST",
				Route:    "/users/{id}/save",
				Params:   []string{"id"},
				GoFile:   "users/by_id/actions.go",
				Function: "PostSave",
				Suffix:   "Save",
				Segment:  "save",
			},
		},
		Middlewares: []ManifestMiddleware{
			{
				RoutePrefix: "/",
				GoFile:      "middleware.go",
			},
			{
				RoutePrefix: "/users/{id}",
				Params:      []string{"id"},
				GoFile:      "users/by_id/middleware.go",
			},
		},
		Routes: []ManifestRouteDeclaration{
			{
				Route:  "/users/{id}",
				Params: []string{"id"},
				GoFile: "users/by_id/route.go",
				Kind:   routeDeclarationKindLocal,
				Name:   "users.show",
				Title:  "User",
				Meta:   []RouteMetaLabel{{Key: "nav", Value: "users"}},
				Page:   &RouteHandlerDeclaration{Handler: "page"},
				Fragments: []RouteFragmentDeclaration{
					{Name: "summary", Segment: "summary", SymbolName: "Summary", Handler: "summary"},
				},
				Actions: []RouteActionDeclaration{
					{Method: "POST", Name: "save", Segment: "save", SymbolName: "Save", Handler: "postSave"},
				},
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildManifest() = %#v, want %#v", got, want)
	}
}

func TestBuildManifestSortsOutput(t *testing.T) {
	tree := Tree{
		Pages: []Page{
			{Route: "/zeta", GoFile: "zeta/page.go"},
			{Route: "/alpha", GoFile: "alpha/page.go"},
		},
		Layouts: []Layout{
			{RoutePrefix: "/zeta", GoFile: "zeta/layout.go"},
			{RoutePrefix: "/alpha", GoFile: "alpha/layout.go"},
		},
		Fragments: []Fragment{
			{Name: "table", RoutePrefix: "/users", GoFile: "users/frag_table.go"},
			{Name: "row", RoutePrefix: "/users", GoFile: "users/frag_row.go"},
			{Name: "nav", RoutePrefix: "/", GoFile: "frag_nav.go"},
		},
		Actions: []Action{
			{Method: "PUT", Route: "/users/save", GoFile: "users/actions.go", Function: "PutSave"},
			{Method: "POST", Route: "/users/create", GoFile: "users/actions.go", Function: "PostCreate"},
		},
		Middlewares: []Middleware{
			{RoutePrefix: "/zeta", GoFile: "zeta/middleware.go"},
			{RoutePrefix: "/alpha", GoFile: "alpha/middleware.go"},
		},
		Routes: []RouteDeclaration{
			{Route: "/zeta", GoFile: "zeta/route.go"},
			{Route: "/alpha", GoFile: "alpha/route.go"},
		},
	}

	got := BuildManifest(tree)

	if got.Pages[0].Route != "/alpha" || got.Pages[1].Route != "/zeta" {
		t.Fatalf("pages = %#v, want sorted by route", got.Pages)
	}
	if got.Layouts[0].RoutePrefix != "/alpha" || got.Layouts[1].RoutePrefix != "/zeta" {
		t.Fatalf("layouts = %#v, want sorted by route prefix", got.Layouts)
	}
	wantFragments := []ManifestFragment{
		{Name: "nav", RoutePrefix: "/", Unit: RenderUnit{GoFile: "frag_nav.go"}},
		{Name: "row", RoutePrefix: "/users", Unit: RenderUnit{GoFile: "users/frag_row.go"}},
		{Name: "table", RoutePrefix: "/users", Unit: RenderUnit{GoFile: "users/frag_table.go"}},
	}
	if !reflect.DeepEqual(got.Fragments, wantFragments) {
		t.Fatalf("fragments = %#v, want %#v", got.Fragments, wantFragments)
	}
	wantActions := []ManifestAction{
		{Method: "POST", Route: "/users/create", GoFile: "users/actions.go", Function: "PostCreate"},
		{Method: "PUT", Route: "/users/save", GoFile: "users/actions.go", Function: "PutSave"},
	}
	if !reflect.DeepEqual(got.Actions, wantActions) {
		t.Fatalf("actions = %#v, want %#v", got.Actions, wantActions)
	}
	wantMiddlewares := []ManifestMiddleware{
		{RoutePrefix: "/alpha", GoFile: "alpha/middleware.go"},
		{RoutePrefix: "/zeta", GoFile: "zeta/middleware.go"},
	}
	if !reflect.DeepEqual(got.Middlewares, wantMiddlewares) {
		t.Fatalf("middleware = %#v, want %#v", got.Middlewares, wantMiddlewares)
	}
	wantRoutes := []ManifestRouteDeclaration{
		{Route: "/alpha", GoFile: "alpha/route.go"},
		{Route: "/zeta", GoFile: "zeta/route.go"},
	}
	if !reflect.DeepEqual(got.Routes, wantRoutes) {
		t.Fatalf("routes = %#v, want %#v", got.Routes, wantRoutes)
	}
}

func TestBuildManifestClonesParams(t *testing.T) {
	pageParams := []string{"page_id"}
	layoutParams := []string{"layout_id"}
	fragmentParams := []string{"fragment_id"}
	actionParams := []string{"action_id"}
	middlewareParams := []string{"middleware_id"}
	routeParams := []string{"route_id"}
	routeMeta := []RouteMetaLabel{{Key: "scope", Value: "routes"}}
	routeFragments := []RouteFragmentDeclaration{{Name: "table", Segment: "table", SymbolName: "Table", Handler: "table"}}
	routeActions := []RouteActionDeclaration{{Method: "POST", Name: "save", Segment: "save", SymbolName: "Save", Handler: "save"}}
	tree := Tree{
		Pages: []Page{
			{Route: "/pages/{page_id}", Params: pageParams, GoFile: "pages/by_page_id/page.go"},
		},
		Layouts: []Layout{
			{RoutePrefix: "/layouts/{layout_id}", Params: layoutParams, GoFile: "layouts/by_layout_id/layout.go"},
		},
		Fragments: []Fragment{
			{Name: "row", RoutePrefix: "/fragments/{fragment_id}", Params: fragmentParams, GoFile: "fragments/by_fragment_id/frag_row.go"},
		},
		Actions: []Action{
			{Method: "PATCH", Route: "/actions/{action_id}", Params: actionParams, GoFile: "actions/by_action_id/actions.go", Function: "PatchIndex"},
		},
		Middlewares: []Middleware{
			{RoutePrefix: "/middleware/{middleware_id}", Params: middlewareParams, GoFile: "middleware/by_middleware_id/middleware.go"},
		},
		Routes: []RouteDeclaration{
			{
				Route:     "/routes/{route_id}",
				Params:    routeParams,
				GoFile:    "routes/by_route_id/route.go",
				Meta:      routeMeta,
				Page:      &RouteHandlerDeclaration{Handler: "page"},
				Fragments: routeFragments,
				Actions:   routeActions,
				Kit:       &RouteKitDeclaration{KitType: "Kit", New: "New"},
			},
		},
	}

	got := BuildManifest(tree)
	pageParams[0] = "changed"
	layoutParams[0] = "changed"
	fragmentParams[0] = "changed"
	actionParams[0] = "changed"
	middlewareParams[0] = "changed"
	routeParams[0] = "changed"
	routeMeta[0].Key = "changed"
	routeFragments[0].Name = "changed"
	routeActions[0].Name = "changed"
	tree.Routes[0].Page.Handler = "changed"
	tree.Routes[0].Kit.KitType = "changed"

	if got.Pages[0].Params[0] != "page_id" {
		t.Fatalf("page params = %#v, want cloned params", got.Pages[0].Params)
	}
	if got.Layouts[0].Params[0] != "layout_id" {
		t.Fatalf("layout params = %#v, want cloned params", got.Layouts[0].Params)
	}
	if got.Fragments[0].Params[0] != "fragment_id" {
		t.Fatalf("fragment params = %#v, want cloned params", got.Fragments[0].Params)
	}
	if got.Actions[0].Params[0] != "action_id" {
		t.Fatalf("action params = %#v, want cloned params", got.Actions[0].Params)
	}
	if got.Middlewares[0].Params[0] != "middleware_id" {
		t.Fatalf("middleware params = %#v, want cloned params", got.Middlewares[0].Params)
	}
	if got.Routes[0].Params[0] != "route_id" {
		t.Fatalf("route params = %#v, want cloned params", got.Routes[0].Params)
	}
	if got.Routes[0].Meta[0].Key != "scope" {
		t.Fatalf("route meta = %#v, want cloned meta", got.Routes[0].Meta)
	}
	if got.Routes[0].Fragments[0].Name != "table" {
		t.Fatalf("route fragments = %#v, want cloned fragments", got.Routes[0].Fragments)
	}
	if got.Routes[0].Actions[0].Name != "save" {
		t.Fatalf("route actions = %#v, want cloned actions", got.Routes[0].Actions)
	}
	if got.Routes[0].Page.Handler != "page" {
		t.Fatalf("route page = %#v, want cloned page", got.Routes[0].Page)
	}
	if got.Routes[0].Kit.KitType != "Kit" {
		t.Fatalf("route kit = %#v, want cloned kit", got.Routes[0].Kit)
	}
}

func TestBuildManifestPreservesMissingTemplPairs(t *testing.T) {
	tree := Tree{
		Pages: []Page{
			{Route: "/", GoFile: "page.go"},
		},
	}

	got := BuildManifest(tree)
	if got.Pages[0].Unit.HasTempl {
		t.Fatalf("page unit HasTempl = true, want false")
	}
	if got.Pages[0].Unit.TemplFile != "" {
		t.Fatalf("page unit TemplFile = %q, want empty", got.Pages[0].Unit.TemplFile)
	}
}
