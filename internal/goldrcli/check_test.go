package goldrcli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mobiletoly/goldr/internal/goldrcli/templtool"
)

func TestRunCheckCleanApp(t *testing.T) {
	root := tempGenerateApp(t)

	requireRunSuccess(t, "generate", "--app-root", root)
	runTemplGenerate(t, root)
	requireRunSuccess(t, "check", "--app-root", root)
}

func TestRunGenerateAndCheckAcceptPageWithoutTempl(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/pageonly\n\ngo 1.26.3\n")
	writeFile(t, root, "app/routes/page.go", `package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(_ *http.Request) goldr.RouteResponse {
	return goldr.Redirect{Location: "/sign-in", Status: http.StatusSeeOther}
}
`)

	requireRunSuccess(t, "generate", "--app-root", root)
	requireRunSuccess(t, "check", "--app-root", root)
}

func TestRunCheckSkipsAssetsWhenOnlyBuildExists(t *testing.T) {
	root := tempGenerateApp(t)
	requireRunSuccess(t, "generate", "--app-root", root)
	runTemplGenerate(t, root)
	writeFile(t, root, "assets/build/app.css", "body {}\n")

	requireRunSuccess(t, "check", "--app-root", root)
}

func TestRunCheckValidatesManagedAssets(t *testing.T) {
	root := tempGenerateApp(t)
	requireRunSuccess(t, "generate", "--app-root", root)
	runTemplGenerate(t, root)
	writeFile(t, root, "assets/build/app.css", "body {}\n")
	requireRunSuccess(t, "assets", "dist", "--app-root", root)

	requireRunSuccess(t, "check", "--app-root", root)
}

func TestRunCheckReportsStaleManagedAssets(t *testing.T) {
	root := tempGenerateApp(t)
	requireRunSuccess(t, "generate", "--app-root", root)
	runTemplGenerate(t, root)
	writeFile(t, root, "assets/build/app.css", "body {}\n")
	requireRunSuccess(t, "assets", "dist", "--app-root", root)
	writeFile(t, root, "assets/build/app.css", "body { color: black; }\n")

	requireCheckFailureContains(t, root, "goldr check:", checkCodeAssets, "goldr-managed assets are not current", "go tool goldr generate", "assets/dist/app.", "is missing")
}

func TestRunCheckReportsMissingManagedAssetState(t *testing.T) {
	root := tempGenerateApp(t)
	requireRunSuccess(t, "generate", "--app-root", root)
	runTemplGenerate(t, root)
	writeFile(t, root, "assets/build/app.css", "body {}\n")
	requireRunSuccess(t, "assets", "dist", "--app-root", root)
	if err := os.Remove(filepath.Join(root, "assets", ".goldr", "assets.json")); err != nil {
		t.Fatalf("Remove(asset state) error = %v", err)
	}

	requireCheckFailureContains(t, root, "goldr check:", checkCodeAssets, "goldr-managed assets are not current", "assets/.goldr/assets.json", "is missing")
}

func TestRunCheckReportsRootResolutionProblems(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "app/routes/page.go", "package routes\n")
	writeFile(t, root, "app/routes/page.templ", "package routes\n\ntempl PageView() {}\n")

	requireCheckFailureContains(t, root, "goldr check:", checkCodeAppRoot, "could not find go.mod")
}

func TestRunCheckReportsMissingRoutesDirectory(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/missingroutes\n\ngo 1.26.3\n")

	requireCheckFailureContains(t, root, "goldr check:", checkCodeAppRoot, "app/routes")
}

func TestRunCheckReportsStaleAndMissingGeneratedFiles(t *testing.T) {
	root := tempGenerateApp(t)

	requireRunSuccess(t, "generate", "--app-root", root)
	runTemplGenerate(t, root)
	if err := os.WriteFile(filepath.Join(root, "app", "routes", "goldr_gen.go"), []byte("stale"), 0644); err != nil {
		t.Fatalf("WriteFile(stale routes) error = %v", err)
	}
	if err := os.Remove(filepath.Join(root, "app", "urls", "goldr_gen.go")); err != nil {
		t.Fatalf("Remove(urls) error = %v", err)
	}

	requireCheckFailureContains(t, root, "goldr check:", checkCodeGeneratedFiles, "app/routes/goldr_gen.go", "is stale", "app/urls/goldr_gen.go", "is missing")
}

