// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import (
	"errors"
	"net/http"

	"github.com/a-h/templ"
)

var (
	// ErrInvalidRouteResponse reports a route response that cannot be written.
	ErrInvalidRouteResponse = errors.New("invalid route response")
	// ErrNilServerError reports goldr.ServerError{Err: nil}.
	ErrNilServerError = errors.New("server error route response: nil error")
)

// RouteResponse is a response value returned by Goldr page functions and
// writable by Goldr-aware HTTP handlers.
type RouteResponse interface {
	goldrRouteResponse()
}

// Page is a rendered page response.
type Page struct {
	Status    int
	Component templ.Component
	Metadata  PageMetadata
	headers   http.Header
}

func (Page) goldrRouteResponse() {}

// Fragment is a standalone rendered fragment response.
type Fragment struct {
	Status    int
	Component templ.Component
	headers   http.Header
}

func (Fragment) goldrRouteResponse() {}

// Redirect is a route response that redirects without rendering.
type Redirect struct {
	Status   int
	Location string
	headers  http.Header
}

func (Redirect) goldrRouteResponse() {}

// Text is a plain text route response.
type Text struct {
	Status  int
	Body    string
	headers http.Header
}

func (Text) goldrRouteResponse() {}

// ServerError delegates an application error to Goldr error handling.
type ServerError struct {
	Err error
}

func (ServerError) goldrRouteResponse() {}

// RouteResponseKind identifies the response behavior for a resolved route
// response.
type RouteResponseKind uint8

const (
	// RouteResponseInvalid is never returned by valid route responses.
	RouteResponseInvalid RouteResponseKind = iota
	// RouteResponsePage renders a templ component through matching layouts.
	RouteResponsePage
	// RouteResponseFragment renders a standalone templ component without
	// layouts.
	RouteResponseFragment
	// RouteResponseRedirect writes a redirect without rendering.
	RouteResponseRedirect
	// RouteResponseText writes a plain text status response without rendering.
	RouteResponseText
	// RouteResponseServerError delegates an application error to Goldr error
	// handling.
	RouteResponseServerError
)

// ResolvedRouteResponse is the generated-dispatch-facing view of a route
// response.
type ResolvedRouteResponse struct {
	// Kind identifies which response behavior generated dispatch should use.
	Kind RouteResponseKind
	// Component is rendered for RouteResponsePage.
	Component templ.Component
	// Metadata is passed to layouts for RouteResponsePage.
	Metadata PageMetadata
	// Status is the HTTP status written by page, redirect, and text responses.
	Status int
	// Location is the redirect target for RouteResponseRedirect.
	Location string
	// Body is the plain text response body for RouteResponseText.
	Body string
	// Error is the application error for RouteResponseServerError.
	Error error
	// Headers are response headers applied before status and body are written.
	Headers http.Header
}

// PageMetadata is page-owned metadata passed explicitly to layouts.
type PageMetadata struct {
	// Title is the page title.
	Title string
	// Description is the page description.
	Description string
}

// LayoutContext is the explicit layout-facing context for page rendering.
type LayoutContext struct {
	// Child is the already-composed child layout or page component.
	Child templ.Component
	// Metadata is the page metadata visible to the layout chain.
	Metadata PageMetadata
}

// NewPage returns a normal 200 OK rendered page response.
func NewPage(component templ.Component, metadata PageMetadata) Page {
	return Page{
		Status:    http.StatusOK,
		Component: component,
		Metadata:  metadata,
	}
}

// NewFragment returns a normal 200 OK rendered fragment response.
func NewFragment(component templ.Component) Fragment {
	return Fragment{
		Status:    http.StatusOK,
		Component: component,
	}
}

// WithStatus returns page with an explicit HTTP status.
func (page Page) WithStatus(status int) Page {
	page.Status = status
	return page
}

// WithStatus returns fragment with an explicit HTTP status.
func (fragment Fragment) WithStatus(status int) Fragment {
	fragment.Status = status
	return fragment
}

// WithHeader returns page with name set to value.
func (page Page) WithHeader(name, value string) Page {
	page.headers = withResponseHeader(page.headers, name, value)
	return page
}

// AddHeader returns page with value added to name.
func (page Page) AddHeader(name, value string) Page {
	page.headers = addResponseHeader(page.headers, name, value)
	return page
}

// WithHeader returns fragment with name set to value.
func (fragment Fragment) WithHeader(name, value string) Fragment {
	fragment.headers = withResponseHeader(fragment.headers, name, value)
	return fragment
}

// AddHeader returns fragment with value added to name.
func (fragment Fragment) AddHeader(name, value string) Fragment {
	fragment.headers = addResponseHeader(fragment.headers, name, value)
	return fragment
}

// WithHeader returns redirect with name set to value.
func (redirect Redirect) WithHeader(name, value string) Redirect {
	redirect.headers = withResponseHeader(redirect.headers, name, value)
	return redirect
}

// AddHeader returns redirect with value added to name.
func (redirect Redirect) AddHeader(name, value string) Redirect {
	redirect.headers = addResponseHeader(redirect.headers, name, value)
	return redirect
}

// WithHeader returns text with name set to value.
func (text Text) WithHeader(name, value string) Text {
	text.headers = withResponseHeader(text.headers, name, value)
	return text
}

// AddHeader returns text with value added to name.
func (text Text) AddHeader(name, value string) Text {
	text.headers = addResponseHeader(text.headers, name, value)
	return text
}

