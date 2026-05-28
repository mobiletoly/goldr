package wiring

import (
	"testing"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
)

func TestGenerateManifestPassesZeroMetadataToLayouts(t *testing.T) {
	manifest := routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/", Unit: completeUnit("page.go")},
		},
		Layouts: []routing.ManifestLayout{
			{RoutePrefix: "/", Unit: completeUnit("layout.go")},
		},
	}

	tempDir := tempGoldrModule(t)
	writeGeneratedRoutes(t, tempDir, generateOK(t, manifest))
	writeTempFile(t, tempDir, "routes/page.go", `package routes

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	component := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "root")
		return err
	})
	return goldr.NewPage(component, goldr.PageMetadata{})
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
		if layout.Metadata != (goldr.PageMetadata{}) {
			_, err := io.WriteString(writer, "nonzero metadata")
			return err
		}
		if err := layout.Child.Render(ctx, writer); err != nil {
			return err
		}
		_, err := io.WriteString(writer, " zero metadata")
		return err
	})
}
`)
	writeTempFile(t, tempDir, "routes/handler_test.go", `package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultMetadata(t *testing.T) {
	recorder := httptest.NewRecorder()
	Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Body.String() != "root zero metadata" {
		t.Fatalf("body = %q, want zero metadata", recorder.Body.String())
	}
}
`)

	runGoTest(t, tempDir)
}

func TestGenerateManifestCustomErrorHandlers(t *testing.T) {
	manifest := routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/", Unit: completeUnit("page.go")},
		},
		Fragments: []routing.ManifestFragment{
			{Name: "nil", RoutePrefix: "/", Unit: completeUnit("frag_nil.go")},
		},
	}

	tempDir := tempGoldrModule(t)
	writeGeneratedRoutes(t, tempDir, generateOK(t, manifest))
	writeTempFile(t, tempDir, "routes/page.go", `package routes

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	component := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "root")
		return err
	})
	return goldr.NewPage(component, goldr.PageMetadata{})
}
`)
	writeTempFile(t, tempDir, "routes/frag_nil.go", `package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func FragNil(r *http.Request) goldr.FragmentRouteResponse {
	return goldr.NewFragment(nil)
}
`)
	writeTempFile(t, tempDir, "routes/handler_test.go", `package routes

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mobiletoly/goldr"
)

func TestCustomErrorHandlers(t *testing.T) {
	handler := HandlerWithOptions(HandlerOptions{ErrorHandlers: ErrorHandlers{
		NotFound: func(r *http.Request) goldr.RouteResponse {
			return goldr.Text{Status: http.StatusNotFound, Body: "custom missing"}
		},
		MethodNotAllowed: func(r *http.Request) goldr.RouteResponse {
			return goldr.Text{Status: http.StatusMethodNotAllowed, Body: "custom method"}
		},
		InternalServerError: func(r *http.Request, err error) goldr.RouteResponse {
			if !errors.Is(err, goldr.ErrNilComponent) {
				t.Fatalf("internal error = %v, want ErrNilComponent", err)
			}
			return goldr.Text{Status: http.StatusInternalServerError, Body: "custom boom"}
		},
	}})

	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if missing.Code != http.StatusNotFound || missing.Body.String() != "custom missing" {
		t.Fatalf("missing = (%d, %q), want custom 404", missing.Code, missing.Body.String())
	}

	underscoreFragment := httptest.NewRecorder()
	handler.ServeHTTP(underscoreFragment, httptest.NewRequest(http.MethodGet, "/frag_nil", nil))
	if underscoreFragment.Code != http.StatusNotFound || underscoreFragment.Body.String() != "custom missing" {
		t.Fatalf("underscore fragment = (%d, %q), want custom 404", underscoreFragment.Code, underscoreFragment.Body.String())
	}

	method := httptest.NewRecorder()
	handler.ServeHTTP(method, httptest.NewRequest(http.MethodPost, "/", nil))
	if method.Code != http.StatusMethodNotAllowed || method.Body.String() != "custom method" {
		t.Fatalf("method = (%d, %q), want custom 405", method.Code, method.Body.String())
	}
	if method.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("method Allow = %q, want GET, HEAD", method.Header().Get("Allow"))
	}

	internal := httptest.NewRecorder()
	handler.ServeHTTP(internal, httptest.NewRequest(http.MethodGet, "/nil", nil))
	if internal.Code != http.StatusInternalServerError || internal.Body.String() != "custom boom" {
		t.Fatalf("internal = (%d, %q), want custom 500", internal.Code, internal.Body.String())
	}
}

func TestNilErrorHandlersFallBackIndependently(t *testing.T) {
	handler := HandlerWithOptions(HandlerOptions{ErrorHandlers: ErrorHandlers{
		NotFound: func(r *http.Request) goldr.RouteResponse {
			return goldr.Text{Status: http.StatusNotFound, Body: "custom missing"}
		},
	}})

	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if missing.Body.String() != "custom missing" {
		t.Fatalf("missing body = %q, want custom missing", missing.Body.String())
	}

	method := httptest.NewRecorder()
	handler.ServeHTTP(method, httptest.NewRequest(http.MethodPost, "/", nil))
	if method.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method status = %d, want %d", method.Code, http.StatusMethodNotAllowed)
	}

	internal := httptest.NewRecorder()
	handler.ServeHTTP(internal, httptest.NewRequest(http.MethodGet, "/nil", nil))
	if internal.Code != http.StatusInternalServerError {
		t.Fatalf("internal status = %d, want %d", internal.Code, http.StatusInternalServerError)
	}
}
`)

	runGoTest(t, tempDir)
}

