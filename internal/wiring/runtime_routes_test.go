package wiring

import (
	"errors"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr/internal/routing"
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
				{Route: "/users/frag-table", Unit: completeUnit("users/frag_table/page.go")},
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

func TestGenerateManifestRejectsNestedRuntimeWithoutImportPath(t *testing.T) {
	_, err := GenerateManifest(runtimeManifest(), GenerateOptions{PackageName: "routes"})
	if !errors.Is(err, ErrInvalidRouteRootImportPath) {
		t.Fatalf("GenerateManifest() error = %v, want ErrInvalidRouteRootImportPath", err)
	}
}
