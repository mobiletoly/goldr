package wiring

import (
	"testing"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
)

func TestGenerateManifestFragmentRouteResponses(t *testing.T) {
	manifest := routing.Manifest{
		Fragments: []routing.ManifestFragment{
			{Name: "table", RoutePrefix: "/users", Unit: completeUnit("users/frag_table.go")},
		},
	}

	tempDir := tempGoldrModule(t)
	writeGeneratedRoutes(t, tempDir, generateOK(t, manifest))
	writeTempFile(t, tempDir, "routes/users/frag_table.go", `package users

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func FragTable(r *http.Request) goldr.FragmentRouteResponse {
	switch r.URL.Query().Get("mode") {
	case "redirect":
		return goldr.Redirect{Location: "/sign-in", Status: http.StatusSeeOther}.
			WithHeader("Cache-Control", "no-store")
	case "text":
		return goldr.Text{Status: http.StatusForbidden, Body: "forbidden"}.
			WithHeader("X-Robots-Tag", "noindex")
	case "error":
		return goldr.RouteError{Err: errors.New("boom")}
	case "cacheable":
		component := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
			_, err := io.WriteString(writer, "<tbody>Cacheable fragment</tbody>")
			return err
		})
		return goldr.NewFragment(component).
			WithHeader("Cache-Control", "public, max-age=60")
	default:
		component := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
			_, err := io.WriteString(writer, "<tbody>Users fragment</tbody>")
			return err
		})
		return goldr.NewFragment(component).
			WithStatus(http.StatusAccepted).
			WithHeader("Hx-Trigger", "fragment-loaded")
	}
}
`)
	writeTempFile(t, tempDir, "routes/users/frag_table.templ", `package users

templ FragTableView() {}
`)
	writeTempFile(t, tempDir, "routes/handler_test.go", `package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr"
)

func TestFragmentRouteResponses(t *testing.T) {
	normal := httptest.NewRecorder()
	Handler().ServeHTTP(normal, httptest.NewRequest(http.MethodGet, "/users/table", nil))
	if normal.Code != http.StatusAccepted {
		t.Fatalf("normal status = %d, want %d", normal.Code, http.StatusAccepted)
	}
	if normal.Body.String() != "<tbody>Users fragment</tbody>" {
		t.Fatalf("normal body = %q", normal.Body.String())
	}
	if got := normal.Header().Get("Hx-Trigger"); got != "fragment-loaded" {
		t.Fatalf("normal Hx-Trigger = %q, want fragment-loaded", got)
	}
	if got := normal.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("normal Cache-Control = %q, want no-store", got)
	}

	head := httptest.NewRecorder()
	Handler().ServeHTTP(head, httptest.NewRequest(http.MethodHead, "/users/table", nil))
	if head.Code != http.StatusAccepted {
		t.Fatalf("HEAD status = %d, want %d", head.Code, http.StatusAccepted)
	}
	if head.Body.Len() != 0 {
		t.Fatalf("HEAD body length = %d, want 0", head.Body.Len())
	}
	if got := head.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("HEAD Cache-Control = %q, want no-store", got)
	}

	cacheable := httptest.NewRecorder()
	Handler().ServeHTTP(cacheable, httptest.NewRequest(http.MethodGet, "/users/table?mode=cacheable", nil))
	if got := cacheable.Header().Get("Cache-Control"); got != "public, max-age=60" {
		t.Fatalf("cacheable Cache-Control = %q, want public, max-age=60", got)
	}

	redirect := httptest.NewRecorder()
	Handler().ServeHTTP(redirect, httptest.NewRequest(http.MethodGet, "/users/table?mode=redirect", nil))
	if redirect.Code != http.StatusSeeOther || redirect.Header().Get("Location") != "/sign-in" {
		t.Fatalf("redirect = (%d, %q), want 303 /sign-in", redirect.Code, redirect.Header().Get("Location"))
	}
	if got := redirect.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("redirect Cache-Control = %q, want no-store", got)
	}

	text := httptest.NewRecorder()
	Handler().ServeHTTP(text, httptest.NewRequest(http.MethodGet, "/users/table?mode=text", nil))
	if text.Code != http.StatusForbidden || text.Body.String() != "forbidden" {
		t.Fatalf("text = (%d, %q), want 403 forbidden", text.Code, text.Body.String())
	}
	if got := text.Header().Get("X-Robots-Tag"); got != "noindex" {
		t.Fatalf("text X-Robots-Tag = %q, want noindex", got)
	}

	inspectedRedirect := httptest.NewRecorder()
	HandlerWithOptions(HandlerOptions{TemplateInspection: goldr.TemplateInspectionComments}).ServeHTTP(inspectedRedirect, httptest.NewRequest(http.MethodGet, "/users/table?mode=redirect", nil))
	if strings.Contains(inspectedRedirect.Body.String(), "goldr:") {
		t.Fatalf("inspected redirect body leaked marker %q", inspectedRedirect.Body.String())
	}

	inspectedText := httptest.NewRecorder()
	HandlerWithOptions(HandlerOptions{TemplateInspection: goldr.TemplateInspectionComments}).ServeHTTP(inspectedText, httptest.NewRequest(http.MethodGet, "/users/table?mode=text", nil))
	if inspectedText.Body.String() != "forbidden" {
		t.Fatalf("inspected text body = %q, want forbidden", inspectedText.Body.String())
	}

	var serverErr error
	errorResponse := httptest.NewRecorder()
	HandlerWithOptions(HandlerOptions{ErrorHandlers: ErrorHandlers{
		RouteError: func(r *http.Request, err error) goldr.RouteResponse {
			serverErr = err
			return goldr.Text{Status: http.StatusInternalServerError, Body: "custom error"}
		},
	}}).ServeHTTP(errorResponse, httptest.NewRequest(http.MethodGet, "/users/table?mode=error", nil))
	if serverErr == nil || !strings.Contains(serverErr.Error(), "boom") {
		t.Fatalf("serverErr = %v, want boom", serverErr)
	}

}
`)

	runGoTest(t, tempDir)
}