func TestGenerateManifestCustomErrorResponsesUseRouteLayoutContext(t *testing.T) {
	manifest := routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/", Unit: completeUnit("page.go")},
			{Route: "/users", Unit: completeUnit("users/page.go")},
		},
		Layouts: []routing.ManifestLayout{
			{RoutePrefix: "/", Unit: completeUnit("layout.go")},
			{RoutePrefix: "/users", Unit: completeUnit("users/layout.go")},
		},
		Fragments: []routing.ManifestFragment{
			{Name: "table", RoutePrefix: "/users", Unit: completeUnit("users/frag_table.go")},
		},
		Actions: []routing.ManifestAction{
			{Method: "POST", Route: "/users/save", GoFile: "users/action.go", Function: "PostSave"},
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

func textComponent(text string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, text)
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
	return goldr.NewPage(textComponent("home"), goldr.PageMetadata{Title: "home"})
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
	writeTempFile(t, tempDir, "routes/users/page.go", `package users

import (
	"errors"
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	return goldr.ServerError{Err: errors.New("users failed")}
}
`)
	writeTempFile(t, tempDir, "routes/users/layout.go", `package users

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
		if _, err := fmt.Fprintf(writer, "<users title=%q>", layout.Metadata.Title); err != nil {
			return err
		}
		if err := layout.Child.Render(ctx, writer); err != nil {
			return err
		}
		_, err := io.WriteString(writer, "</users>")
		return err
	})
}
`)
	writeTempFile(t, tempDir, "routes/users/frag_table.go", `package users

import (
	"errors"
	"net/http"

	"github.com/mobiletoly/goldr"
)

func FragTable(r *http.Request) goldr.FragmentRouteResponse {
	return goldr.ServerError{Err: errors.New("table failed")}
}
`)
	writeTempFile(t, tempDir, "routes/users/action.go", `package users

import (
	"errors"
	"net/http"

	"github.com/mobiletoly/goldr"
)

func PostSave(r *http.Request) goldr.RouteResponse {
	return goldr.ServerError{Err: errors.New("save failed")}
}
`)
	writeTempFile(t, tempDir, "routes/handler_test.go", `package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/hx"
)

func testHandler() http.Handler {
	return HandlerWithOptions(HandlerOptions{ErrorHandlers: ErrorHandlers{
		NotFound: func(r *http.Request) goldr.RouteResponse {
			return goldr.NewPage(textComponent("missing"), goldr.PageMetadata{
				Title: "missing",
			}).WithStatus(http.StatusNotFound)
		},
		InternalServerError: func(r *http.Request, err error) goldr.RouteResponse {
			if hx.IsRequest(r) {
				return goldr.NewFragment(textComponent("toast")).
					WithHeader(hx.HeaderRetarget, "#toast")
			}
			return goldr.NewPage(textComponent("error"), goldr.PageMetadata{
				Title: "error",
			}).WithStatus(http.StatusInternalServerError)
		},
	}})
}

func TestErrorResponseLayoutContext(t *testing.T) {
	handler := testHandler()

	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if missing.Code != http.StatusNotFound || missing.Body.String() != "<root title=\"missing\">missing</root>" {
		t.Fatalf("missing = (%d, %q)", missing.Code, missing.Body.String())
	}

	users := httptest.NewRecorder()
	handler.ServeHTTP(users, httptest.NewRequest(http.MethodGet, "/users", nil))
	if users.Code != http.StatusInternalServerError || users.Body.String() != "<root title=\"error\"><users title=\"error\">error</users></root>" {
		t.Fatalf("users = (%d, %q)", users.Code, users.Body.String())
	}

	action := httptest.NewRecorder()
	handler.ServeHTTP(action, httptest.NewRequest(http.MethodPost, "/users/save", nil))
	if action.Code != http.StatusInternalServerError || action.Body.String() != "<root title=\"error\"><users title=\"error\">error</users></root>" {
		t.Fatalf("action = (%d, %q)", action.Code, action.Body.String())
	}

	fragment := httptest.NewRecorder()
	handler.ServeHTTP(fragment, httptest.NewRequest(http.MethodGet, "/users/table", nil))
	if fragment.Code != http.StatusInternalServerError || fragment.Body.String() != "<root title=\"error\"><users title=\"error\">error</users></root>" {
		t.Fatalf("fragment = (%d, %q)", fragment.Code, fragment.Body.String())
	}

	hxFragment := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/users/table", nil)
	request.Header.Set(hx.HeaderRequest, "true")
	handler.ServeHTTP(hxFragment, request)
	if hxFragment.Code != http.StatusOK || hxFragment.Body.String() != "toast" {
		t.Fatalf("hx fragment = (%d, %q)", hxFragment.Code, hxFragment.Body.String())
	}
	if hxFragment.Header().Get(hx.HeaderRetarget) != "#toast" {
		t.Fatalf("HX-Retarget = %q", hxFragment.Header().Get(hx.HeaderRetarget))
	}
}
`)

	runGoTest(t, tempDir)
}

