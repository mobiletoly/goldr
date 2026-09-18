package wiring

import (
	"strings"
	"testing"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
)

func TestGenerateManifestAdditionalPageSourceContract(t *testing.T) {
	manifest := routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/", Unit: completeUnit("page.go")},
			{Route: "/users/{id}", Params: []string{"id"}, Unit: completeUnit("users/by_id/page.go")},
			{Route: "/owned-missing", Unit: completeUnit("owned_missing/page.go")},
			{Route: "/owned-error", Unit: completeUnit("owned_error/page.go")},
		},
		Fragments: []routing.ManifestFragment{
			{Name: "panel", RoutePrefix: "/", Unit: completeUnit("frag_panel.go")},
		},
		Actions: []routing.ManifestAction{
			{Method: "POST", Route: "/save", GoFile: "actions.go", Function: "PostSave", Suffix: "Save", Segment: "save"},
		},
		Layouts: []routing.ManifestLayout{
			{RoutePrefix: "/", Unit: completeUnit("layout.go")},
		},
	}

	tempDir := tempGoldrModule(t)
	writeGeneratedRoutes(t, tempDir, generateOK(t, manifest))
	writeTempFile(t, tempDir, "routes/view.go", `package routes

import (
	"context"
	"errors"
	"io"

	"github.com/a-h/templ"
)

func textComponent(text string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, text)
		return err
	})
}

func failingComponent() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		if _, err := io.WriteString(writer, "partial"); err != nil {
			return err
		}
		return errors.New("render failed")
	})
}
`)
	writeTempFile(t, tempDir, "routes/layout.go", `package routes

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, layout goldr.LayoutContext) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		if _, err := fmt.Fprintf(writer, "<root title=%q>", layout.Metadata.Title); err != nil {
			return err
		}
		if err := layout.Child.Render(ctx, writer); err != nil {
			return err
		}
		_, err := io.WriteString(writer, "</root>")
		return err
	})
}
`)
	writeTempFile(t, tempDir, "routes/page.go", `package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	return goldr.NewPage(textComponent("home"), goldr.PageMetadata{Title: "Home"})
}
`)
	writeTempFile(t, tempDir, "routes/users/by_id/page.go", `package by_id

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	return goldr.NewPage(templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "user "+r.PathValue("id"))
		return err
	}), goldr.PageMetadata{Title: "User"})
}
`)
	writeTempFile(t, tempDir, "routes/owned_missing/page.go", `package owned_missing

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	return goldr.Text{Status: http.StatusNotFound, Body: "owned missing"}
}
`)
	writeTempFile(t, tempDir, "routes/owned_error/page.go", `package owned_error

import (
	"errors"
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	return goldr.RouteError{Err: errors.New("owned error")}
}
`)
	writeTempFile(t, tempDir, "routes/frag_panel.go", `package routes

import (
	"net/http"
	"github.com/mobiletoly/goldr"
)

func FragPanel(r *http.Request) goldr.FragmentRouteResponse { return goldr.Text{Body: "panel"} }
`)
	writeTempFile(t, tempDir, "routes/actions.go", `package routes

import (
	"net/http"
	"github.com/mobiletoly/goldr"
)

func PostSave(r *http.Request) goldr.RouteResponse { return goldr.Text{Body: "saved"} }
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

func request(t *testing.T, handler http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, path, nil))
	return recorder
}

func TestAdditionalPageSourceOwnsOnlyRouterMisses(t *testing.T) {
	additionalPageSourceCalls := 0
	notFoundCalls := 0
	errorCalls := 0
	handler := HandlerWithOptions(HandlerOptions{
		AdditionalPageSource: func(r *http.Request) (goldr.PageRouteResponse, bool) {
			additionalPageSourceCalls++
			if r.URL.Path == "/content" {
				return goldr.NewPage(textComponent("content"), goldr.PageMetadata{Title: "Content"}), true
			}
			return goldr.Text{Status: http.StatusTeapot, Body: "ignored"}, false
		},
		ErrorHandlers: ErrorHandlers{
			RouteNotFound: func(r *http.Request) goldr.RouteResponse {
				notFoundCalls++
				return goldr.Text{Status: http.StatusNotFound, Body: "custom missing"}
			},
			RouteError: func(r *http.Request, err error) goldr.RouteResponse {
				errorCalls++
				return goldr.Text{Status: http.StatusInternalServerError, Body: "custom error: " + err.Error()}
			},
		},
	})

	content := request(t, handler, http.MethodGet, "/content?mode=full")
	if content.Code != http.StatusOK || content.Body.String() != "<root title=\"Content\">content</root>" {
		t.Fatalf("content = (%d, %q)", content.Code, content.Body.String())
	}
	if additionalPageSourceCalls != 1 {
		t.Fatalf("content additional page source calls = %d, want 1", additionalPageSourceCalls)
	}

	head := request(t, handler, http.MethodHead, "/content")
	if head.Code != http.StatusOK || head.Body.Len() != 0 {
		t.Fatalf("HEAD content = (%d, %q), want 200 with empty body", head.Code, head.Body.String())
	}

	for _, test := range []struct {
		method string
		path   string
		status int
		body   string
	}{
		{http.MethodGet, "/", http.StatusOK, "<root title=\"Home\">home</root>"},
		{http.MethodGet, "/users/42", http.StatusOK, "<root title=\"User\">user 42</root>"},
		{http.MethodGet, "/owned-missing", http.StatusNotFound, "owned missing"},
		{http.MethodGet, "/owned-error", http.StatusInternalServerError, "custom error: owned error"},
		{http.MethodGet, "/panel", http.StatusOK, "panel"},
		{http.MethodPost, "/save", http.StatusOK, "saved"},
		{http.MethodPost, "/", http.StatusMethodNotAllowed, "method not allowed\n"},
		{http.MethodGet, "/users/", http.StatusNotFound, "custom missing"},
	} {
		before := additionalPageSourceCalls
		response := request(t, handler, test.method, test.path)
		if response.Code != test.status || response.Body.String() != test.body {
			t.Fatalf("%s %s = (%d, %q), want (%d, %q)", test.method, test.path, response.Code, response.Body.String(), test.status, test.body)
		}
		if additionalPageSourceCalls != before {
			t.Fatalf("%s %s called additional page source", test.method, test.path)
		}
	}
	if errorCalls != 1 {
		t.Fatalf("route error calls = %d, want 1", errorCalls)
	}
	if notFoundCalls != 1 {
		t.Fatalf("not found calls = %d, want 1", notFoundCalls)
	}
}

func TestAdditionalPageSourceDeclineAndTerminalResponses(t *testing.T) {
	notFoundCalls := 0
	handler := HandlerWithOptions(HandlerOptions{
		AdditionalPageSource: func(r *http.Request) (goldr.PageRouteResponse, bool) {
			if r.URL.Path == "/handled-missing" {
				return goldr.Text{Status: http.StatusNotFound, Body: "additional page source missing"}, true
			}
			return goldr.Text{Status: http.StatusTeapot, Body: "ignored"}, false
		},
		ErrorHandlers: ErrorHandlers{RouteNotFound: func(r *http.Request) goldr.RouteResponse {
			notFoundCalls++
			return goldr.Text{Status: http.StatusNotFound, Body: "final missing"}
		}},
	})

	declined := request(t, handler, http.MethodGet, "/declined")
	if declined.Code != http.StatusNotFound || declined.Body.String() != "final missing" {
		t.Fatalf("declined = (%d, %q)", declined.Code, declined.Body.String())
	}
	handled := request(t, handler, http.MethodGet, "/handled-missing")
	if handled.Code != http.StatusNotFound || handled.Body.String() != "additional page source missing" {
		t.Fatalf("handled = (%d, %q)", handled.Code, handled.Body.String())
	}
	if notFoundCalls != 1 {
		t.Fatalf("not found calls = %d, want 1", notFoundCalls)
	}

	defaultMissing := request(t, Handler(), http.MethodGet, "/declined")
	if defaultMissing.Code != http.StatusNotFound {
		t.Fatalf("default missing status = %d, want 404", defaultMissing.Code)
	}
}

func TestAdditionalPageSourceChainAndTerminalError(t *testing.T) {
	firstCalls := 0
	secondCalls := 0
	first := func(r *http.Request) (goldr.PageRouteResponse, bool) {
		firstCalls++
		if r.URL.Path == "/chain-error" {
			return goldr.RouteError{Err: errors.New("first failed")}, true
		}
		return nil, false
	}
	second := func(r *http.Request) (goldr.PageRouteResponse, bool) {
		secondCalls++
		if r.URL.Path == "/chain" {
			return goldr.Text{Body: "second"}, true
		}
		return nil, false
	}
	handler := HandlerWithOptions(HandlerOptions{
		AdditionalPageSource: func(r *http.Request) (goldr.PageRouteResponse, bool) {
			if response, handled := first(r); handled {
				return response, true
			}
			return second(r)
		},
		ErrorHandlers: ErrorHandlers{RouteError: func(r *http.Request, err error) goldr.RouteResponse {
			return goldr.Text{Status: http.StatusInternalServerError, Body: err.Error()}
		}},
	})

	if response := request(t, handler, http.MethodGet, "/chain"); response.Body.String() != "second" {
		t.Fatalf("chain body = %q", response.Body.String())
	}
	if response := request(t, handler, http.MethodGet, "/all-decline"); response.Code != http.StatusNotFound {
		t.Fatalf("all-decline status = %d, want 404", response.Code)
	}
	secondBefore := secondCalls
	if response := request(t, handler, http.MethodGet, "/chain-error"); response.Body.String() != "first failed" {
		t.Fatalf("chain error body = %q", response.Body.String())
	}
	if secondCalls != secondBefore {
		t.Fatalf("second source called after terminal error")
	}
	if firstCalls != 3 || secondCalls != 2 {
		t.Fatalf("chain calls = (%d, %d), want (3, 2)", firstCalls, secondCalls)
	}
}

func TestAdditionalPageSourceErrorsUseRouteErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		response goldr.PageRouteResponse
		wantErr  string
	}{
		{"route error", goldr.RouteError{Err: errors.New("load failed")}, "load failed"},
		{"nil response", nil, goldr.ErrInvalidRouteResponse.Error()},
		{"render failure", goldr.NewPage(failingComponent(), goldr.PageMetadata{}), "render failed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errorCalls := 0
			handler := HandlerWithOptions(HandlerOptions{
				AdditionalPageSource: func(r *http.Request) (goldr.PageRouteResponse, bool) {
					return test.response, true
				},
				ErrorHandlers: ErrorHandlers{RouteError: func(r *http.Request, err error) goldr.RouteResponse {
					errorCalls++
					return goldr.Text{Status: http.StatusInternalServerError, Body: "handled: " + err.Error()}
				}},
			})
			response := request(t, handler, http.MethodGet, "/broken")
			if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), test.wantErr) {
				t.Fatalf("response = (%d, %q), want error containing %q", response.Code, response.Body.String(), test.wantErr)
			}
			if strings.Contains(response.Body.String(), "partial") {
				t.Fatalf("response leaked partial render: %q", response.Body.String())
			}
			if errorCalls != 1 {
				t.Fatalf("error calls = %d, want 1", errorCalls)
			}
		})
	}

	defaultError := HandlerWithOptions(HandlerOptions{AdditionalPageSource: func(r *http.Request) (goldr.PageRouteResponse, bool) {
		return goldr.RouteError{Err: errors.New("load failed")}, true
	}})
	if response := request(t, defaultError, http.MethodGet, "/broken"); response.Code != http.StatusInternalServerError {
		t.Fatalf("default error status = %d, want 500", response.Code)
	}

	additionalPageSourceCalls := 0
	hookCalls := 0
	hookFailure := HandlerWithOptions(HandlerOptions{
		AdditionalPageSource: func(r *http.Request) (goldr.PageRouteResponse, bool) {
			additionalPageSourceCalls++
			return goldr.RouteError{Err: errors.New("load failed")}, true
		},
		ErrorHandlers: ErrorHandlers{RouteError: func(r *http.Request, err error) goldr.RouteResponse {
			hookCalls++
			return goldr.RouteError{Err: errors.New("hook failed")}
		}},
	})
	response := request(t, hookFailure, http.MethodGet, "/broken")
	if response.Code != http.StatusInternalServerError || response.Body.String() != "internal server error\n" {
		t.Fatalf("hook failure = (%d, %q)", response.Code, response.Body.String())
	}
	if additionalPageSourceCalls != 1 || hookCalls != 1 {
		t.Fatalf("hook failure calls = (%d, %d), want (1, 1)", additionalPageSourceCalls, hookCalls)
	}
}
`)

	runGoTest(t, tempDir)
}

