package actionscan

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestScanDiscoversSupportedActions(t *testing.T) {
	path := writeActions(t, `package users

import "net/http"

func PostIndex(w http.ResponseWriter, r *http.Request) {}
func PutCreate(w http.ResponseWriter, r *http.Request) {}
func PatchSavePreview(w http.ResponseWriter, r *http.Request) {}
func DeleteAvatar(w http.ResponseWriter, r *http.Request) {}
func helper() {}
func PostHelper(w http.ResponseWriter, r *http.Request) { _ = "body ignored" }
`)

	got, err := Scan(path)
	if err != nil {
		t.Fatalf("Scan() error = %v, want nil", err)
	}

	want := []Action{
		{Method: "POST", Function: "PostIndex", Suffix: "Index"},
		{Method: "PUT", Function: "PutCreate", Suffix: "Create", Segment: "create"},
		{Method: "PATCH", Function: "PatchSavePreview", Suffix: "SavePreview", Segment: "save-preview"},
		{Method: "DELETE", Function: "DeleteAvatar", Suffix: "Avatar", Segment: "avatar"},
		{Method: "POST", Function: "PostHelper", Suffix: "Helper", Segment: "helper"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Scan() = %#v, want %#v", got, want)
	}
}

func TestScanUsesDeterministicAcronymSegments(t *testing.T) {
	path := writeActions(t, `package users

import "net/http"

func PostURLValue(w http.ResponseWriter, r *http.Request) {}
func PostSaveHTTPPreview(w http.ResponseWriter, r *http.Request) {}
func PostVersion2URL(w http.ResponseWriter, r *http.Request) {}
`)

	got, err := Scan(path)
	if err != nil {
		t.Fatalf("Scan() error = %v, want nil", err)
	}

	wantSegments := []string{"url-value", "save-http-preview", "version2-url"}
	for index, want := range wantSegments {
		if got[index].Segment != want {
			t.Fatalf("action %d segment = %q, want %q", index, got[index].Segment, want)
		}
	}
}

func TestScanReportsMalformedActionNames(t *testing.T) {
	path := writeActions(t, `package users

import "net/http"

func Post(w http.ResponseWriter, r *http.Request) {}
func Postcreate(w http.ResponseWriter, r *http.Request) {}
func Patch_Save(w http.ResponseWriter, r *http.Request) {}
func Delete_User(w http.ResponseWriter, r *http.Request) {}
`)

	_, err := Scan(path)
	var scanErr *ScanError
	if !errors.As(err, &scanErr) {
		t.Fatalf("Scan() error = %T, want *ScanError", err)
	}
	if len(scanErr.Problems) != 4 {
		t.Fatalf("problems = %#v, want 4", scanErr.Problems)
	}
}

func TestScanReportsInvalidSignatures(t *testing.T) {
	path := writeActions(t, `package users

import "net/http"

func PostNoParams() {}
func PostOneParam(w http.ResponseWriter) {}
func PostWrongResponse(w *http.ResponseWriter, r *http.Request) {}
func PostWrongRequest(w http.ResponseWriter, r http.Request) {}
func PostReturn(w http.ResponseWriter, r *http.Request) error { return nil }
`)

	_, err := Scan(path)
	var scanErr *ScanError
	if !errors.As(err, &scanErr) {
		t.Fatalf("Scan() error = %T, want *ScanError", err)
	}
	if len(scanErr.Problems) != 5 {
		t.Fatalf("problems = %#v, want 5", scanErr.Problems)
	}
	for _, problem := range scanErr.Problems {
		if problem.Message != "action handlers must use func Name(w http.ResponseWriter, r *http.Request)" {
			t.Fatalf("problem = %#v, want signature message", problem)
		}
	}
}

func TestScanRejectsGetActions(t *testing.T) {
	path := writeActions(t, `package users

import "net/http"

func GetCreate(w http.ResponseWriter, r *http.Request) {}
`)

	_, err := Scan(path)
	var scanErr *ScanError
	if !errors.As(err, &scanErr) {
		t.Fatalf("Scan() error = %T, want *ScanError", err)
	}
	if got := scanErr.Problems[0].Message; got != "GET action handlers are not supported; pages and fragments own GET and HEAD" {
		t.Fatalf("message = %q", got)
	}
}

func writeActions(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), FileName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
