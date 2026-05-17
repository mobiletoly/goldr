package goldr

import (
	"errors"
	"net/http"
	"testing"

	"github.com/a-h/templ"
)

func TestPageRenderResponse(t *testing.T) {
	component := templ.NopComponent
	metadata := PageMetadata{Title: "Users"}

	response, err := RenderPage(component, metadata).Response()

	if err != nil {
		t.Fatalf("Response error = %v, want nil", err)
	}
	if response.Kind != PageResponseRender {
		t.Fatalf("kind = %d, want %d", response.Kind, PageResponseRender)
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

func TestPageResponseValidation(t *testing.T) {
	tests := []struct {
		name string
		page Page
		want error
	}{
		{name: "nil component", page: RenderPage(nil, PageMetadata{}), want: ErrNilComponent},
		{name: "bad redirect", page: Redirect("", http.StatusSeeOther), want: ErrInvalidPageResponse},
		{name: "not modified redirect", page: Redirect("/sign-in", http.StatusNotModified), want: ErrInvalidPageResponse},
		{name: "multiple choices redirect", page: Redirect("/sign-in", http.StatusMultipleChoices), want: ErrInvalidPageResponse},
		{name: "bad text status", page: TextStatus(http.StatusFound, "redirect"), want: ErrInvalidPageResponse},
		{name: "informational text status", page: TextStatus(http.StatusContinue, "continue"), want: ErrInvalidPageResponse},
		{name: "no content text status", page: TextStatus(http.StatusNoContent, "no content"), want: ErrInvalidPageResponse},
		{name: "reset content text status", page: TextStatus(http.StatusResetContent, "reset"), want: ErrInvalidPageResponse},
		{name: "informational component status", page: Status(http.StatusSwitchingProtocols, templ.NopComponent, PageMetadata{}), want: ErrInvalidPageResponse},
		{name: "no content component status", page: Status(http.StatusNoContent, templ.NopComponent, PageMetadata{}), want: ErrInvalidPageResponse},
		{name: "nil error", page: Error(nil), want: ErrNilPageError},
		{name: "zero page", page: Page{}, want: ErrInvalidPageResponse},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := pageResponseError(test.page)
			if !errors.Is(got, test.want) {
				t.Fatalf("error = %v, want %v", got, test.want)
			}
		})
	}
}

func TestPageErrorResponseCarriesApplicationError(t *testing.T) {
	appErr := errors.New("load failed")

	response, err := Error(appErr).Response()
	if err != nil {
		t.Fatalf("Response error = %v, want nil", err)
	}
	if response.Kind != PageResponseError {
		t.Fatalf("kind = %d, want %d", response.Kind, PageResponseError)
	}
	if !errors.Is(response.Error, appErr) {
		t.Fatalf("response error = %v, want %v", response.Error, appErr)
	}
}

func pageResponseError(page Page) error {
	_, err := page.Response()
	return err
}
