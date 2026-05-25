package goldrcli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/goldrcli/templtool"
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
	writeFile(t, root, "app/routes/route.go", routeDeclarationSource("routes", "page", routeDeclarationOptions{Page: true}))

	requireRunSuccess(t, "generate", "--app-root", root)
	requireRunSuccess(t, "check", "--app-root", root)
}

func TestRunCheckAcceptsOpaqueRouteDeclarationMetadata(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/routemeta\n\ngo 1.26.3\n")
	writeFile(t, root, "app/routes/users/route.go", routeDeclarationSource("users", "page", routeDeclarationOptions{
		Page:  true,
		Name:  "users.index",
		Title: "Users",
		Labels: []routeDeclarationLabel{
			{Key: "app.nav", Value: "users"},
			{Key: "app.permission", Value: "view"},
		},
	}))

	requireRunSuccess(t, "generate", "--app-root", root)
	requireRunSuccess(t, "check", "--app-root", root)
}

func TestRunCheckRejectsMissingRouteDeclarationHandler(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/missinghandler\n\ngo 1.26.3\n")
	writeFile(t, root, "app/routes/route.go", routeDeclarationSourceWithoutHandlers("routes", "page", routeDeclarationOptions{Page: true}))

	requireCheckFailureContains(t, root, "goldr check:", checkCodeRouteGenerate, "route.go", "page", "route-package declaration")
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

func TestRunCheckReportsMissingMountedURLHelperFile(t *testing.T) {
	root := tempMountedRouteApp(t)

	requireRunSuccess(t, "generate", "--app-root", root)
	if err := os.Remove(filepath.Join(root, "app", "mounts", "reports", "goldr_gen.go")); err != nil {
		t.Fatalf("Remove(mount urls) error = %v", err)
	}

	requireCheckFailureContains(t, root, "goldr check:", checkCodeGeneratedFiles, "app/mounts/reports/goldr_gen.go", "is missing")
}

func TestRunCheckReportsMissingTemplTool(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/notempltool\n\ngo 1.26.3\n")
	writeFile(t, root, "app/routes/route.go", routeDeclarationSource("routes", "page", routeDeclarationOptions{Page: true}))
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
	writeFile(t, root, "app/routes/layout.go", "package routes\n")

	requireCheckFailureContains(t, root, checkCodeRenderUnit, "app/routes/layout.go", "layout /", "missing matching .templ file")
}

func TestRunCheckRejectsOldRouteSurfaceFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/oldsurface\n\ngo 1.26.3\n")
	writeFile(t, root, "app/routes/page.go", "package routes\n")
	writeFile(t, root, "app/routes/users/frag_table.go", "package users\n")
	writeFile(t, root, "app/routes/users/actions.go", "package users\n")

	requireCheckFailureContains(t, root, checkCodeRouteScan, "app/routes/page.go", "app/routes/users/frag_table.go", "app/routes/users/actions.go", "route surface belongs in route.go")
}

func TestRunCheckReportsRuntimeGenerationErrors(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/ambiguousroutes\n\ngo 1.26.3\n")
	writeFile(t, root, "app/routes/users/by_id/route.go", routeDeclarationSource("by_id", "page", routeDeclarationOptions{Page: true}))
	writeFile(t, root, "app/routes/users/by_id/page.templ", "package by_id\n\ntempl PageView() {}\n")
	writeFile(t, root, "app/routes/users/by_slug/route.go", routeDeclarationSource("by_slug", "page", routeDeclarationOptions{Page: true}))
	writeFile(t, root, "app/routes/users/by_slug/page.templ", "package by_slug\n\ntempl PageView() {}\n")

	requireCheckFailureContains(t, root, "goldr check:", checkCodeRouteGenerate, "ambiguous runtime route", "users/by_id/route.go", "users/by_slug/route.go")
}

func TestRunCheckReportsURLHelperGenerationErrors(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/badurls\n\ngo 1.26.3\n")
	writeFile(t, root, "app/routes/users/route.go", routeDeclarationSource("users", "page", routeDeclarationOptions{
		Actions: []routeDeclarationAction{{Helper: "FuncPost", Name: "path", Func: "postPath"}},
	}))

	requireCheckFailureContains(t, root, "goldr check:", checkCodeURLGenerate, "ambiguous URL helper", "Path method")
}
