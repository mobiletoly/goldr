// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

// Package bind provides small helpers for parsing form requests and carrying
// validation errors back to server-rendered HTML.
package bind

import (
	"errors"
	"mime/multipart"
	"net/http"
	"net/url"
	"slices"
)

// ErrNilRequest reports a missing request passed to a form parser.
var ErrNilRequest = errors.New("nil request")

// Form is a parsed form request plus optional field validation errors.
type Form struct {
	values url.Values
	files  map[string][]*multipart.FileHeader
	errors FieldErrors
}

// ParseForm parses URL-encoded form values from the request.
func ParseForm(r *http.Request) (Form, error) {
	if r == nil {
		return Form{}, ErrNilRequest
	}
	if err := r.ParseForm(); err != nil {
		return Form{}, err
	}
	return Form{values: cloneValues(r.Form)}, nil
}

// ParseMultipartForm parses multipart form values and uploaded file headers.
func ParseMultipartForm(r *http.Request, maxMemory int64) (Form, error) {
	if r == nil {
		return Form{}, ErrNilRequest
	}
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		return Form{}, err
	}
	return Form{
		values: cloneMultipartValues(r),
		files:  cloneFiles(r.MultipartForm),
	}, nil
}

// Value returns the first submitted value for name.
func (f Form) Value(name string) string {
	values := f.values[name]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// Values returns all submitted values for name.
func (f Form) Values(name string) []string {
	return slices.Clone(f.values[name])
}

// File opens the first uploaded file for name.
func (f Form) File(name string) (multipart.File, *multipart.FileHeader, error) {
	headers := f.files[name]
	if len(headers) == 0 {
		return nil, nil, http.ErrMissingFile
	}
	file, err := headers[0].Open()
	if err != nil {
		return nil, nil, err
	}
	return file, headers[0], nil
}

// Files returns all uploaded file headers for name.
func (f Form) Files(name string) []*multipart.FileHeader {
	return slices.Clone(f.files[name])
}

// WithErrors returns a copy of the form with validation errors attached.
func (f Form) WithErrors(errors FieldErrors) Form {
	f.errors = errors.clone()
	return f
}

// FieldError returns the first validation error for name.
func (f Form) FieldError(name string) string {
	return f.errors.First(name)
}

// FieldErrors returns all validation errors for name.
func (f Form) FieldErrors(name string) []string {
	return f.errors.Values(name)
}

// HasFieldError reports whether name has at least one validation error.
func (f Form) HasFieldError(name string) bool {
	return f.errors.Has(name)
}

// HasErrors reports whether the form has any validation errors.
func (f Form) HasErrors() bool {
	return f.errors.Any()
}

// FieldErrors stores validation messages keyed by field name.
//
//nolint:recvcheck // Add mutates zero-value FieldErrors; read methods stay value receivers for returned values.
type FieldErrors struct {
	values map[string][]string
}

// Add appends a validation message for field.
func (e *FieldErrors) Add(field, message string) {
	if e.values == nil {
		e.values = make(map[string][]string)
	}
	e.values[field] = append(e.values[field], message)
}

// First returns the first validation message for field.
func (e FieldErrors) First(field string) string {
	values := e.values[field]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// Values returns all validation messages for field.
func (e FieldErrors) Values(field string) []string {
	return slices.Clone(e.values[field])
}

// Has reports whether field has at least one validation message.
func (e FieldErrors) Has(field string) bool {
	return len(e.values[field]) > 0
}

// Any reports whether any field has validation messages.
func (e FieldErrors) Any() bool {
	for _, values := range e.values {
		if len(values) > 0 {
			return true
		}
	}
	return false
}

func (e FieldErrors) clone() FieldErrors {
	if len(e.values) == 0 {
		return FieldErrors{}
	}
	return FieldErrors{values: cloneErrors(e.values)}
}

func cloneValues(values url.Values) url.Values {
	if len(values) == 0 {
		return nil
	}
	copied := make(url.Values, len(values))
	for key, value := range values {
		copied[key] = slices.Clone(value)
	}
	return copied
}

func cloneMultipartValues(r *http.Request) url.Values {
	var values url.Values
	if r.MultipartForm != nil {
		values = cloneValues(r.MultipartForm.Value)
	}
	for key, queryValues := range r.URL.Query() {
		if values == nil {
			values = make(url.Values)
		}
		values[key] = append(values[key], queryValues...)
	}
	return values
}

func cloneFiles(form *multipart.Form) map[string][]*multipart.FileHeader {
	if form == nil || len(form.File) == 0 {
		return nil
	}
	copied := make(map[string][]*multipart.FileHeader, len(form.File))
	for key, value := range form.File {
		copied[key] = slices.Clone(value)
	}
	return copied
}

func cloneErrors(values map[string][]string) map[string][]string {
	if len(values) == 0 {
		return nil
	}
	copied := make(map[string][]string, len(values))
	for key, value := range values {
		copied[key] = slices.Clone(value)
	}
	return copied
}
