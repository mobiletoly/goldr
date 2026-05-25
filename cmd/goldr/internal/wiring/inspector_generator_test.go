package wiring

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
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
			"route=/users/table",
			"source=app/routes/users/frag_table.templ",
			"go=app/routes/users/frag_table.go",
		},
		"users/by_id": {
			"package by_id",
			"func renderFragRow(component templ.Component) templ.Component",
			"route=/users/{id}/row",
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

func TestGenerateRoutePackageFilesWritesAdaptersAndReconstructsImports(t *testing.T) {
	tempDir := tempGoldrModule(t)
	manifest := routing.Manifest{
		Root: filepath.Join(tempDir, "routes"),
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/reports",
				GoFile: "reports/route.go",
				Imports: []routing.RouteImportDeclaration{
					{Name: "goldr", Path: "github.com/mobiletoly/goldr"},
					{Name: "http", Path: "net/http"},
					{Name: "report", Path: "example.com/app/pages/report"},
				},
				Kind: "local",
				Page: &routing.RouteHandlerDeclaration{Handler: "report.Page"},
				Fragments: []routing.RouteFragmentDeclaration{
					{Name: "preview", Segment: "preview", SymbolName: "Preview", Handler: "preview"},
				},
				Actions: []routing.RouteActionDeclaration{
					{Method: "POST", Index: true, SymbolName: "Index", Handler: "report.PostIndex"},
				},
			},
			{
				Route:  "/cohort",
				GoFile: "cohort/route.go",
				Imports: []routing.RouteImportDeclaration{
					{Name: "cohort", Path: "example.com/app/pages/cohortexplorer", Explicit: true},
					{Name: "goldr", Path: "github.com/mobiletoly/goldr"},
					{Name: "http", Path: "net/http"},
				},
				Kind: "kit",
				Page: &routing.RouteHandlerDeclaration{Handler: "cohort.Kit.Page"},
				Actions: []routing.RouteActionDeclaration{
					{Method: "POST", Name: "export", Segment: "export", SymbolName: "Export", Handler: "cohort.Kit.PostExport"},
				},
				Kit: &routing.RouteKitDeclaration{
					New: "newKit",
				},
			},
		},
	}
	writeTempFile(t, tempDir, "routes/reports/route.go", `package reports

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func preview(_ *http.Request) goldr.RouteResponse {
	return goldr.Text{Body: "preview"}
}
`)
	writeTempFile(t, tempDir, "routes/cohort/route.go", `package cohort

import "net/http"

type portalContext struct{}
type Kit struct{}

func portal(_ *http.Request) portalContext {
	return portalContext{}
}

func newKit(_ *http.Request) Kit {
	return Kit{}
}
`)

	files, err := GenerateRoutePackageFiles(manifest, GenerateOptions{RouteRootImportPath: "example.com/app/routes"})
	if err != nil {
		t.Fatalf("GenerateRoutePackageFiles() error = %v, want nil", err)
	}
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}

	byDir := make(map[string]string)
	for _, file := range files {
		byDir[file.Dir] = string(file.Content)
		if _, err := parser.ParseFile(token.NewFileSet(), GeneratedFileName, file.Content, parser.SkipObjectResolution); err != nil {
			t.Fatalf("ParseFile(%q) error = %v\n%s", file.Dir, err, file.Content)
		}
	}
	for _, want := range []string{
		`"example.com/app/pages/report"`,
		`// Route is read by goldr tooling; this reference keeps editors from marking it unused.`,
		`var _ = Route`,
		`func GoldrRoutePage(r *http.Request) goldr.RouteResponse`,
		`return report.Page(r)`,
		`func GoldrRouteFragPreview(r *http.Request) goldr.RouteResponse`,
		`return preview(r)`,
		`func GoldrRoutePostIndex(r *http.Request) goldr.RouteResponse`,
		`return report.PostIndex(r)`,
	} {
		if !strings.Contains(byDir["reports"], want) {
			t.Fatalf("reports generated file missing %q:\n%s", want, byDir["reports"])
		}
	}
	for _, want := range []string{
		`cohort "example.com/app/pages/cohortexplorer"`,
		`var _ = Route`,
		`goldrKit := newKit(r)`,
		`return cohort.Kit.Page(goldrKit, r)`,
		`return cohort.Kit.PostExport(goldrKit, r)`,
	} {
		if !strings.Contains(byDir["cohort"], want) {
			t.Fatalf("cohort generated file missing %q:\n%s", want, byDir["cohort"])
		}
	}
}

