package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr/csrf"
)

func TestReactIslandPageAndSaveContract(t *testing.T) {
	server := httptest.NewServer(exampleHandler())
	t.Cleanup(server.Close)

	client := server.Client()
	response, err := client.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("GET / error = %v", err)
	}
	body := readBody(t, response)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	for _, want := range []string{
		`data-client-island="project-editor"`,
		`data-save-url="/save"`,
		`data-cancel-url="/about"`,
		`meta name="csrf-token"`,
		`hx-boost:inherited="true"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET / body missing %q: %s", want, body)
		}
	}

	token := metaContent(t, body, csrf.MetaName)
	cookie := findCookie(t, response.Cookies(), csrf.DefaultCookieName)

	saved := postJSON(t, client, server.URL+"/save", cookie, token, `{"name":"  Calm editor  ","pinned":true}`)
	if saved.StatusCode != http.StatusOK {
		t.Fatalf("POST /save status = %d, want %d: %s", saved.StatusCode, http.StatusOK, readBody(t, saved))
	}
	if got := saved.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if got := saved.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	var payload struct {
		Project struct {
			Name   string `json:"name"`
			Pinned bool   `json:"pinned"`
		} `json:"project"`
	}
	decodeJSON(t, saved, &payload)
	if payload.Project.Name != "Calm editor" || !payload.Project.Pinned {
		t.Fatalf("saved project = %#v", payload.Project)
	}

	response, err = client.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("second GET / error = %v", err)
	}
	body = readBody(t, response)
	if !strings.Contains(body, `data-project-name="Calm editor"`) || !strings.Contains(body, `data-project-pinned="true"`) {
		t.Fatalf("persisted page body = %s", body)
	}
}

func TestReactIslandSaveRejectsInvalidRequests(t *testing.T) {
	server := httptest.NewServer(exampleHandler())
	t.Cleanup(server.Close)
	client := server.Client()

	response, err := client.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("GET / error = %v", err)
	}
	body := readBody(t, response)
	token := metaContent(t, body, csrf.MetaName)
	cookie := findCookie(t, response.Cookies(), csrf.DefaultCookieName)

	tests := []struct {
		name   string
		body   string
		token  string
		status int
		want   string
	}{
		{name: "missing csrf", body: `{"name":"A","pinned":false}`, status: http.StatusForbidden, want: `"error":"forbidden"`},
		{name: "empty name", body: `{"name":"   ","pinned":false}`, token: token, status: http.StatusUnprocessableEntity, want: `"name":"Enter a project name."`},
		{name: "unknown field", body: `{"name":"A","pinned":false,"extra":1}`, token: token, status: http.StatusBadRequest, want: `"error":"bad request"`},
		{name: "trailing value", body: `{"name":"A","pinned":false}{}`, token: token, status: http.StatusBadRequest, want: `"error":"bad request"`},
		{name: "oversized", body: `{"name":"` + strings.Repeat("A", 5<<10) + `","pinned":false}`, token: token, status: http.StatusBadRequest, want: `"error":"bad request"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := postJSON(t, client, server.URL+"/save", cookie, test.token, test.body)
			gotBody := readBody(t, got)
			if got.StatusCode != test.status || !strings.Contains(gotBody, test.want) {
				t.Fatalf("response = %d %s, want %d containing %q", got.StatusCode, gotBody, test.status, test.want)
			}
		})
	}
}

func postJSON(t *testing.T, client *http.Client, url string, cookie *http.Cookie, token, body string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set(csrf.HeaderName, token)
	}
	request.AddCookie(cookie)
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("POST /save error = %v", err)
	}
	return response
}

func readBody(t *testing.T, response *http.Response) string {
	t.Helper()
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	return string(body)
}

func decodeJSON(t *testing.T, response *http.Response, value any) {
	t.Helper()
	defer response.Body.Close()
	if err := json.NewDecoder(response.Body).Decode(value); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
}

func metaContent(t *testing.T, body, name string) string {
	t.Helper()
	prefix := `<meta name="` + name + `" content="`
	start := strings.Index(body, prefix)
	if start < 0 {
		t.Fatalf("meta %q not found in %s", name, body)
	}
	start += len(prefix)
	end := strings.Index(body[start:], `"`)
	if end < 0 {
		t.Fatalf("meta %q content not terminated", name)
	}
	return body[start : start+end]
}

func findCookie(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %q not found", name)
	return nil
}
