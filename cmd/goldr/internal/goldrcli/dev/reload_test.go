// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package dev

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestResolveReloadPathsCanonicalizesAndCollapsesContainedPaths(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(root, "content")
	shared := filepath.Join(root, "shared")
	writeFile(t, root, "content/page.html", "first")
	writeFile(t, root, "shared/legal.html", "legal")
	t.Chdir(root)
	canonicalContent, err := filepath.EvalSymlinks(content)
	if err != nil {
		t.Fatal(err)
	}
	canonicalLegal, err := filepath.EvalSymlinks(filepath.Join(shared, "legal.html"))
	if err != nil {
		t.Fatal(err)
	}

	paths, err := resolveReloadPaths([]string{
		"content",
		"content/page.html",
		"./content",
		filepath.Join(shared, "legal.html"),
	})
	if err != nil {
		t.Fatalf("resolveReloadPaths() error = %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("len(paths) = %d, want 2: %#v", len(paths), paths)
	}
	if paths[0].path != canonicalContent || !paths[0].directory {
		t.Fatalf("paths[0] = %#v, want directory %q", paths[0], canonicalContent)
	}
	if paths[1].path != canonicalLegal || paths[1].directory {
		t.Fatalf("paths[1] = %#v, want file %q", paths[1], canonicalLegal)
	}
}

func TestResolveReloadPathsRejectsMissingAndEmptyPaths(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
		want string
	}{
		{name: "empty", path: "  ", want: "must not be empty"},
		{name: "missing", path: filepath.Join(t.TempDir(), "missing"), want: "resolve --reload-path"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveReloadPaths([]string{tc.path})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("resolveReloadPaths() error = %v, want containing %q", err, tc.want)
			}
		})
	}
}

func TestTemplControlsReloadPathOnlyInsideAppRoot(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct {
		name string
		path string
		want bool
	}{
		{name: "go", path: filepath.Join(root, "app", "route.go"), want: true},
		{name: "templ", path: filepath.Join(root, "app", "page.templ"), want: true},
		{name: "asset build", path: filepath.Join(root, "assets", "build", "app.css"), want: true},
		{name: "content", path: filepath.Join(root, "content", "page.html"), want: false},
		{name: "external go", path: filepath.Join(filepath.Dir(root), "external", "page.go"), want: false},
		{name: "generated route", path: filepath.Join(root, "app", "routes", "goldr_gen.go"), want: true},
		{name: "generated asset", path: filepath.Join(root, "assets", "dist", "app.12345678.css"), want: true},
		{name: "asset state", path: filepath.Join(root, "assets", ".goldr", "assets.json"), want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := templControlsReloadPath(root, tc.path); got != tc.want {
				t.Fatalf("templControlsReloadPath(%q) = %t, want %t", tc.path, got, tc.want)
			}
		})
	}
}

