package wiring

import (
	"strings"
	"testing"

	"github.com/mobiletoly/goldr/internal/routing"
)

func TestGenerateManifestRuntimeDispatchAndLayoutStack(t *testing.T) {
	tempDir := tempGoldrModule(t)
	source := generateOK(t, runtimeManifest())
	if !strings.Contains(source, `"github.com/mobiletoly/goldr"`) {
		t.Fatalf("generated source missing root goldr package import:\n%s", source)
	}
	if !strings.Contains(source, "goldr.LayoutContext{Metadata: metadata}") {
		t.Fatalf("generated source missing layout context wiring:\n%s", source)
	}
	writeGeneratedRoutes(t, tempDir, source)
	writeTempFile(t, tempDir, "routes/page.go", `package routes

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.RouteResponse {
	component := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "<h1>Root</h1>")
		return err
	})
	return goldr.NewPage(component, goldr.PageMetadata{Title: "Root"})
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
	writeTempFile(t, tempDir, "routes/users/page.go", `package users

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.RouteResponse {
	fragment := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "<tbody>Users fragment</tbody>")
		return err
	})
	component := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		if _, err := io.WriteString(writer, "<h1>Users</h1>"); err != nil {
			return err
		}
		return renderFragTable(fragment).Render(ctx, writer)
	})
	return goldr.NewPage(component, goldr.PageMetadata{Title: "Users", Description: "users"})
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
	writeTempFile(t, tempDir, "routes/users/by_id/page.go", `package by_id

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.RouteResponse {
	id := r.PathValue("id")
	component := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "<h1>User "+id+"</h1>")
		return err
	})
	return goldr.NewPage(component, goldr.PageMetadata{Title: "User " + id, Description: "users"})
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
		id := r.PathValue("id")
		if _, err := io.WriteString(writer, "<user id=\""+id+"\" title=\""+layout.Metadata.Title+"\">"); err != nil {
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
	writeTempFile(t, tempDir, "routes/users/frag_table.go", `package users

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func FragTable(r *http.Request) goldr.RouteResponse {
	return goldr.NewFragment(templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "<tbody>Users fragment</tbody>")
		return err
	}))
}
`)
	writeTempFile(t, tempDir, "routes/users/by_id/frag_row.go", `package by_id

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func FragRow(r *http.Request) goldr.RouteResponse {
	return goldr.NewFragment(templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		id := r.PathValue("id")
		_, err := io.WriteString(writer, "<tr>User "+id+"</tr>")
		return err
	}))
}
`)
	writeGeneratedFragmentWrappers(t, tempDir, runtimeManifest())
	writeTempFile(t, tempDir, "routes/handler_test.go", `package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr"
)

func TestHandlerRoutes(t *testing.T) {
	tests := []struct {
		path string
		body string
	}{
		{"/", "<root title=\"Root\"><h1>Root</h1></root>"},
		{"/users", "<root title=\"Users\"><users section=\"users\"><h1>Users</h1><tbody>Users fragment</tbody></users></root>"},
		{"/users/42", "<root title=\"User 42\"><users section=\"users\"><user id=\"42\" title=\"User 42\"><h1>User 42</h1></user></users></root>"},
		{"/users/42%2F43", "<root title=\"User 42/43\"><users section=\"users\"><user id=\"42/43\" title=\"User 42/43\"><h1>User 42/43</h1></user></users></root>"},
		{"/users/frag-table", "<tbody>Users fragment</tbody>"},
		{"/users/42/frag-row", "<tr>User 42</tr>"},
		{"/users/a%20b/frag-row", "<tr>User a b</tr>"},
	}
	for _, test := range tests {
		recorder := httptest.NewRecorder()
		Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", test.path, recorder.Code, http.StatusOK)
		}
		if recorder.Body.String() != test.body {
			t.Fatalf("%s body = %q, want %q", test.path, recorder.Body.String(), test.body)
		}
	}
}

