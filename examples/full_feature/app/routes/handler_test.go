package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/csrf"
	"github.com/mobiletoly/goldr/examples/full_feature/app/deps"
	"github.com/mobiletoly/goldr/examples/full_feature/app/security"
	"github.com/mobiletoly/goldr/examples/full_feature/assets"
	"github.com/mobiletoly/goldr/examples/full_feature/internal/testutil"
	"github.com/mobiletoly/goldr/hx"
)

func testDependencies() *deps.Dependencies {
	return &deps.Dependencies{CSRF: security.CSRF}
}

func testHandler() http.Handler {
	return deps.Middleware(testDependencies(), Handler())
}

func testHandlerWithOptions(options HandlerOptions) http.Handler {
	return deps.Middleware(testDependencies(), HandlerWithOptions(options))
}

func TestTemplateInspectionOverlayIncludesScript(t *testing.T) {
	recorder := httptest.NewRecorder()
	testHandlerWithOptions(HandlerOptions{TemplateInspection: goldr.TemplateInspectionOverlay}).ServeHTTP(recorder, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/users", nil))

	if !strings.Contains(recorder.Body.String(), `<!--goldr:start id=g_pageusers_route_go`) {
		t.Fatalf("body missing inspector marker:\n%s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `<script src="/goldr/goldr-template-inspector.js" defer></script>`) {
		t.Fatalf("body missing template inspector script:\n%s", recorder.Body.String())
	}
}

func recordRoute(t *testing.T, method string, path string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	testHandler().ServeHTTP(recorder, httptest.NewRequestWithContext(context.Background(), method, path, nil))
	return recorder
}

func recordForm(t *testing.T, path string, values url.Values, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	testHandler().ServeHTTP(recorder, request)
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
			name: "protected resource demo",
			path: "/protected-resource-demo",
			want: []string{
				"Goldr Example",
				"Protected Resource Demo",
				"Signed out",
				`href="/sign-in?next=%2Fprotected-resource-demo"`,
				`href="/admin"`,
				`<a href="/protected-resource-demo" aria-current="page">Protected</a>`,
				`<title>Protected Resource Demo - Goldr Example</title>`,
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
				`hx-encoding="multipart/form-data"`,
				`id="users-table-slot"`,
				`hx-get="/users/table" hx-target="#users-table-slot" hx-swap="innerHTML"`,
				`hx-get="/users/table?status=active" hx-target="#users-table-slot" hx-swap="innerHTML"`,
				`hx-get="/users/table?status=inactive" hx-target="#users-table-slot" hx-swap="innerHTML"`,
				"Active only",
				"Inactive only",
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
			for _, want := range []string{
				`<meta name="csrf-token" content="`,
				`hx-headers="{&#34;` + csrf.HeaderName + `&#34;:`,
			} {
				if !strings.Contains(body, want) {
					t.Fatalf("body = %q, want %q", body, want)
				}
			}
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
				for _, want := range []string{
					`src="https://cdn.jsdelivr.net/npm/htmx.org@4.0.0-beta3"`,
					`integrity="sha384-bq4nTap5u8w4XlVP8JHkDioQVZBI5wUx5PxNwlbCq27H5QJ+q0CSeJcTYU+PLdCp"`,
					`href="` + assets.Path("app.css") + `"`,
					`src="` + assets.Path("app.js") + `"`,
					`data-js-enhance="open-users"`,
				} {
					if !strings.Contains(body, want) {
						t.Fatalf("body = %q, want %q", body, want)
					}
				}
			}
		})
	}
}

func TestHandlerProtectedResourceDemoSignedInState(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected-resource-demo", nil)
	request.AddCookie(&http.Cookie{Name: security.DemoAuthCookie, Value: security.RoleAdmin})
	recorder := httptest.NewRecorder()

	testHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	for _, want := range []string{
		"Signed in as admin",
		`action="/protected-resource-demo/reveal-secret"`,
		`action="/protected-resource-demo/sign-out"`,
		`name="csrf_token"`,
		`href="/admin"`,
	} {
		if !strings.Contains(recorder.Body.String(), want) {
			t.Fatalf("body = %q, want %q", recorder.Body.String(), want)
		}
	}
}

