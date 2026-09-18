// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestExampleHandler(t *testing.T) {
	files := exampleContentFS()
	var logs bytes.Buffer
	handler, err := exampleHandler(files, &logs, false)
	if err != nil {
		t.Fatalf("exampleHandler() error = %v", err)
	}

	tests := []struct {
		name       string
		path       string
		status     int
		contains   []string
		notContain string
	}{
		{
			name:     "HTML content uses root layout and metadata",
			path:     "/privacy",
			status:   http.StatusOK,
			contains: []string{"<title>Privacy</title>", `name="description" content="Privacy description"`, `<script data-trusted="html">window.contentPage = true</script>`, `onclick="trusted()"`, "outer middleware"},
		},
		{
			name:     "nested Markdown keeps original URL",
			path:     "/privacy/p1?source=test",
			status:   http.StatusOK,
			contains: []string{"<title>Privacy Part One</title>", "Nested Markdown body", `<span data-trusted="markdown">raw HTML</span>`, `<a href="javascript:trusted()">trusted destination</a>`, `data-request-uri="/privacy/p1?source=test"`, "outer middleware"},
		},
		{
			name:       "generated route wins content collision",
			path:       "/about",
			status:     http.StatusOK,
			contains:   []string{"Application route wins"},
			notContain: "Shadowed content body",
		},
		{
			name:     "final custom not found",
			path:     "/missing",
			status:   http.StatusNotFound,
			contains: []string{"Page not found", "No application route or content page matches /missing."},
		},
		{
			name:       "content sources are not public files",
			path:       "/content/privacy/body.html",
			status:     http.StatusNotFound,
			contains:   []string{"Page not found"},
			notContain: "HTML privacy body",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d; body = %s", recorder.Code, test.status, recorder.Body.String())
			}
			if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
				t.Fatalf("Cache-Control = %q, want no-store", got)
			}
			for _, want := range test.contains {
				if !strings.Contains(recorder.Body.String(), want) {
					t.Fatalf("body missing %q: %s", want, recorder.Body.String())
				}
			}
			if test.notContain != "" && strings.Contains(recorder.Body.String(), test.notContain) {
				t.Fatalf("body unexpectedly contains %q: %s", test.notContain, recorder.Body.String())
			}
		})
	}

	files["privacy/body.html"].Data = []byte("private invalid body \xff")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/privacy", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("invalid content status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(recorder.Body.String(), "Content unavailable") || strings.Contains(recorder.Body.String(), "private invalid body") {
		t.Fatalf("invalid content body = %q", recorder.Body.String())
	}
	if !strings.Contains(logs.String(), "privacy/body.html") || strings.Contains(logs.String(), "private invalid body") {
		t.Fatalf("invalid content log = %q", logs.String())
	}
}

func TestCheckContentModeProcess(t *testing.T) {
	validRoot := writeContentTree(t, `<p>Valid body</p>`)
	valid := runCheckHelper(t, validRoot)
	if valid.ExitCode() != 0 {
		t.Fatalf("valid check exit = %d; output = %s", valid.ExitCode(), valid.String())
	}
	if !strings.Contains(valid.String(), "content is valid") {
		t.Fatalf("valid check output = %q", valid.String())
	}

	invalidRoot := writeContentTree(t, " \n\t")
	invalid := runCheckHelper(t, invalidRoot)
	if invalid.ExitCode() == 0 {
		t.Fatalf("invalid check exit = 0; output = %s", invalid.String())
	}
	if !strings.Contains(invalid.String(), "body must not be empty or whitespace-only") {
		t.Fatalf("invalid check output = %q", invalid.String())
	}
}

func TestCheckContentProcess(t *testing.T) {
	if os.Getenv("CONTENT_PAGES_CHECK_HELPER") != "1" {
		return
	}
	if err := run(context.Background(), []string{"-check-content", "-addr", "not-a-listen-address"}, os.Stdout, os.Stderr); err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

func exampleContentFS() fstest.MapFS {
	return fstest.MapFS{
		"about/page.json":      &fstest.MapFile{Data: []byte(`{"title":"Content About"}`)},
		"about/body.md":        &fstest.MapFile{Data: []byte("# Shadowed content body\n")},
		"privacy/page.json":    &fstest.MapFile{Data: []byte(`{"title":"Privacy","description":"Privacy description"}`)},
		"privacy/body.html":    &fstest.MapFile{Data: []byte(`<script data-trusted="html">window.contentPage = true</script><h1 onclick="trusted()">HTML privacy body</h1>`)},
		"privacy/p1/page.json": &fstest.MapFile{Data: []byte(`{"title":"Privacy Part One"}`)},
		"privacy/p1/body.md":   &fstest.MapFile{Data: []byte("# Nested Markdown body\n\n<span data-trusted=\"markdown\">raw HTML</span> and [trusted destination](javascript:trusted()).\n")},
	}
}

func writeContentTree(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	directory := filepath.Join(root, "content", "page")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "page.json"), []byte(`{"title":"Page"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "body.html"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

type checkResult struct {
	output   string
	exitCode int
}

func (result checkResult) String() string { return result.output }
func (result checkResult) ExitCode() int  { return result.exitCode }

func runCheckHelper(t *testing.T, directory string) checkResult {
	t.Helper()
	command := exec.Command(os.Args[0], "-test.run=^TestCheckContentProcess$")
	command.Dir = directory
	command.Env = append(os.Environ(), "CONTENT_PAGES_CHECK_HELPER=1")
	output, err := command.CombinedOutput()
	if err == nil {
		return checkResult{output: string(output)}
	}
	if exitError, ok := errors.AsType[*exec.ExitError](err); ok {
		return checkResult{output: string(output), exitCode: exitError.ExitCode()}
	}
	t.Fatalf("run check helper: %v", err)
	return checkResult{}
}
