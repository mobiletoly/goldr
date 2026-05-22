// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldrcli

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestDevTemplArgsUseProxyWatchAndWrapper(t *testing.T) {
	config := devConfig{
		root:        filepath.Join("tmp", "goldr-app"),
		appURL:      "http://127.0.0.1:8080",
		proxyBind:   "127.0.0.1",
		proxyPort:   7331,
		wrapperPath: filepath.Join("tmp", "goldr-dev-wrapper"),
	}

	args := templArgs(config)

	want := []string{
		"tool",
		"templ",
		"generate",
		"-path", config.root,
		"-watch",
		"-watch-pattern", devWatchPattern(),
		"-ignore-pattern", devIgnorePattern(),
		"-proxy", config.appURL,
		"-proxybind", config.proxyBind,
		"-proxyport", strconv.Itoa(config.proxyPort),
		"-open-browser=false",
		"-cmd", config.wrapperPath,
	}
	if strings.Join(args, "\n") != strings.Join(want, "\n") {
		t.Fatalf("templ args = %#v, want %#v", args, want)
	}
}

func TestDevWatchPatternIncludesRoutesTemplatesAndAssetBuild(t *testing.T) {
	pattern := devWatchPattern()
	for _, want := range []string{
		`.go$`,
		`.templ$`,
		`assets`,
		`build`,
	} {
		if !strings.Contains(pattern, want) {
			t.Fatalf("watch pattern = %q, want %q", pattern, want)
		}
	}
}

func TestDevIgnorePatternExcludesGoldrGeneratedOutputs(t *testing.T) {
	pattern := devIgnorePattern()
	for _, want := range []string{
		`app`,
		`routes`,
		`urls`,
		`goldr_gen\.go`,
		`assets`,
		`goldr_assets_gen\.go`,
		`dist`,
		`\.goldr`,
	} {
		if !strings.Contains(pattern, want) {
			t.Fatalf("ignore pattern = %q, want %q", pattern, want)
		}
	}
}

func TestDevWrapperRunsGoldrGenerateThenAppCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell wrapper content is Unix-specific")
	}
	root := t.TempDir()
	config := devConfig{
		root:            root,
		goldrExecutable: filepath.Join(root, "bin", "goldr test"),
		proxyBind:       "127.0.0.1",
		proxyPort:       7331,
		command:         `go run . --message "hello dev"`,
	}

	wrapperTempDir := t.TempDir()
	wrapper, err := writeUnixDevWrapper(config, wrapperTempDir)
	if err != nil {
		t.Fatalf("writeUnixDevWrapper() error = %v", err)
	}
	defer func() {
		_ = os.Remove(wrapper)
	}()

	source := readFile(t, wrapper)
	generateCommand := "TEMPL_DEV_MODE_ROOT=" + shellQuote(devGenerateTemplRoot(wrapper)) + " " + shellQuote(config.goldrExecutable) + " generate --root " + shellQuote(root)
	for _, want := range []string{
		"#!/bin/sh",
		"set -eu",
		generateCommand,
		"cd " + shellQuote(root),
		"printf '%s\\n' 'goldr dev live reload proxy'",
		"printf '%s\\n' 'Open this URL in your browser:'",
		"printf '%s\\n' '  http://127.0.0.1:7331'",
		"printf '%s\\n' 'Do not open the app server URL directly.'",
		"exec /bin/sh -c " + shellQuote(config.command),
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("wrapper = %q, want %q", source, want)
		}
	}
	if strings.Contains(generateCommand, shellQuote(wrapperTempDir)+" ") {
		t.Fatalf("generate command = %q, must not use templ's default temp root", generateCommand)
	}
	assertBefore(t, source, generateCommand, "printf '%s\\n' '  http://127.0.0.1:7331'")
	assertBefore(t, source, "printf '%s\\n' '  http://127.0.0.1:7331'", "exec /bin/sh -c "+shellQuote(config.command))
	if strings.Contains(source, "assets dist") {
		t.Fatalf("wrapper = %q, must not run assets dist separately", source)
	}
	if strings.Contains(source, "unset TEMPL_DEV_MODE") {
		t.Fatalf("wrapper = %q, must keep templ dev mode available to the app", source)
	}
}

func TestDevProxyURLUsesHostPortFormatting(t *testing.T) {
	config := devConfig{
		proxyBind: "::1",
		proxyPort: 7331,
	}

	got := devProxyURL(config)
	if got != "http://[::1]:7331" {
		t.Fatalf("devProxyURL() = %q, want %q", got, "http://[::1]:7331")
	}
}

func assertBefore(t *testing.T, source string, first string, second string) {
	t.Helper()
	firstIndex := strings.Index(source, first)
	if firstIndex < 0 {
		t.Fatalf("source = %q, want %q", source, first)
	}
	secondIndex := strings.Index(source, second)
	if secondIndex < 0 {
		t.Fatalf("source = %q, want %q", source, second)
	}
	if firstIndex >= secondIndex {
		t.Fatalf("source = %q, want %q before %q", source, first, second)
	}
}

