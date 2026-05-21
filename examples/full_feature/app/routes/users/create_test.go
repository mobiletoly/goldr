package users

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr/csrf"
	"github.com/mobiletoly/goldr/examples/full_feature/app/deps"
	"github.com/mobiletoly/goldr/examples/full_feature/app/security"
	"github.com/mobiletoly/goldr/examples/full_feature/internal/testcsrf"
	"github.com/mobiletoly/goldr/examples/full_feature/internal/testmultipart"
	"github.com/mobiletoly/goldr/hx"
)

func requestWithDependencies(request *http.Request) *http.Request {
	return deps.WithRequest(request, &deps.Dependencies{CSRF: security.CSRF})
}

func TestPostCreateRedisplaysFieldErrors(t *testing.T) {
	resetContactsForTest()
	t.Cleanup(resetContactsForTest)

	cookie, token := testcsrf.Pair(t, security.CSRF)
	body, contentType := testmultipart.Body(t, map[string]string{
		csrf.FieldName: token,
		"name":         "",
		"status":       "Missing",
	}, nil)
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/users/create", body)
	request.Header.Set("Content-Type", contentType)
	request.AddCookie(cookie)
	request = requestWithDependencies(request)
	recorder := httptest.NewRecorder()

	PostCreate(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnprocessableEntity)
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

	cookie, token := testcsrf.Pair(t, security.CSRF)
	body, contentType := testmultipart.Body(t, map[string]string{
		csrf.FieldName: token,
		"name":         "Hedy Lamarr",
		"status":       "Inactive",
	}, map[string]testmultipart.Upload{
		"avatar": {Filename: "hedy.txt", Content: "example avatar"},
	})
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/users/create", body)
	request.Header.Set("Content-Type", contentType)
	request.AddCookie(cookie)
	request = requestWithDependencies(request)
	recorder := httptest.NewRecorder()

	PostCreate(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get(hx.HeaderTrigger); got != "user:created" {
		t.Fatalf("%s = %q, want user:created", hx.HeaderTrigger, got)
	}
	response := recorder.Body.String()
	for _, want := range []string{"Hedy Lamarr", "Inactive", "hedy.txt", "User Table Fragment"} {
		if !strings.Contains(response, want) {
			t.Fatalf("body = %q, want %q", response, want)
		}
	}
	contact, ok := ContactByID("43")
	if !ok {
		t.Fatal("ContactByID(43) ok = false, want true")
	}
	if contact.AvatarFilename != "hedy.txt" {
		t.Fatalf("AvatarFilename = %q, want %q", contact.AvatarFilename, "hedy.txt")
	}
}

func TestPostCreateRejectsMissingCSRF(t *testing.T) {
	resetContactsForTest()
	t.Cleanup(resetContactsForTest)

	body, contentType := testmultipart.Body(t, map[string]string{
		"name":   "Hedy Lamarr",
		"status": "Inactive",
	}, nil)
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/users/create", body)
	request.Header.Set("Content-Type", contentType)
	request = requestWithDependencies(request)
	recorder := httptest.NewRecorder()

	PostCreate(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}
