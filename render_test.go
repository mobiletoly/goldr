package goldr

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/a-h/templ"
)

func TestRenderBuffersHTMLResponse(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		wantBody string
	}{
		{name: "get", method: http.MethodGet, wantBody: "<p>Hello</p>"},
		{name: "head", method: http.MethodHead, wantBody: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequestWithContext(context.Background(), tt.method, "/", nil)

			response, err := Render(request, stringComponent("<p>Hello</p>"))

			if err != nil {
				t.Fatalf("Render() error = %v, want nil", err)
			}
			recorder := httptest.NewRecorder()
			if err := response.Write(recorder, request); err != nil {
				t.Fatalf("Write() error = %v, want nil", err)
			}
			if got := recorder.Result().StatusCode; got != http.StatusOK {
				t.Fatalf("status = %d, want %d", got, http.StatusOK)
			}
			if got := recorder.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
				t.Fatalf("content-type = %q, want text/html; charset=utf-8", got)
			}
			if got := recorder.Body.String(); got != tt.wantBody {
				t.Fatalf("body = %q, want %q", got, tt.wantBody)
			}
		})
	}
}

func TestRenderRejectsNilRequest(t *testing.T) {
	_, err := Render(nil, stringComponent("<p>Hello</p>"))
	if !errors.Is(err, ErrNilRequest) {
		t.Fatalf("Render() error = %v, want ErrNilRequest", err)
	}
}

func TestRenderRejectsNilComponent(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

	_, err := Render(request, nil)

	if !errors.Is(err, ErrNilComponent) {
		t.Fatalf("Render() error = %v, want ErrNilComponent", err)
	}
}

func TestRenderReturnsComponentErrorsWithoutWriting(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	componentErr := errors.New("render failed")

	_, err := Render(request, templ.ComponentFunc(func(_ context.Context, writer io.Writer) error {
		_, _ = io.WriteString(writer, "partial")
		return componentErr
	}))

	if !errors.Is(err, componentErr) {
		t.Fatalf("Render() error = %v, want component error", err)
	}
}

func TestHTMLResponseAllowsHeadersAfterRenderBeforeWrite(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	response, err := Render(request, stringComponent("<p>Hello</p>"))
	if err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	recorder := httptest.NewRecorder()
	recorder.Header().Set("HX-Trigger", "saved")
	if err := response.Write(recorder, request); err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}

	if got := recorder.Header().Get("HX-Trigger"); got != "saved" {
		t.Fatalf("HX-Trigger = %q, want saved", got)
	}
	if got := recorder.Body.String(); got != "<p>Hello</p>" {
		t.Fatalf("body = %q, want rendered HTML", got)
	}
}

func TestHTMLResponseWriteStatus(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		wantBody string
	}{
		{name: "get", method: http.MethodGet, wantBody: "<p>Missing</p>"},
		{name: "head", method: http.MethodHead, wantBody: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequestWithContext(context.Background(), tt.method, "/", nil)
			response, err := Render(request, stringComponent("<p>Missing</p>"))
			if err != nil {
				t.Fatalf("Render() error = %v, want nil", err)
			}

			recorder := httptest.NewRecorder()
			if err := response.WriteStatus(recorder, request, http.StatusNotFound); err != nil {
				t.Fatalf("WriteStatus() error = %v, want nil", err)
			}

			if got := recorder.Result().StatusCode; got != http.StatusNotFound {
				t.Fatalf("status = %d, want %d", got, http.StatusNotFound)
			}
			if got := recorder.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
				t.Fatalf("content-type = %q, want text/html; charset=utf-8", got)
			}
			if got := recorder.Body.String(); got != tt.wantBody {
				t.Fatalf("body = %q, want %q", got, tt.wantBody)
			}
		})
	}
}

func stringComponent(value string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, value)
		return err
	})
}
