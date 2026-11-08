package templtool

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestScopeLiteralPathsAndTemplateCompanions(t *testing.T) {
	root := filepath.Join(t.TempDir(), "app[review]+")
	if err := os.Mkdir(root, 0755); err != nil {
		t.Fatal(err)
	}
	writeIgnore(t, root, " # Archives\r\n\r\n tmp/ \r\nold[copy]+\nviews/page.templ\nviews/other_templ.go\n")
	scope, err := LoadScope(root)
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(scope.IgnorePattern())
	for _, test := range []struct {
		path string
		want bool
	}{
		{"tmp", true}, {"tmp/archive/page.templ", true}, {"tmp-other/page.templ", false},
		{"other/tmp/page.templ", false}, {"old[copy]+/page.templ", true}, {"oldcopy/page.templ", false},
		{"views/page.templ", true}, {"views/page_templ.go", true}, {"views/page2.templ", false},
		{"views/other.templ", true}, {"views/other_templ.go", true}, {"views/another.templ", false},
	} {
		t.Run(test.path, func(t *testing.T) {
			name := filepath.Join(root, filepath.FromSlash(test.path))
			for _, path := range []string{name, strings.ReplaceAll(filepath.ToSlash(name), "/", `\`)} {
				if got := pattern.MatchString(path); got != test.want {
					t.Errorf("ignore(%q) = %v, want %v", path, got, test.want)
				}
			}
		})
	}
	if pattern.MatchString(filepath.Join(root+"-other", "tmp", "page.templ")) {
		t.Fatal("matched outside the application root")
	}
}

func TestScopeHasFilesHonorsOnlyRootConfiguration(t *testing.T) {
	parent := t.TempDir()
	writeIgnore(t, parent, "app/\n")
	root := filepath.Join(parent, "app")
	if err := os.MkdirAll(filepath.Join(root, "tmp"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tmp", "page.templ"), []byte("invalid archived template"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		content string
		want    bool
	}{
		{"missing", "", true},
		{"empty", "", true},
		{"comments", " # comment\n\n", true},
		{"excluded", "tmp\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.name != "missing" {
				writeIgnore(t, root, test.content)
			}
			scope, err := LoadScope(root)
			if err != nil {
				t.Fatal(err)
			}
			if got, err := scope.HasFiles(); err != nil || got != test.want {
				t.Fatalf("HasFiles() = %v, %v; want %v", got, err, test.want)
			}
			if test.name != "excluded" && scope.IgnorePattern() != "" {
				t.Fatal("empty configuration produced an ignore regexp")
			}
		})
	}
	if err := os.WriteFile(filepath.Join(root, "page.templ"), []byte("maintained"), 0644); err != nil {
		t.Fatal(err)
	}
	scope, err := LoadScope(root)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := scope.HasFiles(); err != nil || !got {
		t.Fatalf("maintained template: HasFiles() = %v, %v", got, err)
	}
}

func TestScopeRejectsInvalidPaths(t *testing.T) {
	for _, entry := range []string{"/archive", ".", "./", "./archive", "../archive", "a/../b", "a/./b", "a//b", "a//", `tmp\archive`, "C:/archive", "C:archive", "bad\x00name"} {
		t.Run(entry, func(t *testing.T) {
			root := t.TempDir()
			writeIgnore(t, root, "# comment\n\n"+entry+"\n")
			_, err := LoadScope(root)
			if err == nil || !strings.Contains(err.Error(), ".goldrignore:3: invalid app-root-relative path") {
				t.Fatalf("LoadScope() error = %v, want file/line diagnostic", err)
			}
		})
	}
}

func TestScopeReadError(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ignoreFileName), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadScope(root); err == nil || !strings.Contains(err.Error(), ".goldrignore") {
		t.Fatalf("LoadScope() error = %v, want read diagnostic", err)
	}
}

func writeIgnore(t *testing.T, root, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, ignoreFileName), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