func TestGenerateManifestAdditionalPageSourceCompilesWithoutLayoutsAndWithActionsOnly(t *testing.T) {
	emptyDir := tempGoldrModule(t)
	writeGeneratedRoutes(t, emptyDir, generateOK(t, routing.Manifest{}))
	writeTempFile(t, emptyDir, "routes/handler_test.go", `package routes

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func TestZeroEndpointHandler(t *testing.T) {
	missing := httptest.NewRecorder()
	Handler().ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if missing.Code != http.StatusNotFound { t.Fatalf("missing status = %d", missing.Code) }

	handler := HandlerWithOptions(HandlerOptions{AdditionalPageSource: func(r *http.Request) (goldr.PageRouteResponse, bool) {
		if r.PathValue("id") != "" {
			return goldr.Text{Status: http.StatusInternalServerError, Body: "unexpected path value"}, true
		}
		return goldr.NewPage(templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
			_, err := io.WriteString(writer, "content")
			return err
		}), goldr.PageMetadata{}), true
	}})
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/content", nil))
	if page.Code != http.StatusOK || page.Body.String() != "content" { t.Fatalf("page = (%d, %q)", page.Code, page.Body.String()) }
	trailing := httptest.NewRecorder()
	handler.ServeHTTP(trailing, httptest.NewRequest(http.MethodGet, "/content/", nil))
	if trailing.Code != http.StatusOK || trailing.Body.String() != "content" { t.Fatalf("trailing page = (%d, %q)", trailing.Code, trailing.Body.String()) }
}
`)
	runGoTest(t, emptyDir)

	noLayout := routing.Manifest{
		Pages: []routing.ManifestPage{{Route: "/", Unit: completeUnit("page.go")}},
	}
	noLayoutDir := tempGoldrModule(t)
	writeGeneratedRoutes(t, noLayoutDir, generateOK(t, noLayout))
	writeTempFile(t, noLayoutDir, "routes/page.go", `package routes

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	return goldr.NewPage(templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "home")
		return err
	}), goldr.PageMetadata{})
}
`)
	writeTempFile(t, noLayoutDir, "routes/handler_test.go", `package routes

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func TestAdditionalPageSourceWithoutLayout(t *testing.T) {
	handler := HandlerWithOptions(HandlerOptions{AdditionalPageSource: func(r *http.Request) (goldr.PageRouteResponse, bool) {
		return goldr.NewPage(templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
			_, err := io.WriteString(writer, "content")
			return err
		}), goldr.PageMetadata{}), true
	}})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/content", nil))
	if recorder.Body.String() != "content" {
		t.Fatalf("body = %q, want content", recorder.Body.String())
	}
}
`)
	runGoTest(t, noLayoutDir)

	actionOnly := routing.Manifest{
		Actions: []routing.ManifestAction{{Method: "POST", Route: "/create", GoFile: "actions.go", Function: "PostCreate", Suffix: "Create", Segment: "create"}},
	}
	actionDir := tempGoldrModule(t)
	source := generateOK(t, actionOnly)
	for _, want := range []string{
		"AdditionalPageSource func(*http.Request) (goldr.PageRouteResponse, bool)",
		"goldrRouteMiss(options, additionalPageHandlers, w, r, routePath)",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("action-only source missing %q:\n%s", want, source)
		}
	}
	writeGeneratedRoutes(t, actionDir, source)
	writeTempFile(t, actionDir, "routes/actions.go", `package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func PostCreate(r *http.Request) goldr.RouteResponse {
	return goldr.Text{Body: "created"}
}
`)
	writeTempFile(t, actionDir, "routes/handler_test.go", `package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mobiletoly/goldr"
)

func TestActionOnlyAdditionalPageSource(t *testing.T) {
	handler := HandlerWithOptions(HandlerOptions{AdditionalPageSource: func(r *http.Request) (goldr.PageRouteResponse, bool) {
		return goldr.Text{Body: "content"}, true
	}})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/content", nil))
	if recorder.Body.String() != "content" {
		t.Fatalf("body = %q, want content", recorder.Body.String())
	}
}
`)
	runGoTest(t, actionDir)
}