func TestTemplateInspectorMarkers(t *testing.T) {
	page := httptest.NewRecorder()
	HandlerWithOptions(HandlerOptions{TemplateInspection: goldr.TemplateInspectionComments}).ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/users", nil))
	if page.Code != http.StatusOK {
		t.Fatalf("inspected page status = %d, want %d", page.Code, http.StatusOK)
	}
	for _, want := range []string{
		"<!--goldr:start id=g_layoutlayout_templ kind=layout route=/ source=app/routes/layout.templ go=app/routes/layout.go-->",
		"<!--goldr:start id=g_layoutusers_layout_templ kind=layout route=/users source=app/routes/users/layout.templ go=app/routes/users/layout.go-->",
		"<!--goldr:start id=g_pageusers_page_templ kind=page route=/users source=app/routes/users/page.templ go=app/routes/users/page.go-->",
		"<!--goldr:start id=g_fragmentusers_frag_table_templ kind=fragment route=/users/frag-table source=app/routes/users/frag_table.templ go=app/routes/users/frag_table.go-->",
		"<tbody>Users fragment</tbody>",
		"<!--goldr:end id=g_fragmentusers_frag_table_templ-->",
		"<!--goldr:end id=g_pageusers_page_templ-->",
		"<!--goldr:end id=g_layoutusers_layout_templ-->",
		"<!--goldr:end id=g_layoutlayout_templ-->",
	} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("inspected page body missing %q:\n%s", want, page.Body.String())
		}
	}
	if strings.Index(page.Body.String(), "id=g_layoutlayout_templ") > strings.Index(page.Body.String(), "id=g_layoutusers_layout_templ") {
		t.Fatalf("root layout marker should wrap users layout marker:\n%s", page.Body.String())
	}

	overlay := httptest.NewRecorder()
	HandlerWithOptions(HandlerOptions{TemplateInspection: goldr.TemplateInspectionOverlay}).ServeHTTP(overlay, httptest.NewRequest(http.MethodGet, "/users", nil))
	if !strings.Contains(overlay.Body.String(), "<!--goldr:start id=g_pageusers_page_templ") {
		t.Fatalf("overlay inspection body missing page marker:\n%s", overlay.Body.String())
	}

	fragment := httptest.NewRecorder()
	HandlerWithOptions(HandlerOptions{TemplateInspection: goldr.TemplateInspectionComments}).ServeHTTP(fragment, httptest.NewRequest(http.MethodGet, "/users/frag-table", nil))
	if fragment.Code != http.StatusOK {
		t.Fatalf("inspected fragment status = %d, want %d", fragment.Code, http.StatusOK)
	}
	for _, want := range []string{
		"<!--goldr:start id=g_fragmentusers_frag_table_templ kind=fragment route=/users/frag-table source=app/routes/users/frag_table.templ go=app/routes/users/frag_table.go-->",
		"<tbody>Users fragment</tbody>",
		"<!--goldr:end id=g_fragmentusers_frag_table_templ-->",
	} {
		if !strings.Contains(fragment.Body.String(), want) {
			t.Fatalf("inspected fragment body missing %q:\n%s", want, fragment.Body.String())
		}
	}

	head := httptest.NewRecorder()
	HandlerWithOptions(HandlerOptions{TemplateInspection: goldr.TemplateInspectionComments}).ServeHTTP(head, httptest.NewRequest(http.MethodHead, "/users/frag-table", nil))
	if head.Body.Len() != 0 {
		t.Fatalf("inspected HEAD body length = %d, want 0", head.Body.Len())
	}
}

func TestHandlerHeadAndErrors(t *testing.T) {
	head := httptest.NewRecorder()
	Handler().ServeHTTP(head, httptest.NewRequest(http.MethodHead, "/users/42/frag-row", nil))
	if head.Code != http.StatusOK {
		t.Fatalf("HEAD status = %d, want %d", head.Code, http.StatusOK)
	}
	if head.Body.Len() != 0 {
		t.Fatalf("HEAD body length = %d, want 0", head.Body.Len())
	}

	missing := httptest.NewRecorder()
	Handler().ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want %d", missing.Code, http.StatusNotFound)
	}

	trailing := httptest.NewRecorder()
	Handler().ServeHTTP(trailing, httptest.NewRequest(http.MethodGet, "/users/", nil))
	if trailing.Code != http.StatusNotFound {
		t.Fatalf("trailing status = %d, want %d", trailing.Code, http.StatusNotFound)
	}

	method := httptest.NewRecorder()
	Handler().ServeHTTP(method, httptest.NewRequest(http.MethodPost, "/users/frag-table", nil))
	if method.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method status = %d, want %d", method.Code, http.StatusMethodNotAllowed)
	}
	if method.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("allow = %q", method.Header().Get("Allow"))
	}
}
`)

	runGoTest(t, tempDir)
}

func TestGenerateManifestActionOnlyRuntimeDispatch(t *testing.T) {
	manifest := routing.Manifest{
		Actions: []routing.ManifestAction{
			{Method: "POST", Route: "/create", GoFile: "actions.go", Function: "PostCreate", Suffix: "Create", Segment: "create"},
		},
	}
	source := generateOK(t, manifest)
	for _, want := range []string{
		"goldr.WithRoutePageRenderer",
		"github.com/a-h/templ",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("action-only source missing %q:\n%s", want, source)
		}
	}
	for _, unwant := range []string{
		"goldr.WithRouteResponseWriter",
		"func goldrWriteComponentResponse",
		"\"bytes\"",
	} {
		if strings.Contains(source, unwant) {
			t.Fatalf("action-only source contains obsolete %q:\n%s", unwant, source)
		}
	}

	tempDir := tempGoldrModule(t)
	writeGeneratedRoutes(t, tempDir, source)
	writeTempFile(t, tempDir, "routes/actions.go", `package routes

