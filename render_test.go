package goldr

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
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
	redirect.Header().Set("Location", "/middleware")
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
	if err := WriteRouteResponse(
		redirect,
		request,
		Redirect{Location: "/sign-in", Status: http.StatusSeeOther}.
			WithHeader("Cache-Control", "no-store").
			WithHeader("Location", "/ignored").
			AddHeader("Location", "/also-ignored"),
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

func TestWriteRouteResponseAddHeaderPreservesMiddlewareCookie(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	http.SetCookie(recorder, &http.Cookie{Name: "middleware_cookie", Value: "synthetic"})
	response := Redirect{Location: "/next", Status: http.StatusSeeOther}.
		AddHeader("Set-Cookie", "route_cookie=synthetic; Path=/")

	if err := WriteRouteResponse(recorder, request, response); err != nil {
		t.Fatalf("WriteRouteResponse() error = %v, want nil", err)
	}
	result := recorder.Result()
	if result.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", result.StatusCode, http.StatusSeeOther)
	}
	want := []string{"middleware_cookie=synthetic", "route_cookie=synthetic; Path=/"}
	if got := result.Header.Values("Set-Cookie"); !slices.Equal(got, want) {
		t.Fatalf("Set-Cookie = %q, want %q", got, want)
	}
}

func TestWriteRouteResponseAddHeaderPreservesOtherResponseCookies(t *testing.T) {
	const routeCookie = "route_cookie=synthetic; Path=/"
	tests := []struct {
		name     string
		response RouteResponse
		status   int
	}{
		{"page", NewPage(templ.NopComponent, PageMetadata{}).AddHeader("Set-Cookie", routeCookie), http.StatusOK},
		{"fragment", NewFragment(templ.NopComponent).AddHeader("Set-Cookie", routeCookie), http.StatusOK},
		{"text", Text{Body: "ok"}.AddHeader("Set-Cookie", routeCookie), http.StatusOK},
		{"no content", NoContent{}.AddHeader("Set-Cookie", routeCookie), http.StatusNoContent},
	}
	request := WithRoutePageRenderer(
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil),
		func(_ *http.Request, page Page) (templ.Component, error) { return page.Component, nil },
	)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			http.SetCookie(recorder, &http.Cookie{Name: "middleware_cookie", Value: "synthetic"})
			recorder.Header().Set("X-Middleware", "keep")
			if err := WriteRouteResponse(recorder, request, test.response); err != nil {
				t.Fatalf("WriteRouteResponse() error = %v, want nil", err)
			}
			result := recorder.Result()
			if result.StatusCode != test.status {
				t.Fatalf("status = %d, want %d", result.StatusCode, test.status)
			}
			want := []string{"middleware_cookie=synthetic", routeCookie}
			if got := result.Header.Values("Set-Cookie"); !slices.Equal(got, want) {
				t.Fatalf("Set-Cookie = %q, want %q", got, want)
			}
			if got := result.Header.Get("X-Middleware"); got != "keep" {
				t.Fatalf("X-Middleware = %q, want keep", got)
			}
		})
	}
}

func TestWriteRouteResponseHeaderOperations(t *testing.T) {
	base := Redirect{Location: "/next", Status: http.StatusSeeOther}
	tests := []struct {
		name     string
		response Redirect
		want     []string
	}{
		{"none", base, []string{"middleware-one", "middleware-two"}},
		{"add", base.AddHeader("x-values", "a"), []string{"middleware-one", "middleware-two", "a"}},
		{"add add", base.AddHeader("X-Values", "a").AddHeader("x-values", "b"), []string{"middleware-one", "middleware-two", "a", "b"}},
		{"with", base.WithHeader("x-values", "a"), []string{"a"}},
		{"with add", base.WithHeader("X-Values", "a").AddHeader("x-values", "b"), []string{"a", "b"}},
		{"add with", base.AddHeader("X-Values", "a").WithHeader("x-values", "b"), []string{"b"}},
		{"with with", base.WithHeader("X-Values", "a").WithHeader("X-VALUES", "b"), []string{"b"}},
	}
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			recorder.Header().Add("X-Values", "middleware-one")
			recorder.Header().Add("X-Values", "middleware-two")
			if err := WriteRouteResponse(recorder, request, test.response); err != nil {
				t.Fatalf("WriteRouteResponse() error = %v, want nil", err)
			}
			if got := recorder.Result().Header.Values("X-Values"); !slices.Equal(got, test.want) {
				t.Fatalf("X-Values = %q, want %q", got, test.want)
			}
		})
	}
}

