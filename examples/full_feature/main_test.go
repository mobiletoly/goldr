package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

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

	fragmentRequest, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+listener.Addr().String()+"/users/frag_table", nil)
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

	helperRequest, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://"+listener.Addr().String()+"/users/save-preview", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext(helper) error = %v", err)
	}
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
	if got := helperResponse.Header.Get(hx.HeaderRetarget); got != "#users-table" {
		t.Fatalf("helper %s = %q, want %q", hx.HeaderRetarget, got, "#users-table")
	}
	if got := helperResponse.Header.Get(hx.HeaderReswap); got != "outerHTML" {
		t.Fatalf("helper %s = %q, want %q", hx.HeaderReswap, got, "outerHTML")
	}
	helperBody, err := io.ReadAll(helperResponse.Body)
	if err != nil {
		t.Fatalf("ReadAll(helper) error = %v", err)
	}
	if !strings.Contains(string(helperBody), "User Table Fragment") {
		t.Fatalf("helper body = %q", helperBody)
	}

	createRequest, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://"+listener.Addr().String()+"/users/create", strings.NewReader("name=Hedy+Lamarr&status=Inactive"))
	if err != nil {
		t.Fatalf("NewRequestWithContext(create) error = %v", err)
	}
	createRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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

	assetRequest, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+listener.Addr().String()+"/assets/app.css", nil)
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
	if got := assetResponse.Header.Get("Cache-Control"); got != "public, max-age=60" {
		t.Fatalf("asset Cache-Control = %q, want %q", got, "public, max-age=60")
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