func TestReloadWatcherHandlesRecursiveEditorEventsAndDebounces(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first")
	second := filepath.Join(root, "second")
	if err := os.Mkdir(first, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(second, 0755); err != nil {
		t.Fatal(err)
	}
	paths, err := resolveReloadPaths([]string{first, second})
	if err != nil {
		t.Fatal(err)
	}
	watcher, err := newReloadWatcher(context.Background(), paths, filepath.Join(root, "app"), 40*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reloads := make(chan struct{}, 10)
	errs := make(chan error, 1)
	go func() {
		errs <- watcher.Run(ctx, func(context.Context) error {
			reloads <- struct{}{}
			return nil
		}, io.Discard)
	}()

	writeFile(t, first, "page.html", "one")
	writeFile(t, first, "page.html", "two")
	waitReload(t, reloads)
	assertNoReload(t, reloads)

	nested := filepath.Join(second, "new", "nested")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, nested, "page.json", `{}`)
	waitReload(t, reloads)

	temporary := filepath.Join(first, "page.tmp")
	if err := os.WriteFile(temporary, []byte("atomic"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(temporary, filepath.Join(first, "page.html")); err != nil {
		t.Fatal(err)
	}
	waitReload(t, reloads)

	if err := os.Remove(filepath.Join(first, "page.html")); err != nil {
		t.Fatal(err)
	}
	waitReload(t, reloads)

	cancel()
	if err := <-errs; err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestNewReloadWatcherStopsCanceledStartup(t *testing.T) {
	root := t.TempDir()
	paths, err := resolveReloadPaths([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	watcher, err := newReloadWatcher(ctx, paths, filepath.Join(root, "app"), 30*time.Millisecond)
	if watcher != nil {
		_ = watcher.Close()
		t.Fatal("newReloadWatcher() returned a watcher after startup cancellation")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("newReloadWatcher() error = %v, want context canceled", err)
	}
}

func TestReloadWatcherExactFileSupportsRemovalAndRecreation(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "page.html", "first")
	path := filepath.Join(root, "page.html")
	paths, err := resolveReloadPaths([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	watcher, err := newReloadWatcher(context.Background(), paths, filepath.Join(root, "app"), 30*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reloads := make(chan struct{}, 4)
	go func() {
		_ = watcher.Run(ctx, func(context.Context) error {
			reloads <- struct{}{}
			return nil
		}, io.Discard)
	}()

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	waitReload(t, reloads)
	if err := os.WriteFile(path, []byte("second"), 0644); err != nil {
		t.Fatal(err)
	}
	waitReload(t, reloads)
}

func TestReloadWatcherExactFileWatchesOnlyItsParent(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "unrelated", "nested")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "page.html", "page")
	paths, err := resolveReloadPaths([]string{filepath.Join(root, "page.html")})
	if err != nil {
		t.Fatal(err)
	}
	watcher, err := newReloadWatcher(context.Background(), paths, filepath.Join(root, "app"), 30*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	parent := filepath.Dir(paths[0].path)
	if !watcher.watchedDirs[parent] {
		t.Fatalf("watched directories = %#v, want parent %q", watcher.watchedDirs, parent)
	}
	if len(watcher.watchedDirs) != 1 {
		t.Fatalf("watched directories = %#v, want no descendants of exact-file parent", watcher.watchedDirs)
	}

	later := filepath.Join(parent, "later", "nested")
	if err := os.MkdirAll(later, 0755); err != nil {
		t.Fatal(err)
	}
	matched, err := watcher.handleEvent(context.Background(), fsnotify.Event{
		Name: filepath.Dir(later),
		Op:   fsnotify.Create,
	})
	if err != nil {
		t.Fatalf("handleEvent() error = %v", err)
	}
	if matched {
		t.Fatal("handleEvent() matched an unrelated directory for an exact-file target")
	}
	if len(watcher.watchedDirs) != 1 {
		t.Fatalf("watched directories after create = %#v, want only exact-file parent", watcher.watchedDirs)
	}
}

func TestReloadWatcherDefersTemplOwnedInputsInsideAppRoot(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(root, "content")
	if err := os.Mkdir(content, 0755); err != nil {
		t.Fatal(err)
	}
	for _, directory := range []string{
		filepath.Join(root, "app", "routes"),
		filepath.Join(root, "assets", "dist"),
		filepath.Join(root, "assets", ".goldr"),
	} {
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
	}
	paths, err := resolveReloadPaths([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	watcher, err := newReloadWatcher(context.Background(), paths, paths[0].path, 30*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reloads := make(chan struct{}, 2)
	go func() {
		_ = watcher.Run(ctx, func(context.Context) error {
			reloads <- struct{}{}
			return nil
		}, io.Discard)
	}()

	writeFile(t, root, "route.go", "package app")
	assertNoReload(t, reloads)
	writeFile(t, root, "app/routes/goldr_gen.go", "package routes")
	writeFile(t, root, "assets/dist/app.12345678.css", "body{}")
	writeFile(t, root, "assets/.goldr/assets.json", `{}`)
	assertNoReload(t, reloads)
	writeFile(t, content, "page.html", "page")
	waitReload(t, reloads)
}

func TestReloadWatcherWarnsAndContinuesAfterNotificationFailure(t *testing.T) {
	root := t.TempDir()
	paths, err := resolveReloadPaths([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	watcher, err := newReloadWatcher(context.Background(), paths, filepath.Join(root, "app"), 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var attempts atomic.Int32
	var stderr strings.Builder
	succeeded := make(chan struct{}, 1)
	go func() {
		_ = watcher.Run(ctx, func(context.Context) error {
			if attempts.Add(1) == 1 {
				return errors.New("proxy unavailable")
			}
			succeeded <- struct{}{}
			return nil
		}, &stderr)
	}()

	writeFile(t, root, "first.html", "first")
	deadline := time.Now().Add(5 * time.Second)
	for attempts.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	writeFile(t, root, "second.html", "second")
	waitReload(t, succeeded)
	if !strings.Contains(stderr.String(), "browser reload notification failed") {
		t.Fatalf("stderr = %q, want notification warning", stderr.String())
	}
}

func TestReloadWatcherConfiguredDirectoryRemovalIsFatal(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(root, "content")
	if err := os.Mkdir(content, 0755); err != nil {
		t.Fatal(err)
	}
	paths, err := resolveReloadPaths([]string{content})
	if err != nil {
		t.Fatal(err)
	}
	watcher, err := newReloadWatcher(context.Background(), paths, filepath.Join(root, "app"), 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	errs := make(chan error, 1)
	go func() {
		errs <- watcher.Run(context.Background(), func(context.Context) error { return nil }, io.Discard)
	}()
	if err := os.Rename(content, filepath.Join(root, "moved")); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-errs:
		if err == nil || !strings.Contains(err.Error(), "configured reload directory") {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for fatal root removal")
	}
}

func TestNotifyDevProxyPostsReloadAndClosesResponse(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodPost || r.URL.Path != devProxyReloadPath {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := notifyDevProxy(context.Background(), server.Client(), server.URL, time.Millisecond, time.Second)
	if err != nil {
		t.Fatalf("notifyDevProxy() error = %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("requests = %d, want 1", requests.Load())
	}
}

func TestNotifyDevProxyRetriesStartupFailure(t *testing.T) {
	var attempts atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		if attempts.Add(1) < 3 {
			return nil, errors.New("proxy not ready")
		}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})}

	err := notifyDevProxy(context.Background(), client, "http://127.0.0.1:7331", time.Millisecond, time.Second)
	if err != nil {
		t.Fatalf("notifyDevProxy() error = %v", err)
	}
	if attempts.Load() != 3 {
		t.Fatalf("attempts = %d, want 3", attempts.Load())
	}
}

func TestNotifyDevProxyClosesResponseBody(t *testing.T) {
	closed := &atomic.Bool{}
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Status:     "204 No Content",
			Body:       &trackingBody{closed: closed},
			Header:     make(http.Header),
		}, nil
	})}

	if err := notifyDevProxy(context.Background(), client, "http://127.0.0.1:7331", time.Millisecond, time.Second); err != nil {
		t.Fatalf("notifyDevProxy() error = %v", err)
	}
	if !closed.Load() {
		t.Fatal("response body was not closed")
	}
}

func TestNotifyDevProxyReportsStatusAndCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	err := notifyDevProxy(context.Background(), server.Client(), server.URL, time.Millisecond, 20*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("notifyDevProxy() error = %v, want status 503", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = notifyDevProxy(ctx, server.Client(), server.URL, time.Millisecond, time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("notifyDevProxy() error = %v, want context canceled", err)
	}
}

func waitReload(t *testing.T, reloads <-chan struct{}) {
	t.Helper()
	select {
	case <-reloads:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for browser reload")
	}
}

func assertNoReload(t *testing.T, reloads <-chan struct{}) {
	t.Helper()
	select {
	case <-reloads:
		t.Fatal("unexpected duplicate browser reload")
	case <-time.After(150 * time.Millisecond):
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type trackingBody struct {
	closed *atomic.Bool
}

func (b *trackingBody) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (b *trackingBody) Close() error {
	b.closed.Store(true)
	return nil
}