// ResolveRouteResponse returns the generated-dispatch-facing view of response.
//
// A non-nil error reports an invalid Goldr route response contract, such as a
// nil render component, invalid status, empty redirect location, or nil
// response.
func ResolveRouteResponse(response RouteResponse) (ResolvedRouteResponse, error) {
	switch response := response.(type) {
	case Page:
		return resolvePageResponse(response)
	case *Page:
		if response == nil {
			return ResolvedRouteResponse{}, ErrInvalidRouteResponse
		}
		return resolvePageResponse(*response)
	case Fragment:
		return resolveFragmentResponse(response)
	case *Fragment:
		if response == nil {
			return ResolvedRouteResponse{}, ErrInvalidRouteResponse
		}
		return resolveFragmentResponse(*response)
	case Redirect:
		return resolveRedirectResponse(response)
	case *Redirect:
		if response == nil {
			return ResolvedRouteResponse{}, ErrInvalidRouteResponse
		}
		return resolveRedirectResponse(*response)
	case Text:
		return resolveTextResponse(response)
	case *Text:
		if response == nil {
			return ResolvedRouteResponse{}, ErrInvalidRouteResponse
		}
		return resolveTextResponse(*response)
	case ServerError:
		return resolveServerErrorResponse(response)
	case *ServerError:
		if response == nil {
			return ResolvedRouteResponse{}, ErrInvalidRouteResponse
		}
		return resolveServerErrorResponse(*response)
	default:
		return ResolvedRouteResponse{}, ErrInvalidRouteResponse
	}
}

func resolvePageResponse(response Page) (ResolvedRouteResponse, error) {
	status := response.Status
	if status == 0 {
		status = http.StatusOK
	}
	if err := validateRenderResponse(response.Component, status); err != nil {
		return ResolvedRouteResponse{}, err
	}
	return ResolvedRouteResponse{
		Kind:      RouteResponsePage,
		Component: response.Component,
		Metadata:  response.Metadata,
		Status:    status,
		Headers:   cloneResponseHeaders(response.headers),
	}, nil
}

func resolveFragmentResponse(response Fragment) (ResolvedRouteResponse, error) {
	status := response.Status
	if status == 0 {
		status = http.StatusOK
	}
	if err := validateRenderResponse(response.Component, status); err != nil {
		return ResolvedRouteResponse{}, err
	}
	return ResolvedRouteResponse{
		Kind:      RouteResponseFragment,
		Component: response.Component,
		Status:    status,
		Headers:   cloneResponseHeaders(response.headers),
	}, nil
}

func resolveRedirectResponse(response Redirect) (ResolvedRouteResponse, error) {
	if err := validateRedirectResponse(response.Location, response.Status); err != nil {
		return ResolvedRouteResponse{}, err
	}
	return ResolvedRouteResponse{
		Kind:     RouteResponseRedirect,
		Status:   response.Status,
		Location: response.Location,
		Headers:  cloneResponseHeaders(response.headers),
	}, nil
}

func resolveTextResponse(response Text) (ResolvedRouteResponse, error) {
	if err := validateTextResponse(response.Status); err != nil {
		return ResolvedRouteResponse{}, err
	}
	return ResolvedRouteResponse{
		Kind:    RouteResponseText,
		Status:  response.Status,
		Body:    response.Body,
		Headers: cloneResponseHeaders(response.headers),
	}, nil
}

func resolveServerErrorResponse(response ServerError) (ResolvedRouteResponse, error) {
	if response.Err == nil {
		return ResolvedRouteResponse{}, ErrNilServerError
	}
	return ResolvedRouteResponse{
		Kind:  RouteResponseServerError,
		Error: response.Err,
	}, nil
}

func withResponseHeader(headers http.Header, name, value string) http.Header {
	next := cloneResponseHeaders(headers)
	if next == nil {
		next = make(http.Header)
	}
	next.Set(name, value)
	return next
}

func addResponseHeader(headers http.Header, name, value string) http.Header {
	next := cloneResponseHeaders(headers)
	if next == nil {
		next = make(http.Header)
	}
	next.Add(name, value)
	return next
}

func cloneResponseHeaders(headers http.Header) http.Header {
	if len(headers) == 0 {
		return nil
	}

	clone := make(http.Header, len(headers))
	for name, values := range headers {
		clone[name] = append([]string(nil), values...)
	}
	return clone
}

func validateRenderResponse(component templ.Component, status int) error {
	if component == nil {
		return ErrNilComponent
	}
	if !validPageStatus(status) {
		return ErrInvalidRouteResponse
	}
	return nil
}

func validateRedirectResponse(location string, status int) error {
	if location == "" {
		return ErrInvalidRouteResponse
	}
	if !validRedirectStatus(status) {
		return ErrInvalidRouteResponse
	}
	return nil
}

func validateTextResponse(status int) error {
	if !validPageStatus(status) {
		return ErrInvalidRouteResponse
	}
	return nil
}

func validRedirectStatus(status int) bool {
	switch status {
	case http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusSeeOther,
		http.StatusTemporaryRedirect,
		http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

func validPageStatus(status int) bool {
	switch status {
	case http.StatusNoContent, http.StatusResetContent:
		return false
	}
	return (status >= http.StatusOK && status < http.StatusMultipleChoices) ||
		(status >= http.StatusBadRequest && status <= 599)
}
