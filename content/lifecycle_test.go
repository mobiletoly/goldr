// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package content

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/mobiletoly/goldr"
)

//go:embed testdata/embedded
var embeddedContent embed.FS

func TestResolveObservesExternalChangesAndRecovers(t *testing.T) {
	directory := t.TempDir()
	writeContentFile(t, directory, "about/page.json", `{"title":"About"}`)
	writeContentFile(t, directory, "about/body.html", `<p>Version one</p>`)

	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = root.Close()
	}()
	pages, err := New(Config{FS: root.FS()})
	if err != nil {
		t.Fatal(err)
	}

	if body := renderPage(t, mustPage(t, pages, "/about")); !strings.Contains(body, "Version one") {
		t.Fatalf("initial body = %q", body)
	}
	writeContentFile(t, directory, "about/body.html", `<script data-version="two">trusted()</script><p onclick="trusted()">Version two</p>`)
	if body := renderPage(t, mustPage(t, pages, "/about")); !strings.Contains(body, `<script data-version="two">trusted()</script>`) || !strings.Contains(body, `onclick="trusted()"`) {
		t.Fatalf("edited body = %q", body)
	}

	writeContentFile(t, directory, "privacy/p1/page.json", `{"title":"Part One"}`)
	writeContentFile(t, directory, "privacy/p1/body.md", "# New page\n")
	if body := renderPage(t, mustPage(t, pages, "/privacy/p1")); !strings.Contains(body, `<h1 id="new-page">New page</h1>`) {
		t.Fatalf("new page body = %q", body)
	}

	writeContentFile(t, directory, "about/body.html", " \n\t")
	response, handled := pages.Resolve(newRequest(http.MethodGet, "/about", ""))
	if !handled {
		t.Fatal("invalid edit handled = false, want true")
	}
	if routeError, ok := response.(goldr.RouteError); !ok || routeError.Err == nil || !strings.Contains(routeError.Err.Error(), "body.html") || !strings.Contains(routeError.Err.Error(), "whitespace-only") {
		t.Fatalf("invalid edit response = %#v", response)
	}

	writeContentFile(t, directory, "about/body.html", `<p>Recovered</p>`)
	if body := renderPage(t, mustPage(t, pages, "/about")); !strings.Contains(body, "Recovered") {
		t.Fatalf("recovered body = %q", body)
	}

	if err := os.Remove(filepath.Join(directory, "about", "body.html")); err != nil {
		t.Fatal(err)
	}
	response, handled = pages.Resolve(newRequest(http.MethodGet, "/about", ""))
	if !handled {
		t.Fatal("missing required file handled = false, want true")
	}
	if routeError, ok := response.(goldr.RouteError); !ok || routeError.Err == nil || !strings.Contains(routeError.Err.Error(), "exactly one") {
		t.Fatalf("missing required file response = %#v", response)
	}
}

func TestEmbeddedAndConcurrentResolve(t *testing.T) {
	sub, err := fs.Sub(embeddedContent, "testdata/embedded")
	if err != nil {
		t.Fatal(err)
	}
	pages, err := New(Config{FS: sub})
	if err != nil {
		t.Fatal(err)
	}
	if body := renderPage(t, mustPage(t, pages, "/about")); !strings.Contains(body, `<h1 id="embedded">Embedded</h1>`) {
		t.Fatalf("embedded body = %q", body)
	}

	const workers = 24
	const requestsPerWorker = 40
	var wait sync.WaitGroup
	errors := make(chan error, workers)
	for range workers {
		wait.Go(func() {
			for range requestsPerWorker {
				response, handled := pages.Resolve(newRequest(http.MethodGet, "/about", ""))
				if !handled {
					errors <- fmt.Errorf("Resolve() handled = false")
					return
				}
				page, ok := response.(goldr.Page)
				if !ok || page.Metadata.Title != "Embedded About" {
					errors <- fmt.Errorf("Resolve() response = %#v", response)
					return
				}
			}
		})
	}
	wait.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}

func TestCheckRejectsRealSymlink(t *testing.T) {
	directory := t.TempDir()
	writeContentFile(t, directory, "about/page.json", `{"title":"About"}`)
	writeContentFile(t, directory, "target.html", `<p>Target</p>`)
	if err := os.Symlink(filepath.Join(directory, "target.html"), filepath.Join(directory, "about", "body.html")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = root.Close()
	}()
	if err := Check(root.FS()); err == nil || !strings.Contains(err.Error(), "symbolic links") {
		t.Fatalf("Check() error = %v, want symlink rejection", err)
	}
}

func writeContentFile(t *testing.T, root, name, content string) {
	t.Helper()
	filename := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
