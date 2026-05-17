package goldr

import (
	"errors"
	"net/http"
	"testing"

	"github.com/a-h/templ"
)

func TestNewPageRouteResponse(t *testing.T) {
	component := templ.NopComponent
	metadata := PageMetadata{Title: "Users"}

	response, err := ResolveRouteResponse(NewPage(component, metadata))

	if err != nil {
		t.Fatalf("ResolveRouteResponse() error = %v, want nil", err)
	}
	if response.Kind != RouteResponsePage {
		t.Fatalf("kind = %d, want %d", response.Kind, RouteResponsePage)
	}
	if response.Component == nil {
		t.Fatalf("component = nil, want component")
	}
	if response.Metadata != metadata {
		t.Fatalf("metadata = %#v, want %#v", response.Metadata, metadata)
	}
	if response.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Status, http.StatusOK)
	}
}

func TestPageWithStatus(t *testing.T) {
	response, err := ResolveRouteResponse(NewPage(templ.NopComponent, PageMetadata{}).WithStatus(http.StatusForbidden))
	if err != nil {
		t.Fatalf("ResolveRouteResponse() error = %v, want nil", err)
	}
	if response.Status != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Status, http.StatusForbidden)
	}
}

func TestNewFragmentRouteResponse(t *testing.T) {
	response, err := ResolveRouteResponse(
		NewFragment(templ.NopComponent).
			WithStatus(http.StatusAccepted).
			WithHeader("Hx-Trigger", "fragment-loaded"),
	)
	if err != nil {
		t.Fatalf("ResolveRouteResponse() error = %v, want nil", err)
	}
	if response.Kind != RouteResponseFragment {
		t.Fatalf("kind = %d, want %d", response.Kind, RouteResponseFragment)
	}
	if response.Component == nil {
		t.Fatalf("component = nil, want component")
	}
	if response.Status != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Status, http.StatusAccepted)
	}
	if got := response.Headers.Get("Hx-Trigger"); got != "fragment-loaded" {
		t.Fatalf("Hx-Trigger = %q, want fragment-loaded", got)
	}
}

func TestResolveRouteResponseAcceptsPointers(t *testing.T) {
	appErr := errors.New("load failed")
	metadata := PageMetadata{Title: "Users"}
	page := NewPage(templ.NopComponent, metadata)
	fragment := NewFragment(templ.NopComponent)
	redirect := Redirect{Location: "/sign-in", Status: http.StatusSeeOther}
	text := Text{Status: http.StatusForbidden, Body: "forbidden"}
	serverErr := ServerError{Err: appErr}

	tests := []struct {
		name     string
		response RouteResponse
		kind     RouteResponseKind
	}{
		{name: "page", response: &page, kind: RouteResponsePage},
		{name: "fragment", response: &fragment, kind: RouteResponseFragment},
		{name: "redirect", response: &redirect, kind: RouteResponseRedirect},
		{name: "text", response: &text, kind: RouteResponseText},
		{name: "server error", response: &serverErr, kind: RouteResponseServerError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := ResolveRouteResponse(test.response)
			if err != nil {
				t.Fatalf("ResolveRouteResponse() error = %v, want nil", err)
			}
			if response.Kind != test.kind {
				t.Fatalf("kind = %d, want %d", response.Kind, test.kind)
			}
		})
	}
}

func TestRouteResponseWithHeader(t *testing.T) {
	response, err := ResolveRouteResponse(
		NewPage(templ.NopComponent, PageMetadata{}).WithHeader("Cache-Control", "no-store"),
	)
	if err != nil {
		t.Fatalf("ResolveRouteResponse() error = %v, want nil", err)
	}
	if got := response.Headers.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}

	response.Headers.Set("Cache-Control", "mutated")
	again, err := ResolveRouteResponse(
		NewPage(templ.NopComponent, PageMetadata{}).WithHeader("Cache-Control", "no-store"),
	)
	if err != nil {
		t.Fatalf("ResolveRouteResponse() error = %v, want nil", err)
	}
	if got := again.Headers.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control after mutation = %q, want no-store", got)
	}
}

