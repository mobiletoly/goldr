package testutil

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"

	"github.com/mobiletoly/goldr/csrf"
)

type MultipartUpload struct {
	Filename string
	Content  string
}

type TB interface {
	Helper()
	Fatalf(format string, args ...any)
}

func CSRFPair(t TB, guard *csrf.Guard) (*http.Cookie, string) {
	t.Helper()

	recorder := httptest.NewRecorder()
	handler := guard.TokenMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(guard.Token(r)))
	}))
	handler.ServeHTTP(recorder, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil))

	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == csrf.DefaultCookieName {
			return cookie, recorder.Body.String()
		}
	}
	t.Fatalf("CSRF cookie %q not found", csrf.DefaultCookieName)
	return nil, ""
}

func MultipartBody(t TB, fields map[string]string, files map[string]MultipartUpload) (*bytes.Buffer, string) {
	t.Helper()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatalf("WriteField() error = %v", err)
		}
	}
	for name, upload := range files {
		part, err := writer.CreateFormFile(name, upload.Filename)
		if err != nil {
			t.Fatalf("CreateFormFile() error = %v", err)
		}
		if _, err := part.Write([]byte(upload.Content)); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return body, writer.FormDataContentType()
}
