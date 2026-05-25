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
	// ErrRoutePageRendererUnavailable reports a page route response written
	// outside generated Goldr route dispatch.
	ErrRoutePageRendererUnavailable = errors.New("route page renderer unavailable")
)

type routePageRendererContextKey struct{}

// RoutePageRenderer renders a validated page response through generated route
// dispatch.
type RoutePageRenderer func(*http.Request, Page) (templ.Component, error)

// WithRoutePageRenderer returns a request that can write layout-aware page
// responses from ordinary action handlers.
func WithRoutePageRenderer(r *http.Request, render RoutePageRenderer) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), routePageRendererContextKey{}, render))
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
	return writeRouteResponse(w, r, response, routePageRendererFromContext(r), true, true)
}

// WritePageRouteResponse writes a page route response from generated route
// dispatch.
func WritePageRouteResponse(w http.ResponseWriter, r *http.Request, response RouteResponse, render RoutePageRenderer) error {
	return writeRouteResponse(w, r, response, render, true, true)
}

// WriteFragmentRouteResponse writes a fragment route response from generated
// route dispatch.
func WriteFragmentRouteResponse(w http.ResponseWriter, r *http.Request, response RouteResponse) error {
	return writeRouteResponse(w, r, response, nil, false, true)
}

func writeRouteResponse(w http.ResponseWriter, r *http.Request, response RouteResponse, render RoutePageRenderer, allowPage bool, allowFragment bool) error {
	if w == nil {
		return ErrNilResponseWriter
	}
	if r == nil {
		return ErrNilRequest
	}

	resolved, err := resolveRouteResponse(response)
	if err != nil {
		return err
	}

	switch resolved.kind {
	case routeResponsePage:
		if !allowPage {
			return ErrInvalidRouteResponse
		}
		if render == nil {
			return ErrRoutePageRendererUnavailable
		}
		component, err := render(r, resolved.page)
		if err != nil {
			return err
		}
		return writeComponentResponse(w, r, resolved.page.Status, component, resolved.page.headers)
	case routeResponseFragment:
		if !allowFragment {
			return ErrInvalidRouteResponse
		}
		return writeComponentResponse(w, r, resolved.fragment.Status, resolved.fragment.Component, resolved.fragment.headers)
	case routeResponseRedirect:
		applyResponseHeaders(w, resolved.redirect.headers)
		writeRedirect(w, resolved.redirect.Location, resolved.redirect.Status)
		return nil
	case routeResponseText:
		applyResponseHeaders(w, resolved.text.headers)
		return writeTextResponse(w, r, resolved.text.Status, resolved.text.Body)
	case routeResponseNoContent:
		applyResponseHeaders(w, resolved.noBody.headers)
		w.WriteHeader(resolved.noBody.Status)
		return nil
	case routeResponseServerError:
		return resolved.err
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

func routePageRendererFromContext(r *http.Request) RoutePageRenderer {
	if r == nil {
		return nil
	}
	render, _ := r.Context().Value(routePageRendererContextKey{}).(RoutePageRenderer)
	return render
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
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	w.WriteHeader(status)
	if r.Method == http.MethodHead {
		return nil
	}
	_, err := w.Write([]byte(body))
	return err
}
