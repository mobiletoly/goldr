package goldrcli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

const (
	checkCodeAppRoot        = "GOLDR001"
	checkCodeRouteScan      = "GOLDR002"
	checkCodeRenderUnit     = "GOLDR003"
	checkCodeRouteGenerate  = "GOLDR004"
	checkCodeURLGenerate    = "GOLDR005"
	checkCodeGeneratedFiles = "GOLDR006"
	checkCodeTemplGenerated = "GOLDR007"
	checkCodeAssets         = "GOLDR008"
)

func fullFeatureRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("Abs(repo root) error = %v", err)
	}
	return filepath.Join(root, "examples", "full_feature")
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

func requireCheckFailureContains(t *testing.T, root string, wants ...string) {
	t.Helper()

	requireCommandArgsFailureContains(t, []string{"check", "--app-root", root}, wants...)
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
	writeTemplToolModule(t, root, "example.com/generateapp")
	writeFile(t, root, "app/routes/route.go", routeDeclarationSource("routes", "page", routeDeclarationOptions{Page: true}))
	writeFile(t, root, "app/routes/page.templ", "package routes\n\ntempl PageView() {<h1>Root</h1>}\n")
	writeFile(t, root, "app/routes/settings/route.go", routeDeclarationSource("settings", "page", routeDeclarationOptions{Page: true}))
	writeFile(t, root, "app/routes/settings/page.templ", "package settings\n\ntempl PageView() {<h1>Settings</h1>}\n")
	return root
}

func tempMountedRouteApp(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeTemplToolModule(t, root, "example.com/mountedapp")
	writeFile(t, root, "app/routes/admin/reports/route.go", `package reports

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	sharedreports "example.com/mountedapp/app/mounts/reports"
)

var Route = goldr.KitRouteMount[sharedreports.Kit]{
	New: newKit,
	Mount: "reports",
}

func newKit(_ *http.Request) sharedreports.Kit {
	return sharedreports.Kit{}
}
`)
	writeFile(t, root, "app/mounts/reports/route.go", `package reports

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

type Kit struct{}

var Route = goldr.KitRouteDef[Kit]{
	Page: Kit.Page,
	Fragments: goldr.KitFragments[Kit]{
		goldr.KitFragmentRoute("/table", Kit.Table),
	},
}

func (kit Kit) Page(_ *http.Request) goldr.PageRouteResponse {
	return goldr.Text{Body: "ok"}
}

func (kit Kit) Table(_ *http.Request) goldr.FragmentRouteResponse {
	return goldr.Text{Body: "ok"}
}
`)
	return root
}

func tempSelectiveMountedRouteApp(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeTemplToolModule(t, root, "example.com/selectivemountedapp")
	writeFile(t, root, "app/routes/admin/reports/route.go", `package reports

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	sharedreports "example.com/selectivemountedapp/app/mounts/reports"
)

var Route = goldr.KitRouteMount[sharedreports.Kit]{
	New: newKit,
	Mount: "reports",
	Routes: goldr.MountRoutes{
		"/",
	},
}

func newKit(_ *http.Request) sharedreports.Kit {
	return sharedreports.Kit{}
}
`)
	writeFile(t, root, "app/mounts/reports/route.go", `package reports

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

type Kit struct{}

var Route = goldr.KitRouteDef[Kit]{
	Page: Kit.Page,
}

func (kit Kit) Page(_ *http.Request) goldr.PageRouteResponse {
	return goldr.Text{Body: "ok"}
}
`)
	writeFile(t, root, "app/mounts/reports/audit/route.go", `package audit

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	sharedreports "example.com/selectivemountedapp/app/mounts/reports"
)

var Route = goldr.KitRouteDef[sharedreports.Kit]{
	Page: sharedreports.Kit.Audit,
}

func (kit sharedreports.Kit) Audit(_ *http.Request) goldr.PageRouteResponse {
	return goldr.Text{Body: "ok"}
}
`)
	return root
}

type routeDeclarationOptions struct {
	Page          bool
	Fragments     []string
	IndexFragment bool
	Actions       []routeDeclarationAction
	Name          string
	Title         string
	Labels        []routeDeclarationLabel
}

type routeDeclarationLabel struct {
	Key   string
	Value string
}

type routeDeclarationAction struct {
	Helper string
	Name   string
	Func   string
}

func routeDeclarationSource(packageName, pageFunc string, options routeDeclarationOptions) string {
	return routeDeclarationSourceWithHandlers(packageName, pageFunc, options, true)
}

func routeDeclarationSourceWithoutHandlers(packageName, pageFunc string, options routeDeclarationOptions) string {
	return routeDeclarationSourceWithHandlers(packageName, pageFunc, options, false)
}

