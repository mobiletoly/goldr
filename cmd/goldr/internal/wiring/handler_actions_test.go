package wiring

import (
	"testing"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
)

func TestGenerateManifestActionWritesRouteResponseWithoutLayouts(t *testing.T) {
	manifest := routing.Manifest{
		Actions: []routing.ManifestAction{
			{Method: "POST", Route: "/create", GoFile: "actions.go", Function: "PostCreate"},
		},
	}

	tempDir := tempGoldrModule(t)
	writeGeneratedRoutes(t, tempDir, generateOK(t, manifest))
	writeTempFile(t, tempDir, "routes/actions.go", `package routes

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func PostCreate(r *http.Request) goldr.RouteResponse {
	component := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "<p>created</p>")
		return err
	})
	return goldr.NewPage(
		component,
		goldr.PageMetadata{Title: "Created"},
	).WithStatus(http.StatusCreated).WithHeader("Cache-Control", "no-store")
}
`)
	writeTempFile(t, tempDir, "routes/handler_test.go", `package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestActionRouteResponseWithoutLayouts(t *testing.T) {
	recorder := httptest.NewRecorder()
	Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/create", nil))

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %q", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", recorder.Header().Get("Cache-Control"))
	}
	if recorder.Body.String() != "<p>created</p>" {
		t.Fatalf("body = %q, want created page", recorder.Body.String())
	}
}
`)

	runGoTest(t, tempDir)
}