import "net/http"

func PostCreate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Action", "create")
	_, _ = w.Write([]byte("created"))
}
`)
	writeTempFile(t, tempDir, "routes/handler_test.go", `package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestActionOnlyHandler(t *testing.T) {
	recorder := httptest.NewRecorder()
	Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/create", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("POST status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.String() != "created" {
		t.Fatalf("POST body = %q, want created", recorder.Body.String())
	}
	if recorder.Header().Get("X-Action") != "create" {
		t.Fatalf("X-Action = %q", recorder.Header().Get("X-Action"))
	}

	method := httptest.NewRecorder()
	Handler().ServeHTTP(method, httptest.NewRequest(http.MethodGet, "/create", nil))
	if method.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status = %d, want %d", method.Code, http.StatusMethodNotAllowed)
	}
	if method.Header().Get("Allow") != "POST" {
		t.Fatalf("Allow = %q, want POST", method.Header().Get("Allow"))
	}
}
`)

	runGoTest(t, tempDir)
}

func TestGenerateManifestRouteTreeMiddleware(t *testing.T) {
	manifest := routing.Manifest{
		Pages: []routing.ManifestPage{
			{Route: "/", Unit: completeUnit("page.go")},
			{Route: "/users", Unit: completeUnit("users/page.go")},
			{Route: "/users/admin", Unit: completeUnit("users/admin/page.go")},
			{Route: "/users/sibling", Unit: completeUnit("users/sibling/page.go")},
		},
		Layouts: []routing.ManifestLayout{
			{RoutePrefix: "/", Unit: completeUnit("layout.go")},
		},
		Fragments: []routing.ManifestFragment{
			{Name: "table", RoutePrefix: "/users", Unit: completeUnit("users/frag_table.go")},
		},
		Actions: []routing.ManifestAction{
			{Method: "POST", Route: "/users/create", GoFile: "users/actions.go", Function: "PostCreate", Suffix: "Create", Segment: "create"},
		},
		Middlewares: []routing.ManifestMiddleware{
			{RoutePrefix: "/", GoFile: "middleware.go"},
			{RoutePrefix: "/users", GoFile: "users/middleware.go"},
			{RoutePrefix: "/users/admin", GoFile: "users/admin/middleware.go"},
			{RoutePrefix: "/users/sibling", GoFile: "users/sibling/middleware.go"},
			{RoutePrefix: "/users/create", GoFile: "users/create/middleware.go"},
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
	values := append(OrderValues(r), value)
	return r.WithContext(context.WithValue(r.Context(), key{}, values))
}

func Order(r *http.Request) string {
	return strings.Join(OrderValues(r), ">")
}

func OrderValues(r *http.Request) []string {
	values, _ := r.Context().Value(key{}).([]string)
	return append([]string(nil), values...)
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
	writeTempFile(t, tempDir, "routes/users/middleware.go", `package users

import (
	"net/http"

	"example.com/app/routectx"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Middleware", "users")
		next.ServeHTTP(w, routectx.Append(r, "users"))
	})
}
`)
	writeTempFile(t, tempDir, "routes/users/admin/middleware.go", `package admin

