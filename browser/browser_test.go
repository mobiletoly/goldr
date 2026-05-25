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
	assertFSContains(t, SSEEventHelperPath, `registerExtension("goldr-sse-event"`)
}

func TestFSContainsTemplateInspectorHelper(t *testing.T) {
	assertFSContains(t, TemplateInspectorHelperPath, `data-goldr-template-inspector`)
}

func TestFSContainsTemplateInspectorStackingAndSingleOrdering(t *testing.T) {
	assertFSContains(t, TemplateInspectorHelperPath, `data-goldr-template-stack`)
	assertFSContains(t, TemplateInspectorHelperPath, `data-goldr-template-label-row`)
	assertFSContains(t, TemplateInspectorHelperPath, `sameAnchor`)
	assertFSContains(t, TemplateInspectorHelperPath, `depth: starts.length`)
	assertFSContains(t, TemplateInspectorHelperPath, `function boxOrder(a, b)`)
	assertFSContains(t, TemplateInspectorHelperPath, `return (a.depth - b.depth) || (a.sequence - b.sequence)`)
}

func TestFSContainsTemplateInspectorControls(t *testing.T) {
	assertFSContains(t, TemplateInspectorHelperPath, `data-goldr-template-inspector-control`)
	assertFSContains(t, TemplateInspectorHelperPath, `data-goldr-template-inspector-mode`)
	assertFSContains(t, TemplateInspectorHelperPath, `data-goldr-template-inspector-next`)
	assertFSContains(t, TemplateInspectorHelperPath, `var controlZIndex = "2147483647"`)
	assertFSContains(t, TemplateInspectorHelperPath, `var overlayZIndex = "2147483646"`)
	assertFSContains(t, TemplateInspectorHelperPath, `appendModeButton(root, "All", "all")`)
	assertFSContains(t, TemplateInspectorHelperPath, `appendModeButton(root, "Off", "off")`)
	assertFSContains(t, TemplateInspectorHelperPath, `appendModeButton(root, "Off", "off");
    appendNextButton(root);`)
	assertFSContains(t, TemplateInspectorHelperPath, `";margin-left:10px"`)
	assertFSContains(t, TemplateInspectorHelperPath, `"cursor:not-allowed"`)
	assertFSContains(t, TemplateInspectorHelperPath, `"opacity:.72"`)
	assertFSContains(t, TemplateInspectorHelperPath, `buttonStyle(mode !== "off" && selectedIndex !== null, disabled)`)
	assertFSContains(t, TemplateInspectorHelperPath, `if (selectedIndex === null)`)
	assertFSContains(t, TemplateInspectorHelperPath, `selectedIndex = (selectedIndex + 1) % boxes.length`)
	assertFSContains(t, TemplateInspectorHelperPath, `data-goldr-template-copy`)
	assertFSContains(t, TemplateInspectorHelperPath, `button.textContent = "\u29c9"`)
	assertFSContains(t, TemplateInspectorHelperPath, `navigator.clipboard.writeText(path)`)
	assertFSContains(t, TemplateInspectorHelperPath, `fallbackCopySourcePath(path)`)
	assertFSContains(t, TemplateInspectorHelperPath, `event.stopPropagation()`)
	assertFSContains(t, TemplateInspectorHelperPath, `appendCopyButton(label, sourcePath(meta))`)
	assertFSNotContains(t, TemplateInspectorHelperPath, `appendModeButton(root, "Single"`)
	assertFSNotContains(t, TemplateInspectorHelperPath, `mode === "single"`)
	assertFSNotContains(t, TemplateInspectorHelperPath, `root.setAttribute("aria-hidden", "true")`)
}

func TestFSContainsTemplateInspectorControlsPersistAllAndOffOnly(t *testing.T) {
	assertFSContains(t, TemplateInspectorHelperPath, `var storageKey = "goldr.templateInspector.mode"`)
	assertFSContains(t, TemplateInspectorHelperPath, `window.localStorage.getItem(storageKey) === "off"`)
	assertFSContains(t, TemplateInspectorHelperPath, `window.localStorage.setItem(storageKey, value === "off" ? "off" : "all")`)
	assertFSContains(t, TemplateInspectorHelperPath, `persistMode("all")`)
	assertFSNotContains(t, TemplateInspectorHelperPath, `sessionStorage`)
	assertFSNotContains(t, TemplateInspectorHelperPath, `document.cookie`)
}

func assertFSContains(t *testing.T, name string, want string) {
	t.Helper()

	body := helperBody(t, name)
	if !strings.Contains(body, want) {
		t.Fatalf("helper body = %q, want %q", body, want)
	}
}

func assertFSNotContains(t *testing.T, name string, unwanted string) {
	t.Helper()

	body := helperBody(t, name)
	if strings.Contains(body, unwanted) {
		t.Fatalf("helper body contains %q", unwanted)
	}
}

func helperBody(t *testing.T, name string) string {
	t.Helper()

	file, err := FS().Open(name)
	if err != nil {
		t.Fatalf("FS().Open(%q) error = %v, want nil", name, err)
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
	return string(body)
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

func TestHandlerServesTemplateInspectorHelper(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/goldr-template-inspector.js", nil)
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
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v, want nil", err)
	}
	if !strings.Contains(string(body), `goldr:start`) {
		t.Fatalf("body = %q, want template inspector script", body)
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