func TestGenerateRoutePackageFilesQualifiesMountedLocalHandlers(t *testing.T) {
	tempDir := tempGoldrModule(t)
	manifest := routing.Manifest{
		Root: filepath.Join(tempDir, "routes"),
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/admin/reports",
				GoFile: "admin/reports/route.go",
				Kind:   "mounted-kit",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Kit.Page"},
				Kit: &routing.RouteKitDeclaration{
					New: "newKit",
				},
				Source:  "../mounts/reports/route.go",
				Adapter: "MountReports",
				Mount:   &routing.RouteMountDeclaration{Path: "reports", Owner: "admin/reports/route.go"},
			},
		},
	}
	writeTempFile(t, tempDir, "routes/admin/reports/route.go", `package reports

import "net/http"

type Kit struct{}

func newKit(_ *http.Request) Kit {
	return Kit{}
}
`)
	writeTempFile(t, tempDir, "mounts/reports/route.go", `package reports

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

type Kit struct{}

func (Kit) Page(_ *http.Request) goldr.RouteResponse {
	return goldr.Text{Body: "ok"}
}
`)

	files, err := GenerateRoutePackageFiles(manifest, GenerateOptions{RouteRootImportPath: "example.com/app/routes"})
	if err != nil {
		t.Fatalf("GenerateRoutePackageFiles() error = %v, want nil", err)
	}
	if len(files) != 1 {
		t.Fatalf("len(files) = %d, want 1", len(files))
	}
	source := string(files[0].Content)
	for _, want := range []string{
		`goldrmount_reports "example.com/app/mounts/reports"`,
		`var _ = goldrmount_reports.Route`,
		`func GoldrRouteMountReportsPage(r *http.Request) goldr.RouteResponse`,
		`return goldrmount_reports.Kit.Page(goldrKit, r)`,
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("generated file missing %q:\n%s", want, source)
		}
	}
	if _, err := parser.ParseFile(token.NewFileSet(), GeneratedFileName, files[0].Content, parser.SkipObjectResolution); err != nil {
		t.Fatalf("ParseFile() error = %v\n%s", err, source)
	}
}

func TestGenerateRoutePackageFilesRejectsImportedSelectorWithoutMatchingImportName(t *testing.T) {
	tempDir := tempGoldrModule(t)
	manifest := routing.Manifest{
		Root: filepath.Join(tempDir, "routes"),
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/reports",
				GoFile: "reports/route.go",
				Imports: []routing.RouteImportDeclaration{
					{Name: "goldr", Path: "github.com/mobiletoly/goldr"},
					{Name: "handlers", Path: "example.com/app/pages/handlers"},
				},
				Kind: "local",
				Page: &routing.RouteHandlerDeclaration{Handler: "view.Page"},
			},
		},
	}
	writeTempFile(t, tempDir, "routes/reports/route.go", `package reports

import (
	"example.com/app/pages/handlers"
	"github.com/mobiletoly/goldr"
)

var Route = goldr.RouteDef{
	Page: goldr.FuncPage(view.Page),
}
`)

	_, err := GenerateRoutePackageFiles(manifest, GenerateOptions{RouteRootImportPath: "example.com/app/routes"})
	if err == nil {
		t.Fatal("GenerateRoutePackageFiles() error = nil, want missing import alias error")
	}
	for _, want := range []string{"reports/route.go", "view", "explicit import alias"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("GenerateRoutePackageFiles() error = %q, want containing %q", err, want)
		}
	}
}

func TestGenerateRoutePackageFilesRejectsMissingLocalHandlers(t *testing.T) {
	tests := []struct {
		name  string
		route routing.ManifestRouteDeclaration
		want  string
	}{
		{
			name: "page",
			route: routing.ManifestRouteDeclaration{
				Route:  "/reports",
				GoFile: "reports/route.go",
				Kind:   "local",
				Page:   &routing.RouteHandlerDeclaration{Handler: "page"},
			},
			want: "page",
		},
		{
			name: "fragment",
			route: routing.ManifestRouteDeclaration{
				Route:  "/reports",
				GoFile: "reports/route.go",
				Kind:   "local",
				Fragments: []routing.RouteFragmentDeclaration{
					{Name: "table", Segment: "table", SymbolName: "Table", Handler: "fragTable"},
				},
			},
			want: "fragTable",
		},
		{
			name: "action",
			route: routing.ManifestRouteDeclaration{
				Route:  "/reports",
				GoFile: "reports/route.go",
				Kind:   "local",
				Actions: []routing.RouteActionDeclaration{
					{Method: "POST", Name: "create", Segment: "create", SymbolName: "Create", Handler: "postCreate"},
				},
			},
			want: "postCreate",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tempDir := tempGoldrModule(t)
			manifest := routing.Manifest{
				Root:   filepath.Join(tempDir, "routes"),
				Routes: []routing.ManifestRouteDeclaration{test.route},
			}
			writeTempFile(t, tempDir, "routes/reports/route.go", `package reports

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{}
`)

			_, err := GenerateRoutePackageFiles(manifest, GenerateOptions{RouteRootImportPath: "example.com/app/routes"})
			if err == nil {
				t.Fatal("GenerateRoutePackageFiles() error = nil, want missing handler error")
			}
			for _, want := range []string{"reports/route.go", test.want, "route-package declaration"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("GenerateRoutePackageFiles() error = %q, want containing %q", err, want)
				}
			}
		})
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