func TestAdditionalPageSourceUsesStaticLayoutAndMiddlewareAncestry(t *testing.T) {
	manifest := routing.Manifest{
		Layouts: []routing.ManifestLayout{
			{RoutePrefix: "/", Unit: completeUnit("layout.go")},
			{RoutePrefix: "/app", Unit: completeUnit("app/layout.go")},
			{RoutePrefix: "/app/manager", Unit: completeUnit("app/manager/layout.go")},
			{RoutePrefix: "/application", Unit: completeUnit("application/layout.go")},
			{RoutePrefix: "/layout-only", Unit: completeUnit("layout_only/layout.go")},
			{RoutePrefix: "/users/{id}", Params: []string{"id"}, Unit: completeUnit("users/by_id/layout.go")},
			{RoutePrefix: "/users/{id}/help", Params: []string{"id"}, Unit: completeUnit("users/by_id/help/layout.go")},
		},
		Middlewares: []routing.ManifestMiddleware{
			{RoutePrefix: "/", GoFile: "middleware.go"},
			{RoutePrefix: "/app", GoFile: "app/middleware.go"},
			{RoutePrefix: "/app/manager", GoFile: "app/manager/middleware.go"},
			{RoutePrefix: "/application", GoFile: "application/middleware.go"},
			{RoutePrefix: "/middleware-only", GoFile: "middleware_only/middleware.go"},
			{RoutePrefix: "/users/{id}", Params: []string{"id"}, GoFile: "users/by_id/middleware.go"},
			{RoutePrefix: "/users/{id}/help", Params: []string{"id"}, GoFile: "users/by_id/help/middleware.go"},
		},
	}

	tempDir := tempGoldrModule(t)
	writeGeneratedRoutes(t, tempDir, generateOK(t, manifest))
	writeTempFile(t, tempDir, "routectx/routectx.go", `package routectx

import (
	"context"
	"net/http"
	"strings"
)

type key struct{}

func Append(r *http.Request, value string) *http.Request {
	values := append(Values(r), value)
	return r.WithContext(context.WithValue(r.Context(), key{}, values))
}

func Order(r *http.Request) string { return strings.Join(Values(r), ">") }

func Values(r *http.Request) []string {
	values, _ := r.Context().Value(key{}).([]string)
	return append([]string(nil), values...)
}
`)
	writeTempFile(t, tempDir, "routes/view.go", `package routes

import (
	"context"
	"io"

	"github.com/a-h/templ"
)

func text(value string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, value)
		return err
	})
}
`)
	writeTempFile(t, tempDir, "routes/layout.go", `package routes

import (
	"context"
	"io"
	"net/http"

	"example.com/app/routectx"
	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, layout goldr.LayoutContext) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, _ = io.WriteString(writer, "<root:"+routectx.Order(r)+">")
		if err := layout.Child.Render(ctx, writer); err != nil { return err }
		_, err := io.WriteString(writer, "</root>")
		return err
	})
}
`)
	writeTempFile(t, tempDir, "routes/middleware.go", `package routes

import (
	"net/http"
	"example.com/app/routectx"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Middleware", "root")
		next.ServeHTTP(w, routectx.Append(r, "root"))
	})
}
`)
	writeTempFile(t, tempDir, "routes/app/layout.go", `package app

import (
	"context"
	"io"
	"net/http"
	"example.com/app/routectx"
	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, layout goldr.LayoutContext) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, _ = io.WriteString(writer, "<app:"+routectx.Order(r)+">")
		if err := layout.Child.Render(ctx, writer); err != nil { return err }
		_, err := io.WriteString(writer, "</app>")
		return err
	})
}
`)
	writeTempFile(t, tempDir, "routes/app/middleware.go", `package app

import (
	"net/http"
	"example.com/app/routectx"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Middleware", "app")
		if r.URL.Query().Get("stop") == "1" { http.Error(w, "stopped", http.StatusUnauthorized); return }
		next.ServeHTTP(w, routectx.Append(r, "app"))
	})
}
`)
	writeTempFile(t, tempDir, "routes/app/manager/layout.go", `package manager

import (
	"context"
	"io"
	"net/http"
	"example.com/app/routectx"
	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, layout goldr.LayoutContext) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, _ = io.WriteString(writer, "<manager:"+routectx.Order(r)+">")
		if err := layout.Child.Render(ctx, writer); err != nil { return err }
		_, err := io.WriteString(writer, "</manager>")
		return err
	})
}
`)
	writeTempFile(t, tempDir, "routes/app/manager/middleware.go", `package manager

import (
	"net/http"
	"example.com/app/routectx"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Middleware", "manager")
		r = routectx.Append(r, "manager")
		if r.URL.Query().Get("rewrite") == "1" {
			copy := r.Clone(r.Context())
			urlCopy := *r.URL
			urlCopy.Path = "/application/changed"
			urlCopy.RawPath = ""
			copy.URL = &urlCopy
			r = copy
		}
		next.ServeHTTP(w, r)
	})
}
`)
	writeTempFile(t, tempDir, "routes/application/layout.go", `package application

import (
	"net/http"
	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, layout goldr.LayoutContext) templ.Component { return layout.Child }
`)
	writeTempFile(t, tempDir, "routes/application/middleware.go", `package application

import "net/http"

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Middleware", "application")
		next.ServeHTTP(w, r)
	})
}
`)
	writeTempFile(t, tempDir, "routes/layout_only/layout.go", `package layout_only

import (
	"context"
	"io"
	"net/http"
	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, layout goldr.LayoutContext) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, _ = io.WriteString(writer, "<layout-only>")
		if err := layout.Child.Render(ctx, writer); err != nil { return err }
		_, err := io.WriteString(writer, "</layout-only>"); return err
	})
}
`)
	writeTempFile(t, tempDir, "routes/middleware_only/middleware.go", `package middleware_only

import (
	"net/http"
	"example.com/app/routectx"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Header().Add("X-Middleware", "middleware-only"); next.ServeHTTP(w, routectx.Append(r, "middleware-only")) })
}
`)
	writeTempFile(t, tempDir, "routes/handler_test.go", `package routes

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"example.com/app/routectx"
	"github.com/mobiletoly/goldr"
)

func TestStaticAncestry(t *testing.T) {
	calls := 0
	handler := HandlerWithOptions(HandlerOptions{
		AdditionalPageSource: func(r *http.Request) (goldr.PageRouteResponse, bool) {
			calls++
			if r.PathValue("id") != "" { return goldr.Text{Status: http.StatusInternalServerError, Body: "bound path value"}, true }
			if r.URL.Path == "/decline" || r.URL.Path == "/app/decline" { return nil, false }
			if r.URL.Path == "/app/error" { return goldr.RouteError{Err: errors.New("source failed")}, true }
			return goldr.NewPage(text("page:"+routectx.Order(r)+":"+r.URL.Path), goldr.PageMetadata{}), true
		},
		ErrorHandlers: ErrorHandlers{RouteNotFound: func(r *http.Request) goldr.RouteResponse {
			return goldr.Text{Status: http.StatusNotFound, Body: "missing:"+routectx.Order(r)}
		}, RouteError: func(r *http.Request, err error) goldr.RouteResponse {
			return goldr.Text{Status: http.StatusInternalServerError, Body: "error:"+routectx.Order(r)}
		}},
	})

	tests := []struct { path, body string; middleware []string }{
		{"/app/manager/about", "<root:root>app>manager>\x3capp:root>app>manager>\x3cmanager:root>app>manager>page:root>app>manager:/app/manager/about</manager></app></root>", []string{"root", "app", "manager"}},
		{"/application", "<root:root>page:root:/application</root>", []string{"root", "application"}},
		{"/users/42/help", "<root:root>page:root:/users/42/help</root>", []string{"root"}},
		{"/layout-only/page", "<root:root><layout-only>page:root:/layout-only/page</layout-only></root>", []string{"root"}},
		{"/middleware-only/page", "<root:root>middleware-only>page:root>middleware-only:/middleware-only/page</root>", []string{"root", "middleware-only"}},
		{"/app/manager/about?rewrite=1", "<root:root>app>manager>\x3capp:root>app>manager>\x3cmanager:root>app>manager>page:root>app>manager:/application/changed</manager></app></root>", []string{"root", "app", "manager"}},
	}
	for _, test := range tests {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
		if recorder.Body.String() != test.body { t.Fatalf("%s body = %q, want %q", test.path, recorder.Body.String(), test.body) }
		if got := recorder.Header().Values("X-Middleware"); !reflect.DeepEqual(got, test.middleware) { t.Fatalf("%s middleware = %#v, want %#v", test.path, got, test.middleware) }
	}

	for _, test := range []struct { path string; middleware []string }{
		{"/app//x", []string{"root", "app"}},
		{"/app/../x", []string{"root", "app"}},
		{"/app/%2Fx", []string{"root", "app"}},
		{"/app%2Fx", []string{"root"}},
		{"/%61pp/x", []string{"root"}},
	} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
		if got := recorder.Header().Values("X-Middleware"); !reflect.DeepEqual(got, test.middleware) { t.Fatalf("escaped %s middleware = %#v, want %#v", test.path, got, test.middleware) }
	}

	stoppedCalls := calls
	stopped := httptest.NewRecorder()
	handler.ServeHTTP(stopped, httptest.NewRequest(http.MethodGet, "/app/private?stop=1", nil))
	if stopped.Code != http.StatusUnauthorized || calls != stoppedCalls { t.Fatalf("short circuit = (%d, %d calls), want 401 and %d calls", stopped.Code, calls, stoppedCalls) }

	declined := httptest.NewRecorder()
	handler.ServeHTTP(declined, httptest.NewRequest(http.MethodGet, "/app/decline", nil))
	if declined.Code != http.StatusNotFound || declined.Body.String() != "missing:root>app" { t.Fatalf("decline = (%d, %q)", declined.Code, declined.Body.String()) }

	failed := httptest.NewRecorder()
	handler.ServeHTTP(failed, httptest.NewRequest(http.MethodGet, "/app/error", nil))
	if failed.Code != http.StatusInternalServerError || failed.Body.String() != "error:root>app" || !reflect.DeepEqual(failed.Header().Values("X-Middleware"), []string{"root", "app"}) { t.Fatalf("error = (%d, %q, %#v)", failed.Code, failed.Body.String(), failed.Header()) }

	defaultDecline := httptest.NewRecorder()
	HandlerWithOptions(HandlerOptions{AdditionalPageSource: func(r *http.Request) (goldr.PageRouteResponse, bool) { return nil, false }}).ServeHTTP(defaultDecline, httptest.NewRequest(http.MethodGet, "/app/decline", nil))
	if defaultDecline.Code != http.StatusNotFound || !reflect.DeepEqual(defaultDecline.Header().Values("X-Middleware"), []string{"root", "app"}) { t.Fatalf("default decline = (%d, %#v)", defaultDecline.Code, defaultDecline.Header()) }

	nilSource := httptest.NewRecorder()
	Handler().ServeHTTP(nilSource, httptest.NewRequest(http.MethodGet, "/app/missing", nil))
	if nilSource.Code != http.StatusNotFound || len(nilSource.Header().Values("X-Middleware")) != 0 { t.Fatalf("nil source = (%d, %#v)", nilSource.Code, nilSource.Header()) }
}
`)

	runGoTest(t, tempDir)
}

