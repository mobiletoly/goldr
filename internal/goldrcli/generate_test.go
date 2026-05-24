package goldrcli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr/internal/goldrcli/project"
	"github.com/mobiletoly/goldr/internal/goldrcli/templtool"
)

func TestRunGenerateWritesGeneratedFiles(t *testing.T) {
	root := tempGenerateApp(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(context.Background(), []string{"goldr", "generate", "--app-root", root}, &stdout, &stderr, "dev")

	if code != 0 {
		t.Fatalf("Run() exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	routesFile := filepath.Join(root, "app", "routes", "goldr_gen.go")
	routesSource := readFile(t, routesFile)
	if !strings.Contains(routesSource, `package routes`) {
		t.Fatalf("%s = %q, want package routes", routesFile, routesSource)
	}
	if !strings.Contains(routesSource, `"example.com/generateapp/app/routes/settings"`) {
		t.Fatalf("%s = %q, want nested route import", routesFile, routesSource)
	}
	requireExistingFile(t, filepath.Join(root, "app", "routes", "page_templ.go"))
	requireExistingFile(t, filepath.Join(root, "app", "routes", "settings", "page_templ.go"))

	urlsFile := filepath.Join(root, "app", "urls", "goldr_gen.go")
	urlsSource := readFile(t, urlsFile)
	if !strings.Contains(urlsSource, `package urls`) {
		t.Fatalf("%s = %q, want package urls", urlsFile, urlsSource)
	}
	if !strings.Contains(urlsSource, `var Settings = newSettingsRoute("")`) {
		t.Fatalf("%s = %q, want settings helper", urlsFile, urlsSource)
	}
	if !strings.Contains(urlsSource, `func WithBasePath(basePath string) MountedRoutes`) {
		t.Fatalf("%s = %q, want mounted helper", urlsFile, urlsSource)
	}
}

func TestRunGenerateWritesAssetsWhenBuildExists(t *testing.T) {
	root := tempGenerateApp(t)
	writeFile(t, root, "assets/build/app.css", "body {}\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"goldr", "generate", "--app-root", root}, &stdout, &stderr, "dev")

	if code != 0 {
		t.Fatalf("Run(generate) exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	generated := readFile(t, filepath.Join(root, "assets", "goldr_assets_gen.go"))
	for _, want := range []string{
		"package assets",
		`Path: "/assets/app.`,
		"func FS() fs.FS",
	} {
		if !strings.Contains(generated, want) {
			t.Fatalf("assets generated source = %q, want %q", generated, want)
		}
	}
	matches, err := filepath.Glob(filepath.Join(root, "assets", "dist", "app.*.css"))
	if err != nil {
		t.Fatalf("Glob(dist app css) error = %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("dist matches = %#v, want one app css file", matches)
	}
	if got := readFile(t, matches[0]); got != "body {}\n" {
		t.Fatalf("%s = %q, want app css content", matches[0], got)
	}
}

func TestRunGenerateCheckReportsStaleAssets(t *testing.T) {
	root := tempGenerateApp(t)
	writeFile(t, root, "assets/build/app.css", "body {}\n")
	requireRunSuccess(t, "generate", "--app-root", root)
	writeFile(t, root, "assets/build/app.css", "body { color: black; }\n")

	requireCommandArgsFailureContains(
		t,
		[]string{"generate", "--app-root", root, "--check"},
		"goldr generate:",
		"goldr-managed assets are not current",
		"go tool goldr generate",
		"assets/dist/app.",
		"is missing",
	)
}

func TestRunGenerateReportsMissingTemplTool(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/notempltool\n\ngo 1.26.3\n")
	writeFile(t, root, "app/routes/page.go", "package routes\n")
	writeFile(t, root, "app/routes/page.templ", "package routes\n\ntempl PageView() {}\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"goldr", "generate", "--app-root", root}, &stdout, &stderr, "dev")

	if code != 1 {
		t.Fatalf("Run(generate) exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	errText := stderr.String()
	for _, want := range []string{"goldr generate:", "go tool templ is not available", templtool.InstallCommand} {
		if !strings.Contains(errText, want) {
			t.Fatalf("stderr = %q, want %q", errText, want)
		}
	}
}

func TestRunGenerateCheck(t *testing.T) {
	root := tempGenerateApp(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run(context.Background(), []string{"goldr", "generate", "--app-root", root}, &stdout, &stderr, "dev"); code != 0 {
		t.Fatalf("generate exit code = %d; stderr = %q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code := Run(context.Background(), []string{"goldr", "generate", "--app-root", root, "--check"}, &stdout, &stderr, "dev")

	if code != 0 {
		t.Fatalf("Run(--check) exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunGenerateCheckReportsStaleTemplGeneratedFiles(t *testing.T) {
	root := tempGenerateApp(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run(context.Background(), []string{"goldr", "generate", "--app-root", root}, &stdout, &stderr, "dev"); code != 0 {
		t.Fatalf("generate exit code = %d; stderr = %q", code, stderr.String())
	}
	writeFile(t, root, "app/routes/page.templ", "package routes\n\ntempl PageView() {<h1>Changed</h1>}\n")

	stdout.Reset()
	stderr.Reset()
	code := Run(context.Background(), []string{"goldr", "generate", "--app-root", root, "--check"}, &stdout, &stderr, "dev")

	if code != 1 {
		t.Fatalf("Run(--check) exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	errText := stderr.String()
	for _, want := range []string{"goldr generate:", "templ generated files are not up to date", "go tool goldr generate", "generated files are not up to date"} {
		if !strings.Contains(errText, want) {
			t.Fatalf("stderr = %q, want %q", errText, want)
		}
	}
}

func TestRunGenerateCheckReportsStaleAndMissingFiles(t *testing.T) {
	root := tempGenerateApp(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run(context.Background(), []string{"goldr", "generate", "--app-root", root}, &stdout, &stderr, "dev"); code != 0 {
		t.Fatalf("generate exit code = %d; stderr = %q", code, stderr.String())
	}

	if err := os.WriteFile(filepath.Join(root, "app", "routes", "goldr_gen.go"), []byte("stale"), 0644); err != nil {
		t.Fatalf("WriteFile(stale routes) error = %v", err)
	}
	if err := os.Remove(filepath.Join(root, "app", "urls", "goldr_gen.go")); err != nil {
		t.Fatalf("Remove(urls) error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	code := Run(context.Background(), []string{"goldr", "generate", "--app-root", root, "--check"}, &stdout, &stderr, "dev")

	if code != 1 {
		t.Fatalf("Run(--check) exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	errText := stderr.String()
	for _, want := range []string{"goldr generate:", "app/routes/goldr_gen.go", "is stale", "app/urls/goldr_gen.go", "is missing"} {
		if !strings.Contains(errText, want) {
			t.Fatalf("stderr = %q, want %q", errText, want)
		}
	}
	if strings.Contains(errText, "GOLDR") {
		t.Fatalf("stderr = %q, must not contain goldr check diagnostic codes", errText)
	}
}

func TestGenerateFilesDerivesFullFeatureImportPath(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Abs(repo root) error = %v", err)
	}

	files, err := project.GenerateFiles(context.Background(), filepath.Join(repoRoot, "examples", "full_feature"))
	if err != nil {
		t.Fatalf("generateFiles(full_feature) error = %v", err)
	}

	var routesSource string
	for _, file := range files {
		if filepath.Base(filepath.Dir(file.Path)) == "routes" {
			routesSource = string(file.Content)
		}
	}
	if !strings.Contains(routesSource, `"github.com/mobiletoly/goldr/examples/full_feature/app/routes/settings"`) {
		t.Fatalf("routes source = %q, want full-feature settings import", routesSource)
	}
}

func TestGenerateFilesFullFeatureGeneratedFilesAreCurrent(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Abs(repo root) error = %v", err)
	}

	files, err := project.GenerateFiles(context.Background(), filepath.Join(repoRoot, "examples", "full_feature"))
	if err != nil {
		t.Fatalf("generateFiles(full_feature) error = %v", err)
	}

	for _, file := range files {
		got, err := os.ReadFile(file.Path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", file.Path, err)
		}
		if !bytes.Equal(got, file.Content) {
			t.Fatalf("%s is stale\n--- got ---\n%s\n--- want ---\n%s", file.Path, got, file.Content)
		}
	}
}

func TestRunGenerateRequiresModule(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "app/routes/page.go", "package routes\n")
	writeFile(t, root, "app/routes/page.templ", "package routes\n\ntempl PageView() {}\n")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(context.Background(), []string{"goldr", "generate", "--app-root", root}, &stdout, &stderr, "dev")

	if code != 1 {
		t.Fatalf("Run() exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "could not find go.mod") {
		t.Fatalf("stderr = %q, want missing go.mod error", stderr.String())
	}
}

func TestRunGenerateRequiresRoutesDirectory(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/missingroutes\n\ngo 1.26.3\n")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(context.Background(), []string{"goldr", "generate", "--app-root", root}, &stdout, &stderr, "dev")

	if code != 1 {
		t.Fatalf("Run() exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "app/routes") {
		t.Fatalf("stderr = %q, want app/routes error", stderr.String())
	}
}
