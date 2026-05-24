package wiring

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mobiletoly/goldr/internal/routing"
)

func generateOK(t *testing.T, manifest routing.Manifest) string {
	t.Helper()

	source, err := GenerateManifest(manifest, GenerateOptions{
		PackageName:         "routes",
		RouteRootImportPath: "example.com/app/routes",
	})
	if err != nil {
		t.Fatalf("GenerateManifest() error = %v, want nil", err)
	}
	return string(source)
}

func inspectorSupportOK(t *testing.T) string {
	t.Helper()

	source, err := GenerateInspectorSupport("goldrinspect")
	if err != nil {
		t.Fatalf("GenerateInspectorSupport() error = %v, want nil", err)
	}
	return string(source)
}

func writeGeneratedRoutes(t *testing.T, root string, source string) {
	t.Helper()

	writeTempFile(t, root, "routes/goldr_gen.go", source)
	writeTempFile(t, root, "internal/goldrinspect/goldr_gen.go", inspectorSupportOK(t))
}

func writeGeneratedFragmentWrappers(t *testing.T, root string, manifest routing.Manifest) {
	t.Helper()

	manifest.Root = filepath.Join(root, "routes")
	files, err := GenerateFragmentWrappers(manifest, GenerateOptions{
		RouteRootImportPath: "example.com/app/routes",
	})
	if err != nil {
		t.Fatalf("GenerateFragmentWrappers() error = %v, want nil", err)
	}
	for _, file := range files {
		writeTempFile(t, root, filepath.Join("routes", filepath.FromSlash(file.Dir), GeneratedFileName), string(file.Content))
	}
}

func generateURLHelpersOK(t *testing.T, manifest routing.Manifest) string {
	t.Helper()

	source, err := GenerateURLHelpers(manifest, GenerateURLOptions{PackageName: "urls"})
	if err != nil {
		t.Fatalf("GenerateURLHelpers() error = %v, want nil", err)
	}
	return string(source)
}

func trimGeneratedLineIndent(source string) string {
	lines := strings.Split(source, "\n")
	for index, line := range lines {
		lines[index] = strings.TrimLeft(line, "\t ")
	}
	return strings.Join(lines, "\n")
}

func tempGoldrModule(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Abs(repo root) error = %v", err)
	}
	writeTempFile(t, tempDir, "go.mod", `module example.com/app

go 1.26.3

require (
	github.com/a-h/templ v0.3.1020
	github.com/mobiletoly/goldr v0.0.0
)

replace github.com/mobiletoly/goldr => `+filepath.ToSlash(repoRoot)+`
`)
	writeTempFile(t, tempDir, "go.sum", `github.com/a-h/templ v0.3.1020 h1:ypAT/L5ySWEnZ6Zft/5yfoWXYYkhFNvEFOeeqecg4tw=
github.com/a-h/templ v0.3.1020/go.mod h1:A2DlK61v+K+NRoGnhmYbNYVmtYHcFO5/AisMvBdDxTM=
`)
	return tempDir
}

func runGoTest(t *testing.T, dir string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, "go", "test", "./...")
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go test ./... error = %v\n%s", err, output)
	}
}

func rootManifest() routing.Manifest {
	return routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/", Unit: completeUnit("page.go")},
		},
		Layouts: []routing.ManifestLayout{
			{RoutePrefix: "/", Unit: completeUnit("layout.go")},
		},
	}
}

func runtimeManifest() routing.Manifest {
	return routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/", Unit: completeUnit("page.go")},
			{Route: "/users", Unit: completeUnit("users/page.go")},
			{Route: "/users/{id}", Params: []string{"id"}, Unit: completeUnit("users/by_id/page.go")},
		},
		Layouts: []routing.ManifestLayout{
			{RoutePrefix: "/", Unit: completeUnit("layout.go")},
			{RoutePrefix: "/users", Unit: completeUnit("users/layout.go")},
			{RoutePrefix: "/users/{id}", Params: []string{"id"}, Unit: completeUnit("users/by_id/layout.go")},
		},
		Fragments: []routing.ManifestFragment{
			{Name: "table", RoutePrefix: "/users", Unit: completeUnit("users/frag_table.go")},
			{Name: "row", RoutePrefix: "/users/{id}", Params: []string{"id"}, Unit: completeUnit("users/by_id/frag_row.go")},
		},
	}
}