func TestAdditionalPageSourcePreservesPageAndErrorSemantics(t *testing.T) {
	manifest := routing.Manifest{
		Layouts: []routing.ManifestLayout{
			{RoutePrefix: "/", Unit: completeUnit("layout.go")},
			{RoutePrefix: "/app", Unit: completeUnit("app/layout.go")},
			{RoutePrefix: "/app/manager", Unit: completeUnit("app/manager/layout.go")},
		},
	}

	tempDir := tempGoldrModule(t)
	writeGeneratedRoutes(t, tempDir, generateOK(t, manifest))
	writeTempFile(t, tempDir, "layoutstate/state.go", `package layoutstate

import "github.com/mobiletoly/goldr"

var Key = goldr.NewLayoutKey[string]("additional-page-test")
`)
	writeTempFile(t, tempDir, "routes/view.go", `package routes

import (
	"context"
	"errors"
	"io"

	"github.com/a-h/templ"
)

func text(value string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, value)
		return err
	})
}

func broken() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, _ = io.WriteString(writer, "page-partial")
		return errors.New("page failed")
	})
}
`)
	writeTempFile(t, tempDir, "routes/layout.go", `package routes

import (
	"context"
	"errors"
	"io"
	"net/http"

	"example.com/app/layoutstate"
	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, layout goldr.LayoutContext) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		value, _ := goldr.LayoutValue(layout, layoutstate.Key)
		_, _ = io.WriteString(writer, "<root:"+layout.Metadata.Title+":"+value+">")
		if layout.Metadata.Title == "Error" && r.URL.Query().Get("rootfail") == "1" {
			return errors.New("root failed")
		}
		if err := layout.Child.Render(ctx, writer); err != nil { return err }
		_, err := io.WriteString(writer, "</root>")
		return err
	})
}
`)
	writeTempFile(t, tempDir, "routes/app/layout.go", `package app

import (
	"context"
	"io"
	"net/http"

	"example.com/app/layoutstate"
	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, layout goldr.LayoutContext) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		value, _ := goldr.LayoutValue(layout, layoutstate.Key)
		_, _ = io.WriteString(writer, "<app:"+layout.Metadata.Title+":"+value+">")
		if err := layout.Child.Render(ctx, writer); err != nil { return err }
		_, err := io.WriteString(writer, "</app>")
		return err
	})
}
`)
	writeTempFile(t, tempDir, "routes/app/manager/layout.go", `package manager

import (
	"context"
	"errors"
	"io"
	"net/http"

	"example.com/app/layoutstate"
	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, layout goldr.LayoutContext) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		value, _ := goldr.LayoutValue(layout, layoutstate.Key)
		_, _ = io.WriteString(writer, "<manager:"+layout.Metadata.Title+":"+value+">")
		if r.URL.Query().Get("nestedfail") == "1" { return errors.New("nested failed") }
		if err := layout.Child.Render(ctx, writer); err != nil { return err }
		_, err := io.WriteString(writer, "</manager>")
		return err
	})
}
`)
	writeTempFile(t, tempDir, "routes/handler_test.go", `package routes

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/app/layoutstate"
	"github.com/mobiletoly/goldr"
)

func additionalResponse(r *http.Request) (goldr.PageRouteResponse, bool) {
	switch r.URL.Path {
	case "/app/manager/redirect":
		return goldr.Redirect{Location: "/target", Status: http.StatusTemporaryRedirect}, true
	case "/app/manager/text":
		return goldr.Text{Status: http.StatusAccepted, Body: "plain"}.WithHeader("X-Page", "text"), true
	case "/app/manager/error":
		return goldr.RouteError{Err: errors.New("source failed")}, true
	case "/app/manager/page-fail":
		return goldr.NewPage(broken(), goldr.PageMetadata{Title: "Broken"}), true
	}
	page := goldr.NewPage(text("page"), goldr.PageMetadata{Title: "Meta", Description: "Description"}).
		WithStatus(http.StatusCreated).
		WithHeader("X-Page", "yes")
	return goldr.WithLayoutValue(page, layoutstate.Key, "data"), true
}

func testHandler(errorCalls *int) http.Handler {
	return HandlerWithOptions(HandlerOptions{
		AdditionalPageSource: additionalResponse,
		ErrorHandlers: ErrorHandlers{RouteError: func(r *http.Request, err error) goldr.RouteResponse {
			*errorCalls++
			return goldr.NewPage(text("error:"+err.Error()), goldr.PageMetadata{Title: "Error"}).
				WithStatus(http.StatusInternalServerError).
				WithHeader("X-Error", "yes")
		}},
	})
}

func TestPageMetadataDataStatusHeadersAndHead(t *testing.T) {
	errorCalls := 0
	handler := testHandler(&errorCalls)
	want := "<root:Meta:data><app:Meta:data><manager:Meta:data>page</manager></app></root>"
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(method, "/app/manager/page", nil))
		if recorder.Code != http.StatusCreated || recorder.Header().Get("X-Page") != "yes" { t.Fatalf("%s status/header = (%d, %q)", method, recorder.Code, recorder.Header().Get("X-Page")) }
		if method == http.MethodGet && recorder.Body.String() != want { t.Fatalf("GET body = %q, want %q", recorder.Body.String(), want) }
		if method == http.MethodHead && recorder.Body.Len() != 0 { t.Fatalf("HEAD body = %q", recorder.Body.String()) }
	}
	if errorCalls != 0 { t.Fatalf("error calls = %d", errorCalls) }
}

func TestRedirectAndTextBypassLayouts(t *testing.T) {
	errorCalls := 0
	handler := testHandler(&errorCalls)
	redirect := httptest.NewRecorder()
	handler.ServeHTTP(redirect, httptest.NewRequest(http.MethodGet, "/app/manager/redirect", nil))
	if redirect.Code != http.StatusTemporaryRedirect || redirect.Header().Get("Location") != "/target" || redirect.Body.Len() != 0 { t.Fatalf("redirect = (%d, %q, %q)", redirect.Code, redirect.Header().Get("Location"), redirect.Body.String()) }
	plain := httptest.NewRecorder()
	handler.ServeHTTP(plain, httptest.NewRequest(http.MethodGet, "/app/manager/text", nil))
	if plain.Code != http.StatusAccepted || plain.Header().Get("X-Page") != "text" || plain.Body.String() != "plain" { t.Fatalf("text = (%d, %q, %q)", plain.Code, plain.Header().Get("X-Page"), plain.Body.String()) }
}

func TestAdditionalErrorsUseOnlyLiveRoot(t *testing.T) {
	for _, path := range []string{"/app/manager/error", "/app/manager/page-fail", "/app/manager/page?nestedfail=1"} {
		errorCalls := 0
		recorder := httptest.NewRecorder()
		testHandler(&errorCalls).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusInternalServerError || recorder.Header().Get("X-Error") != "yes" { t.Fatalf("%s status/header = (%d, %q)", path, recorder.Code, recorder.Header().Get("X-Error")) }
		if !strings.HasPrefix(recorder.Body.String(), "<root:Error:>") || strings.Contains(recorder.Body.String(), "<app:") || strings.Contains(recorder.Body.String(), "<manager:") || strings.Contains(recorder.Body.String(), "partial") { t.Fatalf("%s body = %q", path, recorder.Body.String()) }
		if errorCalls != 1 { t.Fatalf("%s error calls = %d", path, errorCalls) }
	}
}

func TestLiveRootErrorFailureIsTerminal(t *testing.T) {
	errorCalls := 0
	recorder := httptest.NewRecorder()
	testHandler(&errorCalls).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/app/manager/error?rootfail=1", nil))
	if recorder.Code != http.StatusInternalServerError || recorder.Body.String() != "internal server error\n" { t.Fatalf("response = (%d, %q)", recorder.Code, recorder.Body.String()) }
	if errorCalls != 1 { t.Fatalf("error calls = %d", errorCalls) }
}

func TestAdditionalPageTemplateInspectionMarksLayoutsOnly(t *testing.T) {
	recorder := httptest.NewRecorder()
	HandlerWithOptions(HandlerOptions{AdditionalPageSource: additionalResponse, TemplateInspection: goldr.TemplateInspectionComments}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/app/manager/page", nil))
	body := recorder.Body.String()
	if strings.Count(body, "kind=layout") != 3 { t.Fatalf("layout marker count = %d:\n%s", strings.Count(body, "kind=layout"), body) }
	for _, forbidden := range []string{"kind=page", "kind=route", "kind=content", "handler="} {
		if strings.Contains(body, forbidden) { t.Fatalf("body contains %q:\n%s", forbidden, body) }
	}
}
`)

	runGoTest(t, tempDir)
}