import (
	"net/http"

	"example.com/app/routectx"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Middleware", "admin")
		r = routectx.Append(r, "admin")
		if r.URL.Query().Get("stop") == "1" {
			w.WriteHeader(http.StatusTeapot)
			_, _ = w.Write([]byte("stopped " + routectx.Order(r)))
			return
		}
		next.ServeHTTP(w, r)
	})
}
`)
	writeTempFile(t, tempDir, "routes/users/sibling/middleware.go", `package sibling

import (
	"net/http"

	"example.com/app/routectx"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Middleware", "sibling")
		next.ServeHTTP(w, routectx.Append(r, "sibling"))
	})
}
`)
	writeTempFile(t, tempDir, "routes/page.go", `package routes

import (
	"context"
	"io"
	"net/http"

	"example.com/app/routectx"
	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.RouteResponse {
	order := routectx.Order(r)
	return goldr.NewPage(templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "page:"+order)
		return err
	}), goldr.PageMetadata{})
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
	order := routectx.Order(r)
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		if _, err := io.WriteString(writer, "layout:"+order+"|"); err != nil {
			return err
		}
		return layout.Child.Render(ctx, writer)
	})
}
`)
	writeTempFile(t, tempDir, "routes/users/page.go", `package users

import (
	"context"
	"io"
	"net/http"

	"example.com/app/routectx"
	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.RouteResponse {
	order := routectx.Order(r)
	return goldr.NewPage(templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "page:"+order)
		return err
	}), goldr.PageMetadata{})
}
`)
	writeTempFile(t, tempDir, "routes/users/admin/page.go", `package admin

import (
	"context"
	"io"
	"net/http"

	"example.com/app/routectx"
	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.RouteResponse {
	order := routectx.Order(r)
	return goldr.NewPage(templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "page:"+order)
		return err
	}), goldr.PageMetadata{})
}
`)
	writeTempFile(t, tempDir, "routes/users/sibling/page.go", `package sibling

import (
	"context"
	"io"
	"net/http"

	"example.com/app/routectx"
	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.RouteResponse {
	order := routectx.Order(r)
	return goldr.NewPage(templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "page:"+order)
		return err
	}), goldr.PageMetadata{})
}
`)
	writeTempFile(t, tempDir, "routes/users/frag_table.go", `package users

import (
	"context"
	"io"
	"net/http"

	"example.com/app/routectx"
	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func FragTable(r *http.Request) goldr.RouteResponse {
	order := routectx.Order(r)
	return goldr.NewFragment(templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "fragment:"+order)
		return err
	}))
}
`)
	writeTempFile(t, tempDir, "routes/users/actions.go", `package users

import (
	"net/http"

	"example.com/app/routectx"
)

func PostCreate(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("action:" + routectx.Order(r)))
}
`)
	writeTempFile(t, tempDir, "routes/handler_test.go", `package routes

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestMiddlewareRoutes(t *testing.T) {
	tests := []struct {
		method string
		path string
		body string
		middleware []string
	}{
		{http.MethodGet, "/", "layout:root|page:root", []string{"root"}},
		{http.MethodGet, "/users", "layout:root>users|page:root>users", []string{"root", "users"}},
		{http.MethodGet, "/users/admin", "layout:root>users>admin|page:root>users>admin", []string{"root", "users", "admin"}},
		{http.MethodGet, "/users/sibling", "layout:root>users>sibling|page:root>users>sibling", []string{"root", "users", "sibling"}},
		{http.MethodGet, "/users/frag-table", "fragment:root>users", []string{"root", "users"}},
		{http.MethodPost, "/users/create", "action:root>users", []string{"root", "users"}},
	}
	for _, test := range tests {
		recorder := httptest.NewRecorder()
		Handler().ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s %s status = %d, want 200; body = %q", test.method, test.path, recorder.Code, recorder.Body.String())
		}
		if recorder.Body.String() != test.body {
			t.Fatalf("%s %s body = %q, want %q", test.method, test.path, recorder.Body.String(), test.body)
		}
		if got := recorder.Header().Values("X-Middleware"); !reflect.DeepEqual(got, test.middleware) {
			t.Fatalf("%s %s middleware = %#v, want %#v", test.method, test.path, got, test.middleware)
		}
	}
}

