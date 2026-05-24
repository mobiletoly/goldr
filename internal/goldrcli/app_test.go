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
	writeFile(t, root, "app/routes/page.go", `package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(_ *http.Request) goldr.RouteResponse { return goldr.NewPage(PageView(), goldr.PageMetadata{}) }
`)
	writeFile(t, root, "app/routes/page.templ", "package routes\n\ntempl PageView() {<h1>Root</h1>}\n")
	writeFile(t, root, "app/routes/settings/page.go", `package settings

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(_ *http.Request) goldr.RouteResponse { return goldr.NewPage(PageView(), goldr.PageMetadata{}) }
`)
	writeFile(t, root, "app/routes/settings/page.templ", "package settings\n\ntempl PageView() {<h1>Settings</h1>}\n")
	return root
}

func writeTemplToolModule(t *testing.T, root string, module string) string {
	t.Helper()

	goMod := readRepoFile(t, "go.mod")
	const repoModule = "module github.com/mobiletoly/goldr"
	if !strings.HasPrefix(goMod, repoModule+"\n") {
		t.Fatalf("repo go.mod = %q, want module header %q", goMod, repoModule)
	}
	goMod = strings.Replace(goMod, repoModule, "module "+module, 1)
	writeFile(t, root, "go.mod", goMod)
	writeFile(t, root, "go.sum", readRepoFile(t, "go.sum"))
	return goMod
}

func readRepoFile(t *testing.T, name string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(name)))
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