func TestAdditionalPageSourceExcludesMountedLayoutsAndUsesLiveOwnerMiddleware(t *testing.T) {
	manifest := routing.Manifest{
		Layouts: []routing.ManifestLayout{
			{RoutePrefix: "/", Unit: completeUnit("layout.go")},
			{RoutePrefix: "/", Unit: completeUnit("../mounts/shared/layout.go")},
			{RoutePrefix: "/admin", Unit: completeUnit("admin/layout.go")},
			{RoutePrefix: "/admin", Unit: completeUnit("../mounts/shared/admin/layout.go")},
		},
		Middlewares: []routing.ManifestMiddleware{
			{RoutePrefix: "/admin", GoFile: "admin/middleware.go"},
		},
	}

	tempDir := tempGoldrModule(t)
	writeGeneratedRoutes(t, tempDir, generateOK(t, manifest))
	writeTempFile(t, tempDir, "routes/view.go", `package routes

import (
	"context"
	"io"
	"github.com/a-h/templ"
)

func text(value string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error { _, err := io.WriteString(writer, value); return err })
}
`)
	writeTempFile(t, tempDir, "routes/layout.go", `package routes

import (
	"context"
	"io"
	"net/http"
	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, layout goldr.LayoutContext) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, _ = io.WriteString(writer, "<live-root>")
		if err := layout.Child.Render(ctx, writer); err != nil { return err }
		_, err := io.WriteString(writer, "</live-root>"); return err
	})
}
`)
	writeTempFile(t, tempDir, "routes/admin/layout.go", `package admin

import (
	"context"
	"io"
	"net/http"
	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, layout goldr.LayoutContext) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, _ = io.WriteString(writer, "<live-admin>")
		if err := layout.Child.Render(ctx, writer); err != nil { return err }
		_, err := io.WriteString(writer, "</live-admin>"); return err
	})
}
`)
	writeTempFile(t, tempDir, "routes/admin/middleware.go", `package admin

import "net/http"

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Header().Add("X-Live-Admin", "yes"); next.ServeHTTP(w, r) })
}
`)
	writeTempFile(t, tempDir, "mounts/shared/layout.go", `package shared

import (
	"context"
	"io"
	"net/http"
	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, layout goldr.LayoutContext) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error { _, _ = io.WriteString(writer, "<mounted-root>"); if err := layout.Child.Render(ctx, writer); err != nil { return err }; _, err := io.WriteString(writer, "</mounted-root>"); return err })
}
`)
	writeTempFile(t, tempDir, "mounts/shared/admin/layout.go", `package admin

import (
	"context"
	"io"
	"net/http"
	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, layout goldr.LayoutContext) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error { _, _ = io.WriteString(writer, "<mounted-admin>"); if err := layout.Child.Render(ctx, writer); err != nil { return err }; _, err := io.WriteString(writer, "</mounted-admin>"); return err })
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

func TestMountedLayoutsAreExcluded(t *testing.T) {
	handler := HandlerWithOptions(HandlerOptions{
		AdditionalPageSource: func(r *http.Request) (goldr.PageRouteResponse, bool) {
			if strings.HasSuffix(r.URL.Path, "/error") { return goldr.RouteError{Err: errors.New("failed")}, true }
			return goldr.NewPage(text("page"), goldr.PageMetadata{}), true
		},
		ErrorHandlers: ErrorHandlers{RouteError: func(r *http.Request, err error) goldr.RouteResponse {
			return goldr.NewPage(text("error"), goldr.PageMetadata{}).WithStatus(http.StatusInternalServerError)
		}},
	})
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/admin/page", nil))
	if page.Body.String() != "<live-root><live-admin>page</live-admin></live-root>" || page.Header().Get("X-Live-Admin") != "yes" { t.Fatalf("page = (%q, %q)", page.Body.String(), page.Header().Get("X-Live-Admin")) }
	errorPage := httptest.NewRecorder()
	handler.ServeHTTP(errorPage, httptest.NewRequest(http.MethodGet, "/admin/error", nil))
	if errorPage.Body.String() != "<live-root>error</live-root>" || errorPage.Header().Get("X-Live-Admin") != "yes" { t.Fatalf("error = (%q, %q)", errorPage.Body.String(), errorPage.Header().Get("X-Live-Admin")) }
	if strings.Contains(page.Body.String()+errorPage.Body.String(), "mounted-") { t.Fatalf("mounted layout rendered: %q %q", page.Body.String(), errorPage.Body.String()) }
}
`)
	runGoTest(t, tempDir)

	noRoot := routing.Manifest{Layouts: []routing.ManifestLayout{{RoutePrefix: "/app", Unit: completeUnit("app/layout.go")}}}
	noRootDir := tempGoldrModule(t)
	writeGeneratedRoutes(t, noRootDir, generateOK(t, noRoot))
	writeTempFile(t, noRootDir, "routes/app/layout.go", `package app

import (
	"net/http"
	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, layout goldr.LayoutContext) templ.Component { return layout.Child }
`)
	writeTempFile(t, noRootDir, "routes/handler_test.go", `package routes

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func TestErrorPageRendersDirectlyWithoutLiveRoot(t *testing.T) {
	handler := HandlerWithOptions(HandlerOptions{
		AdditionalPageSource: func(r *http.Request) (goldr.PageRouteResponse, bool) { return goldr.RouteError{Err: errors.New("failed")}, true },
		ErrorHandlers: ErrorHandlers{RouteError: func(r *http.Request, err error) goldr.RouteResponse { return goldr.NewPage(templ.Raw("error"), goldr.PageMetadata{}).WithStatus(http.StatusInternalServerError) }},
	})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/app/error", nil))
	if recorder.Code != http.StatusInternalServerError || recorder.Body.String() != "error" { t.Fatalf("response = (%d, %q)", recorder.Code, recorder.Body.String()) }
}
`)
	runGoTest(t, noRootDir)
}

