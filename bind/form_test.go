package bind

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
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

func TestParseMultipartFormReadsTextFields(t *testing.T) {
	request := multipartRequest(t, map[string]string{
		"name":   "Ada",
		"status": "Active",
	}, nil)

	form, err := ParseMultipartForm(request, 1<<20)
	if err != nil {
		t.Fatalf("ParseMultipartForm() error = %v, want nil", err)
	}

	if got := form.Value("name"); got != "Ada" {
		t.Fatalf("Value(name) = %q, want %q", got, "Ada")
	}
	if got := form.Value("status"); got != "Active" {
		t.Fatalf("Value(status) = %q, want %q", got, "Active")
	}
}

func TestParseMultipartFormUsesBodyBeforeQueryValues(t *testing.T) {
	request := multipartRequest(t, map[string]string{
		"name": "Body",
	}, nil)
	request.URL.RawQuery = "name=Query"

	form, err := ParseMultipartForm(request, 1<<20)
	if err != nil {
		t.Fatalf("ParseMultipartForm() error = %v, want nil", err)
	}

	if got := form.Value("name"); got != "Body" {
		t.Fatalf("Value(name) = %q, want body value", got)
	}
	if got := form.Values("name"); len(got) != 2 || got[0] != "Body" || got[1] != "Query" {
		t.Fatalf("Values(name) = %#v, want body then query", got)
	}
}

func TestParseMultipartFormFileReturnsStandardLibraryTypes(t *testing.T) {
	request := multipartRequest(t, nil, map[string][]testUpload{
		"avatar": {
			{filename: "ada.txt", content: "hello ada"},
		},
	})

	form, err := ParseMultipartForm(request, 1<<20)
	if err != nil {
		t.Fatalf("ParseMultipartForm() error = %v, want nil", err)
	}

	file, header, err := form.File("avatar")
	if err != nil {
		t.Fatalf("File(avatar) error = %v, want nil", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	if header.Filename != "ada.txt" {
		t.Fatalf("header.Filename = %q, want %q", header.Filename, "ada.txt")
	}
	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(content) != "hello ada" {
		t.Fatalf("file content = %q, want %q", content, "hello ada")
	}
}

func TestParseMultipartFormMissingFileReturnsError(t *testing.T) {
	request := multipartRequest(t, map[string]string{"name": "Ada"}, nil)

	form, err := ParseMultipartForm(request, 1<<20)
	if err != nil {
		t.Fatalf("ParseMultipartForm() error = %v, want nil", err)
	}

	if _, _, err := form.File("avatar"); !errors.Is(err, http.ErrMissingFile) {
		t.Fatalf("File(avatar) error = %v, want ErrMissingFile", err)
	}
	if got := form.Files("avatar"); got != nil {
		t.Fatalf("Files(avatar) = %#v, want nil", got)
	}
}

func TestParseMultipartFormFilesReturnsCopiedSlice(t *testing.T) {
	request := multipartRequest(t, nil, map[string][]testUpload{
		"attachments": {
			{filename: "first.txt", content: "first"},
			{filename: "second.txt", content: "second"},
		},
	})

	form, err := ParseMultipartForm(request, 1<<20)
	if err != nil {
		t.Fatalf("ParseMultipartForm() error = %v, want nil", err)
	}

	files := form.Files("attachments")
	if len(files) != 2 {
		t.Fatalf("Files(attachments) len = %d, want 2", len(files))
	}
	files[0] = nil
	if got := form.Files("attachments"); got[0] == nil || got[0].Filename != "first.txt" {
		t.Fatalf("Files(attachments)[0] = %#v, want copied first header", got[0])
	}
}

func TestParseMultipartFormNilRequestReturnsError(t *testing.T) {
	if _, err := ParseMultipartForm(nil, 1<<20); !errors.Is(err, ErrNilRequest) {
		t.Fatalf("ParseMultipartForm(nil) error = %v, want ErrNilRequest", err)
	}
}

func TestParseMultipartFormMalformedBodyReturnsError(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/users", strings.NewReader("not multipart"))
	request.Header.Set("Content-Type", "multipart/form-data; boundary=missing")

	if _, err := ParseMultipartForm(request, 1<<20); err == nil {
		t.Fatal("ParseMultipartForm() error = nil, want error")
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

type testUpload struct {
	filename string
	content  string
}

func multipartRequest(t *testing.T, fields map[string]string, files map[string][]testUpload) *http.Request {
	t.Helper()

	body := new(strings.Builder)
	writer := multipart.NewWriter(body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatalf("WriteField() error = %v", err)
		}
	}
	for name, uploads := range files {
		for _, upload := range uploads {
			part, err := writer.CreateFormFile(name, upload.filename)
			if err != nil {
				t.Fatalf("CreateFormFile() error = %v", err)
			}
			if _, err := io.Copy(part, strings.NewReader(upload.content)); err != nil {
				t.Fatalf("Copy() error = %v", err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/users", strings.NewReader(body.String()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}