func TestHandlerProtectedPageResponses(t *testing.T) {
	redirect := recordRoute(t, http.MethodGet, "/admin")
	if redirect.Code != http.StatusSeeOther {
		t.Fatalf("redirect status = %d, want %d", redirect.Code, http.StatusSeeOther)
	}
	if redirect.Header().Get("Location") != "/sign-in?next=%2Fadmin" {
		t.Fatalf("redirect Location = %q, want /sign-in?next=%%2Fadmin", redirect.Header().Get("Location"))
	}
	if strings.Contains(redirect.Body.String(), "Protected admin") {
		t.Fatalf("redirect body = %q, want no protected page render", redirect.Body.String())
	}
	signIn := recordRoute(t, http.MethodGet, redirect.Header().Get("Location"))
	if signIn.Code != http.StatusOK {
		t.Fatalf("sign-in status = %d, want %d", signIn.Code, http.StatusOK)
	}
	if !strings.Contains(signIn.Body.String(), "Sign in to open the protected admin page.") {
		t.Fatalf("sign-in body = %q, want protected page redirect notice", signIn.Body.String())
	}
	if !strings.Contains(signIn.Body.String(), `class="auth-notice"`) {
		t.Fatalf("sign-in body = %q, want auth notice class", signIn.Body.String())
	}
	if strings.Contains(signIn.Body.String(), "Open protected admin page") {
		t.Fatalf("sign-in body = %q, want no direct admin link", signIn.Body.String())
	}
	for _, want := range []string{
		`action="/sign-in"`,
		`name="credential" value="admin"`,
		`name="credential" value="member"`,
		`name="credential" value="unknown"`,
		`name="next" value="/admin"`,
	} {
		if !strings.Contains(signIn.Body.String(), want) {
			t.Fatalf("sign-in body = %q, want %q", signIn.Body.String(), want)
		}
	}

	memberRequest := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/admin", nil)
	memberRequest.AddCookie(&http.Cookie{Name: security.DemoAuthCookie, Value: security.RoleMember})
	member := httptest.NewRecorder()
	testHandler().ServeHTTP(member, memberRequest)
	if member.Code != http.StatusForbidden {
		t.Fatalf("member status = %d, want %d", member.Code, http.StatusForbidden)
	}
	for _, want := range []string{
		"Goldr Example",
		"Forbidden",
		"requires the admin demo role",
		`href="/protected-resource-demo"`,
		`<title>Protected admin - Goldr Example</title>`,
	} {
		if !strings.Contains(member.Body.String(), want) {
			t.Fatalf("member body = %q, want %q", member.Body.String(), want)
		}
	}
	if strings.Contains(member.Body.String(), "Switch demo role") {
		t.Fatalf("member body = %q, want no switch role link", member.Body.String())
	}

	adminRequest := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/admin", nil)
	adminRequest.AddCookie(&http.Cookie{Name: security.DemoAuthCookie, Value: security.RoleAdmin})
	admin := httptest.NewRecorder()
	testHandler().ServeHTTP(admin, adminRequest)
	if admin.Code != http.StatusOK {
		t.Fatalf("admin status = %d, want %d", admin.Code, http.StatusOK)
	}
	if !strings.Contains(admin.Body.String(), "route-local auth check passed") {
		t.Fatalf("admin body = %q", admin.Body.String())
	}
	if !strings.Contains(admin.Body.String(), `href="/protected-resource-demo"`) {
		t.Fatalf("admin body = %q, want protected resource demo link", admin.Body.String())
	}
	if strings.Contains(admin.Body.String(), "Switch demo role") {
		t.Fatalf("admin body = %q, want no switch role link", admin.Body.String())
	}
}