func TestAdditionalPageSourceImportAliasCollisionsCompile(t *testing.T) {
	tempDir := tempGoldrModule(t)
	writeTempFile(t, tempDir, "routes/route.go", `package routes

import (
	goldrroute_a_b "example.com/app/dependency"
	"github.com/mobiletoly/goldr"
)

var Route = goldr.RouteDef{Page: goldrroute_a_b.Page}
`)
	writeTempFile(t, tempDir, "dependency/page.go", `package dependency

import (
	"net/http"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.PageRouteResponse { return goldr.Text{Body: "owned"} }
`)
	for _, file := range []string{"routes/a/b/layout.go", "routes/a_b/layout.go"} {
		packageName := "b"
		if strings.Contains(file, "a_b") {
			packageName = "a_b"
		}
		writeTempFile(t, tempDir, file, `package `+packageName+`

import (
	"net/http"
	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, layout goldr.LayoutContext) templ.Component { return layout.Child }
`)
	}
	for _, item := range []struct{ file, packageName string }{
		{"routes/c/d/middleware.go", "d"},
		{"routes/c_d/middleware.go", "c_d"},
	} {
		writeTempFile(t, tempDir, item.file, `package `+item.packageName+`

import "net/http"

func Middleware(next http.Handler) http.Handler { return next }
`)
	}
	manifest := routing.Manifest{
		Root: tempDir + "/routes",
		Routes: []routing.ManifestRouteDeclaration{{
			Route:  "/owned",
			GoFile: "route.go",
			Kind:   "local",
			Imports: []routing.RouteImportDeclaration{{
				Name: "goldrroute_a_b", Path: "example.com/app/dependency", Explicit: true,
			}},
			Page: &routing.RouteHandlerDeclaration{Handler: "goldrroute_a_b.Page"},
		}},
		Layouts: []routing.ManifestLayout{
			{RoutePrefix: "/a", Unit: completeUnit("a/b/layout.go")},
			{RoutePrefix: "/a-b", Unit: completeUnit("a_b/layout.go")},
		},
		Middlewares: []routing.ManifestMiddleware{
			{RoutePrefix: "/c", GoFile: "c/d/middleware.go"},
			{RoutePrefix: "/c-d", GoFile: "c_d/middleware.go"},
		},
	}
	source, err := GenerateManifest(manifest, GenerateOptions{PackageName: "routes", RouteRootImportPath: "example.com/app/routes"})
	if err != nil {
		t.Fatalf("GenerateManifest() error = %v", err)
	}
	generated := string(source)
	for _, want := range []string{
		`goldrroute_a_b "example.com/app/dependency"`,
		`goldrroute_a_b_2 "example.com/app/routes/a/b"`,
		`goldrroute_a_b_3 "example.com/app/routes/a_b"`,
		`goldrroute_c_d "example.com/app/routes/c/d"`,
		`goldrroute_c_d_2 "example.com/app/routes/c_d"`,
	} {
		if !strings.Contains(generated, want) {
			t.Fatalf("generated source missing %q:\n%s", want, generated)
		}
	}
	writeGeneratedRoutes(t, tempDir, generated)

	writeTempFile(t, tempDir, "routes/handler_test.go", `package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNilSourceCompilesWithCollisionImports(t *testing.T) {
	recorder := httptest.NewRecorder()
	Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if recorder.Code != http.StatusNotFound { t.Fatalf("status = %d", recorder.Code) }
}
`)
	runGoTest(t, tempDir)
}

