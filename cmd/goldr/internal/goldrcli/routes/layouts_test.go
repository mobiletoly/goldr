package routes

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/goldrcli/ansi"
	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
	"github.com/mobiletoly/goldr/cmd/goldr/internal/wiring"
)

func TestRenderRouteLayoutMapStylesSemanticLabels(t *testing.T) {
	routesDir, layoutMap := fullFeatureLayoutMap(t)
	var stdout bytes.Buffer

	if err := renderLayoutMap(&stdout, routesDir, layoutMap, ansi.New(true)); err != nil {
		t.Fatalf("renderLayoutMap() error = %v, want nil", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"\x1b[1m" + "Layout map" + "\x1b[0m",
		"\x1b[1m" + "/" + "\x1b[0m",
		"\x1b[1m" + "users/" + "\x1b[0m",
		"\x1b[36m" + "layout" + "\x1b[0m",
		"\x1b[32m" + "page" + "\x1b[0m",
		"\x1b[33m" + "fragment (not wrapped)" + "\x1b[0m",
		"\x1b[35m" + "action (layout-aware)" + "\x1b[0m",
		"\x1b[2m" + "params:" + "\x1b[0m",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("styled layout map missing %q\n%s", want, output)
		}
	}

	sourcePath := routeSourceDisplayPath(routesDir, "users/by_id/route.go")
	if !strings.Contains(output, sourcePath) {
		t.Fatalf("styled layout map missing plain source path %q\n%s", sourcePath, output)
	}
	if strings.Contains(output, "\x1b[32m"+sourcePath) || strings.Contains(output, "\x1b[36m"+sourcePath) {
		t.Fatalf("styled layout map styled source path %q\n%s", sourcePath, output)
	}
}

func TestRenderRouteLayoutMapShowsMountedOwner(t *testing.T) {
	layoutMap := wiring.RouteLayoutMap{
		Root: &wiring.RouteLayoutMapNode{
			Name: "/",
			Pages: []wiring.RouteLayoutMapPage{
				{
					Methods: []string{"GET", "HEAD"},
					Route:   "/admin/reports",
					Source:  "../mounts/reports/route.go",
					Owner:   "admin/reports/route.go",
				},
			},
		},
	}
	var stdout bytes.Buffer

	if err := renderLayoutMap(&stdout, "/app/routes", layoutMap, ansi.New(false)); err != nil {
		t.Fatalf("renderLayoutMap() error = %v, want nil", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"mounts/reports/route.go",
		"owner: admin/reports/route.go",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("layout map = %q, want %q", output, want)
		}
	}
}

func fullFeatureLayoutMap(t *testing.T) (string, wiring.RouteLayoutMap) {
	t.Helper()

	routesDir, err := filepath.EvalSymlinks(filepath.Join("..", "..", "..", "..", "..", "examples", "full_feature", "app", "routes"))
	if err != nil {
		t.Fatalf("EvalSymlinks(routes dir) error = %v", err)
	}
	tree, err := routing.Scan(routesDir)
	if err != nil {
		t.Fatalf("Scan(routes dir) error = %v", err)
	}
	layoutMap, err := wiring.BuildRouteLayoutMap(routing.BuildManifest(*tree))
	if err != nil {
		t.Fatalf("BuildRouteLayoutMap() error = %v", err)
	}
	return routesDir, layoutMap
}
