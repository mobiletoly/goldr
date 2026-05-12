package assets

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDistCheckListAndClean(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "assets/build/app.css", "body {}\n")
	writeTestFile(t, root, "assets/build/fonts/inter.woff2", "font\n")

	if err := runDist(options{root: root}); err != nil {
		t.Fatalf("runDist() error = %v", err)
	}

	appHash := shortHash([]byte("body {}\n"))
	fontHash := shortHash([]byte("font\n"))
	appDist := filepath.Join(root, "assets", "dist", "app."+appHash+".css")
	fontDist := filepath.Join(root, "assets", "dist", "fonts", "inter."+fontHash+".woff2")
	requireFileContent(t, appDist, "body {}\n")
	requireFileContent(t, fontDist, "font\n")

	generated := readTestFile(t, filepath.Join(root, "assets", "goldr_assets_gen.go"))
	for _, want := range []string{
		"package assets",
		`Path: "/assets/app.` + appHash + `.css"`,
		`Path: "/assets/fonts/inter.` + fontHash + `.woff2"`,
		"func Path(name string) string",
		"func Lookup(name string) (string, bool)",
		"func FS() fs.FS",
	} {
		if !strings.Contains(generated, want) {
			t.Fatalf("generated source = %q, want %q", generated, want)
		}
	}

	if err := runCheck(options{root: root}); err != nil {
		t.Fatalf("runCheck() error = %v", err)
	}

	var table bytes.Buffer
	if err := runList(options{root: root}, &table); err != nil {
		t.Fatalf("runList() error = %v", err)
	}
	for _, want := range []string{
		"Logical asset",
		"/assets/app." + appHash + ".css",
		"/assets/fonts/inter." + fontHash + ".woff2",
	} {
		if !strings.Contains(table.String(), want) {
			t.Fatalf("table = %q, want %q", table.String(), want)
		}
	}

	var jsonOut bytes.Buffer
	if err := runList(options{root: root, json: true}, &jsonOut); err != nil {
		t.Fatalf("runList(json) error = %v", err)
	}
	var rows []listRow
	if err := json.Unmarshal(jsonOut.Bytes(), &rows); err != nil {
		t.Fatalf("Unmarshal(list json) error = %v; json = %q", err, jsonOut.String())
	}
	if len(rows) != 2 || rows[0].Name != "app.css" || rows[1].Name != "fonts/inter.woff2" {
		t.Fatalf("json rows = %#v", rows)
	}

	writeTestFile(t, root, "assets/build/app.css", "body { color: black; }\n")
	if err := runDist(options{root: root}); err != nil {
		t.Fatalf("runDist(updated) error = %v", err)
	}
	if _, err := os.Stat(appDist); err != nil {
		t.Fatalf("old dist file Stat() error = %v", err)
	}
	writeTestFile(t, root, "assets/dist/keep.txt", "keep\n")

	if err := runClean(options{root: root}); err != nil {
		t.Fatalf("runClean() error = %v", err)
	}
	if _, err := os.Stat(appDist); !os.IsNotExist(err) {
		t.Fatalf("old dist Stat() error = %v, want missing", err)
	}
	requireFileContent(t, filepath.Join(root, "assets", "dist", "keep.txt"), "keep\n")
	if err := runCheck(options{root: root}); err != nil {
		t.Fatalf("runCheck(after clean) error = %v", err)
	}
}

func TestRunCheckReportsStaleGeneratedFile(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "assets/build/app.css", "body {}\n")

	if err := runDist(options{root: root}); err != nil {
		t.Fatalf("runDist() error = %v", err)
	}
	writeTestFile(t, root, "assets/goldr_assets_gen.go", "stale\n")

	err := runCheck(options{root: root})
	if err == nil {
		t.Fatal("runCheck() error = nil, want stale generated file")
	}
	if !strings.Contains(err.Error(), "goldr_assets_gen.go") || !strings.Contains(err.Error(), "is stale") {
		t.Fatalf("runCheck() error = %v, want stale generated file", err)
	}
}

func TestRunDistEscapesGeneratedAssetURLs(t *testing.T) {
	root := t.TempDir()
	content := "body {}\n"
	writeTestFile(t, root, "assets/build/theme#dark.css", content)

	if err := runDist(options{root: root}); err != nil {
		t.Fatalf("runDist() error = %v", err)
	}

	hash := shortHash([]byte(content))
	generated := readTestFile(t, filepath.Join(root, "assets", "goldr_assets_gen.go"))
	wantPath := `Path: "/assets/theme%23dark.` + hash + `.css"`
	if !strings.Contains(generated, wantPath) {
		t.Fatalf("generated source = %q, want %q", generated, wantPath)
	}
	if strings.Contains(generated, "/assets/theme#dark.") {
		t.Fatalf("generated source = %q, want escaped asset URL", generated)
	}

	var jsonOut bytes.Buffer
	if err := runList(options{root: root, json: true}, &jsonOut); err != nil {
		t.Fatalf("runList(json) error = %v", err)
	}
	var rows []listRow
	if err := json.Unmarshal(jsonOut.Bytes(), &rows); err != nil {
		t.Fatalf("Unmarshal(list json) error = %v; json = %q", err, jsonOut.String())
	}
	if len(rows) != 1 {
		t.Fatalf("json rows = %#v, want one row", rows)
	}
	if want := "/assets/theme%23dark." + hash + ".css"; rows[0].Path != want {
		t.Fatalf("json row path = %q, want %q", rows[0].Path, want)
	}
	if want := "assets/dist/theme#dark." + hash + ".css"; rows[0].Dist != want {
		t.Fatalf("json row dist = %q, want %q", rows[0].Dist, want)
	}
}

func TestRunDistCreatesDistForEmptyBuild(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "assets", "build"), 0755); err != nil {
		t.Fatalf("MkdirAll(build) error = %v", err)
	}

	if err := runDist(options{root: root}); err != nil {
		t.Fatalf("runDist() error = %v", err)
	}
	info, err := os.Stat(filepath.Join(root, "assets", "dist"))
	if err != nil {
		t.Fatalf("Stat(dist) error = %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("dist IsDir() = false")
	}
	if err := runCheck(options{root: root}); err != nil {
		t.Fatalf("runCheck() error = %v", err)
	}
}

func TestRunDistRejectsSymlinkInput(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "assets/build/app.css", "body {}\n")
	if err := os.Symlink("app.css", filepath.Join(root, "assets", "build", "link.css")); err != nil {
		t.Skipf("Symlink() error = %v", err)
	}

	err := runDist(options{root: root})
	if err == nil {
		t.Fatal("runDist() error = nil, want symlink error")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("runDist() error = %v, want symlink error", err)
	}
}

func writeTestFile(t *testing.T, root string, name string, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(content)
}

func requireFileContent(t *testing.T, path string, want string) {
	t.Helper()

	got := readTestFile(t, path)
	if got != want {
		t.Fatalf("%s = %q, want %q", path, got, want)
	}
}
