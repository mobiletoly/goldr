package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mobiletoly/goldr/csrf"
	"github.com/mobiletoly/goldr/examples/full_feature/assets"
	"github.com/mobiletoly/goldr/examples/full_feature/internal/testmultipart"
	"github.com/mobiletoly/goldr/hx"
)

func TestExampleAppServesRootPageOverHTTP(t *testing.T) {
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}

	server := &http.Server{
		Handler:           exampleHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		_ = server.Close()
	})

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+listener.Addr().String()+"/users/42", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}

	client := http.Client{Timeout: 5 * time.Second}
	baseURL := "http://" + listener.Addr().String()
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if got := response.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("route X-Content-Type-Options = %q, want %q", got, "nosniff")
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !strings.Contains(string(body), "Ada Lovelace") {
		t.Fatalf("body = %q", body)
	}
	if !strings.Contains(string(body), "Goldr Example") {
		t.Fatalf("body = %q", body)
	}
	for _, want := range []string{
		`<title>Ada Lovelace - Goldr Example</title>`,
		`<a href="/users" aria-current="page">Users</a>`,
	} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("body = %q, want %q", body, want)
		}
	}
	if strings.Contains(string(body), `rel="canonical"`) {
		t.Fatalf("body = %q, want no canonical link", body)
	}
	for _, want := range []string{`href="/users"`, `href="/settings"`} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("body = %q, want %q", body, want)
		}
	}
	if !strings.Contains(string(body), "people section shell") {
		t.Fatalf("body = %q", body)
	}

	settingsRequest, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+listener.Addr().String()+"/settings", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext(settings) error = %v", err)
	}
	settingsResponse, err := client.Do(settingsRequest)
	if err != nil {
		t.Fatalf("Do(settings) error = %v", err)
	}
	defer func() {
		if err := settingsResponse.Body.Close(); err != nil {
			t.Errorf("Close(settings) error = %v", err)
		}
	}()
	if settingsResponse.StatusCode != http.StatusOK {
		t.Fatalf("settings status = %d, want %d", settingsResponse.StatusCode, http.StatusOK)
	}
	settingsBody, err := io.ReadAll(settingsResponse.Body)
	if err != nil {
		t.Fatalf("ReadAll(settings) error = %v", err)
	}
	if !strings.Contains(string(settingsBody), "Settings") {
		t.Fatalf("settings body = %q", settingsBody)
	}
	for _, want := range []string{
		`<title>Settings - Goldr Example</title>`,
		`<a href="/settings" aria-current="page">Settings</a>`,
	} {
		if !strings.Contains(string(settingsBody), want) {
			t.Fatalf("settings body = %q, want %q", settingsBody, want)
		}
	}
	if strings.Contains(string(settingsBody), `rel="canonical"`) {
		t.Fatalf("settings body = %q, want no canonical link", settingsBody)
	}

	fragmentRequest, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+listener.Addr().String()+"/users/frag-table", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}
	fragmentResponse, err := client.Do(fragmentRequest)
	if err != nil {
		t.Fatalf("Do(fragment) error = %v", err)
	}
	defer func() {
		if err := fragmentResponse.Body.Close(); err != nil {
			t.Errorf("Close(fragment) error = %v", err)
		}
	}()
	if fragmentResponse.StatusCode != http.StatusOK {
		t.Fatalf("fragment status = %d, want %d", fragmentResponse.StatusCode, http.StatusOK)
	}
	fragmentBody, err := io.ReadAll(fragmentResponse.Body)
	if err != nil {
		t.Fatalf("ReadAll(fragment) error = %v", err)
	}
	if !strings.Contains(string(fragmentBody), "User Table Fragment") {
		t.Fatalf("fragment body = %q", fragmentBody)
	}
	if strings.Contains(string(fragmentBody), "Goldr Example") {
		t.Fatalf("fragment body = %q", fragmentBody)
	}

	csrfCookie, csrfToken := fetchUserCSRF(t, client, baseURL)

	helperRequest, err := http.NewRequestWithContext(context.Background(), http.MethodPost, baseURL+"/users/save-preview", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext(helper) error = %v", err)
	}
	helperRequest.AddCookie(csrfCookie)
	helperRequest.Header.Set(csrf.HeaderName, csrfToken)
	helperResponse, err := client.Do(helperRequest)
	if err != nil {
		t.Fatalf("Do(helper) error = %v", err)
	}
	defer func() {
		if err := helperResponse.Body.Close(); err != nil {
			t.Errorf("Close(helper) error = %v", err)
		}
	}()
	if helperResponse.StatusCode != http.StatusOK {
		t.Fatalf("helper status = %d, want %d", helperResponse.StatusCode, http.StatusOK)
	}
	if got := helperResponse.Header.Get(hx.HeaderTrigger); got != "user:saved" {
		t.Fatalf("helper %s = %q, want %q", hx.HeaderTrigger, got, "user:saved")
	}
	if got := helperResponse.Header.Get(hx.HeaderRetarget); got != "#users-table-slot" {
		t.Fatalf("helper %s = %q, want %q", hx.HeaderRetarget, got, "#users-table-slot")
	}
	if got := helperResponse.Header.Get(hx.HeaderReswap); got != "innerHTML" {
		t.Fatalf("helper %s = %q, want %q", hx.HeaderReswap, got, "innerHTML")
	}
	helperBody, err := io.ReadAll(helperResponse.Body)
	if err != nil {
		t.Fatalf("ReadAll(helper) error = %v", err)
	}
	if !strings.Contains(string(helperBody), "User Table Fragment") {
		t.Fatalf("helper body = %q", helperBody)
	}

	createBodyReader, createContentType := testmultipart.Body(t, map[string]string{
		csrf.FieldName: csrfToken,
		"name":         "Hedy Lamarr",
		"status":       "Inactive",
	}, map[string]testmultipart.Upload{
		"avatar": {Filename: "hedy.txt", Content: "example avatar"},
	})
	createRequest, err := http.NewRequestWithContext(context.Background(), http.MethodPost, baseURL+"/users/create", createBodyReader)
	if err != nil {
		t.Fatalf("NewRequestWithContext(create) error = %v", err)
	}
	createRequest.Header.Set("Content-Type", createContentType)
	createRequest.AddCookie(csrfCookie)
	createResponse, err := client.Do(createRequest)
	if err != nil {
		t.Fatalf("Do(create) error = %v", err)
	}
	defer func() {
		if err := createResponse.Body.Close(); err != nil {
			t.Errorf("Close(create) error = %v", err)
		}
	}()
	if createResponse.StatusCode != http.StatusOK {
		t.Fatalf("create status = %d, want %d", createResponse.StatusCode, http.StatusOK)
	}
	if got := createResponse.Header.Get(hx.HeaderTrigger); got != "user:created" {
		t.Fatalf("create %s = %q, want %q", hx.HeaderTrigger, got, "user:created")
	}
	createBody, err := io.ReadAll(createResponse.Body)
	if err != nil {
		t.Fatalf("ReadAll(create) error = %v", err)
	}
	if !strings.Contains(string(createBody), "Hedy Lamarr") {
		t.Fatalf("create body = %q", createBody)
	}
	if !strings.Contains(string(createBody), "hedy.txt") {
		t.Fatalf("create body = %q", createBody)
	}

	assetRequest, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+listener.Addr().String()+assets.Path("app.css"), nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext(asset) error = %v", err)
	}
	assetResponse, err := client.Do(assetRequest)
	if err != nil {
		t.Fatalf("Do(asset) error = %v", err)
	}
	defer func() {
		if err := assetResponse.Body.Close(); err != nil {
			t.Errorf("Close(asset) error = %v", err)
		}
	}()
	if assetResponse.StatusCode != http.StatusOK {
		t.Fatalf("asset status = %d, want %d", assetResponse.StatusCode, http.StatusOK)
	}
	if got := assetResponse.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/css") {
		t.Fatalf("asset content-type = %q, want text/css", got)
	}
	if got := assetResponse.Header.Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("asset Cache-Control = %q, want %q", got, "public, max-age=31536000, immutable")
	}

	jsRequest, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+listener.Addr().String()+assets.Path("app.js"), nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext(js) error = %v", err)
	}
	jsResponse, err := client.Do(jsRequest)
	if err != nil {
		t.Fatalf("Do(js) error = %v", err)
	}
	defer func() {
		if err := jsResponse.Body.Close(); err != nil {
			t.Errorf("Close(js) error = %v", err)
		}
	}()
	if jsResponse.StatusCode != http.StatusOK {
		t.Fatalf("js status = %d, want %d", jsResponse.StatusCode, http.StatusOK)
	}
	if got := jsResponse.Header.Get("Content-Type"); !strings.Contains(got, "javascript") {
		t.Fatalf("js content-type = %q, want javascript", got)
	}
	if got := jsResponse.Header.Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("js Cache-Control = %q, want %q", got, "public, max-age=31536000, immutable")
	}
	jsBody, err := io.ReadAll(jsResponse.Body)
	if err != nil {
		t.Fatalf("ReadAll(js) error = %v", err)
	}
	if !strings.Contains(string(jsBody), `dataset.goldrJs = "ready"`) {
		t.Fatalf("js body = %q", jsBody)
	}
	if strings.Contains(string(jsBody), "htmx:beforeSwap") {
		t.Fatalf("js body = %q, want htmx 4 validation responses without custom swap handling", jsBody)
	}

	missingRequest, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+listener.Addr().String()+"/missing", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext(missing) error = %v", err)
	}
	missingResponse, err := client.Do(missingRequest)
	if err != nil {
		t.Fatalf("Do(missing) error = %v", err)
	}
	defer func() {
		if err := missingResponse.Body.Close(); err != nil {
			t.Errorf("Close(missing) error = %v", err)
		}
	}()
	if missingResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("missing status = %d, want %d", missingResponse.StatusCode, http.StatusNotFound)
	}
	if got := missingResponse.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("missing X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
	missingBody, err := io.ReadAll(missingResponse.Body)
	if err != nil {
		t.Fatalf("ReadAll(missing) error = %v", err)
	}
	if !strings.Contains(string(missingBody), "Page not found") {
		t.Fatalf("missing body = %q", missingBody)
	}
	if !strings.Contains(string(missingBody), "Goldr Example") {
		t.Fatalf("missing body = %q", missingBody)
	}
}

func fetchUserCSRF(t *testing.T, client http.Client, baseURL string) (*http.Cookie, string) {
	t.Helper()

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/users", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext(csrf) error = %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do(csrf) error = %v", err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("Close(csrf) error = %v", err)
		}
	}()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("csrf status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("ReadAll(csrf) error = %v", err)
	}
	token := hiddenCSRFToken(t, string(body))
	for _, cookie := range response.Cookies() {
		if cookie.Name == csrf.DefaultCookieName {
			return cookie, token
		}
	}
	t.Fatalf("CSRF cookie %q not found", csrf.DefaultCookieName)
	return nil, ""
}

func hiddenCSRFToken(t *testing.T, body string) string {
	t.Helper()

	marker := `name="` + csrf.FieldName + `" value="`
	start := strings.Index(body, marker)
	if start < 0 {
		t.Fatalf("body = %q, want CSRF hidden input", body)
	}
	start += len(marker)
	end := strings.Index(body[start:], `"`)
	if end < 0 {
		t.Fatalf("body = %q, want CSRF token value", body)
	}
	token := body[start : start+end]
	if token == "" {
		t.Fatalf("body = %q, want non-empty CSRF token", body)
	}
	return token
}
