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

	response, err := resolveRouteResponse(NewPage(component, metadata))

	if err != nil {
		t.Fatalf("resolveRouteResponse() error = %v, want nil", err)
	}
	if response.kind != routeResponsePage {
		t.Fatalf("kind = %d, want %d", response.kind, routeResponsePage)
	}
	if response.page.Component == nil {
		t.Fatalf("component = nil, want component")
	}
	if response.page.Metadata != metadata {
		t.Fatalf("metadata = %#v, want %#v", response.page.Metadata, metadata)
	}
	if response.page.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.page.Status, http.StatusOK)
	}
}

func TestPageWithStatus(t *testing.T) {
	response, err := resolveRouteResponse(NewPage(templ.NopComponent, PageMetadata{}).WithStatus(http.StatusForbidden))
	if err != nil {
		t.Fatalf("resolveRouteResponse() error = %v, want nil", err)
	}
	if response.page.Status != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.page.Status, http.StatusForbidden)
	}
}

func TestNewFragmentRouteResponse(t *testing.T) {
	response, err := resolveRouteResponse(
		NewFragment(templ.NopComponent).
			WithStatus(http.StatusAccepted).
			WithHeader("Hx-Trigger", "fragment-loaded"),
	)
	if err != nil {
		t.Fatalf("resolveRouteResponse() error = %v, want nil", err)
	}
	if response.kind != routeResponseFragment {
		t.Fatalf("kind = %d, want %d", response.kind, routeResponseFragment)
	}
	if response.fragment.Component == nil {
		t.Fatalf("component = nil, want component")
	}
	if response.fragment.Status != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.fragment.Status, http.StatusAccepted)
	}
	if got := response.fragment.headers.Get("Hx-Trigger"); got != "fragment-loaded" {
		t.Fatalf("Hx-Trigger = %q, want fragment-loaded", got)
	}
	if got := response.fragment.headers.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
}

func TestFragmentCacheControlCanBeOverridden(t *testing.T) {
	response, err := resolveRouteResponse(
		NewFragment(templ.NopComponent).WithHeader("Cache-Control", "public, max-age=60"),
	)
	if err != nil {
		t.Fatalf("resolveRouteResponse() error = %v, want nil", err)
	}
	if got := response.fragment.headers.Get("Cache-Control"); got != "public, max-age=60" {
		t.Fatalf("Cache-Control = %q, want public, max-age=60", got)
	}
}

func TestResolveRouteResponseAcceptsPointers(t *testing.T) {
	appErr := errors.New("load failed")
	metadata := PageMetadata{Title: "Users"}
	page := NewPage(templ.NopComponent, metadata)
	fragment := NewFragment(templ.NopComponent)
	redirect := Redirect{Location: "/sign-in", Status: http.StatusSeeOther}
	text := Text{Status: http.StatusForbidden, Body: "forbidden"}
	noContent := NoContent{}
	routeErr := RouteError{Err: appErr}

	tests := []struct {
		name     string
		response RouteResponse
		kind     routeResponseKind
	}{
		{name: "page", response: &page, kind: routeResponsePage},
		{name: "fragment", response: &fragment, kind: routeResponseFragment},
		{name: "redirect", response: &redirect, kind: routeResponseRedirect},
		{name: "text", response: &text, kind: routeResponseText},
		{name: "no content", response: &noContent, kind: routeResponseNoContent},
		{name: "route error", response: &routeErr, kind: routeResponseRouteError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := resolveRouteResponse(test.response)
			if err != nil {
				t.Fatalf("resolveRouteResponse() error = %v, want nil", err)
			}
			if response.kind != test.kind {
				t.Fatalf("kind = %d, want %d", response.kind, test.kind)
			}
		})
	}
}

func TestEndpointResponseInterfaceMembership(_ *testing.T) {
	var _ RouteResponse = Page{}
	var _ PageRouteResponse = Page{}

	var _ RouteResponse = Fragment{}
	var _ FragmentRouteResponse = Fragment{}

	var _ RouteResponse = Redirect{}
	var _ PageRouteResponse = Redirect{}
	var _ FragmentRouteResponse = Redirect{}

	var _ RouteResponse = Text{}
	var _ PageRouteResponse = Text{}
	var _ FragmentRouteResponse = Text{}

	var _ RouteResponse = RouteError{}
	var _ PageRouteResponse = RouteError{}
	var _ FragmentRouteResponse = RouteError{}

	var _ RouteResponse = NoContent{}
}

func TestRouteResponseWithHeader(t *testing.T) {
	response, err := resolveRouteResponse(
		NewPage(templ.NopComponent, PageMetadata{}).WithHeader("Cache-Control", "no-store"),
	)
	if err != nil {
		t.Fatalf("resolveRouteResponse() error = %v, want nil", err)
	}
	if got := response.page.headers.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}

	response.page.headers.Set("Cache-Control", "mutated")
	again, err := resolveRouteResponse(
		NewPage(templ.NopComponent, PageMetadata{}).WithHeader("Cache-Control", "no-store"),
	)
	if err != nil {
		t.Fatalf("resolveRouteResponse() error = %v, want nil", err)
	}
	if got := again.page.headers.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control after mutation = %q, want no-store", got)
	}
}

