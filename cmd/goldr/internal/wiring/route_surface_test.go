package wiring

import (
	"errors"
	"reflect"
	"testing"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
)

func TestRouteSurfaceRows(t *testing.T) {
	rows, err := RouteSurface(routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/", Unit: completeUnit("page.go")},
			{Route: "/users", Unit: completeUnit("users/page.go")},
			{Route: "/users/{id}", Params: []string{"id"}, Unit: completeUnit("users/by_id/page.go")},
			{Route: "/orgs/{org_id}/users/{user_id}", Params: []string{"org_id", "user_id"}, Unit: completeUnit("orgs/by_org_id/users/by_user_id/page.go")},
		},
		Layouts: []routing.ManifestLayout{
			{RoutePrefix: "/", Unit: completeUnit("layout.go")},
			{RoutePrefix: "/users", Unit: completeUnit("users/layout.go")},
		},
		Fragments: []routing.ManifestFragment{
			{Name: "table", RoutePrefix: "/users", Unit: completeUnit("users/frag_table.go")},
		},
		Actions: []routing.ManifestAction{
			{Method: "POST", Route: "/users/create", GoFile: "users/actions.go", Function: "PostCreate"},
			{Method: "PATCH", Route: "/users/{id}", Params: []string{"id"}, GoFile: "users/by_id/actions.go", Function: "PatchIndex"},
		},
	})
	if err != nil {
		t.Fatalf("RouteSurface() error = %v, want nil", err)
	}

	want := []RouteSurfaceRow{
		{Kind: "layout", Path: "/", Source: "layout.go"},
		{Kind: "page", Methods: []string{"GET", "HEAD"}, Path: "/", Source: "page.go", Helper: "urls.Root.Path()"},
		{Kind: "action", Methods: []string{"POST"}, Path: "/users/create", Source: "users/actions.go:PostCreate", Helper: "urls.Users.Create.Path()"},
		{Kind: "fragment", Methods: []string{"GET", "HEAD"}, Path: "/users/table", Source: "users/frag_table.go", Helper: "urls.Users.Table.Path()"},
		{Kind: "page", Methods: []string{"GET", "HEAD"}, Path: "/orgs/{org_id}/users/{user_id}", Params: []string{"org_id", "user_id"}, Source: "orgs/by_org_id/users/by_user_id/page.go", Helper: "urls.Orgs.ByOrgID(orgID).Users.ByUserID(userID).Path()"},
		{Kind: "layout", Path: "/users", Source: "users/layout.go"},
		{Kind: "page", Methods: []string{"GET", "HEAD"}, Path: "/users", Source: "users/page.go", Helper: "urls.Users.Path()"},
		{Kind: "page", Methods: []string{"GET", "HEAD"}, Path: "/users/{id}", Params: []string{"id"}, Source: "users/by_id/page.go", Helper: "urls.Users.ByID(id).Path()"},
		{Kind: "action", Methods: []string{"PATCH"}, Path: "/users/{id}", Params: []string{"id"}, Source: "users/by_id/actions.go:PatchIndex", Helper: "urls.Users.ByID(id).Path()"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows = %#v, want %#v", rows, want)
	}
}

func TestRouteSurfaceIncludesMissingRenderUnitPairs(t *testing.T) {
	rows, err := RouteSurface(routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/", Unit: routing.RenderUnit{GoFile: "page.go"}},
		},
	})
	if err != nil {
		t.Fatalf("RouteSurface() error = %v, want nil", err)
	}
	want := []RouteSurfaceRow{
		{Kind: "page", Methods: []string{"GET", "HEAD"}, Path: "/", Source: "page.go", Helper: "urls.Root.Path()"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows = %#v, want %#v", rows, want)
	}
}

