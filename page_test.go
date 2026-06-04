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

func TestPageLayoutValue(t *testing.T) {
	type layoutState struct {
		ActiveTab string
	}
	key := NewLayoutKey[layoutState]("test.layout")
	page := WithLayoutValue(
		NewPage(templ.NopComponent, PageMetadata{}),
		key,
		layoutState{ActiveTab: "summary"},
	)

	got, ok := LayoutValue(LayoutContext{Data: page.Data}, key)

	if !ok {
		t.Fatalf("LayoutValue() ok = false, want true")
	}
	if got.ActiveTab != "summary" {
		t.Fatalf("ActiveTab = %q, want summary", got.ActiveTab)
	}
}

func TestPageLayoutValueMissingAndWrongType(t *testing.T) {
	type layoutState struct {
		ActiveTab string
	}
	stateKey := NewLayoutKey[layoutState]("test.layout")
	missingKey := NewLayoutKey[layoutState]("missing")
	page := WithLayoutValue(
		NewPage(templ.NopComponent, PageMetadata{}),
		stateKey,
		layoutState{ActiveTab: "summary"},
	)
	wrongTypeData := LayoutData{values: map[*layoutKeyID]any{stateKey.id: "summary"}}

	if got, ok := LayoutValue(LayoutContext{}, stateKey); ok || got != (layoutState{}) {
		t.Fatalf("zero LayoutValue() = (%#v, %v), want zero false", got, ok)
	}
	if got, ok := LayoutValue(LayoutContext{Data: page.Data}, missingKey); ok || got != (layoutState{}) {
		t.Fatalf("missing LayoutValue() = (%#v, %v), want zero false", got, ok)
	}
	if got, ok := LayoutValue(LayoutContext{Data: wrongTypeData}, stateKey); ok || got != (layoutState{}) {
		t.Fatalf("wrong-type LayoutValue() = (%#v, %v), want zero false", got, ok)
	}
}

func TestPageLayoutValueZeroKey(t *testing.T) {
	var key LayoutKey[string]

	if got, ok := LayoutValue(LayoutContext{}, key); ok || got != "" {
		t.Fatalf("zero-key LayoutValue() = (%q, %v), want empty false", got, ok)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("WithLayoutValue() with zero key did not panic")
		}
	}()
	_ = WithLayoutValue(NewPage(templ.NopComponent, PageMetadata{}), key, "ignored")
}

func TestPageLayoutValueKeysWithSameNameDoNotCollide(t *testing.T) {
	type layoutState struct {
		ActiveTab string
	}
	firstKey := NewLayoutKey[layoutState]("test.layout")
	secondKey := NewLayoutKey[layoutState]("test.layout")
	page := WithLayoutValue(NewPage(templ.NopComponent, PageMetadata{}), firstKey, layoutState{ActiveTab: "summary"})
	page = WithLayoutValue(page, secondKey, layoutState{ActiveTab: "details"})

	firstValue, firstOK := LayoutValue(LayoutContext{Data: page.Data}, firstKey)
	secondValue, secondOK := LayoutValue(LayoutContext{Data: page.Data}, secondKey)

	if !firstOK || firstValue.ActiveTab != "summary" {
		t.Fatalf("first LayoutValue() = (%#v, %v), want summary true", firstValue, firstOK)
	}
	if !secondOK || secondValue.ActiveTab != "details" {
		t.Fatalf("second LayoutValue() = (%#v, %v), want details true", secondValue, secondOK)
	}
}

func TestPageLayoutValueKeysWithSameNameAndDifferentTypesDoNotCollide(t *testing.T) {
	type layoutState struct {
		ActiveTab string
	}
	stateKey := NewLayoutKey[layoutState]("test.layout")
	stringKey := NewLayoutKey[string]("test.layout")
	page := WithLayoutValue(NewPage(templ.NopComponent, PageMetadata{}), stateKey, layoutState{ActiveTab: "summary"})
	page = WithLayoutValue(page, stringKey, "settings")

	stateValue, stateOK := LayoutValue(LayoutContext{Data: page.Data}, stateKey)
	stringValue, stringOK := LayoutValue(LayoutContext{Data: page.Data}, stringKey)

	if !stateOK || stateValue.ActiveTab != "summary" {
		t.Fatalf("state LayoutValue() = (%#v, %v), want summary true", stateValue, stateOK)
	}
	if !stringOK || stringValue != "settings" {
		t.Fatalf("string LayoutValue() = (%q, %v), want settings true", stringValue, stringOK)
	}
}

func TestPageLayoutValueCopyOnWrite(t *testing.T) {
	key := NewLayoutKey[string]("test.layout")
	original := WithLayoutValue(NewPage(templ.NopComponent, PageMetadata{}), key, "one")
	next := WithLayoutValue(original, key, "two")

	originalValue, originalOK := LayoutValue(LayoutContext{Data: original.Data}, key)
	nextValue, nextOK := LayoutValue(LayoutContext{Data: next.Data}, key)

	if !originalOK || originalValue != "one" {
		t.Fatalf("original LayoutValue() = (%q, %v), want one true", originalValue, originalOK)
	}
	if !nextOK || nextValue != "two" {
		t.Fatalf("next LayoutValue() = (%q, %v), want two true", nextValue, nextOK)
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
