package wiring

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
)

func TestGenerateManifestRejectsAmbiguousPageRoutes(t *testing.T) {
	manifest := routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/users/{id}", Params: []string{"id"}, Unit: completeUnit("users/by_id/page.go")},
			{Route: "/users/{slug}", Params: []string{"slug"}, Unit: completeUnit("users/by_slug/page.go")},
		},
	}

	_, err := GenerateManifest(manifest, GenerateOptions{PackageName: "routes", RouteRootImportPath: "example.com/app/routes"})
	if !errors.Is(err, ErrAmbiguousPageRoute) {
		t.Fatalf("GenerateManifest() error = %v, want ErrAmbiguousPageRoute", err)
	}
}

func TestGenerateManifestRejectsAmbiguousFragmentRoutes(t *testing.T) {
	tests := []routing.Manifest{
		{
			Pages: []routing.ManifestPage{
				{Route: "/users/table", Unit: completeUnit("users/frag_table/page.go")},
			},
			Fragments: []routing.ManifestFragment{
				{Name: "table", RoutePrefix: "/users", Unit: completeUnit("users/frag_table.go")},
			},
		},
		{
			Fragments: []routing.ManifestFragment{
				{Name: "row", RoutePrefix: "/users/{id}", Params: []string{"id"}, Unit: completeUnit("users/by_id/frag_row.go")},
				{Name: "row", RoutePrefix: "/users/{slug}", Params: []string{"slug"}, Unit: completeUnit("users/by_slug/frag_row.go")},
			},
		},
	}

	for _, test := range tests {
		_, err := GenerateManifest(test, GenerateOptions{PackageName: "routes", RouteRootImportPath: "example.com/app/routes"})
		if !errors.Is(err, ErrAmbiguousRuntimeRoute) {
			t.Fatalf("GenerateManifest() error = %v, want ErrAmbiguousRuntimeRoute", err)
		}
	}
}

