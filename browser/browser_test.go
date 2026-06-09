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
	assertFSContains(t, TemplateInspectorHelperPath, `parents: starts.map(function (start) { return start.meta; })`)
	assertFSContains(t, TemplateInspectorHelperPath, `parents: starts[index].parents`)
	assertFSContains(t, TemplateInspectorHelperPath, `key: (pair.meta.id || "marker") + ":" + pair.sequence`)
	assertFSContains(t, TemplateInspectorHelperPath, `sameAnchor`)
	assertFSContains(t, TemplateInspectorHelperPath, `depth: starts.length`)
	assertFSContains(t, TemplateInspectorHelperPath, `function boxOrder(a, b)`)
	assertFSContains(t, TemplateInspectorHelperPath, `return (a.depth - b.depth) || (a.sequence - b.sequence)`)
}

func TestFSContainsTemplateInspectorLabelsPaintAboveBorders(t *testing.T) {
	assertFSContains(t, TemplateInspectorHelperPath, `"z-index:1"`)
}

func TestFSContainsTemplateInspectorComponentLabels(t *testing.T) {
	assertFSContains(t, TemplateInspectorHelperPath, `component: "#7c3aed"`)
	assertFSContains(t, TemplateInspectorHelperPath, `function decodeMetaValue(value)`)
	assertFSContains(t, TemplateInspectorHelperPath, `.replace(/%2D/gi, "-")`)
	assertFSContains(t, TemplateInspectorHelperPath, `.replace(/%3E/gi, ">")`)
	assertFSContains(t, TemplateInspectorHelperPath, `.replace(/%20/gi, " ")`)
	assertFSContains(t, TemplateInspectorHelperPath, `.replace(/%25/gi, "%")`)
	assertFSContains(t, TemplateInspectorHelperPath, `.replace(/&gt;/g, ">")`)
	assertFSContains(t, TemplateInspectorHelperPath, `.replace(/- -/g, "--")`)
	assertFSContains(t, TemplateInspectorHelperPath, `function renderContext(box)`)
	assertFSContains(t, TemplateInspectorHelperPath, `function badgeParts(box)`)
	assertFSContains(t, TemplateInspectorHelperPath, `return { kind: "component", primary: meta.label, secondary: contextPath }`)
	assertFSContains(t, TemplateInspectorHelperPath, `return { kind: "component", primary: meta.label, secondary: unitText(context) }`)
	assertFSContains(t, TemplateInspectorHelperPath, `return { kind: "component", primary: meta.label, secondary: "" }`)
	assertFSContains(t, TemplateInspectorHelperPath, `if (parts.kind)`)
	assertFSContains(t, TemplateInspectorHelperPath, `appendStyledText(text, parts.kind + " ", "kind")`)
	assertFSContains(t, TemplateInspectorHelperPath, `appendStyledText(text, ": ", "separator")`)
	assertFSContains(t, TemplateInspectorHelperPath, `data-goldr-template-badge-text`)
	assertFSContains(t, TemplateInspectorHelperPath, `data-goldr-template-text`)
	assertFSContains(t, TemplateInspectorHelperPath, `font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace`)
	assertFSContains(t, TemplateInspectorHelperPath, `background:rgba(255,255,255,.16)`)
	assertFSContains(t, TemplateInspectorHelperPath, `overflow:visible`)
	assertFSContains(t, TemplateInspectorHelperPath, `flex-wrap:wrap`)
	assertFSContains(t, TemplateInspectorHelperPath, `padding:0 3px`)
	assertFSContains(t, TemplateInspectorHelperPath, `role === "component-label"`)
	assertFSContains(t, TemplateInspectorHelperPath, `appendDetailRow(details, "component", box.meta.label || "component")`)
	assertFSContains(t, TemplateInspectorHelperPath, `appendDetailRow(details, "unit", box.meta.kind || "template")`)
	assertFSContains(t, TemplateInspectorHelperPath, `appendPathDetailRow(details, "route pattern", box.meta.route, "chain-path")`)
	assertFSContains(t, TemplateInspectorHelperPath, `appendDetailRow(details, "handler", box.meta.handler)`)
	assertFSContains(t, TemplateInspectorHelperPath, `appendPathDetailRow(details, "source context", path)`)
	assertFSContains(t, TemplateInspectorHelperPath, `appendPathDetailRow(details, "source", path)`)
	assertFSContains(t, TemplateInspectorHelperPath, `appendMetaDetailRow(details, "rendered in", context)`)
	assertFSContains(t, TemplateInspectorHelperPath, `function renderChain(box)`)
	assertFSContains(t, TemplateInspectorHelperPath, `chain = chain.filter(function (meta) { return meta.kind !== "component" || meta.label; })`)
	assertFSContains(t, TemplateInspectorHelperPath, `appendRenderChainRow(details, "render chain", chain)`)
	assertFSContains(t, TemplateInspectorHelperPath, `data-goldr-template-chain`)
	assertFSContains(t, TemplateInspectorHelperPath, `data-goldr-template-chain-item`)
	assertFSContains(t, TemplateInspectorHelperPath, `function appendRenderChainItem(root, meta, depth)`)
	assertFSContains(t, TemplateInspectorHelperPath, `appendMetaValue(body, meta, "chain")`)
	assertFSContains(t, TemplateInspectorHelperPath, `var pathRole = context === "chain" ? "chain-path" : "path"`)
	assertFSContains(t, TemplateInspectorHelperPath, `role === "path" || role === "chain-path"`)
	assertFSContains(t, TemplateInspectorHelperPath, `overflow-wrap:anywhere`)
	assertFSContains(t, TemplateInspectorHelperPath, `return meta.source || meta.go || ""`)
	assertFSContains(t, TemplateInspectorHelperPath, `if (meta.kind === "component")`)
	assertFSContains(t, TemplateInspectorHelperPath, `appendStyledText(root, "component ", "kind")`)
	assertFSContains(t, TemplateInspectorHelperPath, `appendStyledText(root, meta.label || "component", "component-label")`)
	assertFSNotContains(t, TemplateInspectorHelperPath, `background:rgba(255,255,255,.12)`)
	assertFSNotContains(t, TemplateInspectorHelperPath, `appendDetailRow(details, "stack", stackText(box))`)
	assertFSNotContains(t, TemplateInspectorHelperPath, `appendMetaDetailRow(details, "render unit", box.meta)`)
	assertFSNotContains(t, TemplateInspectorHelperPath, `appendStyledText(value, " > ", "separator")`)
	assertFSNotContains(t, TemplateInspectorHelperPath, `return "component: " + meta.label`)
}

