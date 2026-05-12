package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr/hx"
)

func recordRoute(t *testing.T, method string, path string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	Handler().ServeHTTP(recorder, httptest.NewRequestWithContext(context.Background(), method, path, nil))
	return recorder
}

func TestHandlerGetPages(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    []string
		omit    []string
		wantCSS bool
	}{
		{
			name: "root",
			path: "/",
			want: []string{
				"Goldr Example",
				`<title>Goldr Example</title>`,
				`<meta name="description" content="Server-rendered pages, nested layouts, HTMX fragments, actions, and custom error views.">`,
				`href="/users"`,
				`href="/settings"`,
			},
			omit:    []string{`aria-current="page"`, "people section shell", `rel="canonical"`},
			wantCSS: true,
		},
		{
			name: "settings",
			path: "/settings",
			want: []string{
				"Goldr Example",
				"Settings",
				"Application preferences",
				`<title>Settings - Goldr Example</title>`,
				`<meta name="description" content="Application preferences and account controls.">`,
				`<a href="/settings" aria-current="page">Settings</a>`,
			},
			omit: []string{"people section shell", `rel="canonical"`},
		},
		{
			name: "users",
			path: "/users",
			want: []string{
				"Goldr Example",
				"people section shell",
				"Active accounts",
				"User Directory",
				`<title>Users - Goldr Example</title>`,
				`<meta name="description" content="Browse and manage example contacts.">`,
				`hx-post="/users/create"`,
				`href="/users/42"`,
				`<a href="/users" aria-current="page">Users</a>`,
				`name="name"`,
			},
			omit: []string{`rel="canonical"`},
		},
		{
			name: "dynamic user",
			path: "/users/42",
			want: []string{
				"Goldr Example",
				"people section shell",
				"Active accounts",
				"Ada Lovelace",
				"Route id",
				`<title>Ada Lovelace - Goldr Example</title>`,
				`<meta name="description" content="Contact details for Ada Lovelace.">`,
				`<a href="/users" aria-current="page">Users</a>`,
			},
			omit: []string{"User Directory", `rel="canonical"`},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := recordRoute(t, http.MethodGet, test.path)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
			}
			if recorder.Header().Get("Content-Type") != "text/html; charset=utf-8" {
				t.Fatalf("content-type = %q", recorder.Header().Get("Content-Type"))
			}
			body := recorder.Body.String()
			for _, want := range test.want {
				if !strings.Contains(body, want) {
					t.Fatalf("body = %q, want %q", body, want)
				}
			}
			for _, omit := range test.omit {
				if strings.Contains(body, omit) {
					t.Fatalf("body = %q, want no %q", body, omit)
				}
			}
			if test.wantCSS {
				for _, want := range []string{`src="https://unpkg.com/htmx.org@2.0.4"`, `href="/assets/app.css"`} {
					if !strings.Contains(body, want) {
						t.Fatalf("body = %q, want %q", body, want)
					}
				}
			}
		})
	}
}

func TestHandlerGetFragmentPartial(t *testing.T) {
	recorder := recordRoute(t, http.MethodGet, "/users/frag_table")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "User Table Fragment") {
		t.Fatalf("body = %q", body)
	}
	if !strings.Contains(body, `href="/users/42"`) {
		t.Fatalf("body = %q", body)
	}
	for _, layoutText := range []string{"goldr app shell", "people section shell"} {
		if strings.Contains(body, layoutText) {
			t.Fatalf("body = %q, want no layout text %q", body, layoutText)
		}
	}
}

func TestHandlerHeadRoot(t *testing.T) {
	recorder := recordRoute(t, http.MethodHead, "/users/42")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("body length = %d, want 0", recorder.Body.Len())
	}
}

func TestHandlerMissingPath(t *testing.T) {
	recorder := recordRoute(t, http.MethodGet, "/missing")

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestHandlerWithErrorsCustomNotFound(t *testing.T) {
	handler := HandlerWithErrors(ErrorHandlers{
		NotFound: NotFound,
	})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/missing", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
	if !strings.Contains(recorder.Body.String(), "Page not found") {
		t.Fatalf("body = %q, want Page not found", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "No goldr route matches /missing.") {
		t.Fatalf("body = %q, want missing path", recorder.Body.String())
	}
}

func TestHandlerPostCreateAction(t *testing.T) {
	body := strings.NewReader("name=Grace+Hopper&status=Active")
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/users/create", body)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get(hx.HeaderTrigger); got != "user:created" {
		t.Fatalf("%s = %q, want user:created", hx.HeaderTrigger, got)
	}
	if got := recorder.Header().Get(hx.HeaderRetarget); got != "#users-directory" {
		t.Fatalf("%s = %q, want #users-directory", hx.HeaderRetarget, got)
	}
	if !strings.Contains(recorder.Body.String(), "Grace Hopper") {
		t.Fatalf("body = %q, want created contact", recorder.Body.String())
	}
}

func TestHandlerPostCreateRedisplaysErrors(t *testing.T) {
	body := strings.NewReader("name=&status=Missing")
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/users/create", body)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	for _, want := range []string{"Name is required.", "Choose a valid status.", "User Table Fragment"} {
		if !strings.Contains(recorder.Body.String(), want) {
			t.Fatalf("body = %q, want %q", recorder.Body.String(), want)
		}
	}
	if got := recorder.Header().Get(hx.HeaderTrigger); got != "" {
		t.Fatalf("%s = %q, want empty", hx.HeaderTrigger, got)
	}
}

func TestHandlerPostSavePreviewAction(t *testing.T) {
	recorder := recordRoute(t, http.MethodPost, "/users/save-preview")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get(hx.HeaderTrigger); got != "user:saved" {
		t.Fatalf("%s = %q, want user:saved", hx.HeaderTrigger, got)
	}
	if got := recorder.Header().Get(hx.HeaderRetarget); got != "#users-table" {
		t.Fatalf("%s = %q, want #users-table", hx.HeaderRetarget, got)
	}
	if got := recorder.Header().Get(hx.HeaderReswap); got != "outerHTML" {
		t.Fatalf("%s = %q, want outerHTML", hx.HeaderReswap, got)
	}
	if !strings.Contains(recorder.Body.String(), "User Table Fragment") {
		t.Fatalf("body = %q", recorder.Body.String())
	}
}

func TestHandlerRejectsMethods(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		allow  string
	}{
		{name: "page", method: http.MethodPost, path: "/users", allow: "GET, HEAD"},
		{name: "action", method: http.MethodGet, path: "/users/create", allow: "POST"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := recordRoute(t, test.method, test.path)

			if recorder.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
			}
			if recorder.Header().Get("Allow") != test.allow {
				t.Fatalf("allow = %q, want %q", recorder.Header().Get("Allow"), test.allow)
			}
		})
	}
}