func TestGenerateManifestRejectsAmbiguousActionRoutes(t *testing.T) {
	tests := []struct {
		name     string
		manifest routing.Manifest
	}{
		{
			name: "same method and path",
			manifest: routing.Manifest{
				Actions: []routing.ManifestAction{
					{Method: "POST", Route: "/users/create", GoFile: "users/actions.go", Function: "PostCreate"},
					{Method: "POST", Route: "/users/create", GoFile: "users/actions.go", Function: "PostCreateAgain"},
				},
			},
		},
		{
			name: "same dynamic shape with different methods",
			manifest: routing.Manifest{
				Actions: []routing.ManifestAction{
					{Method: "POST", Route: "/users/{id}", Params: []string{"id"}, GoFile: "users/by_id/actions.go", Function: "PostIndex"},
					{Method: "DELETE", Route: "/users/{slug}", Params: []string{"slug"}, GoFile: "users/by_slug/actions.go", Function: "DeleteIndex"},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := GenerateManifest(test.manifest, GenerateOptions{PackageName: "routes", RouteRootImportPath: "example.com/app/routes"})
			if !errors.Is(err, ErrAmbiguousRuntimeRoute) {
				t.Fatalf("GenerateManifest() error = %v, want ErrAmbiguousRuntimeRoute", err)
			}
		})
	}
}

func TestGenerateManifestRejectsDeclarationEndpointCollisions(t *testing.T) {
	tests := []struct {
		name     string
		manifest routing.Manifest
	}{
		{
			name: "declared page collides with old page",
			manifest: routing.Manifest{
				Pages: []routing.ManifestPage{
					{Route: "/users", Unit: completeUnit("users/page.go")},
				},
				Routes: []routing.ManifestRouteDeclaration{
					{
						Route:  "/users",
						GoFile: "users_decl/route.go",
						Kind:   "local",
						Page:   &routing.RouteHandlerDeclaration{Handler: "page"},
					},
				},
			},
		},
		{
			name: "declared action collides with old action",
			manifest: routing.Manifest{
				Actions: []routing.ManifestAction{
					{Method: "POST", Route: "/users/save", GoFile: "users/actions.go", Function: "PostSave"},
				},
				Routes: []routing.ManifestRouteDeclaration{
					{
						Route:  "/users",
						GoFile: "users_decl/route.go",
						Kind:   "local",
						Actions: []routing.RouteActionDeclaration{
							{Method: "POST", Name: "save", Segment: "save", SymbolName: "Save", Handler: "postSave"},
						},
					},
				},
			},
		},
		{
			name: "declared index fragment collides with old page",
			manifest: routing.Manifest{
				Pages: []routing.ManifestPage{
					{Route: "/users", Unit: completeUnit("users/page.go")},
				},
				Routes: []routing.ManifestRouteDeclaration{
					{
						Route:  "/users",
						GoFile: "users_decl/route.go",
						Kind:   "local",
						Fragments: []routing.RouteFragmentDeclaration{
							{Name: "index", SymbolName: "Index", Index: true, Handler: "options"},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := GenerateManifest(test.manifest, GenerateOptions{PackageName: "routes", RouteRootImportPath: "example.com/app/routes"})
			if !errors.Is(err, ErrAmbiguousRuntimeRoute) {
				t.Fatalf("GenerateManifest() error = %v, want ErrAmbiguousRuntimeRoute", err)
			}
			if !strings.Contains(err.Error(), "route.go") {
				t.Fatalf("GenerateManifest() error = %v, want route.go source", err)
			}
		})
	}
}

func TestGenerateManifestRejectsDuplicateNavKeyInCanonicalTrail(t *testing.T) {
	manifest := routing.Manifest{
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/device-models/{model_id}",
				Params: []string{"model_id"},
				GoFile: "device_models/by_model_id/route.go",
				Kind:   "local",
				Nav:    routing.RouteNavDeclaration{Key: "model"},
				Page:   &routing.RouteHandlerDeclaration{Handler: "page"},
			},
			{
				Route:  "/device-models/{model_id}/firmware-models/{firmware_model_id}",
				Params: []string{"model_id", "firmware_model_id"},
				GoFile: "device_models/by_model_id/firmware_models/by_firmware_model_id/route.go",
				Kind:   "local",
				Nav:    routing.RouteNavDeclaration{Key: "model"},
				Page:   &routing.RouteHandlerDeclaration{Handler: "page"},
			},
		},
	}

	_, err := GenerateManifest(manifest, GenerateOptions{PackageName: "routes", RouteRootImportPath: "example.com/app/routes"})
	if !errors.Is(err, ErrAmbiguousRuntimeRoute) {
		t.Fatalf("GenerateManifest() error = %v, want ErrAmbiguousRuntimeRoute", err)
	}
	if !strings.Contains(err.Error(), `duplicate Nav.Key "model"`) {
		t.Fatalf("GenerateManifest() error = %v, want duplicate Nav.Key detail", err)
	}
}

func TestGenerateManifestDoesNotRejectURLHelperCollisions(t *testing.T) {
	manifest := routing.Manifest{
		Actions: []routing.ManifestAction{
			{Method: "POST", Route: "/users/path", GoFile: "users/actions.go", Function: "PostPath"},
		},
	}

	source, err := GenerateManifest(manifest, GenerateOptions{
		PackageName:         "routes",
		RouteRootImportPath: "example.com/app/routes",
	})
	if err != nil {
		t.Fatalf("GenerateManifest() error = %v, want nil", err)
	}
	if !strings.Contains(string(source), "users/actions.go:PostPath") {
		t.Fatalf("generated source missing action route surface row:\n%s", source)
	}
}

func TestRuntimeRoutesUseMountedLiveRoutePathForMiddleware(t *testing.T) {
	manifest := routing.Manifest{
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:            "/admin/reports/table",
				GoFile:           "admin/reports/route.go",
				MiddlewareGoFile: "admin/reports/table/route.go",
				Kind:             "mounted-kit",
				Source:           "../mounts/reports/table/route.go",
				Adapter:          "MountReportsTable",
				Mount:            &routing.RouteMountDeclaration{Path: "reports", Owner: "admin/reports/route.go"},
				Page:             &routing.RouteHandlerDeclaration{Handler: "shared.Kit.Table"},
				Kit:              &routing.RouteKitDeclaration{New: "newReportKit"},
			},
		},
		Middlewares: []routing.ManifestMiddleware{
			{RoutePrefix: "/", GoFile: "middleware.go"},
			{RoutePrefix: "/admin/reports", GoFile: "admin/reports/middleware.go"},
			{RoutePrefix: "/admin/reports/table", GoFile: "admin/reports/table/middleware.go"},
		},
	}

	routes, err := runtimeRoutes(manifest)
	if err != nil {
		t.Fatalf("runtimeRoutes() error = %v, want nil", err)
	}
	if got, want := len(routes), 1; got != want {
		t.Fatalf("len(routes) = %d, want %d", got, want)
	}
	got := manifestMiddlewareGoFiles(routes[0].page.middlewares)
	want := []string{"middleware.go", "admin/reports/middleware.go", "admin/reports/table/middleware.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("middleware = %#v, want %#v", got, want)
	}
}

func TestGenerateManifestRejectsNestedRuntimeWithoutImportPath(t *testing.T) {
	_, err := GenerateManifest(runtimeManifest(), GenerateOptions{PackageName: "routes"})
	if !errors.Is(err, ErrInvalidRouteRootImportPath) {
		t.Fatalf("GenerateManifest() error = %v, want ErrInvalidRouteRootImportPath", err)
	}
}

func manifestMiddlewareGoFiles(middlewares []routing.ManifestMiddleware) []string {
	result := make([]string, 0, len(middlewares))
	for _, middleware := range middlewares {
		result = append(result, middleware.GoFile)
	}
	return result
}