func TestMiddlewareCanShortCircuit(t *testing.T) {
	recorder := httptest.NewRecorder()
	Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/users/admin?stop=1", nil))
	if recorder.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusTeapot)
	}
	if recorder.Body.String() != "stopped root>users>admin" {
		t.Fatalf("body = %q", recorder.Body.String())
	}
}

func TestMiddlewareDoesNotWrapGeneratedErrors(t *testing.T) {
	missing := httptest.NewRecorder()
	Handler().ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want %d", missing.Code, http.StatusNotFound)
	}
	if got := missing.Header().Values("X-Middleware"); len(got) != 0 {
		t.Fatalf("missing middleware = %#v, want none", got)
	}

	method := httptest.NewRecorder()
	Handler().ServeHTTP(method, httptest.NewRequest(http.MethodPost, "/", nil))
	if method.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method status = %d, want %d", method.Code, http.StatusMethodNotAllowed)
	}
	if got := method.Header().Values("X-Middleware"); len(got) != 0 {
		t.Fatalf("method middleware = %#v, want none", got)
	}
}
`)

	runGoTest(t, tempDir)
}

func TestGenerateManifestStaticRouteWinsOverDynamic(t *testing.T) {
	tempDir := tempGoldrModule(t)
	writeGeneratedRoutes(t, tempDir, generateOK(t, staticPriorityManifest()))
	writeTempFile(t, tempDir, "routes/users/profile/page.go", `package profile

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.RouteResponse {
	component := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "profile")
		return err
	})
	return goldr.NewPage(component, goldr.PageMetadata{})
}
`)
	writeTempFile(t, tempDir, "routes/users/profile/actions.go", `package profile

import "net/http"

func PostProfile(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("profile action"))
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

func Page(r *http.Request) goldr.RouteResponse {
	component := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "dynamic")
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

func FragTable(r *http.Request) goldr.RouteResponse {
	return goldr.NewFragment(templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "fragment")
		return err
	}))
}
`)
	writeTempFile(t, tempDir, "routes/handler_test.go", `package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStaticRouteWins(t *testing.T) {
	recorder := httptest.NewRecorder()
	Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/users/profile", nil))
	if recorder.Body.String() != "profile" {
		t.Fatalf("body = %q, want profile", recorder.Body.String())
	}

	fragment := httptest.NewRecorder()
	Handler().ServeHTTP(fragment, httptest.NewRequest(http.MethodGet, "/users/frag-table", nil))
	if fragment.Body.String() != "fragment" {
		t.Fatalf("fragment body = %q, want fragment", fragment.Body.String())
	}

	action := httptest.NewRecorder()
	Handler().ServeHTTP(action, httptest.NewRequest(http.MethodPost, "/users/profile", nil))
	if action.Body.String() != "profile action" {
		t.Fatalf("action body = %q, want profile action", action.Body.String())
	}
}
`)

	runGoTest(t, tempDir)
}

func TestGenerateManifestDispatchHelperNamesAreUnique(t *testing.T) {
	manifest := routing.Manifest{
		Actions: []routing.ManifestAction{
			{Method: "POST", Route: "/users/save-preview", GoFile: "users/actions.go", Function: "PostSavePreview", Suffix: "SavePreview", Segment: "save-preview"},
			{Method: "POST", Route: "/users/save_preview", GoFile: "users/actions.go", Function: "PostSavePreviewUnderscore", Suffix: "SavePreviewUnderscore", Segment: "save_preview"},
		},
	}

	tempDir := tempGoldrModule(t)
	writeGeneratedRoutes(t, tempDir, generateOK(t, manifest))
	writeTempFile(t, tempDir, "routes/users/actions.go", `package users

import "net/http"

func PostSavePreview(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("dash"))
}

func PostSavePreviewUnderscore(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("underscore"))
}
`)
	writeTempFile(t, tempDir, "routes/handler_test.go", `package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDispatchHelperNameCollisionRoutes(t *testing.T) {
	tests := []struct {
		path string
		body string
	}{
		{"/users/save-preview", "dash"},
		{"/users/save_preview", "underscore"},
	}
	for _, test := range tests {
		recorder := httptest.NewRecorder()
		Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, test.path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", test.path, recorder.Code, http.StatusOK)
		}
		if recorder.Body.String() != test.body {
			t.Fatalf("%s body = %q, want %q", test.path, recorder.Body.String(), test.body)
		}
	}
}
`)

	runGoTest(t, tempDir)
}