func TestRouteSurfaceRowsIncludeIndexFragments(t *testing.T) {
	rows, err := RouteSurface(routing.Manifest{
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
	})
	if err != nil {
		t.Fatalf("RouteSurface() error = %v, want nil", err)
	}
	want := []RouteSurfaceRow{
		{
			Kind:    "fragment",
			Methods: []string{"GET", "HEAD"},
			Path:    "/users/status-options",
			Source:  "users/status_options/route.go",
			Helper:  "urls.Users.StatusOptions.Path()",
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
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows = %#v, want %#v", rows, want)
	}
}

func TestRouteSurfaceRejectsURLHelperCollisions(t *testing.T) {
	_, err := RouteSurface(routing.Manifest{
		Actions: []routing.ManifestAction{
			{Method: "POST", Route: "/users/path", GoFile: "users/actions.go", Function: "PostPath"},
		},
	})
	if !errors.Is(err, ErrAmbiguousURLHelper) {
		t.Fatalf("RouteSurface() error = %v, want ErrAmbiguousURLHelper", err)
	}
}

func TestRouteSurfaceRowsIncludeDeclarationMetadata(t *testing.T) {
	rows, err := RouteSurface(routing.Manifest{
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/users",
				GoFile: "users/route.go",
				Kind:   "local",
				Name:   "users.index",
				Title:  "Users",
				Meta: []routing.RouteMetaLabel{
					{Key: "zeta", Value: "last"},
					{Key: "alpha", Value: "first value"},
				},
				Page: &routing.RouteHandlerDeclaration{Handler: "page"},
				Fragments: []routing.RouteFragmentDeclaration{
					{Name: "table", Segment: "table", SymbolName: "Table", Handler: "table"},
				},
				Actions: []routing.RouteActionDeclaration{
					{Method: "POST", Index: true, SymbolName: "Index", Handler: "postIndex"},
					{Method: "PATCH", Name: "save-profile", Segment: "save-profile", SymbolName: "SaveProfile", Handler: "patchSaveProfile"},
				},
			},
			{
				Route:  "/reports",
				GoFile: "reports/route.go",
				Kind:   "kit",
				Page:   &routing.RouteHandlerDeclaration{Handler: "reports.Kit.Page"},
				Fragments: []routing.RouteFragmentDeclaration{
					{Name: "panel", Segment: "panel", SymbolName: "Panel", Handler: "reports.Kit.Panel"},
				},
				Actions: []routing.RouteActionDeclaration{
					{Method: "POST", Name: "export", Segment: "export", SymbolName: "Export", Handler: "reports.Kit.PostExport"},
				},
				Kit: &routing.RouteKitDeclaration{
					KitType: "reports.Kit",
					New:     "newKit",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RouteSurface() error = %v, want nil", err)
	}

	want := []RouteSurfaceRow{
		{
			Kind:    "action",
			Methods: []string{"POST"},
			Path:    "/reports/export",
			Source:  "reports/route.go:GoldrRoutePostExport",
			Helper:  "urls.Reports.Export.Path()",
			Declaration: &RouteDeclarationInfo{
				Source: "reports/route.go",
				Kind:   "kit",
				Labels: nil,
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
		{
			Kind:    "fragment",
			Methods: []string{"GET", "HEAD"},
			Path:    "/reports/panel",
			Source:  "reports/route.go",
			Helper:  "urls.Reports.Panel.Path()",
			Declaration: &RouteDeclarationInfo{
				Source: "reports/route.go",
				Kind:   "kit",
				Labels: nil,
				Kit: &RouteDeclarationKit{
					KitType: "reports.Kit",
					New:     "newKit",
				},
				Fragment: &RouteDeclarationFragment{
					Name:    "panel",
					Segment: "panel",
					Handler: "reports.Kit.Panel",
					Adapter: "GoldrRouteFragPanel",
				},
			},
		},
		{
			Kind:    "action",
			Methods: []string{"PATCH"},
			Path:    "/users/save-profile",
			Source:  "users/route.go:GoldrRoutePatchSaveProfile",
			Helper:  "urls.Users.SaveProfile.Path()",
			Declaration: &RouteDeclarationInfo{
				Source: "users/route.go",
				Kind:   "local",
				Name:   "users.index",
				Title:  "Users",
				Labels: []RouteDeclarationLabel{
					{Key: "alpha", Value: "first value"},
					{Key: "zeta", Value: "last"},
				},
				Action: &RouteDeclarationAction{
					Method:  "PATCH",
					Name:    "save-profile",
					Segment: "save-profile",
					Handler: "patchSaveProfile",
					Adapter: "GoldrRoutePatchSaveProfile",
				},
			},
		},
		{
			Kind:    "fragment",
			Methods: []string{"GET", "HEAD"},
			Path:    "/users/table",
			Source:  "users/route.go",
			Helper:  "urls.Users.Table.Path()",
			Declaration: &RouteDeclarationInfo{
				Source: "users/route.go",
				Kind:   "local",
				Name:   "users.index",
				Title:  "Users",
				Labels: []RouteDeclarationLabel{
					{Key: "alpha", Value: "first value"},
					{Key: "zeta", Value: "last"},
				},
				Fragment: &RouteDeclarationFragment{
					Name:    "table",
					Segment: "table",
					Handler: "table",
					Adapter: "GoldrRouteFragTable",
				},
			},
		},
		{
			Kind:    "page",
			Methods: []string{"GET", "HEAD"},
			Path:    "/reports",
			Source:  "reports/route.go",
			Helper:  "urls.Reports.Path()",
			Declaration: &RouteDeclarationInfo{
				Source: "reports/route.go",
				Kind:   "kit",
				Labels: nil,
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
		{
			Kind:    "page",
			Methods: []string{"GET", "HEAD"},
			Path:    "/users",
			Source:  "users/route.go",
			Helper:  "urls.Users.Path()",
			Declaration: &RouteDeclarationInfo{
				Source: "users/route.go",
				Kind:   "local",
				Name:   "users.index",
				Title:  "Users",
				Labels: []RouteDeclarationLabel{
					{Key: "alpha", Value: "first value"},
					{Key: "zeta", Value: "last"},
				},
				Page: &RouteDeclarationPage{
					Handler: "page",
					Adapter: "GoldrRoutePage",
				},
			},
		},
		{
			Kind:    "action",
			Methods: []string{"POST"},
			Path:    "/users",
			Source:  "users/route.go:GoldrRoutePostIndex",
			Helper:  "urls.Users.Path()",
			Declaration: &RouteDeclarationInfo{
				Source: "users/route.go",
				Kind:   "local",
				Name:   "users.index",
				Title:  "Users",
				Labels: []RouteDeclarationLabel{
					{Key: "alpha", Value: "first value"},
					{Key: "zeta", Value: "last"},
				},
				Action: &RouteDeclarationAction{
					Method:  "POST",
					Index:   true,
					Handler: "postIndex",
					Adapter: "GoldrRoutePostIndex",
				},
			},
		},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows = %#v, want %#v", rows, want)
	}
}

func TestBuildRouteLayoutMap(t *testing.T) {
	layoutMap, err := BuildRouteLayoutMap(routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/", Unit: completeUnit("page.go")},
			{Route: "/settings", Unit: completeUnit("settings/page.go")},
			{Route: "/users", Unit: completeUnit("users/page.go")},
			{Route: "/users/{id}", Params: []string{"id"}, Unit: completeUnit("users/by_id/page.go")},
		},
		Layouts: []routing.ManifestLayout{
			{RoutePrefix: "/", Unit: completeUnit("layout.go")},
			{RoutePrefix: "/users", Unit: completeUnit("users/layout.go")},
		},
		Fragments: []routing.ManifestFragment{
			{Name: "table", RoutePrefix: "/users", Unit: completeUnit("users/frag_table.go")},
		},
		Actions: []routing.ManifestAction{
			{Method: "POST", Route: "/users/create", GoFile: "users/actions.go", Function: "PostCreate"},
			{Method: "POST", Route: "/users/save-preview", GoFile: "users/actions.go", Function: "PostSavePreview"},
		},
	})
	if err != nil {
		t.Fatalf("BuildRouteLayoutMap() error = %v, want nil", err)
	}

	if layoutMap.Root == nil {
		t.Fatal("Root = nil, want root node")
	}
	if got := layoutMap.Root.Layout.Source; got != "layout.go" {
		t.Fatalf("root layout source = %q, want layout.go", got)
	}
	if got := layoutMap.Root.Pages[0].Source; got != "page.go" {
		t.Fatalf("root page source = %q, want page.go", got)
	}
	if got, want := childNames(layoutMap.Root), []string{"settings", "users"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("root children = %#v, want %#v", got, want)
	}

	users := childNode(t, layoutMap.Root, "users")
	if got := users.Layout.Source; got != "users/layout.go" {
		t.Fatalf("users layout source = %q, want users/layout.go", got)
	}
	if got := users.Pages[0].Route; got != "/users" {
		t.Fatalf("users page route = %q, want /users", got)
	}
	if got := users.Fragments[0].Route; got != "/users/table" {
		t.Fatalf("users fragment route = %q, want /users/table", got)
	}
	if got := users.Actions[0].Function; got != "PostCreate" {
		t.Fatalf("users first action = %q, want PostCreate", got)
	}
	if got, want := layoutSources(users.Actions[0].Layouts), []string{"layout.go", "users/layout.go"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("users action layouts = %#v, want %#v", got, want)
	}
	if got := users.Actions[1].Function; got != "PostSavePreview" {
		t.Fatalf("users second action = %q, want PostSavePreview", got)
	}

	byID := childNode(t, users, "by_id")
	page := byID.Pages[0]
	if got := page.Route; got != "/users/{id}" {
		t.Fatalf("dynamic page route = %q, want /users/{id}", got)
	}
	if got := page.Params; !reflect.DeepEqual(got, []string{"id"}) {
		t.Fatalf("dynamic page params = %#v, want id", got)
	}
	if got, want := layoutSources(page.Layouts), []string{"layout.go", "users/layout.go"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("dynamic page layouts = %#v, want %#v", got, want)
	}
}

func childNode(t *testing.T, node *RouteLayoutMapNode, name string) *RouteLayoutMapNode {
	t.Helper()

	for _, child := range node.Children {
		if child.Name == name {
			return child
		}
	}
	t.Fatalf("children = %#v, want child %q", childNames(node), name)
	return nil
}

func childNames(node *RouteLayoutMapNode) []string {
	names := make([]string, 0, len(node.Children))
	for _, child := range node.Children {
		names = append(names, child.Name)
	}
	return names
}

func layoutSources(layouts []RouteLayoutMapLayout) []string {
	sources := make([]string, 0, len(layouts))
	for _, layout := range layouts {
		sources = append(sources, layout.Source)
	}
	return sources
}