func TestGenerateManifestActionWritesRouteResponseWithLayoutStack(t *testing.T) {
	manifest := routing.Manifest{
		Layouts: []routing.ManifestLayout{
			{RoutePrefix: "/", Unit: completeUnit("layout.go")},
			{RoutePrefix: "/users", Unit: completeUnit("users/layout.go")},
			{RoutePrefix: "/users/{id}", Params: []string{"id"}, Unit: completeUnit("users/by_id/layout.go")},
		},
		Actions: []routing.ManifestAction{
			{Method: "POST", Route: "/users/{id}/keys/create", Params: []string{"id"}, GoFile: "users/by_id/keys/actions.go", Function: "PostCreate"},
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
		if _, err := io.WriteString(writer, "<root title=\""+layout.Metadata.Title+"\">"); err != nil {
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
		if _, err := io.WriteString(writer, "<users section=\""+layout.Metadata.Description+"\">"); err != nil {
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
	writeTempFile(t, tempDir, "routes/users/by_id/layout.go", `package by_id

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, layout goldr.LayoutContext) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		if _, err := io.WriteString(writer, "<user id=\""+r.PathValue("id")+"\">"); err != nil {
			return err
		}
		if err := layout.Child.Render(ctx, writer); err != nil {
			return err
		}
		_, err := io.WriteString(writer, "</user>")
		return err
	})
}
`)
	writeTempFile(t, tempDir, "routes/users/by_id/keys/actions.go", `package keys

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func PostCreate(r *http.Request) goldr.RouteResponse {
	id := r.PathValue("id")
	component := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "<p>created "+id+"</p>")
		return err
	})
	return goldr.NewPage(
		component,
		goldr.PageMetadata{Title: "Created " + id, Description: "keys"},
	).WithStatus(http.StatusCreated).WithHeader("Cache-Control", "no-store")
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

func TestActionRouteResponseUsesLayoutStack(t *testing.T) {
	recorder := httptest.NewRecorder()
	Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/users/42/keys/create", nil))

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", recorder.Header().Get("Cache-Control"))
	}
	want := "<root title=\"Created 42\"><users section=\"keys\"><user id=\"42\"><p>created 42</p></user></users></root>"
	if recorder.Body.String() != want {
		t.Fatalf("body = %q, want %q", recorder.Body.String(), want)
	}

	inspected := httptest.NewRecorder()
	HandlerWithOptions(HandlerOptions{TemplateInspection: goldr.TemplateInspectionComments}).ServeHTTP(inspected, httptest.NewRequest(http.MethodPost, "/users/42/keys/create", nil))
	if inspected.Code != http.StatusCreated {
		t.Fatalf("inspected status = %d, want %d", inspected.Code, http.StatusCreated)
	}
	for _, want := range []string{
		"<!--goldr:start id=g_layoutlayout_templ kind=layout route=/ source=app/routes/layout.templ go=app/routes/layout.go-->",
		"<!--goldr:start id=g_layoutusers_layout_templ kind=layout route=/users source=app/routes/users/layout.templ go=app/routes/users/layout.go-->",
		"<p>created 42</p>",
		"<!--goldr:end id=g_layoutusers_layout_templ-->",
		"<!--goldr:end id=g_layoutlayout_templ-->",
	} {
		if !strings.Contains(inspected.Body.String(), want) {
			t.Fatalf("inspected body missing %q:\n%s", want, inspected.Body.String())
		}
	}
	if strings.Contains(inspected.Body.String(), "kind=page") {
		t.Fatalf("inspected action response should not claim an action-owned page template:\n%s", inspected.Body.String())
	}
}
`)

	runGoTest(t, tempDir)
}

func TestGenerateManifestPageFragmentActionDispatch(t *testing.T) {
	manifest := routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/users", Unit: completeUnit("users/page.go")},
		},
		Fragments: []routing.ManifestFragment{
			{Name: "table", RoutePrefix: "/users", Unit: completeUnit("users/frag_table.go")},
		},
		Actions: []routing.ManifestAction{
			{Method: "POST", Route: "/users", GoFile: "users/actions.go", Function: "PostIndex", Suffix: "Index"},
			{Method: "POST", Route: "/users/table", GoFile: "users/actions.go", Function: "PostTable", Suffix: "Table", Segment: "table"},
			{Method: "PATCH", Route: "/users/{id}/profile", Params: []string{"id"}, GoFile: "users/by_id/actions.go", Function: "PatchProfile", Suffix: "Profile", Segment: "profile"},
		},
	}

	tempDir := tempGoldrModule(t)
	writeGeneratedRoutes(t, tempDir, generateOK(t, manifest))
	writeTempFile(t, tempDir, "routes/users/page.go", `package users

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	component := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "users page")
		return err
	})
	return goldr.NewPage(component, goldr.PageMetadata{})
}
`)
	writeTempFile(t, tempDir, "routes/users/frag_table.go", `package users

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func FragTable(r *http.Request) goldr.FragmentRouteResponse {
	return goldr.NewFragment(templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "users fragment")
		return err
	}))
}
`)
	writeTempFile(t, tempDir, "routes/users/actions.go", `package users

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func PostIndex(r *http.Request) goldr.RouteResponse {
	return goldr.Text{Status: http.StatusOK, Body: "users action"}
}

func PostTable(r *http.Request) goldr.RouteResponse {
	return goldr.Text{Status: http.StatusOK, Body: "table action"}
}
`)
	writeTempFile(t, tempDir, "routes/users/by_id/actions.go", `package by_id

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func PatchProfile(r *http.Request) goldr.RouteResponse {
	id := r.PathValue("id")
	return goldr.Text{Status: http.StatusOK, Body: "profile " + id}
}
`)
	writeTempFile(t, tempDir, "routes/handler_test.go", `package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestActionDispatch(t *testing.T) {
	tests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/users", "users page"},
		{http.MethodPost, "/users", "users action"},
		{http.MethodGet, "/users/table", "users fragment"},
		{http.MethodPost, "/users/table", "table action"},
		{http.MethodPatch, "/users/42/profile", "profile 42"},
		{http.MethodPatch, "/users/42%2F43/profile", "profile 42/43"},
	}
	for _, test := range tests {
		recorder := httptest.NewRecorder()
		Handler().ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s %s status = %d, want %d", test.method, test.path, recorder.Code, http.StatusOK)
		}
		if recorder.Body.String() != test.body {
			t.Fatalf("%s %s body = %q, want %q", test.method, test.path, recorder.Body.String(), test.body)
		}
	}

	method := httptest.NewRecorder()
	Handler().ServeHTTP(method, httptest.NewRequest(http.MethodPut, "/users", nil))
	if method.Code != http.StatusMethodNotAllowed {
		t.Fatalf("PUT /users status = %d, want %d", method.Code, http.StatusMethodNotAllowed)
	}
	if method.Header().Get("Allow") != "GET, HEAD, POST" {
		t.Fatalf("Allow = %q", method.Header().Get("Allow"))
	}
}
`)

	runGoTest(t, tempDir)
}
