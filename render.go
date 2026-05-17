// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/a-h/templ"
)

var (
	// ErrNilRequest reports a missing request for rendering or writing.
	ErrNilRequest = errors.New("write component: nil request")
	// ErrNilResponseWriter reports a missing response writer.
	ErrNilResponseWriter = errors.New("write component: nil response writer")
	// ErrNilComponent reports a nil templ component.
	ErrNilComponent = errors.New("write component: nil component")
	// ErrInvalidHTMLStatus reports an HTTP status that cannot carry rendered
	// HTML.
	ErrInvalidHTMLStatus = errors.New("invalid html response status")
	// ErrRouteResponseWriterUnavailable reports a layout-wrapped page response
	// written outside generated Goldr route dispatch.
	ErrRouteResponseWriterUnavailable = errors.New("route response writer unavailable")
)

type routeResponseWriterContextKey struct{}

// RouteResponseWriter writes a resolved route response for generated Goldr
// route dispatch.
type RouteResponseWriter func(http.ResponseWriter, *http.Request, ResolvedRouteResponse) error

// WithRouteResponseWriter returns a request that can write layout-aware route
// responses.
func WithRouteResponseWriter(r *http.Request, writer RouteResponseWriter) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), routeResponseWriterContextKey{}, writer))
}

// WriteComponent renders component and writes it as an HTML response with
// status.
func WriteComponent(w http.ResponseWriter, r *http.Request, status int, component templ.Component) error {
	return writeComponentResponse(w, r, status, component, nil)
}

func writeComponentResponse(w http.ResponseWriter, r *http.Request, status int, component templ.Component, headers http.Header) error {
	if w == nil {
		return ErrNilResponseWriter
	}
	if r == nil {
		return ErrNilRequest
	}
	if component == nil {
		return ErrNilComponent
	}
	if !validPageStatus(status) {
		return ErrInvalidHTMLStatus
	}

	var body bytes.Buffer
	if err := component.Render(r.Context(), &body); err != nil {
		return fmt.Errorf("write component: %w", err)
	}

	applyResponseHeaders(w, headers)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if r.Method == http.MethodHead {
		return nil
	}
	_, err := w.Write(body.Bytes())
	return err
}

// WriteRouteResponse writes response from an ordinary HTTP handler.
func WriteRouteResponse(w http.ResponseWriter, r *http.Request, response RouteResponse) error {
	if w == nil {
		return ErrNilResponseWriter
	}
	if r == nil {
		return ErrNilRequest
	}

	resolved, err := ResolveRouteResponse(response)
	if err != nil {
		return err
	}

	switch resolved.Kind {
	case RouteResponsePage:
		writer := routeResponseWriterFromContext(r)
		if writer == nil {
			return ErrRouteResponseWriterUnavailable
		}
		return writer(w, r, resolved)
	case RouteResponseFragment:
		return writeComponentResponse(w, r, resolved.Status, resolved.Component, resolved.Headers)
	case RouteResponseRedirect:
		applyResponseHeaders(w, resolved.Headers)
		writeRedirect(w, resolved.Location, resolved.Status)
		return nil
	case RouteResponseText:
		applyResponseHeaders(w, resolved.Headers)
		return writeTextResponse(w, r, resolved.Status, resolved.Body)
	case RouteResponseServerError:
		return resolved.Error
	default:
		return ErrInvalidRouteResponse
	}
}

func applyResponseHeaders(w http.ResponseWriter, headers http.Header) {
	target := w.Header()
	for name, values := range headers {
		target.Del(name)
		for _, value := range values {
			target.Add(name, value)
		}
	}
}

func routeResponseWriterFromContext(r *http.Request) RouteResponseWriter {
	if r == nil {
		return nil
	}
	writer, _ := r.Context().Value(routeResponseWriterContextKey{}).(RouteResponseWriter)
	return writer
}

func writeRedirect(w http.ResponseWriter, location string, status int) {
	w.Header().Set("Location", location)
	w.WriteHeader(status)
}

func writeTextResponse(w http.ResponseWriter, r *http.Request, status int, body string) error {
	if w == nil {
		return ErrNilResponseWriter
	}
	if r == nil {
		return ErrNilRequest
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	if r.Method == http.MethodHead {
		return nil
	}
	_, err := w.Write([]byte(body))
	return err
}