func TestRunCheckReportsMissingTemplTool(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/notempltool\n\ngo 1.26.3\n")
	writeFile(t, root, "app/routes/page.go", `package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(_ *http.Request) goldr.RouteResponse {
	return goldr.NewPage(nil, goldr.PageMetadata{})
}
`)
	writeFile(t, root, "app/routes/page.templ", "package routes\n\ntempl PageView() {}\n")

	requireCheckFailureContains(t, root, "goldr check:", checkCodeTemplGenerated, "go tool templ is not available", templtool.InstallCommand)
}

func TestRunCheckReportsStaleTemplGeneratedFiles(t *testing.T) {
	root := tempGenerateApp(t)

	requireRunSuccess(t, "generate", "--app-root", root)
	writeFile(t, root, "app/routes/page.templ", "package routes\n\ntempl PageView() {<h1>Changed</h1>}\n")

	requireCheckFailureContains(t, root, "goldr check:", checkCodeTemplGenerated, "templ generated files are not up to date", "go tool goldr generate", "generated files are not up to date")
}

func TestRunCheckReportsInvalidRouteNames(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/invalidroutes\n\ngo 1.26.3\n")
	writeFile(t, root, "app/routes/Users/page.go", "package Users\n")

	requireCheckFailureContains(t, root, "goldr check:", checkCodeRouteScan, "app/routes/Users", "static route directories must use lowercase Go-safe names")
}

func TestRunCheckReportsMissingRenderUnitPairs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/missingpairs\n\ngo 1.26.3\n")
	writeFile(t, root, "app/routes/page.go", `package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(_ *http.Request) goldr.RouteResponse {
	return goldr.Redirect{Location: "/sign-in", Status: http.StatusSeeOther}
}
`)
	writeFile(t, root, "app/routes/layout.go", "package routes\n")
	writeFile(t, root, "app/routes/frag_row.go", "package routes\n")

	requireCheckFailureContains(t, root, checkCodeRenderUnit, "app/routes/layout.go", "layout /", "app/routes/frag_row.go", "fragment /:row", "missing matching .templ file")
}

func TestRunCheckReportsInvalidPageWithoutTempl(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name: "invalid signature",
			source: `package routes

import "net/http"

func Page(_ *http.Request) string { return "" }
`,
		},
		{
			name: "missing page function",
			source: `package routes

func Helper() {}
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, "go.mod", "module example.com/badpage\n\ngo 1.26.3\n")
			writeFile(t, root, "app/routes/page.go", test.source)

			requireCheckFailureContains(t, root, checkCodeRenderUnit, "app/routes/page.go", "page /", "page handlers must use func Page(*http.Request) goldr.RouteResponse")
		})
	}
}

func TestRunCheckReportsActionProblems(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/badactions\n\ngo 1.26.3\n")
	writeFile(t, root, "app/routes/users/actions.go", `package users

import "net/http"

func GetCreate(w http.ResponseWriter, r *http.Request) {}
func PostCreate(w http.ResponseWriter) {}
`)

	requireCheckFailureContains(t, root, checkCodeRouteScan, "app/routes/users/actions.go", "GetCreate", "GET action handlers are not supported", "PostCreate", "action handlers must use func Name")
}

func TestRunCheckReportsRuntimeGenerationErrors(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/ambiguousroutes\n\ngo 1.26.3\n")
	writeFile(t, root, "app/routes/users/by_id/page.go", `package by_id

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(_ *http.Request) goldr.RouteResponse {
	return goldr.NewPage(nil, goldr.PageMetadata{})
}
`)
	writeFile(t, root, "app/routes/users/by_id/page.templ", "package by_id\n\ntempl PageView() {}\n")
	writeFile(t, root, "app/routes/users/by_slug/page.go", `package by_slug

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(_ *http.Request) goldr.RouteResponse {
	return goldr.NewPage(nil, goldr.PageMetadata{})
}
`)
	writeFile(t, root, "app/routes/users/by_slug/page.templ", "package by_slug\n\ntempl PageView() {}\n")

	requireCheckFailureContains(t, root, "goldr check:", checkCodeRouteGenerate, "ambiguous runtime route", "users/by_id/page.go", "users/by_slug/page.go")
}

func TestRunCheckReportsURLHelperGenerationErrors(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/badurls\n\ngo 1.26.3\n")
	writeFile(t, root, "app/routes/users/actions.go", `package users

import "net/http"

func PostPath(w http.ResponseWriter, r *http.Request) {}
`)

	requireCheckFailureContains(t, root, "goldr check:", checkCodeURLGenerate, "ambiguous URL helper", "Path method")
}