func TestWriteRouteResponseWithHeaderReplacesMiddlewareCookies(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	http.SetCookie(recorder, &http.Cookie{Name: "middleware_one", Value: "1"})
	http.SetCookie(recorder, &http.Cookie{Name: "middleware_two", Value: "2"})
	response := Redirect{Location: "/next", Status: http.StatusSeeOther}.
		AddHeader("Set-Cookie", "discarded=1").
		WithHeader("set-cookie", "session=1").
		AddHeader("SET-COOKIE", "csrf=2")
	if err := WriteRouteResponse(recorder, request, response); err != nil {
		t.Fatalf("WriteRouteResponse() error = %v, want nil", err)
	}
	want := []string{"session=1", "csrf=2"}
	if got := recorder.Result().Header.Values("Set-Cookie"); !slices.Equal(got, want) {
		t.Fatalf("Set-Cookie = %q, want %q", got, want)
	}
}

func TestWriteRouteResponseHeaderCopies(t *testing.T) {
	base := Redirect{Location: "/next", Status: http.StatusSeeOther}.
		WithHeader("X-Existing", "base").
		AddHeader("X-Values", "base")
	replaced := base.WithHeader("x-values", "replacement")
	appended := base.AddHeader("x-values", "appended")
	tests := []struct {
		name     string
		response Redirect
		want     []string
	}{
		{"base", base, []string{"middleware", "base"}},
		{"replaced", replaced, []string{"replacement"}},
		{"appended", appended, []string{"middleware", "base", "appended"}},
	}
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			recorder.Header().Set("X-Values", "middleware")
			recorder.Header().Set("X-Existing", "middleware")
			if err := WriteRouteResponse(recorder, request, test.response); err != nil {
				t.Fatalf("WriteRouteResponse() error = %v, want nil", err)
			}
			result := recorder.Result()
			if got := result.Header.Values("X-Values"); !slices.Equal(got, test.want) {
				t.Fatalf("X-Values = %q, want %q", got, test.want)
			}
			if got := result.Header.Values("X-Existing"); !slices.Equal(got, []string{"base"}) {
				t.Fatalf("X-Existing = %q, want [base]", got)
			}
		})
	}
}

func TestWriteRouteResponseWritesFragment(t *testing.T) {
	recorder := httptest.NewRecorder()
	recorder.Header().Set("Cache-Control", "public, max-age=60")
	recorder.Header().Set("Content-Type", "text/plain")
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
	if got := recorder.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/html; charset=utf-8", got)
	}
}

func TestWriteRouteResponseDoesNotApplyFragmentHeadersOnRenderError(t *testing.T) {
	recorder := httptest.NewRecorder()
	http.SetCookie(recorder, &http.Cookie{Name: "middleware", Value: "keep"})
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	componentErr := errors.New("render failed")

	err := WriteRouteResponse(
		recorder,
		request,
		NewFragment(templ.ComponentFunc(func(_ context.Context, writer io.Writer) error {
			_, _ = io.WriteString(writer, "partial")
			return componentErr
		})).WithHeader("Hx-Trigger", "fragment-loaded").
			WithHeader("Set-Cookie", "replacement=1").
			AddHeader("Set-Cookie", "appended=2"),
	)

	if !errors.Is(err, componentErr) {
		t.Fatalf("WriteRouteResponse(fragment) error = %v, want component error", err)
	}
	if got := recorder.Header().Get("Hx-Trigger"); got != "" {
		t.Fatalf("Hx-Trigger = %q, want empty", got)
	}
	if got := recorder.Header().Values("Set-Cookie"); !slices.Equal(got, []string{"middleware=keep"}) {
		t.Fatalf("Set-Cookie = %q, want [middleware=keep]", got)
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