func TestGenerateManifestFragmentOnlyRouteImportsInheritedLayout(t *testing.T) {
	manifest := routing.Manifest{
		Layouts: []routing.ManifestLayout{
			{RoutePrefix: "/users", Unit: completeUnit("users/layout.go")},
		},
		Fragments: []routing.ManifestFragment{
			{Name: "table", RoutePrefix: "/users", Unit: completeUnit("users/table/frag_table.go")},
		},
	}

	tempDir := tempGoldrModule(t)
	writeGeneratedRoutes(t, tempDir, generateOK(t, manifest))
	writeTempFile(t, tempDir, "routes/users/layout.go", `package users

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, layout goldr.LayoutContext) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		if err := layout.Child.Render(ctx, writer); err != nil {
			return err
		}
		_, err := io.WriteString(writer, " users layout")
		return err
	})
}
`)
	writeTempFile(t, tempDir, "routes/users/table/frag_table.go", `package table

import (
	"errors"
	"net/http"

	"github.com/mobiletoly/goldr"
)

func FragTable(r *http.Request) goldr.FragmentRouteResponse {
	return goldr.RouteError{Err: errors.New("table failed")}
}
`)
	writeTempFile(t, tempDir, "routes/handler_test.go", `package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func TestFragmentOnlyRouteImportsInheritedLayout(t *testing.T) {
	handler := HandlerWithOptions(HandlerOptions{ErrorHandlers: ErrorHandlers{
		RouteError: func(r *http.Request, err error) goldr.RouteResponse {
			return goldr.NewPage(templ.NopComponent, goldr.PageMetadata{
				Title: "error",
			}).WithStatus(http.StatusInternalServerError)
		},
	}})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/users/table", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if recorder.Body.String() != " users layout" {
		t.Fatalf("body = %q, want inherited users layout", recorder.Body.String())
	}
}
`)

	runGoTest(t, tempDir)
}