func TestGenerateManifestPageResponses(t *testing.T) {
	manifest := routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/redirect", Unit: completeUnit("redirect/page.go")},
			{Route: "/forbidden", Unit: completeUnit("forbidden/page.go")},
			{Route: "/plain", Unit: completeUnit("plain/page.go")},
			{Route: "/error", Unit: completeUnit("errorpage/page.go")},
			{Route: "/badredirect", Unit: completeUnit("badredirect/page.go")},
		},
		Layouts: []routing.ManifestLayout{
			{RoutePrefix: "/", Unit: completeUnit("layout.go")},
		},
	}

	tempDir := tempGoldrModule(t)
	writeGeneratedRoutes(t, tempDir, generateOK(t, manifest))
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
		if _, err := io.WriteString(writer, "<layout title=\""+layout.Metadata.Title+"\">"); err != nil {
			return err
		}
		if err := layout.Child.Render(ctx, writer); err != nil {
			return err
		}
		_, err := io.WriteString(writer, "</layout>")
		return err
	})
}
`)
	writeTempFile(t, tempDir, "routes/redirect/page.go", `package redirect

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	return goldr.Redirect{Location: "/sign-in", Status: http.StatusSeeOther}.WithHeader("Cache-Control", "no-store")
}
`)
	writeTempFile(t, tempDir, "routes/forbidden/page.go", `package forbidden

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	component := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "<p>forbidden</p>")
		return err
	})
	return goldr.NewPage(component, goldr.PageMetadata{Title: "Forbidden"}).
		WithStatus(http.StatusForbidden).
		WithHeader("X-Robots-Tag", "noindex")
}
`)
	writeTempFile(t, tempDir, "routes/plain/page.go", `package plain

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	return goldr.Text{Status: http.StatusForbidden, Body: "plain forbidden"}.WithHeader("Cache-Control", "private")
}
`)
	writeTempFile(t, tempDir, "routes/errorpage/page.go", `package errorpage

