package dev

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func TestGoldrIgnoreDevUsesAppRootAndPreservesBuiltInIgnores(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/devapp\n\ngo 1.26.0\n")
	writeFile(t, root, "app/routes/route.go", "package routes\n")
	writeFile(t, root, ".goldrignore", "tmp/\n")
	commandDir := t.TempDir()
	writeFile(t, commandDir, ".goldrignore", "app/\n")
	config, err := resolveDevConfig(context.Background(), devOptions{
		root: root, cmdDir: commandDir, command: defaultDevCommand,
		appURL: defaultDevAppURL, proxyAddr: defaultDevProxyAddr,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(config.wrapperPath) }()
	args := templArgs(config)
	pattern := regexp.MustCompile(args[slices.Index(args, "-ignore-pattern")+1])
	for _, test := range []struct {
		name string
		want bool
	}{
		{"tmp/archive/page.templ", true}, {"tmp/archive/route.go", true},
		{"app/routes/goldr_gen.go", true}, {"assets/dist/app.css", true},
		{"app/routes/route.go", false}, {"tmp-other/route.go", false},
	} {
		if got := pattern.MatchString(filepath.Join(config.root, filepath.FromSlash(test.name))); got != test.want {
			t.Errorf("ignore(%s) = %v, want %v", test.name, got, test.want)
		}
	}
	if !templControlsReloadPath(config.root, filepath.Join(config.root, "tmp/archive/route.go")) {
		t.Fatal("ignored Go source became a reload-only input")
	}
}

func TestGoldrIgnoreDevRejectsConfigurationBeforeStartup(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/devapp\n\ngo 1.26.0\n")
	writeFile(t, root, "app/routes/route.go", "package routes\n")
	writeFile(t, root, ".goldrignore", "../archive\n")
	_, err := resolveDevConfig(context.Background(), devOptions{
		root: root, command: defaultDevCommand,
		appURL: defaultDevAppURL, proxyAddr: defaultDevProxyAddr,
	})
	if err == nil || !strings.Contains(err.Error(), ".goldrignore:1:") {
		t.Fatalf("resolveDevConfig() error = %v, want configuration error before startup", err)
	}
}