func staticPriorityManifest() routing.Manifest {
	return routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/users/{id}", Params: []string{"id"}, Unit: completeUnit("users/by_id/page.go")},
			{Route: "/users/profile", Unit: completeUnit("users/profile/page.go")},
		},
		Fragments: []routing.ManifestFragment{
			{Name: "table", RoutePrefix: "/users", Unit: completeUnit("users/frag_table.go")},
		},
		Actions: []routing.ManifestAction{
			{Method: "POST", Route: "/users/profile", GoFile: "users/profile/actions.go", Function: "PostProfile", Suffix: "Profile", Segment: "profile"},
		},
	}
}

func urlHelperManifest() routing.Manifest {
	return routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/", Unit: completeUnit("page.go")},
			{Route: "/{slug}", Params: []string{"slug"}, Unit: completeUnit("by_slug/page.go")},
			{Route: "/settings/build-info", Unit: completeUnit("settings/build_info/page.go")},
			{Route: "/users", Unit: completeUnit("users/page.go")},
			{Route: "/users/{id}", Params: []string{"id"}, Unit: completeUnit("users/by_id/page.go")},
			{Route: "/orgs/{org_id}/users/{user_id}", Params: []string{"org_id", "user_id"}, Unit: completeUnit("orgs/by_org_id/users/by_user_id/page.go")},
		},
		Fragments: []routing.ManifestFragment{
			{Name: "table", RoutePrefix: "/users", Unit: completeUnit("users/frag_table.go")},
		},
		Actions: []routing.ManifestAction{
			{Method: "POST", Route: "/users/create", GoFile: "users/actions.go", Function: "PostCreate", Suffix: "Create", Segment: "create"},
			{Method: "POST", Route: "/users/save-preview", GoFile: "users/actions.go", Function: "PostSavePreview", Suffix: "SavePreview", Segment: "save-preview"},
			{Method: "PATCH", Route: "/users/{id}/profile", Params: []string{"id"}, GoFile: "users/by_id/actions.go", Function: "PatchProfile", Suffix: "Profile", Segment: "profile"},
			{Method: "DELETE", Route: "/users/{id}/profile", Params: []string{"id"}, GoFile: "users/by_id/actions.go", Function: "DeleteProfile", Suffix: "Profile", Segment: "profile"},
		},
	}
}

func fullFeatureManifest() routing.Manifest {
	return routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/", Unit: completeUnit("page.go")},
			{Route: "/admin", Unit: completeUnit("admin/page.go")},
			{Route: "/protected-resource-demo", Unit: completeUnit("protected_resource_demo/page.go")},
			{Route: "/settings", Unit: completeUnit("settings/page.go")},
			{Route: "/sign-in", Unit: completeUnit("sign_in/page.go")},
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
			{Method: "POST", Route: "/protected-resource-demo/reveal-secret", GoFile: "protected_resource_demo/actions.go", Function: "PostRevealSecret", Suffix: "RevealSecret", Segment: "reveal-secret"},
			{Method: "POST", Route: "/protected-resource-demo/sign-out", GoFile: "protected_resource_demo/actions.go", Function: "PostSignOut", Suffix: "SignOut", Segment: "sign-out"},
			{Method: "POST", Route: "/sign-in", GoFile: "sign_in/actions.go", Function: "PostIndex", Suffix: "Index"},
			{Method: "POST", Route: "/users/create", GoFile: "users/actions.go", Function: "PostCreate", Suffix: "Create", Segment: "create"},
			{Method: "POST", Route: "/users/save-preview", GoFile: "users/actions.go", Function: "PostSavePreview", Suffix: "SavePreview", Segment: "save-preview"},
		},
		Middlewares: []routing.ManifestMiddleware{
			{RoutePrefix: "/", GoFile: "middleware.go"},
		},
	}
}

func completeUnit(goFile string) routing.RenderUnit {
	return routing.RenderUnit{
		GoFile:    goFile,
		TemplFile: strings.TrimSuffix(goFile, ".go") + ".templ",
		HasTempl:  true,
	}
}

func writeTempFile(t *testing.T, root, name, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