func TestHandlerProtectedPageErrorResponse(t *testing.T) {
	handler := testHandlerWithOptions(HandlerOptions{
		ErrorHandlers: ErrorHandlers{
			RouteError: func(r *http.Request, err error) goldr.RouteResponse {
				return goldr.Text{Status: http.StatusInternalServerError, Body: err.Error()}
			},
		},
	})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/admin?demo_error=1", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if recorder.Body.String() != "demo admin load failed" {
		t.Fatalf("body = %q, want demo admin load failed", recorder.Body.String())
	}
}

func TestHandlerSignInActionSetsDemoRoleAndRedirects(t *testing.T) {
	tests := []struct {
		name       string
		credential string
		next       string
		location   string
		role       string
	}{
		{
			name:       "admin returns to admin",
			credential: security.RoleAdmin,
			next:       "/admin",
			location:   "/admin",
			role:       security.RoleAdmin,
		},
		{
			name:       "member returns to demo",
			credential: security.RoleMember,
			next:       "/protected-resource-demo",
			location:   "/protected-resource-demo",
			role:       security.RoleMember,
		},
		{
			name:       "invalid next defaults to demo",
			credential: security.RoleAdmin,
			next:       "/missing",
			location:   "/protected-resource-demo",
			role:       security.RoleAdmin,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cookie, token := testutil.CSRFPair(t, security.CSRF)
			recorder := recordForm(t, "/sign-in", url.Values{
				csrf.FieldName: {token},
				"credential":   {test.credential},
				"next":         {test.next},
			}, cookie)
			if recorder.Code != http.StatusSeeOther {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusSeeOther)
			}
			if recorder.Header().Get("Location") != test.location {
				t.Fatalf("Location = %q, want %s", recorder.Header().Get("Location"), test.location)
			}
			for _, setCookie := range recorder.Result().Cookies() {
				if setCookie.Name == security.DemoAuthCookie && setCookie.Value == test.role {
					return
				}
			}
			t.Fatalf("Set-Cookie = %v, want %s=%s", recorder.Result().Cookies(), security.DemoAuthCookie, test.role)
		})
	}
}

func TestHandlerSignInActionRejectsUnknownCredentials(t *testing.T) {
	cookie, token := testutil.CSRFPair(t, security.CSRF)
	recorder := recordForm(t, "/sign-in", url.Values{
		csrf.FieldName: {token},
		"credential":   {"unknown"},
		"next":         {"/admin"},
	}, cookie)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusSeeOther)
	}
	if recorder.Header().Get("Location") != "/sign-in?next=%2Fadmin&error=credentials" {
		t.Fatalf("Location = %q, want sign-in credential error", recorder.Header().Get("Location"))
	}
	for _, setCookie := range recorder.Result().Cookies() {
		if setCookie.Name == security.DemoAuthCookie {
			t.Fatalf("Set-Cookie = %v, want no demo role cookie", recorder.Result().Cookies())
		}
	}

	signIn := recordRoute(t, http.MethodGet, recorder.Header().Get("Location"))
	if signIn.Code != http.StatusOK {
		t.Fatalf("sign-in status = %d, want %d", signIn.Code, http.StatusOK)
	}
	for _, want := range []string{
		"Unknown credentials.",
		`class="auth-notice"`,
		`name="next" value="/admin"`,
	} {
		if !strings.Contains(signIn.Body.String(), want) {
			t.Fatalf("sign-in body = %q, want %q", signIn.Body.String(), want)
		}
	}
}

