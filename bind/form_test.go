package bind

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseFormReadsFormBody(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/users", strings.NewReader("name=Ada&status=Active"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	form, err := ParseForm(request)
	if err != nil {
		t.Fatalf("ParseForm() error = %v, want nil", err)
	}

	if got := form.Value("name"); got != "Ada" {
		t.Fatalf("Value(name) = %q, want %q", got, "Ada")
	}
	if got := form.Value("status"); got != "Active" {
		t.Fatalf("Value(status) = %q, want %q", got, "Active")
	}
}

func TestParseFormUsesBodyBeforeQueryValues(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/users?name=Query", strings.NewReader("name=Body"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	form, err := ParseForm(request)
	if err != nil {
		t.Fatalf("ParseForm() error = %v, want nil", err)
	}

	if got := form.Value("name"); got != "Body" {
		t.Fatalf("Value(name) = %q, want body value", got)
	}
	if got := form.Values("name"); len(got) != 2 || got[0] != "Body" || got[1] != "Query" {
		t.Fatalf("Values(name) = %#v, want body then query", got)
	}
}

func TestParseFormCopiesValues(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/users", strings.NewReader("name=Ada&name=Grace"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	form, err := ParseForm(request)
	if err != nil {
		t.Fatalf("ParseForm() error = %v, want nil", err)
	}

	request.Form.Set("name", "Changed")
	values := form.Values("name")
	values[0] = "Mutated"

	if got := form.Value("name"); got != "Ada" {
		t.Fatalf("Value(name) = %q, want copied value", got)
	}
	if got := form.Values("name"); got[0] != "Ada" || got[1] != "Grace" {
		t.Fatalf("Values(name) = %#v, want copied values", got)
	}
}

func TestParseFormNilRequestReturnsError(t *testing.T) {
	if _, err := ParseForm(nil); !errors.Is(err, ErrNilRequest) {
		t.Fatalf("ParseForm(nil) error = %v, want ErrNilRequest", err)
	}
}

func TestZeroValueForm(t *testing.T) {
	var form Form

	if got := form.Value("name"); got != "" {
		t.Fatalf("Value(name) = %q, want empty", got)
	}
	if got := form.Values("name"); got != nil {
		t.Fatalf("Values(name) = %#v, want nil", got)
	}
	if form.HasErrors() {
		t.Fatal("HasErrors() = true, want false")
	}
	if form.HasFieldError("name") {
		t.Fatal("HasFieldError(name) = true, want false")
	}
	if got := form.FieldError("name"); got != "" {
		t.Fatalf("FieldError(name) = %q, want empty", got)
	}
}

func TestFieldErrorsZeroValueAndMultipleMessages(t *testing.T) {
	var errors FieldErrors

	if errors.Any() {
		t.Fatal("Any() = true, want false")
	}
	if errors.Has("name") {
		t.Fatal("Has(name) = true, want false")
	}

	errors.Add("name", "Name is required.")
	errors.Add("name", "Name is too short.")

	if !errors.Any() {
		t.Fatal("Any() = false, want true")
	}
	if !errors.Has("name") {
		t.Fatal("Has(name) = false, want true")
	}
	if got := errors.First("name"); got != "Name is required." {
		t.Fatalf("First(name) = %q, want first error", got)
	}
	if got := errors.Values("name"); len(got) != 2 || got[0] != "Name is required." || got[1] != "Name is too short." {
		t.Fatalf("Values(name) = %#v, want both errors", got)
	}
}

func TestFieldErrorsReturnedValueCanBeRead(t *testing.T) {
	if !fieldErrorsWithMessage().Any() {
		t.Fatal("Any() = false, want true")
	}
	if !fieldErrorsWithMessage().Has("name") {
		t.Fatal("Has(name) = false, want true")
	}
	if got := fieldErrorsWithMessage().First("name"); got != "Name is required." {
		t.Fatalf("First(name) = %q, want first error", got)
	}
	if got := fieldErrorsWithMessage().Values("name"); len(got) != 1 || got[0] != "Name is required." {
		t.Fatalf("Values(name) = %#v, want one error", got)
	}
}

func TestFormWithErrorsCopiesErrors(t *testing.T) {
	var fieldErrors FieldErrors
	fieldErrors.Add("name", "Name is required.")

	form := Form{}.WithErrors(fieldErrors)
	fieldErrors.Add("name", "Changed")
	returned := form.FieldErrors("name")
	returned[0] = "Mutated"

	if got := form.FieldError("name"); got != "Name is required." {
		t.Fatalf("FieldError(name) = %q, want copied error", got)
	}
	if got := form.FieldErrors("name"); len(got) != 1 || got[0] != "Name is required." {
		t.Fatalf("FieldErrors(name) = %#v, want copied errors", got)
	}
}

func fieldErrorsWithMessage() FieldErrors {
	var errors FieldErrors
	errors.Add("name", "Name is required.")
	return errors
}
