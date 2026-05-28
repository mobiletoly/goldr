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

func TestWriteComponent(t *testing.T) {
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
			recorder := httptest.NewRecorder()
			recorder.Header().Set("Hx-Trigger", "saved")

			err := WriteComponent(recorder, request, http.StatusAccepted, stringComponent("<p>Hello</p>"))

			if err != nil {
				t.Fatalf("WriteComponent() error = %v, want nil", err)
			}
			if got := recorder.Result().StatusCode; got != http.StatusAccepted {
				t.Fatalf("status = %d, want %d", got, http.StatusAccepted)
			}
			if got := recorder.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
				t.Fatalf("content-type = %q, want text/html; charset=utf-8", got)
			}
			if got := recorder.Header().Get("Hx-Trigger"); got != "saved" {
				t.Fatalf("HX-Trigger = %q, want saved", got)
			}
			if got := recorder.Body.String(); got != tt.wantBody {
				t.Fatalf("body = %q, want %q", got, tt.wantBody)
			}
		})
	}
}

func TestWriteComponentRejectsInvalidInputs(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	tests := []struct {
		name      string
		writer    http.ResponseWriter
		request   *http.Request
		status    int
		component templ.Component
		want      error
	}{
		{name: "nil writer", writer: nil, request: request, status: http.StatusOK, component: templ.NopComponent, want: ErrNilResponseWriter},
		{name: "nil request", writer: httptest.NewRecorder(), request: nil, status: http.StatusOK, component: templ.NopComponent, want: ErrNilRequest},
		{name: "nil component", writer: httptest.NewRecorder(), request: request, status: http.StatusOK, component: nil, want: ErrNilComponent},
		{name: "no content status", writer: httptest.NewRecorder(), request: request, status: http.StatusNoContent, component: templ.NopComponent, want: ErrInvalidHTMLStatus},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := WriteComponent(test.writer, test.request, test.status, test.component)
			if !errors.Is(err, test.want) {
				t.Fatalf("WriteComponent() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestWriteComponentReturnsComponentErrorsWithoutWriting(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	componentErr := errors.New("render failed")
	recorder := httptest.NewRecorder()

	err := WriteComponent(recorder, request, http.StatusOK, templ.ComponentFunc(func(_ context.Context, writer io.Writer) error {
		_, _ = io.WriteString(writer, "partial")
		return componentErr
	}))

	if !errors.Is(err, componentErr) {
		t.Fatalf("WriteComponent() error = %v, want component error", err)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want implicit 200 because response was not committed", recorder.Code)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", recorder.Body.String())
	}
}

func TestWriteRouteResponseRequiresRoutePageRendererForPage(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	err := WriteRouteResponse(recorder, request, NewPage(templ.NopComponent, PageMetadata{}))

	if !errors.Is(err, ErrRoutePageRendererUnavailable) {
		t.Fatalf("WriteRouteResponse() error = %v, want ErrRoutePageRendererUnavailable", err)
	}
}

func TestWriteRouteResponseUsesRoutePageRendererForPage(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
	request = WithRoutePageRenderer(request, func(_ *http.Request, page Page) (templ.Component, error) {
		if page.Metadata.Title != "Created" {
			t.Fatalf("metadata title = %q, want Created", page.Metadata.Title)
		}
		return stringComponent("<main>Created</main>"), nil
	})
	recorder := httptest.NewRecorder()

	err := WriteRouteResponse(
		recorder,
		request,
		NewPage(templ.NopComponent, PageMetadata{Title: "Created"}).
			WithStatus(http.StatusCreated).
			WithHeader("Cache-Control", "no-store"),
	)

	if err != nil {
		t.Fatalf("WriteRouteResponse(page) error = %v, want nil", err)
	}
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	if got := recorder.Body.String(); got != "<main>Created</main>" {
		t.Fatalf("body = %q, want rendered page", got)
	}
}

func TestWritePageRouteResponseWritesRedirectAndText(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

	redirect := httptest.NewRecorder()
	if err := WritePageRouteResponse(
		redirect,
		request,
		Redirect{Location: "/sign-in", Status: http.StatusSeeOther}.
			WithHeader("Cache-Control", "no-store"),
		func(_ *http.Request, page Page) (templ.Component, error) {
			t.Fatalf("page renderer called for redirect response")
			return page.Component, nil
		},
	); err != nil {
		t.Fatalf("WritePageRouteResponse(redirect) error = %v, want nil", err)
	}
	if redirect.Code != http.StatusSeeOther || redirect.Header().Get("Location") != "/sign-in" {
		t.Fatalf("redirect = (%d, %q), want 303 /sign-in", redirect.Code, redirect.Header().Get("Location"))
	}
	if got := redirect.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("redirect Cache-Control = %q, want no-store", got)
	}

	text := httptest.NewRecorder()
	if err := WritePageRouteResponse(
		text,
		request,
		Text{Status: http.StatusForbidden, Body: "forbidden"}.
			WithHeader("X-Robots-Tag", "noindex"),
		func(_ *http.Request, page Page) (templ.Component, error) {
			t.Fatalf("page renderer called for text response")
			return page.Component, nil
		},
	); err != nil {
		t.Fatalf("WritePageRouteResponse(text) error = %v, want nil", err)
	}
	if text.Code != http.StatusForbidden || text.Body.String() != "forbidden" {
		t.Fatalf("text = (%d, %q), want 403 forbidden", text.Code, text.Body.String())
	}
	if got := text.Header().Get("X-Robots-Tag"); got != "noindex" {
		t.Fatalf("text X-Robots-Tag = %q, want noindex", got)
	}
}

func TestWriteFragmentRouteResponseWritesRedirectAndText(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

	redirect := httptest.NewRecorder()
	if err := WriteFragmentRouteResponse(
		redirect,
		request,
		Redirect{Location: "/sign-in", Status: http.StatusSeeOther}.
			WithHeader("Cache-Control", "no-store"),
	); err != nil {
		t.Fatalf("WriteFragmentRouteResponse(redirect) error = %v, want nil", err)
	}
	if redirect.Code != http.StatusSeeOther || redirect.Header().Get("Location") != "/sign-in" {
		t.Fatalf("redirect = (%d, %q), want 303 /sign-in", redirect.Code, redirect.Header().Get("Location"))
	}
	if got := redirect.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("redirect Cache-Control = %q, want no-store", got)
	}

	text := httptest.NewRecorder()
	if err := WriteFragmentRouteResponse(
		text,
		request,
		Text{Status: http.StatusForbidden, Body: "forbidden"}.
			WithHeader("X-Robots-Tag", "noindex"),
	); err != nil {
		t.Fatalf("WriteFragmentRouteResponse(text) error = %v, want nil", err)
	}
	if text.Code != http.StatusForbidden || text.Body.String() != "forbidden" {
		t.Fatalf("text = (%d, %q), want 403 forbidden", text.Code, text.Body.String())
	}
	if got := text.Header().Get("X-Robots-Tag"); got != "noindex" {
		t.Fatalf("text X-Robots-Tag = %q, want noindex", got)
	}
}

