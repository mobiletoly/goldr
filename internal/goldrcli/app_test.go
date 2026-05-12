package goldrcli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestRunHelp(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no args"},
		{name: "help", args: []string{"help"}},
		{name: "long help", args: []string{"--help"}},
		{name: "short help", args: []string{"-h"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runGoldr(t, tt.args...)

			if code != 0 {
				t.Fatalf("Run() exit code = %d, want 0", code)
			}
			if !strings.Contains(stdout, "USAGE:") {
				t.Fatalf("stdout = %q, want usage text", stdout)
			}
			if !strings.Contains(stdout, "init") {
				t.Fatalf("stdout = %q, want init command", stdout)
			}
			for _, futureCommand := range []string{"new", "dev", "build"} {
				if strings.Contains(stdout, futureCommand) {
					t.Fatalf("stdout = %q, must not mention future command %q", stdout, futureCommand)
				}
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
		})
	}
}

func TestRunInitHelp(t *testing.T) {
	code, stdout, stderr := runGoldr(t, "init", "--help")

	if code != 0 {
		t.Fatalf("Run(init --help) exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "goldr init [--root <dir>]") {
		t.Fatalf("stdout = %q, want init usage", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunVersion(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "version", args: []string{"goldr", "version"}},
		{name: "long version", args: []string{"goldr", "--version"}},
		{name: "dash version", args: []string{"goldr", "-version"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(context.Background(), tt.args, &stdout, &stderr, "dev")

			if code != 0 {
				t.Fatalf("Run() exit code = %d, want 0", code)
			}
			if got := stdout.String(); got != "goldr dev\n" {
				t.Fatalf("stdout = %q, want %q", got, "goldr dev\n")
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(context.Background(), []string{"goldr", "unknown"}, &stdout, &stderr, "dev")

	if code != 2 {
		t.Fatalf("Run() exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	errText := stderr.String()
	if !strings.Contains(errText, `goldr: unknown command "unknown"`) {
		t.Fatalf("stderr = %q, want unknown-command error", errText)
	}
	if !strings.Contains(errText, "USAGE:") {
		t.Fatalf("stderr = %q, want usage text", errText)
	}
}

func TestRunInitCreatesStarterApp(t *testing.T) {
	root := t.TempDir()
	goMod := "module example.com/initapp\n\ngo 1.26.3\n"
	writeFile(t, root, "go.mod", goMod)

	code, stdout, stderr := runGoldr(t, "init", "--root", root)

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
		"app/routes/page.go",
		"app/routes/page.templ",
		"app/routes/layout.go",
		"app/routes/layout.templ",
		"app/routes/goldr_gen.go",
		"app/urls/goldr_gen.go",
	} {
		requireExistingFile(t, filepath.Join(root, filepath.FromSlash(name)))
	}

	pageSource := readFile(t, filepath.Join(root, "app", "routes", "page.go"))
	if !strings.Contains(pageSource, `"github.com/mobiletoly/goldr"`) {
		t.Fatalf("page.go = %q, want goldr import", pageSource)
	}
	layoutTempl := readFile(t, filepath.Join(root, "app", "routes", "layout.templ"))
	if !strings.Contains(layoutTempl, `https://unpkg.com/htmx.org@2.0.4`) {
		t.Fatalf("layout.templ = %q, want HTMX script", layoutTempl)
	}

	files, err := generateFiles(context.Background(), root)
	if err != nil {
		t.Fatalf("generateFiles(init app) error = %v", err)
	}
	for _, file := range files {
		got := readFile(t, file.path)
		if !bytes.Equal([]byte(got), file.content) {
			t.Fatalf("%s is stale\n--- got ---\n%s\n--- want ---\n%s", file.path, got, file.content)
		}
	}

	requireRunSuccess(t, "check", "--root", root)

	code, routesOut, routesErr := runGoldr(t, "routes", "list", "--root", root)
	if code != 0 {
		t.Fatalf("Run(routes list) exit code = %d, want 0; stderr = %q", code, routesErr)
	}
	requireRouteTableRows(t, routesOut, [][]string{
		{"KIND", "METHOD", "PATH", "PARAMS", "SOURCE", "HELPER"},
		{"layout", "-", "/", "-", "layout.go", "-"},
		{"page", "GET,HEAD", "/", "-", "page.go", "urls.Root.Path()"},
	})
	if routesErr != "" {
		t.Fatalf("routes stderr = %q, want empty", routesErr)
	}
}

func TestRunInitRequiresModule(t *testing.T) {
	root := t.TempDir()

	code, stdout, stderr := runGoldr(t, "init", "--root", root)

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

			code, stdout, stderr := runGoldr(t, "init", "--root", root)

			if code != 1 {
				t.Fatalf("Run(init) exit code = %d, want 1", code)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if !strings.Contains(stderr, "app") || !strings.Contains(stderr, "already exists") {
				t.Fatalf("stderr = %q, want existing app error", stderr)
			}
			requireMissingFile(t, filepath.Join(root, "app", "routes", "page.go"))
			requireMissingFile(t, filepath.Join(root, "app", "routes", "goldr_gen.go"))
			requireMissingFile(t, filepath.Join(root, "app", "urls", "goldr_gen.go"))
		})
	}
}

func TestRunGenerateWritesGeneratedFiles(t *testing.T) {
	root := tempGenerateApp(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(context.Background(), []string{"goldr", "generate", "--root", root}, &stdout, &stderr, "dev")

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

	urlsFile := filepath.Join(root, "app", "urls", "goldr_gen.go")
	urlsSource := readFile(t, urlsFile)
	if !strings.Contains(urlsSource, `package urls`) {
		t.Fatalf("%s = %q, want package urls", urlsFile, urlsSource)
	}
	if !strings.Contains(urlsSource, `var Settings = newSettingsRoute()`) {
		t.Fatalf("%s = %q, want settings helper", urlsFile, urlsSource)
	}
}

func TestRunGenerateCheck(t *testing.T) {
	root := tempGenerateApp(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run(context.Background(), []string{"goldr", "generate", "--root", root}, &stdout, &stderr, "dev"); code != 0 {
		t.Fatalf("generate exit code = %d; stderr = %q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code := Run(context.Background(), []string{"goldr", "generate", "--root", root, "--check"}, &stdout, &stderr, "dev")

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

func TestRunGenerateCheckReportsStaleAndMissingFiles(t *testing.T) {
	root := tempGenerateApp(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run(context.Background(), []string{"goldr", "generate", "--root", root}, &stdout, &stderr, "dev"); code != 0 {
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
	code := Run(context.Background(), []string{"goldr", "generate", "--root", root, "--check"}, &stdout, &stderr, "dev")

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

	files, err := generateFiles(context.Background(), filepath.Join(repoRoot, "examples", "full_feature"))
	if err != nil {
		t.Fatalf("generateFiles(full_feature) error = %v", err)
	}

	var routesSource string
	for _, file := range files {
		if filepath.Base(filepath.Dir(file.path)) == "routes" {
			routesSource = string(file.content)
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

	files, err := generateFiles(context.Background(), filepath.Join(repoRoot, "examples", "full_feature"))
	if err != nil {
		t.Fatalf("generateFiles(full_feature) error = %v", err)
	}

	for _, file := range files {
		got, err := os.ReadFile(file.path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", file.path, err)
		}
		if !bytes.Equal(got, file.content) {
			t.Fatalf("%s is stale\n--- got ---\n%s\n--- want ---\n%s", file.path, got, file.content)
		}
	}
}

func TestRunGenerateRequiresModule(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "app/routes/page.go", "package routes\n")
	writeFile(t, root, "app/routes/page.templ", "package routes\n\ntempl PageView() {}\n")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(context.Background(), []string{"goldr", "generate", "--root", root}, &stdout, &stderr, "dev")

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

	code := Run(context.Background(), []string{"goldr", "generate", "--root", root}, &stdout, &stderr, "dev")

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

func TestRunCheckCleanApp(t *testing.T) {
	root := tempGenerateApp(t)

	requireRunSuccess(t, "generate", "--root", root)
	requireRunSuccess(t, "check", "--root", root)
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

	requireRunSuccess(t, "generate", "--root", root)
	if err := os.WriteFile(filepath.Join(root, "app", "routes", "goldr_gen.go"), []byte("stale"), 0644); err != nil {
		t.Fatalf("WriteFile(stale routes) error = %v", err)
	}
	if err := os.Remove(filepath.Join(root, "app", "urls", "goldr_gen.go")); err != nil {
		t.Fatalf("Remove(urls) error = %v", err)
	}

	requireCheckFailureContains(t, root, "goldr check:", checkCodeGeneratedFiles, "app/routes/goldr_gen.go", "is stale", "app/urls/goldr_gen.go", "is missing")
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
	writeFile(t, root, "app/routes/page.go", "package routes\n")
	writeFile(t, root, "app/routes/layout.go", "package routes\n")
	writeFile(t, root, "app/routes/frag_row.go", "package routes\n")

	requireCheckFailureContains(t, root, checkCodeRenderUnit, "app/routes/page.go", "page /", "app/routes/layout.go", "layout /", "app/routes/frag_row.go", "fragment /:row", "missing matching .templ file")
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
	writeFile(t, root, "app/routes/users/by_id/page.go", "package by_id\n")
	writeFile(t, root, "app/routes/users/by_id/page.templ", "package by_id\n\ntempl PageView() {}\n")
	writeFile(t, root, "app/routes/users/by_slug/page.go", "package by_slug\n")
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

func TestRunRoutesPrintsRouteTable(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "app/routes/layout.go", "package routes\n")
	writeFile(t, root, "app/routes/page.go", "package routes\n")
	writeFile(t, root, "app/routes/users/page.go", "package users\n")
	writeFile(t, root, "app/routes/users/frag_table.go", "package users\n")
	writeFile(t, root, "app/routes/users/by_id/page.go", "package by_id\n")
	writeFile(t, root, "app/routes/users/actions.go", `package users

import "net/http"

func PostCreate(w http.ResponseWriter, r *http.Request) {}
`)

	code, stdout, stderr := runGoldr(t, "routes", "list", "--root", root)

	if code != 0 {
		t.Fatalf("Run(routes list) exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	want := [][]string{
		{"KIND", "METHOD", "PATH", "PARAMS", "SOURCE", "HELPER"},
		{"layout", "-", "/", "-", "layout.go", "-"},
		{"page", "GET,HEAD", "/", "-", "page.go", "urls.Root.Path()"},
		{"action", "POST", "/users/create", "-", "users/actions.go:PostCreate", "urls.Users.Create.Path()"},
		{"fragment", "GET,HEAD", "/users/frag_table", "-", "users/frag_table.go", "urls.Users.FragTable.Path()"},
		{"page", "GET,HEAD", "/users", "-", "users/page.go", "urls.Users.Path()"},
		{"page", "GET,HEAD", "/users/{id}", "id", "users/by_id/page.go", "urls.Users.ByID(id).Path()"},
	}
	requireRouteTableRows(t, stdout, want)
	requireMissingFile(t, filepath.Join(root, "app", "routes", "goldr_gen.go"))
	requireMissingFile(t, filepath.Join(root, "app", "urls", "goldr_gen.go"))
}

func TestRunRoutesPrintsJSON(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "app/routes/layout.go", "package routes\n")
	writeFile(t, root, "app/routes/page.go", "package routes\n")
	writeFile(t, root, "app/routes/users/page.go", "package users\n")
	writeFile(t, root, "app/routes/users/frag_table.go", "package users\n")
	writeFile(t, root, "app/routes/users/by_id/page.go", "package by_id\n")
	writeFile(t, root, "app/routes/users/actions.go", `package users

import "net/http"

func PostCreate(w http.ResponseWriter, r *http.Request) {}
`)

	code, stdout, stderr := runGoldr(t, "routes", "list", "--root", root, "--json")

	if code != 0 {
		t.Fatalf("Run(routes list --json) exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if strings.Contains(stdout, "null") {
		t.Fatalf("stdout = %q, must not contain null arrays", stdout)
	}

	var rows []routeSurfaceJSONRow
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil {
		t.Fatalf("Unmarshal(routes list --json) error = %v; stdout = %q", err, stdout)
	}
	want := []routeSurfaceJSONRow{
		{Kind: "layout", Methods: []string{}, Path: "/", Params: []string{}, Source: "layout.go"},
		{Kind: "page", Methods: []string{"GET", "HEAD"}, Path: "/", Params: []string{}, Source: "page.go", Helper: "urls.Root.Path()"},
		{Kind: "action", Methods: []string{"POST"}, Path: "/users/create", Params: []string{}, Source: "users/actions.go:PostCreate", Helper: "urls.Users.Create.Path()"},
		{Kind: "fragment", Methods: []string{"GET", "HEAD"}, Path: "/users/frag_table", Params: []string{}, Source: "users/frag_table.go", Helper: "urls.Users.FragTable.Path()"},
		{Kind: "page", Methods: []string{"GET", "HEAD"}, Path: "/users", Params: []string{}, Source: "users/page.go", Helper: "urls.Users.Path()"},
		{Kind: "page", Methods: []string{"GET", "HEAD"}, Path: "/users/{id}", Params: []string{"id"}, Source: "users/by_id/page.go", Helper: "urls.Users.ByID(id).Path()"},
	}
	if len(rows) != len(want) {
		t.Fatalf("JSON rows = %#v, want %#v", rows, want)
	}
	for index := range want {
		if strings.Join(rows[index].Methods, "\x00") != strings.Join(want[index].Methods, "\x00") ||
			strings.Join(rows[index].Params, "\x00") != strings.Join(want[index].Params, "\x00") ||
			rows[index].Kind != want[index].Kind ||
			rows[index].Path != want[index].Path ||
			rows[index].Source != want[index].Source ||
			rows[index].Helper != want[index].Helper {
			t.Fatalf("JSON row %d = %#v, want %#v", index, rows[index], want[index])
		}
		if rows[index].Methods == nil {
			t.Fatalf("JSON row %d methods = nil, want empty array when empty", index)
		}
		if rows[index].Params == nil {
			t.Fatalf("JSON row %d params = nil, want empty array when empty", index)
		}
	}
	requireMissingFile(t, filepath.Join(root, "app", "routes", "goldr_gen.go"))
	requireMissingFile(t, filepath.Join(root, "app", "urls", "goldr_gen.go"))
}

func TestRunRoutesFullFeatureOutputIsDeterministic(t *testing.T) {
	root := fullFeatureRoot(t)
	stdout := runGoldrDeterministic(t, "routes list", "routes", "list", "--root", root)

	rows := routeTableRows(t, stdout)
	for _, want := range [][]string{
		{"layout", "-", "/", "-", "layout.go", "-"},
		{"page", "GET,HEAD", "/", "-", "page.go", "urls.Root.Path()"},
		{"page", "GET,HEAD", "/settings", "-", "settings/page.go", "urls.Settings.Path()"},
		{"layout", "-", "/users", "-", "users/layout.go", "-"},
		{"page", "GET,HEAD", "/users/{id}", "id", "users/by_id/page.go", "urls.Users.ByID(id).Path()"},
		{"action", "POST", "/users/create", "-", "users/actions.go:PostCreate", "urls.Users.Create.Path()"},
		{"action", "POST", "/users/save-preview", "-", "users/actions.go:PostSavePreview", "urls.Users.SavePreview.Path()"},
	} {
		requireRouteTableContainsRow(t, rows, want)
	}
}

func TestRunRoutesFullFeatureJSONOutputIsDeterministic(t *testing.T) {
	root := fullFeatureRoot(t)
	stdout := runGoldrDeterministic(t, "routes list --json", "routes", "list", "--root", root, "--json")

	var rows []routeSurfaceJSONRow
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil {
		t.Fatalf("Unmarshal(routes list --json) error = %v; stdout = %q", err, stdout)
	}
	requireRouteJSONContainsRow(t, rows, routeSurfaceJSONRow{
		Kind:    "layout",
		Methods: []string{},
		Path:    "/",
		Params:  []string{},
		Source:  "layout.go",
	})
	requireRouteJSONContainsRow(t, rows, routeSurfaceJSONRow{
		Kind:    "page",
		Methods: []string{"GET", "HEAD"},
		Path:    "/users/{id}",
		Params:  []string{"id"},
		Source:  "users/by_id/page.go",
		Helper:  "urls.Users.ByID(id).Path()",
	})
	requireRouteJSONContainsRow(t, rows, routeSurfaceJSONRow{
		Kind:    "action",
		Methods: []string{"POST"},
		Path:    "/users/save-preview",
		Params:  []string{},
		Source:  "users/actions.go:PostSavePreview",
		Helper:  "urls.Users.SavePreview.Path()",
	})
}

func TestRunRoutesFullFeatureLayoutMapOutputIsDeterministic(t *testing.T) {
	root := fullFeatureRoot(t)
	stdout := runGoldrDeterministic(t, "routes layouts", "routes", "layouts", "--root", root)

	want := fullFeatureLayoutMapOutput(t)
	if stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
}

func TestRunRoutesExplainFullURLDynamicPage(t *testing.T) {
	root := fullFeatureRoot(t)
	source := func(name string) string {
		return fullFeatureRouteSourcePath(name)
	}

	code, stdout, stderr := runGoldr(t, "routes", "explain", "--root", root, "http://127.0.0.1:8080/users/a%2Fb?tab=profile#details")

	if code != 0 {
		t.Fatalf("Run(routes explain) exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	want := strings.Join([]string{
		"/users/a%2Fb  GET",
		"",
		"MATCH",
		"  page     /users/{id}",
		"  source   " + source("users/by_id/page.go"),
		"  params   id = \"a/b\"",
		"",
		"LAYOUT STACK",
		"  /      " + source("layout.go"),
		"  /users " + source("users/layout.go"),
	}, "\n") + "\n"
	if stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
}

func TestRunRoutesExplainHonorsRootFlag(t *testing.T) {
	root := fullFeatureRoot(t)

	code, stdout, stderr := runGoldr(t, "routes", "explain", "--root", root, "/users/7")

	if code != 0 {
		t.Fatalf("Run(routes explain --root) exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "  page     /users/{id}") {
		t.Fatalf("stdout = %q, want users by_id route", stdout)
	}
}

func TestRunRoutesExplainActionShowsNoLayout(t *testing.T) {
	root := fullFeatureRoot(t)
	source := fullFeatureRouteSourcePath("users/actions.go")

	code, stdout, stderr := runGoldr(t, "routes", "explain", "--root", root, "--method", "POST", "/users/create")

	if code != 0 {
		t.Fatalf("Run(routes explain action) exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"/users/create  POST",
		"  action   /users/create",
		"  source   " + source + " (PostCreate)",
		"LAYOUT STACK\n  not layout-wrapped",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout = %q, want %q", stdout, want)
		}
	}
}

func TestRunRoutesExplainReportsFailures(t *testing.T) {
	root := fullFeatureRoot(t)

	tests := []struct {
		name  string
		args  []string
		wants []string
	}{
		{
			name:  "method mismatch",
			args:  []string{"routes", "explain", "--root", root, "--method", "DELETE", "/users/7"},
			wants: []string{"goldr routes explain:", "DELETE /users/7", "method not allowed", "allowed: GET,HEAD"},
		},
		{
			name:  "no match",
			args:  []string{"routes", "explain", "--root", root, "/missing"},
			wants: []string{"goldr routes explain:", "GET /missing", "no route matches path"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runGoldr(t, tt.args...)

			if code != 1 {
				t.Fatalf("Run(%s) exit code = %d, want 1", tt.name, code)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			for _, want := range tt.wants {
				if !strings.Contains(stderr, want) {
					t.Fatalf("stderr = %q, want %q", stderr, want)
				}
			}
		})
	}
}

func fullFeatureRoot(t *testing.T) string {
	t.Helper()

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Abs(repo root) error = %v", err)
	}
	return filepath.Join(repoRoot, "examples", "full_feature")
}

func runGoldrDeterministic(t *testing.T, label string, args ...string) string {
	t.Helper()

	stdout := requireGoldrSuccessOutput(t, label, args...)
	stdout2 := requireGoldrSuccessOutput(t, "second "+label, args...)
	if stdout != stdout2 {
		t.Fatalf("second stdout differs for %s\n--- first ---\n%s\n--- second ---\n%s", label, stdout, stdout2)
	}
	return stdout
}

func TestRunRoutesShowsSubcommandHelp(t *testing.T) {
	requireGoldrOutputContains(t, []string{"routes"}, "goldr routes <command> [options]", "list", "layouts", "explain")
}

func TestRunRoutesListHelpShowsRootFlag(t *testing.T) {
	requireGoldrOutputContains(t, []string{"routes", "list", "--help"}, "goldr routes list [--root <dir>] [--json]", "--root string", "--json")
}

func TestRunAssetsShowsSubcommandHelp(t *testing.T) {
	requireGoldrOutputContains(t, []string{"assets"}, "goldr assets <command> [options]", "dist", "check", "clean", "list")
}

func requireGoldrOutputContains(t *testing.T, args []string, wants ...string) {
	t.Helper()

	stdout := requireGoldrSuccessOutput(t, strings.Join(args, " "), args...)
	for _, want := range wants {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout = %q, want %q", stdout, want)
		}
	}
}

func TestRunRoutesOldModeFlagsFail(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "json", args: []string{"routes", "--json"}, want: "json"},
		{name: "layouts", args: []string{"routes", "--layouts"}, want: "layouts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runGoldr(t, tt.args...)
			if code == 0 {
				t.Fatalf("Run(%s) exit code = 0, want failure", strings.Join(tt.args, " "))
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if !strings.Contains(stderr, tt.want) {
				t.Fatalf("stderr = %q, want %q", stderr, tt.want)
			}
		})
	}
}

func TestRunRoutesReportsInvalidRouteNames(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "app/routes/Users/page.go", "package Users\n")

	requireCommandArgsFailureContains(t, []string{"routes", "list", "--root", root}, "goldr routes list:", "app/routes/Users", "static route directories must use lowercase Go-safe names")
}

func TestRunRoutesReportsURLHelperGenerationErrors(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "app/routes/users/actions.go", `package users

import "net/http"

func PostPath(w http.ResponseWriter, r *http.Request) {}
`)

	requireCommandArgsFailureContains(t, []string{"routes", "list", "--root", root}, "goldr routes list:", "ambiguous URL helper", "Path method")
}

func TestRunRoutesJSONReportsErrorsToStderr(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "app/routes/Users/page.go", "package Users\n")

	code, stdout, stderr := runGoldr(t, "routes", "list", "--root", root, "--json")
	if code != 1 {
		t.Fatalf("Run(routes list --json) exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	for _, want := range []string{"goldr routes list:", "app/routes/Users", "static route directories must use lowercase Go-safe names"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr = %q, want %q", stderr, want)
		}
	}
}

func requireRunSuccess(t *testing.T, args ...string) {
	t.Helper()

	stdout := requireGoldrSuccessOutput(t, strings.Join(args, " "), args...)
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
}

func requireGoldrSuccessOutput(t *testing.T, label string, args ...string) string {
	t.Helper()

	code, stdout, stderr := runGoldr(t, args...)
	if code != 0 {
		t.Fatalf("Run(%s) exit code = %d, want 0; stderr = %q", label, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	return stdout
}

func requireRouteTableRows(t *testing.T, stdout string, want [][]string) {
	t.Helper()

	got := routeTableRows(t, stdout)
	if len(got) != len(want) {
		t.Fatalf("route table rows = %#v, want %#v", got, want)
	}
	for index := range want {
		if strings.Join(got[index], "\x00") != strings.Join(want[index], "\x00") {
			t.Fatalf("route table row %d = %#v, want %#v", index, got[index], want[index])
		}
	}
}

func routeTableRows(t *testing.T, stdout string) [][]string {
	t.Helper()

	trimmed := strings.TrimSuffix(stdout, "\n")
	if trimmed == "" {
		t.Fatalf("stdout is empty, want route table")
	}
	lines := strings.Split(trimmed, "\n")
	rows := make([][]string, 0, len(lines))
	for _, line := range lines {
		rows = append(rows, strings.Fields(line))
	}
	return rows
}

func requireRouteTableContainsRow(t *testing.T, rows [][]string, want []string) {
	t.Helper()

	wantText := strings.Join(want, "\x00")
	for _, row := range rows {
		if strings.Join(row, "\x00") == wantText {
			return
		}
	}
	t.Fatalf("route table rows = %#v, want row %#v", rows, want)
}

type routeSurfaceJSONRow struct {
	Kind    string   `json:"kind"`
	Methods []string `json:"methods"`
	Path    string   `json:"path"`
	Params  []string `json:"params"`
	Source  string   `json:"source"`
	Helper  string   `json:"helper"`
}

func requireRouteJSONContainsRow(t *testing.T, rows []routeSurfaceJSONRow, want routeSurfaceJSONRow) {
	t.Helper()

	for _, row := range rows {
		if row.Kind == want.Kind &&
			strings.Join(row.Methods, "\x00") == strings.Join(want.Methods, "\x00") &&
			row.Path == want.Path &&
			strings.Join(row.Params, "\x00") == strings.Join(want.Params, "\x00") &&
			row.Source == want.Source &&
			row.Helper == want.Helper {
			return
		}
	}
	t.Fatalf("route JSON rows = %#v, want row %#v", rows, want)
}

func requireCheckFailureContains(t *testing.T, root string, wants ...string) {
	t.Helper()

	requireCommandArgsFailureContains(t, []string{"check", "--root", root}, wants...)
}

func requireCommandArgsFailureContains(t *testing.T, args []string, wants ...string) {
	t.Helper()

	code, stdout, stderr := runGoldr(t, args...)

	if code != 1 {
		t.Fatalf("Run(%s) exit code = %d, want 1", strings.Join(args, " "), code)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	for _, want := range wants {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr = %q, want %q", stderr, want)
		}
	}
}

func fullFeatureLayoutMapOutput(t *testing.T) string {
	t.Helper()

	source := func(name string) string {
		return fullFeatureRouteSourcePath(name)
	}
	rootPath := fullFeatureRoutesDisplayRoot()
	if !strings.HasSuffix(rootPath, "/") {
		rootPath += "/"
	}
	lines := []string{
		"Layout map",
		"",
		rootPath,
		"└─ /  layout: " + source("layout.go"),
		"   ├─ page: GET,HEAD /  " + source("page.go"),
		"   ├─ settings/",
		"   │  └─ page: GET,HEAD /settings  " + source("settings/page.go"),
		"   └─ users/  layout: " + source("users/layout.go"),
		"      ├─ page: GET,HEAD /users  " + source("users/page.go"),
		"      ├─ by_id/",
		"      │  └─ page: GET,HEAD /users/{id}  params: id  " + source("users/by_id/page.go"),
		"      ├─ fragment (not wrapped): GET,HEAD /users/frag_table  " + source("users/frag_table.go"),
		"      ├─ action (not wrapped): POST /users/create  " + source("users/actions.go") + " (PostCreate)",
		"      └─ action (not wrapped): POST /users/save-preview  " + source("users/actions.go") + " (PostSavePreview)",
		"",
		"Rule:",
		"  pages inherit every layout above them",
		"  fragments and actions do not inherit layouts",
	}
	return strings.Join(lines, "\n") + "\n"
}

func fullFeatureRouteSourcePath(source string) string {
	return filepath.ToSlash(filepath.Join("..", "..", "examples", "full_feature", "app", "routes", filepath.FromSlash(source)))
}

func fullFeatureRoutesDisplayRoot() string {
	return filepath.ToSlash(filepath.Join("..", "..", "examples", "full_feature", "app", "routes"))
}

func runGoldr(t *testing.T, args ...string) (int, string, string) {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	allArgs := append([]string{"goldr"}, args...)
	code := Run(context.Background(), allArgs, &stdout, &stderr, "dev")
	return code, stdout.String(), stderr.String()
}

func tempGenerateApp(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/generateapp\n\ngo 1.26.3\n")
	writeFile(t, root, "app/routes/page.go", `package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(_ *http.Request) goldr.Page { return goldr.Page{Component: PageView()} }
`)
	writeFile(t, root, "app/routes/page.templ", "package routes\n\ntempl PageView() {<h1>Root</h1>}\n")
	writeFile(t, root, "app/routes/settings/page.go", `package settings

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(_ *http.Request) goldr.Page { return goldr.Page{Component: PageView()} }
`)
	writeFile(t, root, "app/routes/settings/page.templ", "package settings\n\ntempl PageView() {<h1>Settings</h1>}\n")
	return root
}

func writeFile(t *testing.T, root string, name string, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(content)
}

func requireMissingFile(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); !os.IsNotExist(err) && !errors.Is(err, syscall.ENOTDIR) {
		t.Fatalf("Stat(%q) error = %v, want missing file", path, err)
	}
}

func requireExistingFile(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v, want file", path, err)
	}
	if info.IsDir() {
		t.Fatalf("Stat(%q) is directory, want file", path)
	}
}
