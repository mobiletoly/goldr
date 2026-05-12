package wiring

import (
	"errors"
	"reflect"
	"testing"

	"github.com/mobiletoly/goldr/internal/routing"
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
		{Kind: "fragment", Methods: []string{"GET", "HEAD"}, Path: "/users/frag-table", Source: "users/frag_table.go", Helper: "urls.Users.FragTable.Path()"},
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
	if got := users.Fragments[0].Route; got != "/users/frag-table" {
		t.Fatalf("users fragment route = %q, want /users/frag-table", got)
	}
	if got := users.Actions[0].Function; got != "PostCreate" {
		t.Fatalf("users first action = %q, want PostCreate", got)
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
