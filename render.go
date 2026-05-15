// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"

	"github.com/a-h/templ"
)

var (
	// ErrNilRequest reports a missing request for rendering or writing.
	ErrNilRequest = errors.New("render html response: nil request")
	// ErrNilComponent reports a nil templ component.
	ErrNilComponent = errors.New("render html response: nil component")
)

// HTMLResponse is a buffered templ HTML response.
type HTMLResponse struct {
	body []byte
}

// Render buffers a templ HTML response using the request context.
func Render(r *http.Request, component templ.Component) (HTMLResponse, error) {
	if r == nil {
		return HTMLResponse{}, ErrNilRequest
	}
	if component == nil {
		return HTMLResponse{}, ErrNilComponent
	}

	var body bytes.Buffer
	if err := component.Render(r.Context(), &body); err != nil {
		return HTMLResponse{}, fmt.Errorf("render html response: %w", err)
	}

	return HTMLResponse{body: body.Bytes()}, nil
}

// Write writes a buffered HTML response.
func (response HTMLResponse) Write(w http.ResponseWriter, r *http.Request) error {
	return response.write(w, r, http.StatusOK, false)
}

// WriteStatus writes a buffered HTML response with an explicit HTTP status.
func (response HTMLResponse) WriteStatus(w http.ResponseWriter, r *http.Request, status int) error {
	return response.write(w, r, status, true)
}

func (response HTMLResponse) write(w http.ResponseWriter, r *http.Request, status int, explicitStatus bool) error {
	if r == nil {
		return ErrNilRequest
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if explicitStatus {
		w.WriteHeader(status)
	}
	if r.Method == http.MethodHead {
		return nil
	}
	_, err := w.Write(response.body)
	return err
}