func TestWriteEndpointResponsesDelegateRouteError(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	appErr := errors.New("product not found")

	page := httptest.NewRecorder()
	err := WritePageRouteResponse(
		page,
		request,
		RouteError{Err: appErr},
		func(_ *http.Request, page Page) (templ.Component, error) {
			t.Fatalf("page renderer called for route error")
			return page.Component, nil
		},
	)
	if !errors.Is(err, appErr) {
		t.Fatalf("WritePageRouteResponse(RouteError) error = %v, want %v", err, appErr)
	}
	if page.Body.Len() != 0 {
		t.Fatalf("page body = %q, want empty", page.Body.String())
	}

	fragment := httptest.NewRecorder()
	err = WriteFragmentRouteResponse(fragment, request, RouteError{Err: appErr})
	if !errors.Is(err, appErr) {
		t.Fatalf("WriteFragmentRouteResponse(RouteError) error = %v, want %v", err, appErr)
	}
	if fragment.Body.Len() != 0 {
		t.Fatalf("fragment body = %q, want empty", fragment.Body.String())
	}

	route := httptest.NewRecorder()
	err = WriteRouteResponse(route, request, RouteError{Err: appErr})
	if !errors.Is(err, appErr) {
		t.Fatalf("WriteRouteResponse(RouteError) error = %v, want %v", err, appErr)
	}
	if route.Body.Len() != 0 {
		t.Fatalf("route body = %q, want empty", route.Body.String())
	}
}

