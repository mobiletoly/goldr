// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package content

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/mobiletoly/goldr"
)

func TestNewAndCheckValidTree(t *testing.T) {
	files := validContentFS()
	pages, err := New(Config{FS: files})
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
	if pages == nil {
		t.Fatal("New() pages = nil")
	}
	if err := Check(files); err != nil {
		t.Fatalf("Check() error = %v, want nil", err)
	}
	if _, err := New(Config{}); !errors.Is(err, errFilesystemRequired) {
		t.Fatalf("New(nil) error = %v, want filesystem required", err)
	}
	if err := Check(nil); !errors.Is(err, errFilesystemRequired) {
		t.Fatalf("Check(nil) error = %v, want filesystem required", err)
	}
}

func TestResolveOutcomes(t *testing.T) {
	pages, err := New(Config{FS: validContentFS()})
	if err != nil {
		t.Fatal(err)
	}

	response, handled := pages.Resolve(newRequest(http.MethodGet, "/privacy/p1", ""))
	if !handled {
		t.Fatal("nested page handled = false, want true")
	}
	page, ok := response.(goldr.Page)
	if !ok {
		t.Fatalf("response type = %T, want goldr.Page", response)
	}
	if page.Metadata != (goldr.PageMetadata{Title: "Part One", Description: "Nested Markdown."}) {
		t.Fatalf("metadata = %#v", page.Metadata)
	}
	body := renderPage(t, page)
	for _, want := range []string{`<article class="goldr-content">`, `<h1 id="part-one">Part One</h1>`, `<a href="/privacy">Privacy</a>`} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q: %s", want, body)
		}
	}

	head, handled := pages.Resolve(newRequest(http.MethodHead, "/about", ""))
	if !handled {
		t.Fatal("HEAD handled = false, want true")
	}
	if _, ok := head.(goldr.Page); !ok {
		t.Fatalf("HEAD response type = %T, want goldr.Page", head)
	}

	withQuery, handled := pages.Resolve(newRequest(http.MethodGet, "/about", "draft=1"))
	if !handled || renderPage(t, withQuery.(goldr.Page)) != renderPage(t, mustPage(t, pages, "/about")) {
		t.Fatal("query string changed content selection")
	}

	for _, path := range []string{"/missing", "/container"} {
		response, handled := pages.Resolve(newRequest(http.MethodGet, path, ""))
		if handled || response != nil {
			t.Fatalf("Resolve(%q) = (%T, %v), want (nil, false)", path, response, handled)
		}
	}

	var zero Pages
	response, handled = zero.Resolve(newRequest(http.MethodGet, "/about", ""))
	if !handled {
		t.Fatal("unconfigured Pages handled = false, want true")
	}
	if routeError, ok := response.(goldr.RouteError); !ok || routeError.Err == nil {
		t.Fatalf("unconfigured response = %#v, want RouteError", response)
	}
}

func TestResolveDeclinesBeforeFilesystemAccess(t *testing.T) {
	counter := &countingFS{files: validContentFS()}
	pages, err := New(Config{FS: counter})
	if err != nil {
		t.Fatal(err)
	}
	counter.opens.Store(0)

	requests := []*http.Request{
		newRequest(http.MethodPost, "/about", ""),
		newRequest(http.MethodPut, "/about", ""),
		newRequest(http.MethodGet, "/", ""),
		newRequest(http.MethodGet, "/about/", ""),
		newRequest(http.MethodGet, "/About", ""),
		newRequest(http.MethodGet, "/about//child", ""),
		newRequest(http.MethodGet, `/about\\child`, ""),
		newEscapedRequest("/privacy/p1", "/privacy%2Fp1"),
		newEscapedRequest("/../privacy", "/%2e%2e/privacy"),
	}
	for _, request := range requests {
		response, handled := pages.Resolve(request)
		if handled || response != nil {
			t.Fatalf("%s %q = (%T, %v), want (nil, false)", request.Method, request.URL.EscapedPath(), response, handled)
		}
	}
	if got := counter.opens.Load(); got != 0 {
		t.Fatalf("filesystem opens = %d, want 0", got)
	}
}

func TestResolveRecognizedInvalidEntryIsTerminal(t *testing.T) {
	files := validContentFS()
	pages, err := New(Config{FS: files})
	if err != nil {
		t.Fatal(err)
	}
	delete(files, "about/body.html")

	response, handled := pages.Resolve(newRequest(http.MethodGet, "/about", ""))
	if !handled {
		t.Fatal("invalid recognized entry handled = false, want true")
	}
	routeError, ok := response.(goldr.RouteError)
	if !ok || routeError.Err == nil || !strings.Contains(routeError.Err.Error(), "about") || !strings.Contains(routeError.Err.Error(), "exactly one") {
		t.Fatalf("response = %#v, want source-local RouteError", response)
	}
}

func TestResolveDeclinesIgnoredFileCollisions(t *testing.T) {
	files := validContentFS()
	files["ignored"] = &fstest.MapFile{Data: []byte("ordinary ignored file")}
	files["privacy/ignored"] = &fstest.MapFile{Data: []byte("nested ordinary ignored file")}
	pages, err := New(Config{FS: files})
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"/ignored", "/privacy/ignored"} {
		response, handled := pages.Resolve(newRequest(http.MethodGet, path, ""))
		if handled || response != nil {
			t.Fatalf("Resolve(%q) = (%T, %v), want (nil, false)", path, response, handled)
		}
	}
}

func validContentFS() fstest.MapFS {
	return fstest.MapFS{
		"about/page.json":       {Data: []byte(`{"title":"About","description":"About this site."}`)},
		"about/body.html":       {Data: []byte(`<h1>About</h1><p class="lead">Hello.</p>`)},
		"about/notes.txt":       {Data: []byte("ignored")},
		"privacy/page.json":     {Data: []byte(`{"title":"Privacy"}`)},
		"privacy/body.html":     {Data: []byte(`<h1>Privacy</h1>`)},
		"privacy/p1/page.json":  {Data: []byte(`{"title":"Part One","description":"Nested Markdown."}`)},
		"privacy/p1/body.md":    {Data: []byte("# Part One\n\nRead [Privacy](/privacy).\n")},
		"privacy/p2/page.json":  {Data: []byte(`{"title":"Part Two"}`)},
		"privacy/p2/body.md":    {Data: []byte("# Part Two\n")},
		"container/readme.txt":  {Data: []byte("organizational container")},
		"empty-container/.keep": {Data: []byte{}},
	}
}

func mustPage(t *testing.T, pages *Pages, path string) goldr.Page {
	t.Helper()
	response, handled := pages.Resolve(newRequest(http.MethodGet, path, ""))
	if !handled {
		t.Fatalf("Resolve(%q) handled = false", path)
	}
	page, ok := response.(goldr.Page)
	if !ok {
		t.Fatalf("Resolve(%q) type = %T, want goldr.Page", path, response)
	}
	return page
}

func renderPage(t *testing.T, page goldr.Page) string {
	t.Helper()
	var output bytes.Buffer
	if err := page.Component.Render(context.Background(), &output); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	return output.String()
}

func newRequest(method, path, rawQuery string) *http.Request {
	return &http.Request{
		Method: method,
		URL: &url.URL{
			Path:     path,
			RawQuery: rawQuery,
		},
	}
}

func newEscapedRequest(path, rawPath string) *http.Request {
	return &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Path:    path,
			RawPath: rawPath,
		},
	}
}

type countingFS struct {
	files fs.FS
	opens atomic.Int64
}

func (files *countingFS) Open(name string) (fs.File, error) {
	files.opens.Add(1)
	return files.files.Open(name)
}