func TestRouteResponseAddHeader(t *testing.T) {
	response, err := ResolveRouteResponse(
		NewPage(templ.NopComponent, PageMetadata{}).
			AddHeader("Set-Cookie", "one=1").
			AddHeader("Set-Cookie", "two=2"),
	)
	if err != nil {
		t.Fatalf("ResolveRouteResponse() error = %v, want nil", err)
	}
	if got := response.Headers.Values("Set-Cookie"); len(got) != 2 || got[0] != "one=1" || got[1] != "two=2" {
		t.Fatalf("Set-Cookie values = %#v, want [one=1 two=2]", got)
	}

	response.Headers.Add("Set-Cookie", "mutated=1")
	again, err := ResolveRouteResponse(
		NewPage(templ.NopComponent, PageMetadata{}).
			AddHeader("Set-Cookie", "one=1").
			AddHeader("Set-Cookie", "two=2"),
	)
	if err != nil {
		t.Fatalf("ResolveRouteResponse() error = %v, want nil", err)
	}
	if got := again.Headers.Values("Set-Cookie"); len(got) != 2 || got[0] != "one=1" || got[1] != "two=2" {
		t.Fatalf("Set-Cookie values after mutation = %#v, want [one=1 two=2]", got)
	}
}

func TestRouteResponseValidation(t *testing.T) {
	var nilPage *Page
	var nilFragment *Fragment
	var nilRedirect *Redirect
	var nilText *Text
	var nilServerError *ServerError

	tests := []struct {
		name     string
		response RouteResponse
		want     error
	}{
		{name: "nil component", response: NewPage(nil, PageMetadata{}), want: ErrNilComponent},
		{name: "nil fragment component", response: NewFragment(nil), want: ErrNilComponent},
		{name: "bad redirect", response: Redirect{Location: "", Status: http.StatusSeeOther}, want: ErrInvalidRouteResponse},
		{name: "not modified redirect", response: Redirect{Location: "/sign-in", Status: http.StatusNotModified}, want: ErrInvalidRouteResponse},
		{name: "multiple choices redirect", response: Redirect{Location: "/sign-in", Status: http.StatusMultipleChoices}, want: ErrInvalidRouteResponse},
		{name: "bad text status", response: Text{Status: http.StatusFound, Body: "redirect"}, want: ErrInvalidRouteResponse},
		{name: "informational text status", response: Text{Status: http.StatusContinue, Body: "continue"}, want: ErrInvalidRouteResponse},
		{name: "no content text status", response: Text{Status: http.StatusNoContent, Body: "no content"}, want: ErrInvalidRouteResponse},
		{name: "reset content text status", response: Text{Status: http.StatusResetContent, Body: "reset"}, want: ErrInvalidRouteResponse},
		{name: "informational component status", response: NewPage(templ.NopComponent, PageMetadata{}).WithStatus(http.StatusSwitchingProtocols), want: ErrInvalidRouteResponse},
		{name: "no content component status", response: NewPage(templ.NopComponent, PageMetadata{}).WithStatus(http.StatusNoContent), want: ErrInvalidRouteResponse},
		{name: "no content fragment status", response: NewFragment(templ.NopComponent).WithStatus(http.StatusNoContent), want: ErrInvalidRouteResponse},
		{name: "nil server error", response: ServerError{}, want: ErrNilServerError},
		{name: "nil route response", response: nil, want: ErrInvalidRouteResponse},
		{name: "nil page pointer", response: nilPage, want: ErrInvalidRouteResponse},
		{name: "nil fragment pointer", response: nilFragment, want: ErrInvalidRouteResponse},
		{name: "nil redirect pointer", response: nilRedirect, want: ErrInvalidRouteResponse},
		{name: "nil text pointer", response: nilText, want: ErrInvalidRouteResponse},
		{name: "nil server error pointer", response: nilServerError, want: ErrInvalidRouteResponse},
		{name: "zero page", response: Page{}, want: ErrNilComponent},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := routeResponseError(test.response)
			if !errors.Is(got, test.want) {
				t.Fatalf("error = %v, want %v", got, test.want)
			}
		})
	}
}

func TestServerErrorRouteResponseCarriesApplicationError(t *testing.T) {
	appErr := errors.New("load failed")

	response, err := ResolveRouteResponse(ServerError{Err: appErr})
	if err != nil {
		t.Fatalf("ResolveRouteResponse() error = %v, want nil", err)
	}
	if response.Kind != RouteResponseServerError {
		t.Fatalf("kind = %d, want %d", response.Kind, RouteResponseServerError)
	}
	if !errors.Is(response.Error, appErr) {
		t.Fatalf("response error = %v, want %v", response.Error, appErr)
	}
}

func routeResponseError(response RouteResponse) error {
	_, err := ResolveRouteResponse(response)
	return err
}