func TestAdditionalPageSourceReusedImportAliasDoesNotShadowMiddlewareParameter(t *testing.T) {
	tempDir := tempGoldrModule(t)
	writeTempFile(t, tempDir, "routes/route.go", `package routes

import (
	next "example.com/app/routes/child"
	"github.com/mobiletoly/goldr"
)

var Route = goldr.RouteDef{Page: next.Page}
`)
	writeTempFile(t, tempDir, "routes/child/page.go", `package child

import (
	"net/http"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.PageRouteResponse { return goldr.Text{Body: "owned"} }
`)
	writeTempFile(t, tempDir, "routes/child/middleware.go", `package child

import "net/http"

func Middleware(next http.Handler) http.Handler { return next }
`)

	manifest := routing.Manifest{
		Root: tempDir + "/routes",
		Routes: []routing.ManifestRouteDeclaration{{
			Route:  "/owned",
			GoFile: "route.go",
			Kind:   "local",
			Imports: []routing.RouteImportDeclaration{{
				Name: "next", Path: "example.com/app/routes/child", Explicit: true,
			}},
			Page: &routing.RouteHandlerDeclaration{Handler: "next.Page"},
		}},
		Middlewares: []routing.ManifestMiddleware{{RoutePrefix: "/child", GoFile: "child/middleware.go"}},
	}
	source, err := GenerateManifest(manifest, GenerateOptions{PackageName: "routes", RouteRootImportPath: "example.com/app/routes"})
	if err != nil {
		t.Fatalf("GenerateManifest() error = %v", err)
	}
	writeGeneratedRoutes(t, tempDir, string(source))
	runGoTest(t, tempDir)
}
