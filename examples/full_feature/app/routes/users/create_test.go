package users

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr/hx"
)

func TestPostCreateRedisplaysFieldErrors(t *testing.T) {
	resetContactsForTest()
	t.Cleanup(resetContactsForTest)

	body := strings.NewReader("name=&status=Missing")
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/users/create", body)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	PostCreate(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get(hx.HeaderRetarget); got != "#users-directory" {
		t.Fatalf("%s = %q, want #users-directory", hx.HeaderRetarget, got)
	}
	if got := recorder.Header().Get(hx.HeaderReswap); got != "outerHTML" {
		t.Fatalf("%s = %q, want outerHTML", hx.HeaderReswap, got)
	}
	response := recorder.Body.String()
	for _, want := range []string{"Name is required.", "Choose a valid status.", "User Table Fragment"} {
		if !strings.Contains(response, want) {
			t.Fatalf("body = %q, want %q", response, want)
		}
	}
	if got := recorder.Header().Get(hx.HeaderTrigger); got != "" {
		t.Fatalf("%s = %q, want empty", hx.HeaderTrigger, got)
	}
}

func TestPostCreateAddsContact(t *testing.T) {
	resetContactsForTest()
	t.Cleanup(resetContactsForTest)

	body := strings.NewReader("name=Hedy+Lamarr&status=Inactive")
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/users/create", body)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	PostCreate(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get(hx.HeaderTrigger); got != "user:created" {
		t.Fatalf("%s = %q, want user:created", hx.HeaderTrigger, got)
	}
	response := recorder.Body.String()
	for _, want := range []string{"Hedy Lamarr", "Inactive", "User Table Fragment"} {
		if !strings.Contains(response, want) {
			t.Fatalf("body = %q, want %q", response, want)
		}
	}
	if _, ok := ContactByID("43"); !ok {
		t.Fatal("ContactByID(43) ok = false, want true")
	}
}