func TestGenerateManifestDoesNotLeakRouteHeadersOnRenderError(t *testing.T) {
	manifest := routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/", Unit: completeUnit("page.go")},
		},
		Fragments: []routing.ManifestFragment{
			{Name: "fail", RoutePrefix: "/", Unit: completeUnit("frag_fail.go")},
		},
	}

	tempDir := tempGoldrModule(t)
	writeGeneratedRoutes(t, tempDir, generateOK(t, manifest))
	writeTempFile(t, tempDir, "routes/page.go", `package routes

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	component := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		return errors.New("page render failed")
	})
	return goldr.NewPage(component, goldr.PageMetadata{}).
		WithHeader("Set-Cookie", "page=success")
}
`)
	writeTempFile(t, tempDir, "routes/frag_fail.go", `package routes

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func FragFail(r *http.Request) goldr.FragmentRouteResponse {
	component := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		return errors.New("fragment render failed")
	})
	return goldr.NewFragment(component).
		WithHeader("Hx-Trigger", "fragment-success")
}
`)
	writeTempFile(t, tempDir, "routes/handler_test.go", `package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mobiletoly/goldr"
)

func TestRouteHeadersAreDelayedUntilRenderSuccess(t *testing.T) {
	handler := HandlerWithOptions(HandlerOptions{ErrorHandlers: ErrorHandlers{
		RouteError: func(r *http.Request, err error) goldr.RouteResponse {
			return goldr.Text{Status: http.StatusInternalServerError, Body: "custom error"}
		},
	}})

	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/", nil))
	if page.Code != http.StatusInternalServerError {
		t.Fatalf("page status = %d, want %d", page.Code, http.StatusInternalServerError)
	}
	if got := page.Header().Values("Set-Cookie"); len(got) != 0 {
		t.Fatalf("page Set-Cookie = %#v, want none", got)
	}

	fragment := httptest.NewRecorder()
	handler.ServeHTTP(fragment, httptest.NewRequest(http.MethodGet, "/fail", nil))
	if fragment.Code != http.StatusInternalServerError {
		t.Fatalf("fragment status = %d, want %d", fragment.Code, http.StatusInternalServerError)
	}
	if got := fragment.Header().Get("Hx-Trigger"); got != "" {
		t.Fatalf("fragment Hx-Trigger = %q, want empty", got)
	}
}
`)

	runGoTest(t, tempDir)
}

func TestGenerateManifestFragmentRenderFailures(t *testing.T) {
	manifest := routing.Manifest{
		Fragments: []routing.ManifestFragment{
			{Name: "nil", RoutePrefix: "/", Unit: completeUnit("frag_nil.go")},
			{Name: "fail", RoutePrefix: "/", Unit: completeUnit("frag_fail.go")},
		},
	}
	tempDir := tempGoldrModule(t)
	writeGeneratedRoutes(t, tempDir, generateOK(t, manifest))
	writeTempFile(t, tempDir, "routes/frag_nil.go", `package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func FragNil(r *http.Request) goldr.FragmentRouteResponse {
	return goldr.NewFragment(nil)
}
`)
	writeTempFile(t, tempDir, "routes/frag_fail.go", `package routes

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func FragFail(r *http.Request) goldr.FragmentRouteResponse {
	return goldr.NewFragment(templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		return errors.New("render failed")
	}))
}
`)
	writeTempFile(t, tempDir, "routes/handler_test.go", `package routes

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr"
)

func TestFragmentRenderFailures(t *testing.T) {
	for _, path := range []string{"/nil", "/fail"} {
		recorder := httptest.NewRecorder()
		Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("%s status = %d, want %d", path, recorder.Code, http.StatusInternalServerError)
		}
	}
}

func TestFragmentRenderFailureErrorsReachHandler(t *testing.T) {
	var internalErr error
	handler := HandlerWithOptions(HandlerOptions{ErrorHandlers: ErrorHandlers{
		RouteError: func(r *http.Request, err error) goldr.RouteResponse {
			internalErr = err
			return goldr.Text{Status: http.StatusInternalServerError}
		},
	}})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/nil", nil))
	if !errors.Is(internalErr, goldr.ErrNilComponent) {
		t.Fatalf("nil component error = %v, want ErrNilComponent", internalErr)
	}

	internalErr = nil
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/fail", nil))
	if internalErr == nil || !strings.Contains(internalErr.Error(), "render failed") {
		t.Fatalf("render error = %v, want render failed", internalErr)
	}
}
`)

	runGoTest(t, tempDir)
}