func TestHandlerProtectedResourceDemoSignOutClearsDemoRole(t *testing.T) {
	cookie, token := testutil.CSRFPair(t, security.CSRF)
	recorder := recordForm(t, "/protected-resource-demo/sign-out", url.Values{
		csrf.FieldName: {token},
	}, cookie, &http.Cookie{Name: security.DemoAuthCookie, Value: security.RoleAdmin})

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusSeeOther)
	}
	if recorder.Header().Get("Location") != "/protected-resource-demo" {
		t.Fatalf("Location = %q, want /protected-resource-demo", recorder.Header().Get("Location"))
	}
	for _, setCookie := range recorder.Result().Cookies() {
		if setCookie.Name == security.DemoAuthCookie && setCookie.MaxAge < 0 {
			return
		}
	}
	t.Fatalf("Set-Cookie = %v, want cleared %s cookie", recorder.Result().Cookies(), security.DemoAuthCookie)
}

func TestHandlerProtectedResourceDemoActionWritesFullPage(t *testing.T) {
	cookie, token := testutil.CSRFPair(t, security.CSRF)
	recorder := recordForm(t, "/protected-resource-demo/reveal-secret", url.Values{
		csrf.FieldName: {token},
	}, cookie, &http.Cookie{Name: security.DemoAuthCookie, Value: security.RoleAdmin})

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if recorder.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q", recorder.Header().Get("Content-Type"))
	}
	for _, want := range []string{
		`<title>One-time secret - Goldr Example</title>`,
		`href="` + assets.Path("app.css") + `"`,
		"One-time secret",
		"This full page was returned from an action after a POST.",
		"goldr-demo-secret-123",
		`href="/protected-resource-demo"`,
	} {
		if !strings.Contains(recorder.Body.String(), want) {
			t.Fatalf("body = %q, want %q", recorder.Body.String(), want)
		}
	}
}

func TestHandlerProtectedResourceDemoActionRequiresAuth(t *testing.T) {
	cookie, token := testutil.CSRFPair(t, security.CSRF)
	recorder := recordForm(t, "/protected-resource-demo/reveal-secret", url.Values{
		csrf.FieldName: {token},
	}, cookie)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusSeeOther)
	}
	if recorder.Header().Get("Location") != "/sign-in?next=%2Fprotected-resource-demo" {
		t.Fatalf("Location = %q, want sign-in redirect", recorder.Header().Get("Location"))
	}
	if strings.Contains(recorder.Body.String(), "goldr-demo-secret-123") {
		t.Fatalf("body = %q, want no secret", recorder.Body.String())
	}
}

func TestHandlerGetFragmentPartials(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    []string
		wantNot []string
	}{
		{
			name:    "table",
			path:    "/users/table",
			want:    []string{"User Table Fragment", `href="/users/42"`},
			wantNot: fragmentLayoutText(),
		},
		{
			name:    "active filter",
			path:    "/users/table?status=active",
			want:    []string{"Ada Lovelace", "Grace Hopper"},
			wantNot: append([]string{"Katherine Johnson"}, fragmentLayoutText()...),
		},
		{
			name:    "inactive filter",
			path:    "/users/table?status=inactive",
			want:    []string{"Katherine Johnson"},
			wantNot: append([]string{"Ada Lovelace", "Grace Hopper"}, fragmentLayoutText()...),
		},
		{
			name: "status options",
			path: "/users/status-options?selected=Inactive",
			want: []string{
				`<option value="Active">Active</option>`,
				`<option value="Inactive" selected>Inactive</option>`,
			},
			wantNot: fragmentLayoutText(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := recordRoute(t, http.MethodGet, test.path)
			assertFragmentResponse(t, recorder, test.want, test.wantNot)
		})
	}
}

func assertFragmentResponse(t *testing.T, recorder *httptest.ResponseRecorder, want []string, wantNot []string) {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	body := recorder.Body.String()
	for _, text := range want {
		if !strings.Contains(body, text) {
			t.Fatalf("body = %q, want %q", body, text)
		}
	}
	for _, text := range wantNot {
		if strings.Contains(body, text) {
			t.Fatalf("body = %q, want no %q", body, text)
		}
	}
}

