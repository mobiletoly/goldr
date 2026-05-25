package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKitRoutesRenderSharedReportPages(t *testing.T) {
	tests := []struct {
		path    string
		title   string
		want    []string
		wantNot []string
	}{
		{
			path:  "/admin/reports",
			title: "Admin Reports - Goldr Kit Routes",
			want: []string{
				"Admin Reports",
				"Operational view across all teams.",
				"Revenue",
				"Churn risk",
				`hx-get="/admin/reports/table"`,
				`value="30d" selected`,
				`href="/admin/reports/audit"`,
				"Admin report tools",
			},
		},
		{
			path:  "/user/reports",
			title: "User Reports - Goldr Kit Routes",
			want: []string{
				"User Reports",
				"Personal report view for the signed-in user.",
				"My tasks",
				"My usage",
				`hx-get="/user/reports/table"`,
				`value="30d" selected`,
			},
			wantNot: []string{
				`href="/user/reports/audit"`,
				"Admin report tools",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			response := serveExample(t, http.MethodGet, test.path)
			defer closeBody(t, response)
			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
			}
			if got := response.Header.Get("X-Content-Type-Options"); got != "nosniff" {
				t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
			}
			body := readBody(t, response)
			for _, want := range test.want {
				if !strings.Contains(body, want) {
					t.Fatalf("body = %q, want %q", body, want)
				}
			}
			for _, text := range test.wantNot {
				if strings.Contains(body, text) {
					t.Fatalf("body = %q, want no %q", body, text)
				}
			}
			if !strings.Contains(body, "<title>"+test.title+"</title>") {
				t.Fatalf("body = %q, want title %q", body, test.title)
			}
		})
	}
}

func TestKitRoutesKeepOwnerOnlyChildUnderAdmin(t *testing.T) {
	response := serveExample(t, http.MethodGet, "/admin/reports/audit")
	defer closeBody(t, response)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	body := readBody(t, response)
	for _, want := range []string{
		"Admin Report Tools",
		"Owner-only report operations",
		`href="/admin/reports"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body = %q, want %q", body, want)
		}
	}

	missing := serveExample(t, http.MethodGet, "/user/reports/audit")
	defer closeBody(t, missing)
	if missing.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", missing.StatusCode, http.StatusNotFound)
	}
}

func TestKitRoutesRenderSharedReportFragments(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    []string
		wantNot []string
	}{
		{
			name:    "admin table",
			path:    "/admin/reports/table",
			want:    []string{"Admin report table", "Revenue", "Churn risk"},
			wantNot: []string{"Goldr Kit Routes</title>"},
		},
		{
			name:    "user table",
			path:    "/user/reports/table",
			want:    []string{"User report table", "My tasks", "My usage"},
			wantNot: []string{"Goldr Kit Routes</title>"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertReportFragment(t, test.path, test.want, test.wantNot)
		})
	}
}

func assertReportFragment(t *testing.T, path string, want []string, wantNot []string) {
	t.Helper()

	response := serveExample(t, http.MethodGet, path)
	defer closeBody(t, response)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if got := response.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	body := readBody(t, response)
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

func serveExample(t *testing.T, method string, path string) *http.Response {
	t.Helper()

	request := httptest.NewRequestWithContext(context.Background(), method, path, nil)
	recorder := httptest.NewRecorder()
	exampleHandler().ServeHTTP(recorder, request)
	return recorder.Result()
}

func readBody(t *testing.T, response *http.Response) string {
	t.Helper()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	return string(body)
}

func closeBody(t *testing.T, response *http.Response) {
	t.Helper()

	if err := response.Body.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