import (
	"errors"
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	return goldr.ServerError{Err: errors.New("load failed")}
}
`)
	writeTempFile(t, tempDir, "routes/badredirect/page.go", `package badredirect

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	return goldr.Redirect{Location: "", Status: http.StatusSeeOther}
}
`)
	writeTempFile(t, tempDir, "routes/handler_test.go", `package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr"
)

func TestPageResponses(t *testing.T) {
	handler := HandlerWithOptions(HandlerOptions{ErrorHandlers: ErrorHandlers{
		InternalServerError: func(r *http.Request, err error) goldr.RouteResponse {
			return goldr.Text{Status: http.StatusInternalServerError, Body: "internal: " + err.Error()}
		},
	}})

	redirect := httptest.NewRecorder()
	handler.ServeHTTP(redirect, httptest.NewRequest(http.MethodGet, "/redirect", nil))
	if redirect.Code != http.StatusSeeOther {
		t.Fatalf("redirect status = %d, want %d", redirect.Code, http.StatusSeeOther)
	}
	if redirect.Header().Get("Location") != "/sign-in" {
		t.Fatalf("redirect Location = %q", redirect.Header().Get("Location"))
	}
	if redirect.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("redirect Cache-Control = %q", redirect.Header().Get("Cache-Control"))
	}
	if strings.Contains(redirect.Body.String(), "layout") {
		t.Fatalf("redirect body = %q, must not render layout", redirect.Body.String())
	}

	forbidden := httptest.NewRecorder()
	handler.ServeHTTP(forbidden, httptest.NewRequest(http.MethodGet, "/forbidden", nil))
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("forbidden status = %d, want %d", forbidden.Code, http.StatusForbidden)
	}
	if forbidden.Body.String() != "<layout title=\"Forbidden\"><p>forbidden</p></layout>" {
		t.Fatalf("forbidden body = %q", forbidden.Body.String())
	}
	if forbidden.Header().Get("X-Robots-Tag") != "noindex" {
		t.Fatalf("forbidden X-Robots-Tag = %q", forbidden.Header().Get("X-Robots-Tag"))
	}

	plain := httptest.NewRecorder()
	handler.ServeHTTP(plain, httptest.NewRequest(http.MethodGet, "/plain", nil))
	if plain.Code != http.StatusForbidden {
		t.Fatalf("plain status = %d, want %d", plain.Code, http.StatusForbidden)
	}
	if plain.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatalf("plain Content-Type = %q", plain.Header().Get("Content-Type"))
	}
	if plain.Header().Get("Cache-Control") != "private" {
		t.Fatalf("plain Cache-Control = %q", plain.Header().Get("Cache-Control"))
	}
	if plain.Body.String() != "plain forbidden" {
		t.Fatalf("plain body = %q", plain.Body.String())
	}

	head := httptest.NewRecorder()
	handler.ServeHTTP(head, httptest.NewRequest(http.MethodHead, "/plain", nil))
	if head.Code != http.StatusForbidden || head.Body.Len() != 0 {
		t.Fatalf("HEAD plain = (%d, %q), want 403 with empty body", head.Code, head.Body.String())
	}

	pageErr := httptest.NewRecorder()
	handler.ServeHTTP(pageErr, httptest.NewRequest(http.MethodGet, "/error", nil))
	if pageErr.Code != http.StatusInternalServerError || pageErr.Body.String() != "internal: load failed" {
		t.Fatalf("page error = (%d, %q)", pageErr.Code, pageErr.Body.String())
	}

	invalid := httptest.NewRecorder()
	handler.ServeHTTP(invalid, httptest.NewRequest(http.MethodGet, "/badredirect", nil))
	if invalid.Code != http.StatusInternalServerError {
		t.Fatalf("bad redirect status = %d, want %d", invalid.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(invalid.Body.String(), goldr.ErrInvalidRouteResponse.Error()) {
		t.Fatalf("bad redirect body = %q, want invalid page response", invalid.Body.String())
	}
}
`)

	runGoTest(t, tempDir)
}