func TestRouteResponseAddHeader(t *testing.T) {
	response, err := resolveRouteResponse(
		NewPage(templ.NopComponent, PageMetadata{}).
			AddHeader("Set-Cookie", "one=1").
			AddHeader("Set-Cookie", "two=2"),
	)
	if err != nil {
		t.Fatalf("resolveRouteResponse() error = %v, want nil", err)
	}
	if got := response.page.headers.Values("Set-Cookie"); len(got) != 2 || got[0] != "one=1" || got[1] != "two=2" {
		t.Fatalf("Set-Cookie values = %#v, want [one=1 two=2]", got)
	}

	response.page.headers.Add("Set-Cookie", "mutated=1")
	again, err := resolveRouteResponse(
		NewPage(templ.NopComponent, PageMetadata{}).
			AddHeader("Set-Cookie", "one=1").
			AddHeader("Set-Cookie", "two=2"),
	)
	if err != nil {
		t.Fatalf("resolveRouteResponse() error = %v, want nil", err)
	}
	if got := again.page.headers.Values("Set-Cookie"); len(got) != 2 || got[0] != "one=1" || got[1] != "two=2" {
		t.Fatalf("Set-Cookie values after mutation = %#v, want [one=1 two=2]", got)
	}
}

func TestNoContentRouteResponse(t *testing.T) {
	tests := []struct {
		name   string
		input  NoContent
		status int
	}{
		{name: "default", input: NoContent{}, status: http.StatusNoContent},
		{name: "reset content", input: NoContent{Status: http.StatusResetContent}, status: http.StatusResetContent},
		{name: "not modified", input: NoContent{Status: http.StatusNotModified}, status: http.StatusNotModified},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := resolveRouteResponse(
				test.input.
					WithHeader("Cache-Control", "no-store").
					AddHeader("Set-Cookie", "one=1").
					AddHeader("Set-Cookie", "two=2"),
			)
			if err != nil {
				t.Fatalf("resolveRouteResponse() error = %v, want nil", err)
			}
			if response.noBody.Status != test.status {
				t.Fatalf("status = %d, want %d", response.noBody.Status, test.status)
			}
			if got := response.noBody.headers.Get("Cache-Control"); got != "no-store" {
				t.Fatalf("Cache-Control = %q, want no-store", got)
			}
			if got := response.noBody.headers.Values("Set-Cookie"); len(got) != 2 || got[0] != "one=1" || got[1] != "two=2" {
				t.Fatalf("Set-Cookie = %#v, want [one=1 two=2]", got)
			}
		})
	}
}

func TestRouteResponseValidation(t *testing.T) {
	var nilPage *Page
	var nilFragment *Fragment
	var nilRedirect *Redirect
	var nilText *Text
	var nilNoContent *NoContent
	var nilRouteError *RouteError

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
		{name: "bad no content status", response: NoContent{Status: http.StatusOK}, want: ErrInvalidRouteResponse},
		{name: "informational component status", response: NewPage(templ.NopComponent, PageMetadata{}).WithStatus(http.StatusSwitchingProtocols), want: ErrInvalidRouteResponse},
		{name: "no content component status", response: NewPage(templ.NopComponent, PageMetadata{}).WithStatus(http.StatusNoContent), want: ErrInvalidRouteResponse},
		{name: "no content fragment status", response: NewFragment(templ.NopComponent).WithStatus(http.StatusNoContent), want: ErrInvalidRouteResponse},
		{name: "nil route error", response: RouteError{}, want: ErrNilRouteError},
		{name: "nil route response", response: nil, want: ErrInvalidRouteResponse},
		{name: "nil page pointer", response: nilPage, want: ErrInvalidRouteResponse},
		{name: "nil fragment pointer", response: nilFragment, want: ErrInvalidRouteResponse},
		{name: "nil redirect pointer", response: nilRedirect, want: ErrInvalidRouteResponse},
		{name: "nil text pointer", response: nilText, want: ErrInvalidRouteResponse},
		{name: "nil no content pointer", response: nilNoContent, want: ErrInvalidRouteResponse},
		{name: "nil route error pointer", response: nilRouteError, want: ErrInvalidRouteResponse},
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

func TestRouteErrorResponseCarriesApplicationError(t *testing.T) {
	appErr := errors.New("load failed")

	response, err := resolveRouteResponse(RouteError{Err: appErr})
	if err != nil {
		t.Fatalf("resolveRouteResponse() error = %v, want nil", err)
	}
	if response.kind != routeResponseRouteError {
		t.Fatalf("kind = %d, want %d", response.kind, routeResponseRouteError)
	}
	if !errors.Is(response.err, appErr) {
		t.Fatalf("response error = %v, want %v", response.err, appErr)
	}
}

func routeResponseError(response RouteResponse) error {
	_, err := resolveRouteResponse(response)
	return err
}
