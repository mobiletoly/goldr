package goldrcli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/goldrcli/project"
)

func TestRunInitCreatesStarterApp(t *testing.T) {
	root := t.TempDir()
	goMod := writeTemplToolModule(t, root, "example.com/initapp")

	code, stdout, stderr := runGoldr(t, "init", "--app-root", root)

	if code != 0 {
		t.Fatalf("Run(init) exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if got := readFile(t, filepath.Join(root, "go.mod")); got != goMod {
		t.Fatalf("go.mod = %q, want unchanged %q", got, goMod)
	}
	requireMissingFile(t, filepath.Join(root, "main.go"))

	for _, name := range []string{
		"app/routes/route.go",
		"app/routes/page.templ",
		"app/routes/layout.go",
		"app/routes/layout.templ",
		"app/routes/goldr_gen.go",
		"app/internal/goldrinspect/goldr_gen.go",
		"app/urls/goldr_gen.go",
	} {
		requireExistingFile(t, filepath.Join(root, filepath.FromSlash(name)))
	}

	routeSource := readFile(t, filepath.Join(root, "app", "routes", "route.go"))
	if !strings.Contains(routeSource, `"github.com/mobiletoly/goldr"`) {
		t.Fatalf("route.go = %q, want goldr import", routeSource)
	}
	if !strings.Contains(routeSource, "var Route = goldr.RouteDef") {
		t.Fatalf("route.go = %q, want RouteDef declaration", routeSource)
	}
	layoutTempl := readFile(t, filepath.Join(root, "app", "routes", "layout.templ"))
	if !strings.Contains(layoutTempl, `https://cdn.jsdelivr.net/npm/htmx.org@4.0.0-beta3`) {
		t.Fatalf("layout.templ = %q, want HTMX script", layoutTempl)
	}

	files, err := project.GenerateFiles(context.Background(), root)
	if err != nil {
		t.Fatalf("generateFiles(init app) error = %v", err)
	}
	for _, file := range files {
		got := readFile(t, file.Path)
		if !bytes.Equal([]byte(got), file.Content) {
			t.Fatalf("%s is stale\n--- got ---\n%s\n--- want ---\n%s", file.Path, got, file.Content)
		}
	}

	runTemplGenerate(t, root)
	requireRunSuccess(t, "check", "--app-root", root)

	code, routesOut, routesErr := runGoldr(t, "routes", "list", "--app-root", root)
	if code != 0 {
		t.Fatalf("Run(routes list) exit code = %d, want 0; stderr = %q", code, routesErr)
	}
	requireRouteTableRows(t, routesOut, [][]string{
		{"KIND", "METHOD", "PATH", "PARAMS", "SOURCE", "OWNER", "DECL", "NAME", "TITLE", "LABELS", "NAV", "TRAIL_KEYS", "HELPER"},
		{"layout", "-", "/", "-", "layout.go", "-", "-", "-", "-", "-", "-", "-", "-"},
		{"page", "GET,HEAD", "/", "-", "route.go", "-", "local", "-", "-", "-", "-", "-", "urls.Root.Path()"},
	})
	if routesErr != "" {
		t.Fatalf("routes stderr = %q, want empty", routesErr)
	}
}

func TestRunInitRequiresModule(t *testing.T) {
	root := t.TempDir()

	code, stdout, stderr := runGoldr(t, "init", "--app-root", root)

	if code != 1 {
		t.Fatalf("Run(init) exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "could not find go.mod") {
		t.Fatalf("stderr = %q, want missing go.mod error", stderr)
	}
	requireMissingFile(t, filepath.Join(root, "app"))
}

func TestRunInitRefusesExistingAppPath(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, root string)
	}{
		{
			name: "directory",
			setup: func(t *testing.T, root string) {
				t.Helper()
				writeFile(t, root, "app/keep.txt", "keep\n")
			},
		},
		{
			name: "file",
			setup: func(t *testing.T, root string) {
				t.Helper()
				writeFile(t, root, "app", "keep\n")
			},
		},
		{
			name: "symlink",
			setup: func(t *testing.T, root string) {
				t.Helper()
				writeFile(t, root, "target/keep.txt", "keep\n")
				if err := os.Symlink("target", filepath.Join(root, "app")); err != nil {
					t.Skipf("Symlink() error = %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, "go.mod", "module example.com/existingapp\n\ngo 1.26.3\n")
			tt.setup(t, root)

			code, stdout, stderr := runGoldr(t, "init", "--app-root", root)

			if code != 1 {
				t.Fatalf("Run(init) exit code = %d, want 1", code)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if !strings.Contains(stderr, "app") || !strings.Contains(stderr, "already exists") {
				t.Fatalf("stderr = %q, want existing app error", stderr)
			}
			requireMissingFile(t, filepath.Join(root, "app", "routes", "route.go"))
			requireMissingFile(t, filepath.Join(root, "app", "routes", "goldr_gen.go"))
			requireMissingFile(t, filepath.Join(root, "app", "urls", "goldr_gen.go"))
		})
	}
}
