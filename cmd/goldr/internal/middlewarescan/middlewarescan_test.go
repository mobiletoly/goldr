package middlewarescan

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestScanAcceptsMiddleware(t *testing.T) {
	path := writeMiddleware(t, `package routes

import "net/http"

func Middleware(next http.Handler) http.Handler {
	return next
}
`)

	if err := Scan(path); err != nil {
		t.Fatalf("Scan() error = %v, want nil", err)
	}
}

func TestScanReportsMissingMiddleware(t *testing.T) {
	path := writeMiddleware(t, `package routes

func helper() {}
`)

	err := Scan(path)
	var scanErr *ScanError
	if !errors.As(err, &scanErr) {
		t.Fatalf("Scan() error = %T, want *ScanError", err)
	}
	want := Problem{Function: FunctionName, Message: "missing Middleware function"}
	if len(scanErr.Problems) != 1 || scanErr.Problems[0] != want {
		t.Fatalf("problems = %#v, want %#v", scanErr.Problems, []Problem{want})
	}
}

func TestScanReportsInvalidMiddlewareSignature(t *testing.T) {
	path := writeMiddleware(t, `package routes

import "net/http"

func Middleware(next http.Handler) {}
`)

	err := Scan(path)
	var scanErr *ScanError
	if !errors.As(err, &scanErr) {
		t.Fatalf("Scan() error = %T, want *ScanError", err)
	}
	want := Problem{Function: FunctionName, Message: "middleware must use exact form func Middleware(next http.Handler) http.Handler with unaliased net/http import"}
	if len(scanErr.Problems) != 1 || scanErr.Problems[0] != want {
		t.Fatalf("problems = %#v, want %#v", scanErr.Problems, []Problem{want})
	}
}

func writeMiddleware(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), FileName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
