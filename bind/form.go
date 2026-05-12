// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package bind

import (
	"errors"
	"net/http"
	"net/url"
	"slices"
)

var ErrNilRequest = errors.New("nil request")

type Form struct {
	values url.Values
	errors FieldErrors
}

func ParseForm(r *http.Request) (Form, error) {
	if r == nil {
		return Form{}, ErrNilRequest
	}
	if err := r.ParseForm(); err != nil {
		return Form{}, err
	}
	return Form{values: cloneValues(r.Form)}, nil
}

func (f Form) Value(name string) string {
	values := f.values[name]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (f Form) Values(name string) []string {
	return slices.Clone(f.values[name])
}

func (f Form) WithErrors(errors FieldErrors) Form {
	f.errors = errors.clone()
	return f
}

func (f Form) FieldError(name string) string {
	return f.errors.First(name)
}

func (f Form) FieldErrors(name string) []string {
	return f.errors.Values(name)
}

func (f Form) HasFieldError(name string) bool {
	return f.errors.Has(name)
}

func (f Form) HasErrors() bool {
	return f.errors.Any()
}

//nolint:recvcheck // Add mutates zero-value FieldErrors; read methods stay value receivers for returned values.
type FieldErrors struct {
	values map[string][]string
}

func (e *FieldErrors) Add(field, message string) {
	if e.values == nil {
		e.values = make(map[string][]string)
	}
	e.values[field] = append(e.values[field], message)
}

func (e FieldErrors) First(field string) string {
	values := e.values[field]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (e FieldErrors) Values(field string) []string {
	return slices.Clone(e.values[field])
}

func (e FieldErrors) Has(field string) bool {
	return len(e.values[field]) > 0
}

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
