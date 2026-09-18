package wiring

import (
	"strings"
	"testing"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
)

func TestGenerateManifestFallbackContract(t *testing.T) {
	manifest := routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/", Unit: completeUnit("page.go")},
			{Route: "/users/{id}", Params: []string{"id"}, Unit: completeUnit("users/by_id/page.go")},
			{Route: "/owned-missing", Unit: completeUnit("owned_missing/page.go")},
			{Route: "/owned-error", Unit: completeUnit("owned_error/page.go")},
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

func TestFallbackOwnsOnlyRouterMisses(t *testing.T) {
	fallbackCalls := 0
	notFoundCalls := 0
	errorCalls := 0
	handler := HandlerWithOptions(HandlerOptions{
		Fallback: func(r *http.Request) (goldr.PageRouteResponse, bool) {
			fallbackCalls++
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
	if fallbackCalls != 1 {
		t.Fatalf("content fallback calls = %d, want 1", fallbackCalls)
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
		{http.MethodPost, "/", http.StatusMethodNotAllowed, "method not allowed\n"},
		{http.MethodGet, "/users/", http.StatusNotFound, "custom missing"},
	} {
		before := fallbackCalls
		response := request(t, handler, test.method, test.path)
		if response.Code != test.status || response.Body.String() != test.body {
			t.Fatalf("%s %s = (%d, %q), want (%d, %q)", test.method, test.path, response.Code, response.Body.String(), test.status, test.body)
		}
		if fallbackCalls != before {
			t.Fatalf("%s %s called fallback", test.method, test.path)
		}
	}
	if errorCalls != 1 {
		t.Fatalf("route error calls = %d, want 1", errorCalls)
	}
	if notFoundCalls != 1 {
		t.Fatalf("not found calls = %d, want 1", notFoundCalls)
	}
}

func TestFallbackDeclineAndTerminalResponses(t *testing.T) {
	notFoundCalls := 0
	handler := HandlerWithOptions(HandlerOptions{
		Fallback: func(r *http.Request) (goldr.PageRouteResponse, bool) {
			if r.URL.Path == "/handled-missing" {
				return goldr.Text{Status: http.StatusNotFound, Body: "fallback missing"}, true
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
	if handled.Code != http.StatusNotFound || handled.Body.String() != "fallback missing" {
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

func TestFallbackChainAndTerminalError(t *testing.T) {
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
		Fallback: func(r *http.Request) (goldr.PageRouteResponse, bool) {
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

func TestFallbackErrorsUseRouteErrorHandling(t *testing.T) {
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
				Fallback: func(r *http.Request) (goldr.PageRouteResponse, bool) {
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

	defaultError := HandlerWithOptions(HandlerOptions{Fallback: func(r *http.Request) (goldr.PageRouteResponse, bool) {
		return goldr.RouteError{Err: errors.New("load failed")}, true
	}})
	if response := request(t, defaultError, http.MethodGet, "/broken"); response.Code != http.StatusInternalServerError {
		t.Fatalf("default error status = %d, want 500", response.Code)
	}

	fallbackCalls := 0
	hookCalls := 0
	hookFailure := HandlerWithOptions(HandlerOptions{
		Fallback: func(r *http.Request) (goldr.PageRouteResponse, bool) {
			fallbackCalls++
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
	if fallbackCalls != 1 || hookCalls != 1 {
		t.Fatalf("hook failure calls = (%d, %d), want (1, 1)", fallbackCalls, hookCalls)
	}
}
`)

	runGoTest(t, tempDir)
}

func TestGenerateManifestFallbackCompilesWithoutLayoutsAndWithActionsOnly(t *testing.T) {
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

func TestFallbackWithoutLayout(t *testing.T) {
	handler := HandlerWithOptions(HandlerOptions{Fallback: func(r *http.Request) (goldr.PageRouteResponse, bool) {
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
		"Fallback           func(*http.Request) (goldr.PageRouteResponse, bool)",
		"goldrRouteMiss(options, w, r)",
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

func TestActionOnlyFallback(t *testing.T) {
	handler := HandlerWithOptions(HandlerOptions{Fallback: func(r *http.Request) (goldr.PageRouteResponse, bool) {
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
