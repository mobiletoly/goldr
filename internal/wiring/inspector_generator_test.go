package wiring

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr/internal/routing"
)

func TestGenerateFragmentWrappersWritesPackageGoldrGenFileContent(t *testing.T) {
	tempDir := tempGoldrModule(t)
	manifest := routing.Manifest{
		Root: filepath.Join(tempDir, "routes"),
		Fragments: []routing.ManifestFragment{
			{Name: "table", RoutePrefix: "/users", Unit: completeUnit("users/frag_table.go")},
			{Name: "row", RoutePrefix: "/users/{id}", Params: []string{"id"}, Unit: completeUnit("users/by_id/frag_row.go")},
		},
	}
	writeTempFile(t, tempDir, "routes/users/frag_table.go", `package users

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func FragTable(r *http.Request) goldr.RouteResponse {
	return goldr.NewFragment(nil)
}
`)
	writeTempFile(t, tempDir, "routes/users/by_id/frag_row.go", `package by_id

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func FragRow(r *http.Request) goldr.RouteResponse {
	return goldr.NewFragment(nil)
}
`)

	files, err := GenerateFragmentWrappers(manifest, GenerateOptions{RouteRootImportPath: "example.com/app/routes"})
	if err != nil {
		t.Fatalf("GenerateFragmentWrappers() error = %v, want nil", err)
	}
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
	tests := map[string][]string{
		"users": {
			"package users",
			`"example.com/app/internal/goldrinspect"`,
			"func renderFragTable(component templ.Component) templ.Component",
			"route=/users/frag-table",
			"source=app/routes/users/frag_table.templ",
			"go=app/routes/users/frag_table.go",
		},
		"users/by_id": {
			"package by_id",
			"func renderFragRow(component templ.Component) templ.Component",
			"route=/users/{id}/frag-row",
		},
	}
	for _, file := range files {
		wants, ok := tests[file.Dir]
		if !ok {
			t.Fatalf("unexpected generated wrapper dir %q", file.Dir)
		}
		for _, want := range wants {
			if !strings.Contains(string(file.Content), want) {
				t.Fatalf("wrapper file %q missing %q:\n%s", file.Dir, want, file.Content)
			}
		}
		if _, err := parser.ParseFile(token.NewFileSet(), GeneratedFileName, file.Content, parser.SkipObjectResolution); err != nil {
			t.Fatalf("ParseFile(%q) error = %v\n%s", file.Dir, err, file.Content)
		}
		delete(tests, file.Dir)
	}
	if len(tests) != 0 {
		t.Fatalf("missing generated wrapper dirs: %v", tests)
	}
}

func TestGenerateManifestFragmentNamesWithEmptyUnderscoreParts(t *testing.T) {
	source := generateOK(t, routing.Manifest{
		Fragments: []routing.ManifestFragment{
			{Name: "table_", RoutePrefix: "/", Unit: completeUnit("frag_table_.go")},
			{Name: "user__row", RoutePrefix: "/", Unit: completeUnit("frag_user__row.go")},
		},
	})

	for _, want := range []string{
		"routeResponse := FragTable_(r)",
		"routeResponse := FragUser_Row(r)",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("generated source missing %q:\n%s", want, source)
		}
	}
	if _, err := parser.ParseFile(token.NewFileSet(), GeneratedFileName, source, parser.SkipObjectResolution); err != nil {
		t.Fatalf("ParseFile() error = %v\n%s", err, source)
	}
}