func TestDevWrapperTempDirSkipsWhitespacePath(t *testing.T) {
	parent := t.TempDir()
	if devPathHasWhitespace(parent) {
		t.Skipf("test temp directory contains whitespace: %s", parent)
	}
	spaced := filepath.Join(parent, "tmp with spaces")
	clean := filepath.Join(parent, "tmp")
	if err := os.Mkdir(spaced, 0755); err != nil {
		t.Fatalf("mkdir spaced temp dir: %v", err)
	}
	if err := os.Mkdir(clean, 0755); err != nil {
		t.Fatalf("mkdir clean temp dir: %v", err)
	}

	got, err := selectDevWrapperTempDir([]string{spaced, clean})
	if err != nil {
		t.Fatalf("selectDevWrapperTempDir() error = %v", err)
	}
	if got != clean {
		t.Fatalf("selectDevWrapperTempDir() = %q, want %q", got, clean)
	}
}

func TestWriteDevWrapperUsesSpaceFreeTempDir(t *testing.T) {
	parent := t.TempDir()
	if devPathHasWhitespace(parent) {
		t.Skipf("test temp directory contains whitespace: %s", parent)
	}
	spaced := filepath.Join(parent, "tmp with spaces")
	clean := filepath.Join(parent, "tmp")
	if err := os.Mkdir(spaced, 0755); err != nil {
		t.Fatalf("mkdir spaced temp dir: %v", err)
	}
	if err := os.Mkdir(clean, 0755); err != nil {
		t.Fatalf("mkdir clean temp dir: %v", err)
	}
	config := devConfig{
		root:            parent,
		goldrExecutable: filepath.Join(parent, "goldr"),
		command:         "go run .",
	}

	wrapper, err := writeDevWrapperInTempDirs(config, []string{spaced, clean})
	if err != nil {
		t.Fatalf("writeDevWrapperInTempDirs() error = %v", err)
	}
	defer func() {
		_ = os.Remove(wrapper)
	}()

	if devPathHasWhitespace(wrapper) {
		t.Fatalf("wrapper path = %q, want no whitespace", wrapper)
	}
	if !strings.HasPrefix(wrapper, clean+string(os.PathSeparator)) {
		t.Fatalf("wrapper path = %q, want under %q", wrapper, clean)
	}
}

func TestWriteDevWrapperSkipsWhitespaceTMPDIR(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("TMPDIR fallback behavior is Unix-specific")
	}
	if _, err := os.Stat("/tmp"); err != nil {
		t.Skipf("/tmp is not available: %v", err)
	}
	parent := t.TempDir()
	spaced := filepath.Join(parent, "tmp with spaces")
	if err := os.Mkdir(spaced, 0755); err != nil {
		t.Fatalf("mkdir spaced temp dir: %v", err)
	}
	t.Setenv("TMPDIR", spaced)
	config := devConfig{
		root:            parent,
		goldrExecutable: filepath.Join(parent, "goldr"),
		command:         "go run .",
	}

	wrapper, err := writeDevWrapper(config)
	if err != nil {
		t.Fatalf("writeDevWrapper() error = %v", err)
	}
	defer func() {
		_ = os.Remove(wrapper)
	}()

	if devPathHasWhitespace(wrapper) {
		t.Fatalf("wrapper path = %q, want no whitespace", wrapper)
	}
}

func TestResolveDevConfigRejectsInvalidOptionsBeforeTemplLookup(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/devapp\n\ngo 1.26.3\n")
	writeFile(t, root, "app/routes/page.go", "package routes\n")
	writeFile(t, root, "app/routes/page.templ", "package routes\n\ntempl PageView() {}\n")

	tests := []struct {
		name string
		opts devOptions
		want string
	}{
		{
			name: "bad app url",
			opts: devOptions{
				root:      root,
				appURL:    "ftp://127.0.0.1:8080",
				proxyAddr: defaultDevProxyAddr,
				command:   defaultDevCommand,
			},
			want: "--app-url must use http or https",
		},
		{
			name: "bad proxy",
			opts: devOptions{
				root:      root,
				appURL:    defaultDevAppURL,
				proxyAddr: "127.0.0.1",
				command:   defaultDevCommand,
			},
			want: "invalid --proxy-addr",
		},
		{
			name: "empty command",
			opts: devOptions{
				root:      root,
				appURL:    defaultDevAppURL,
				proxyAddr: defaultDevProxyAddr,
				command:   "  ",
			},
			want: "--cmd must not be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolveDevConfig(context.Background(), tt.opts)
			if err == nil {
				t.Fatal("resolveDevConfig() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("resolveDevConfig() error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestDevShellQuote(t *testing.T) {
	got := shellQuote(`it's fine`)
	if got != `'it'"'"'s fine'` {
		t.Fatalf("shellQuote() = %q", got)
	}
}