func TestFSContainsTemplateInspectorDOMHelpers(t *testing.T) {
	assertFSContains(t, TemplateInspectorHelperPath, `function css(parts)`)
	assertFSContains(t, TemplateInspectorHelperPath, `function setStyle(element, parts)`)
	assertFSContains(t, TemplateInspectorHelperPath, `element.style.cssText = css(parts)`)
	assertFSContains(t, TemplateInspectorHelperPath, `function badgeButton(text, title, attribute)`)
	assertFSContains(t, TemplateInspectorHelperPath, `button.setAttribute("aria-label", title)`)
	assertFSContains(t, TemplateInspectorHelperPath, `setStyle(button, badgeButtonStyle())`)
	assertFSContains(t, TemplateInspectorHelperPath, `function styledTextStyle(role)`)
	assertFSContains(t, TemplateInspectorHelperPath, `function controlButton(label, title, active, disabled, extraStyle)`)
	assertFSContains(t, TemplateInspectorHelperPath, `setStyle(button, controlButtonStyle(active, disabled).concat(extraStyle || []))`)
	assertFSContains(t, TemplateInspectorHelperPath, `Array.prototype.slice.call(document.querySelectorAll("[" + overlayAttribute + "]")).forEach`)
	assertFSContains(t, TemplateInspectorHelperPath, `window.requestAnimationFrame(draw)`)
	assertFSNotContains(t, TemplateInspectorHelperPath, `button.style.cssText =`)
	assertFSNotContains(t, TemplateInspectorHelperPath, `root.style.cssText = [`)
}

func TestFSContainsTemplateInspectorExpandableBadges(t *testing.T) {
	assertFSContains(t, TemplateInspectorHelperPath, `var expandedKey = null`)
	assertFSContains(t, TemplateInspectorHelperPath, `data-goldr-template-expand`)
	assertFSContains(t, TemplateInspectorHelperPath, `data-goldr-template-details`)
	assertFSContains(t, TemplateInspectorHelperPath, `badgeButton(expanded ? "v" : ">", title, "data-goldr-template-expand")`)
	assertFSContains(t, TemplateInspectorHelperPath, `button.setAttribute("aria-expanded", expanded ? "true" : "false")`)
	assertFSContains(t, TemplateInspectorHelperPath, `expandedKey = expanded ? null : box.key`)
	assertFSContains(t, TemplateInspectorHelperPath, `expandedKey = null`)
	assertFSContains(t, TemplateInspectorHelperPath, `!boxesToDraw.some(function (box) { return box.key === expandedKey; })`)
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
	assertFSContains(t, TemplateInspectorHelperPath, `["margin-left:10px"]`)
	assertFSContains(t, TemplateInspectorHelperPath, `"cursor:not-allowed"`)
	assertFSContains(t, TemplateInspectorHelperPath, `"opacity:.72"`)
	assertFSContains(t, TemplateInspectorHelperPath, `var active = mode !== "off" && selectedIndex !== null`)
	assertFSContains(t, TemplateInspectorHelperPath, `controlButtonStyle(active, disabled)`)
	assertFSContains(t, TemplateInspectorHelperPath, `if (selectedIndex === null)`)
	assertFSContains(t, TemplateInspectorHelperPath, `selectedIndex = (selectedIndex + 1) % boxes.length`)
	assertFSContains(t, TemplateInspectorHelperPath, `data-goldr-template-copy`)
	assertFSContains(t, TemplateInspectorHelperPath, `badgeButton("\u29c9", title, "data-goldr-template-copy")`)
	assertFSContains(t, TemplateInspectorHelperPath, `navigator.clipboard.writeText(path)`)
	assertFSContains(t, TemplateInspectorHelperPath, `fallbackCopySourcePath(path)`)
	assertFSContains(t, TemplateInspectorHelperPath, `if (fallbackCopySourcePath(path))`)
	assertFSContains(t, TemplateInspectorHelperPath, `copyFailed()`)
	assertFSContains(t, TemplateInspectorHelperPath, `return document.execCommand("copy")`)
	assertFSContains(t, TemplateInspectorHelperPath, `Could not copy source path`)
	assertFSContains(t, TemplateInspectorHelperPath, `event.stopPropagation()`)
	assertFSContains(t, TemplateInspectorHelperPath, `var path = sourcePath(box)`)
	assertFSContains(t, TemplateInspectorHelperPath, `if (path)`)
	assertFSContains(t, TemplateInspectorHelperPath, `appendCopyButton(header, path, box.meta.kind === "component")`)
	assertFSContains(t, TemplateInspectorHelperPath, `Copy source context path to clipboard`)
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