func routeDeclarationSourceWithHandlers(packageName, pageFunc string, options routeDeclarationOptions, defineHandlers bool) string {
	var builder strings.Builder
	builder.WriteString("package ")
	builder.WriteString(packageName)
	builder.WriteString(`

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

var Route = goldr.RouteDef{
`)
	if options.Name != "" {
		builder.WriteString("\tName: \"")
		builder.WriteString(options.Name)
		builder.WriteString("\",\n")
	}
	if options.Title != "" {
		builder.WriteString("\tTitle: \"")
		builder.WriteString(options.Title)
		builder.WriteString("\",\n")
	}
	if options.Page {
		builder.WriteString("\tPage: ")
		builder.WriteString(pageFunc)
		builder.WriteString(",\n")
	}
	if len(options.Fragments) > 0 || options.IndexFragment {
		builder.WriteString("\tFragments: goldr.Fragments{\n")
		if options.IndexFragment {
			builder.WriteString("\t\tgoldr.FragmentRoute(\"/\", fragIndex),\n")
		}
		for _, fragment := range options.Fragments {
			builder.WriteString("\t\tgoldr.FragmentRoute(\"/")
			builder.WriteString(fragment)
			builder.WriteString("\", frag")
			builder.WriteString(exportName(fragment))
			builder.WriteString("),\n")
		}
		builder.WriteString("\t},\n")
	}
	if len(options.Actions) > 0 {
		builder.WriteString("\tActions: goldr.Actions{\n")
		for _, action := range options.Actions {
			builder.WriteString("\t\tgoldr.")
			builder.WriteString(action.Helper)
			builder.WriteString("(http.MethodPost, \"")
			if action.Name == "" {
				builder.WriteString("/")
			} else {
				builder.WriteString("/")
				builder.WriteString(action.Name)
			}
			builder.WriteString("\", ")
			builder.WriteString(action.Func)
			builder.WriteString("),\n")
		}
		builder.WriteString("\t},\n")
	}
	if len(options.Labels) > 0 {
		builder.WriteString("\tMeta: goldr.RouteMeta{\n")
		builder.WriteString("\t\tLabels: map[string]string{\n")
		for _, label := range options.Labels {
			builder.WriteString("\t\t\t\"")
			builder.WriteString(label.Key)
			builder.WriteString("\": \"")
			builder.WriteString(label.Value)
			builder.WriteString("\",\n")
		}
		builder.WriteString("\t\t},\n")
		builder.WriteString("\t},\n")
	}
	builder.WriteString("}\n")
	if defineHandlers && options.Page {
		builder.WriteString("\nfunc ")
		builder.WriteString(pageFunc)
		builder.WriteString("(_ *http.Request) goldr.RouteResponse {\n\treturn goldr.Text{Body: \"ok\"}\n}\n")
	}
	if defineHandlers {
		if options.IndexFragment {
			builder.WriteString("\nfunc fragIndex(_ *http.Request) goldr.RouteResponse {\n\treturn goldr.Text{Body: \"ok\"}\n}\n")
		}
		for _, fragment := range options.Fragments {
			builder.WriteString("\nfunc frag")
			builder.WriteString(exportName(fragment))
			builder.WriteString("(_ *http.Request) goldr.RouteResponse {\n\treturn goldr.Text{Body: \"ok\"}\n}\n")
		}
		for _, action := range options.Actions {
			builder.WriteString("\nfunc ")
			builder.WriteString(action.Func)
			builder.WriteString("(w http.ResponseWriter, r *http.Request) {}\n")
		}
	}
	return builder.String()
}

func exportName(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '-' || r == '_'
	})
	var builder strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		builder.WriteString(strings.ToUpper(part[:1]))
		builder.WriteString(part[1:])
	}
	return builder.String()
}

func writeTemplToolModule(t *testing.T, root string, module string) string {
	t.Helper()

	goMod := readRepoFile(t, "examples/full_feature/go.mod")
	goMod = strings.Replace(goMod, "module github.com/mobiletoly/goldr/examples/full_feature", "module "+module, 1)
	goMod = strings.Replace(goMod, "tool github.com/mobiletoly/goldr/cmd/goldr\n\n", "", 1)
	goMod = strings.Replace(goMod, "\tgithub.com/mobiletoly/goldr/cmd/goldr v0.0.0-00010101000000-000000000000 // indirect\n", "", 1)
	goMod = strings.Replace(goMod, "\tgithub.com/urfave/cli/v3 v3.8.0 // indirect\n", "", 1)
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("Abs(repo root) error = %v", err)
	}
	goMod = strings.Replace(goMod, "replace github.com/mobiletoly/goldr => ../..", "replace github.com/mobiletoly/goldr => "+filepath.ToSlash(repoRoot), 1)
	goMod = strings.Replace(goMod, "\nreplace github.com/mobiletoly/goldr/cmd/goldr => ../../cmd/goldr\n", "", 1)
	writeFile(t, root, "go.mod", goMod)
	writeFile(t, root, "go.sum", readRepoFile(t, "examples/full_feature/go.sum"))
	return goMod
}

func readRepoFile(t *testing.T, name string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join("..", "..", "..", "..", filepath.FromSlash(name)))
	if err != nil {
		t.Fatalf("ReadFile(repo %s) error = %v", name, err)
	}
	return string(content)
}

func runTemplGenerate(t *testing.T, root string) {
	t.Helper()

	command := exec.CommandContext(context.Background(), "go", "tool", "templ", "generate", "-path", ".")
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go tool templ generate error = %v\n%s", err, output)
	}
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