func TestWriteRouteResponseWritesRedirectAndText(t *testing.T) {
	redirect := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
	if err := WriteRouteResponse(
		redirect,
		request,
		Redirect{Location: "/sign-in", Status: http.StatusSeeOther}.WithHeader("Cache-Control", "no-store"),
	); err != nil {
		t.Fatalf("WriteRouteResponse(redirect) error = %v, want nil", err)
	}
	if redirect.Code != http.StatusSeeOther || redirect.Header().Get("Location") != "/sign-in" {
		t.Fatalf("redirect = (%d, %q), want 303 /sign-in", redirect.Code, redirect.Header().Get("Location"))
	}
	if got := redirect.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("redirect Cache-Control = %q, want no-store", got)
	}

	text := httptest.NewRecorder()
	if err := WriteRouteResponse(
		text,
		request,
		Text{Status: http.StatusForbidden, Body: "forbidden"}.WithHeader("X-Robots-Tag", "noindex"),
	); err != nil {
		t.Fatalf("WriteRouteResponse(text) error = %v, want nil", err)
	}
	if text.Code != http.StatusForbidden || text.Body.String() != "forbidden" {
		t.Fatalf("text = (%d, %q), want 403 forbidden", text.Code, text.Body.String())
	}
	if got := text.Header().Get("X-Robots-Tag"); got != "noindex" {
		t.Fatalf("text X-Robots-Tag = %q, want noindex", got)
	}

	defaultText := httptest.NewRecorder()
	if err := WriteRouteResponse(defaultText, request, Text{Body: "saved"}); err != nil {
		t.Fatalf("WriteRouteResponse(default text) error = %v, want nil", err)
	}
	if defaultText.Code != http.StatusOK || defaultText.Body.String() != "saved" {
		t.Fatalf("default text = (%d, %q), want 200 saved", defaultText.Code, defaultText.Body.String())
	}

	csv := httptest.NewRecorder()
	if err := WriteRouteResponse(
		csv,
		request,
		Text{Status: http.StatusOK, Body: "id,name\n1,Ada\n"}.WithHeader("Content-Type", "text/csv; charset=utf-8"),
	); err != nil {
		t.Fatalf("WriteRouteResponse(csv text) error = %v, want nil", err)
	}
	if got := csv.Header().Get("Content-Type"); got != "text/csv; charset=utf-8" {
		t.Fatalf("csv Content-Type = %q, want text/csv; charset=utf-8", got)
	}

	noContent := httptest.NewRecorder()
	if err := WriteRouteResponse(
		noContent,
		request,
		NoContent{}.WithHeader("Hx-Trigger", "saved"),
	); err != nil {
		t.Fatalf("WriteRouteResponse(no content) error = %v, want nil", err)
	}
	if noContent.Code != http.StatusNoContent || noContent.Body.Len() != 0 {
		t.Fatalf("no content = (%d, %q), want 204 empty", noContent.Code, noContent.Body.String())
	}
	if got := noContent.Header().Get("Hx-Trigger"); got != "saved" {
		t.Fatalf("no content Hx-Trigger = %q, want saved", got)
	}
}

func TestWriteRouteResponseWritesFragment(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

	err := WriteRouteResponse(
		recorder,
		request,
		NewFragment(stringComponent("<tbody>Users</tbody>")).
			WithStatus(http.StatusAccepted).
			WithHeader("Hx-Trigger", "fragment-loaded"),
	)

	if err != nil {
		t.Fatalf("WriteRouteResponse(fragment) error = %v, want nil", err)
	}
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusAccepted)
	}
	if recorder.Body.String() != "<tbody>Users</tbody>" {
		t.Fatalf("body = %q, want fragment body", recorder.Body.String())
	}
	if got := recorder.Header().Get("Hx-Trigger"); got != "fragment-loaded" {
		t.Fatalf("Hx-Trigger = %q, want fragment-loaded", got)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
}

func TestWriteRouteResponseDoesNotApplyFragmentHeadersOnRenderError(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	componentErr := errors.New("render failed")

	err := WriteRouteResponse(
		recorder,
		request,
		NewFragment(templ.ComponentFunc(func(_ context.Context, writer io.Writer) error {
			_, _ = io.WriteString(writer, "partial")
			return componentErr
		})).WithHeader("Hx-Trigger", "fragment-loaded"),
	)

	if !errors.Is(err, componentErr) {
		t.Fatalf("WriteRouteResponse(fragment) error = %v, want component error", err)
	}
	if got := recorder.Header().Get("Hx-Trigger"); got != "" {
		t.Fatalf("Hx-Trigger = %q, want empty", got)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want implicit 200 because response was not committed", recorder.Code)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", recorder.Body.String())
	}
}

func stringComponent(value string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, value)
		return err
	})
}