func fragmentLayoutText() []string {
	return []string{"goldr app shell", "people section shell"}
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

func TestHandlerWithOptionsCustomNotFound(t *testing.T) {
	handler := testHandlerWithOptions(HandlerOptions{
		ErrorHandlers: ErrorHandlers{
			RouteNotFound: RouteNotFound,
		},
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
	cookie, token := testutil.CSRFPair(t, security.CSRF)
	body, contentType := testutil.MultipartBody(t, map[string]string{
		csrf.FieldName: token,
		"name":         "Grace Hopper",
		"status":       "Active",
	}, map[string]testutil.MultipartUpload{
		"avatar": {Filename: "grace.txt", Content: "example avatar"},
	})
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/users/create", body)
	request.Header.Set("Content-Type", contentType)
	request.AddCookie(cookie)
	recorder := httptest.NewRecorder()

	testHandler().ServeHTTP(recorder, request)

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
	if !strings.Contains(recorder.Body.String(), "grace.txt") {
		t.Fatalf("body = %q, want uploaded filename", recorder.Body.String())
	}
}

func TestHandlerPostCreateRejectsMissingCSRF(t *testing.T) {
	body, contentType := testutil.MultipartBody(t, map[string]string{
		"name":   "Grace Hopper",
		"status": "Active",
	}, nil)
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/users/create", body)
	request.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()

	testHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestHandlerPostCreateRedisplaysErrors(t *testing.T) {
	cookie, token := testutil.CSRFPair(t, security.CSRF)
	body, contentType := testutil.MultipartBody(t, map[string]string{
		csrf.FieldName: token,
		"name":         "",
		"status":       "Missing",
	}, nil)
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/users/create", body)
	request.Header.Set("Content-Type", contentType)
	request.AddCookie(cookie)
	recorder := httptest.NewRecorder()

	testHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnprocessableEntity)
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
	cookie, token := testutil.CSRFPair(t, security.CSRF)
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/users/save-preview", nil)
	request.AddCookie(cookie)
	request.Header.Set(csrf.HeaderName, token)
	recorder := httptest.NewRecorder()

	testHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get(hx.HeaderTrigger); got != "user:saved" {
		t.Fatalf("%s = %q, want user:saved", hx.HeaderTrigger, got)
	}
	if got := recorder.Header().Get(hx.HeaderRetarget); got != "#users-table-slot" {
		t.Fatalf("%s = %q, want #users-table-slot", hx.HeaderRetarget, got)
	}
	if got := recorder.Header().Get(hx.HeaderReswap); got != "innerHTML" {
		t.Fatalf("%s = %q, want innerHTML", hx.HeaderReswap, got)
	}
	if !strings.Contains(recorder.Body.String(), "User Table Fragment") {
		t.Fatalf("body = %q", recorder.Body.String())
	}
}

func TestHandlerRouteMiddlewareUsesInjectedCSRFGuard(t *testing.T) {
	guard, err := csrf.New(csrf.Config{Secret: []byte("full-feature-route-test-csrf-secret")})
	if err != nil {
		t.Fatalf("csrf.New() error = %v", err)
	}
	handler := deps.Middleware(&deps.Dependencies{CSRF: guard}, Handler())

	cookie := routeCSRFCookie(t, handler, "/users")
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/users/save-preview", nil)
	request.AddCookie(cookie)
	request.Header.Set(csrf.HeaderName, cookie.Value)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
}

func routeCSRFCookie(t *testing.T, handler http.Handler, path string) *http.Cookie {
	t.Helper()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want %d; body = %q", path, recorder.Code, http.StatusOK, recorder.Body.String())
	}
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == csrf.DefaultCookieName {
			return cookie
		}
	}
	t.Fatalf("GET %s Set-Cookie = %v, want %s", path, recorder.Result().Cookies(), csrf.DefaultCookieName)
	return nil
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
