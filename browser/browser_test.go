package browser

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFSContainsSSEEventHelper(t *testing.T) {
	file, err := FS().Open(helperPath)
	if err != nil {
		t.Fatalf("FS().Open(%q) error = %v, want nil", helperPath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	body, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("ReadAll() error = %v, want nil", err)
	}
	if !strings.Contains(string(body), `registerExtension("goldr-sse-event"`) {
		t.Fatalf("helper body = %q, want goldr-sse-event extension registration", body)
	}
}

func TestHandlerServesSSEEventHelper(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/goldr-sse-event.js", nil)
	recorder := httptest.NewRecorder()

	Handler().ServeHTTP(recorder, request)

	response := recorder.Result()
	defer closeResponse(t, response)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if got := response.Header.Get("Content-Type"); got != "text/javascript; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/javascript; charset=utf-8", got)
	}
	if got := response.Header.Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", got)
	}
	if got := response.Header.Get("ETag"); got == "" {
		t.Fatalf("ETag = empty, want content-derived validator")
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v, want nil", err)
	}
	if !strings.Contains(string(body), `goldr-sse-event`) {
		t.Fatalf("body = %q, want helper script", body)
	}
}

func TestHandlerRevalidatesSSEEventHelperWithETag(t *testing.T) {
	first := httptest.NewRecorder()
	Handler().ServeHTTP(first, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/goldr-sse-event.js", nil))
	etag := first.Result().Header.Get("ETag")
	if etag == "" {
		t.Fatalf("first ETag = empty, want validator")
	}

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/goldr-sse-event.js", nil)
	request.Header.Set("If-None-Match", etag)
	recorder := httptest.NewRecorder()

	Handler().ServeHTTP(recorder, request)

	response := recorder.Result()
	defer closeResponse(t, response)
	if response.StatusCode != http.StatusNotModified {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusNotModified)
	}
	if got := response.Header.Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", got)
	}
	if got := response.Header.Get("ETag"); got != etag {
		t.Fatalf("ETag = %q, want %q", got, etag)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v, want nil", err)
	}
	if len(body) != 0 {
		t.Fatalf("body = %q, want empty 304 body", body)
	}
}

func TestHandlerRejectsUnknownHelperPath(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/missing.js", nil)
	recorder := httptest.NewRecorder()

	Handler().ServeHTTP(recorder, request)

	response := recorder.Result()
	defer closeResponse(t, response)
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusNotFound)
	}
}

func closeResponse(t *testing.T, response *http.Response) {
	t.Helper()
	if err := response.Body.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
