// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import (
	"errors"
	"maps"
	"net/http"

	"github.com/a-h/templ"
)

const fragmentDefaultCacheControl = "no-store"

var (
	// ErrInvalidRouteResponse reports a route response that cannot be written.
	ErrInvalidRouteResponse = errors.New("invalid route response")
	// ErrNilRouteError reports goldr.RouteError{Err: nil}.
	ErrNilRouteError = errors.New("route error response: nil error")
)

// RouteResponse is a response value writable by Goldr-aware HTTP handlers.
type RouteResponse interface {
	goldrRouteResponse()
}

// PageRouteResponse is a response value returned by Goldr page functions.
type PageRouteResponse interface {
	goldrRouteResponse()
	goldrPageRouteResponse()
}

// FragmentRouteResponse is a response value returned by Goldr fragment
// functions.
type FragmentRouteResponse interface {
	goldrRouteResponse()
	goldrFragmentRouteResponse()
}

// Page is a rendered page response.
type Page struct {
	Status    int
	Component templ.Component
	Metadata  PageMetadata
	Data      LayoutData
	headers   http.Header
}

func (Page) goldrRouteResponse()     {}
func (Page) goldrPageRouteResponse() {}

// Fragment is a standalone rendered fragment response.
type Fragment struct {
	Status    int
	Component templ.Component
	headers   http.Header
}

func (Fragment) goldrRouteResponse()         {}
func (Fragment) goldrFragmentRouteResponse() {}

// Redirect is a route response that redirects without rendering.
type Redirect struct {
	Status   int
	Location string
	headers  http.Header
}

func (Redirect) goldrRouteResponse()         {}
func (Redirect) goldrPageRouteResponse()     {}
func (Redirect) goldrFragmentRouteResponse() {}

// Text is a plain text route response.
type Text struct {
	Status  int
	Body    string
	headers http.Header
}

func (Text) goldrRouteResponse()         {}
func (Text) goldrPageRouteResponse()     {}
func (Text) goldrFragmentRouteResponse() {}

// NoContent is a response that writes headers and status without a body.
type NoContent struct {
	Status  int
	headers http.Header
}

func (NoContent) goldrRouteResponse() {}

// RouteError delegates a matched route error to generated error handling.
type RouteError struct {
	Err error
}

func (RouteError) goldrRouteResponse()         {}
func (RouteError) goldrPageRouteResponse()     {}
func (RouteError) goldrFragmentRouteResponse() {}

type routeResponseKind uint8

const (
	routeResponseInvalid routeResponseKind = iota
	routeResponsePage
	routeResponseFragment
	routeResponseRedirect
	routeResponseText
	routeResponseNoContent
	routeResponseRouteError
)

type resolvedRouteResponse struct {
	kind     routeResponseKind
	page     Page
	fragment Fragment
	redirect Redirect
	text     Text
	noBody   NoContent
	err      error
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
	// Data is app-owned page response data visible to the layout chain.
	Data LayoutData
}

// LayoutData carries app-owned page response data for matching layouts.
type LayoutData struct {
	values map[*layoutKeyID]any
}

type layoutKeyID struct {
	name string
}

// LayoutKey identifies one typed layout data value.
type LayoutKey[T any] struct {
	id *layoutKeyID
}

// NewPage returns a normal 200 OK rendered page response.
func NewPage(component templ.Component, metadata PageMetadata) Page {
	return Page{
		Status:    http.StatusOK,
		Component: component,
		Metadata:  metadata,
	}
}

// NewLayoutKey returns an app-owned key for one layout data value.
//
// The name is for humans; each call returns a distinct key identity even when
// names match.
func NewLayoutKey[T any](name string) LayoutKey[T] {
	return LayoutKey[T]{id: &layoutKeyID{name: name}}
}

// WithLayoutValue returns page with value stored under key for matching layouts.
func WithLayoutValue[T any](page Page, key LayoutKey[T], value T) Page {
	if key.id == nil {
		panic("goldr: zero-value LayoutKey passed to WithLayoutValue")
	}
	page.Data = withLayoutValue(page.Data, key.id, value)
	return page
}

// LayoutValue returns the layout value stored under key.
func LayoutValue[T any](ctx LayoutContext, key LayoutKey[T]) (T, bool) {
	var zero T
	if key.id == nil || ctx.Data.values == nil {
		return zero, false
	}
	value, ok := ctx.Data.values[key.id]
	if !ok {
		return zero, false
	}
	typed, ok := value.(T)
	if !ok {
		return zero, false
	}
	return typed, true
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

// WithStatus returns no-content response with status.
func (noContent NoContent) WithStatus(status int) NoContent {
	noContent.Status = status
	return noContent
}

// WithHeader returns no-content response with name set to value.
func (noContent NoContent) WithHeader(name, value string) NoContent {
	noContent.headers = withResponseHeader(noContent.headers, name, value)
	return noContent
}

// AddHeader returns no-content response with value added to name.
func (noContent NoContent) AddHeader(name, value string) NoContent {
	noContent.headers = addResponseHeader(noContent.headers, name, value)
	return noContent
}

func resolveRouteResponse(response RouteResponse) (resolvedRouteResponse, error) {
	switch response := response.(type) {
	case Page:
		return resolvePageResponse(response)
	case *Page:
		if response == nil {
			return resolvedRouteResponse{}, ErrInvalidRouteResponse
		}
		return resolvePageResponse(*response)
	case Fragment:
		return resolveFragmentResponse(response)
	case *Fragment:
		if response == nil {
			return resolvedRouteResponse{}, ErrInvalidRouteResponse
		}
		return resolveFragmentResponse(*response)
	case Redirect:
		return resolveRedirectResponse(response)
	case *Redirect:
		if response == nil {
			return resolvedRouteResponse{}, ErrInvalidRouteResponse
		}
		return resolveRedirectResponse(*response)
	case Text:
		return resolveTextResponse(response)
	case *Text:
		if response == nil {
			return resolvedRouteResponse{}, ErrInvalidRouteResponse
		}
		return resolveTextResponse(*response)
	case NoContent:
		return resolveNoContentResponse(response)
	case *NoContent:
		if response == nil {
			return resolvedRouteResponse{}, ErrInvalidRouteResponse
		}
		return resolveNoContentResponse(*response)
	case RouteError:
		return resolveRouteErrorResponse(response)
	case *RouteError:
		if response == nil {
			return resolvedRouteResponse{}, ErrInvalidRouteResponse
		}
		return resolveRouteErrorResponse(*response)
	default:
		return resolvedRouteResponse{}, ErrInvalidRouteResponse
	}
}

func resolvePageResponse(response Page) (resolvedRouteResponse, error) {
	status := response.Status
	if status == 0 {
		status = http.StatusOK
	}
	if err := validateRenderResponse(response.Component, status); err != nil {
		return resolvedRouteResponse{}, err
	}
	response.Status = status
	response.headers = cloneResponseHeaders(response.headers)
	return resolvedRouteResponse{
		kind: routeResponsePage,
		page: response,
	}, nil
}

func resolveFragmentResponse(response Fragment) (resolvedRouteResponse, error) {
	status := response.Status
	if status == 0 {
		status = http.StatusOK
	}
	if err := validateRenderResponse(response.Component, status); err != nil {
		return resolvedRouteResponse{}, err
	}
	response.Status = status
	response.headers = cloneResponseHeaders(response.headers)
	if response.headers == nil {
		response.headers = make(http.Header)
	}
	if response.headers.Get("Cache-Control") == "" {
		response.headers.Set("Cache-Control", fragmentDefaultCacheControl)
	}
	return resolvedRouteResponse{
		kind:     routeResponseFragment,
		fragment: response,
	}, nil
}

func resolveRedirectResponse(response Redirect) (resolvedRouteResponse, error) {
	if err := validateRedirectResponse(response.Location, response.Status); err != nil {
		return resolvedRouteResponse{}, err
	}
	response.headers = cloneResponseHeaders(response.headers)
	return resolvedRouteResponse{
		kind:     routeResponseRedirect,
		redirect: response,
	}, nil
}

func resolveTextResponse(response Text) (resolvedRouteResponse, error) {
	status := response.Status
	if status == 0 {
		status = http.StatusOK
	}
	if err := validateTextResponse(status); err != nil {
		return resolvedRouteResponse{}, err
	}
	response.Status = status
	response.headers = cloneResponseHeaders(response.headers)
	return resolvedRouteResponse{
		kind: routeResponseText,
		text: response,
	}, nil
}

func resolveNoContentResponse(response NoContent) (resolvedRouteResponse, error) {
	status := response.Status
	if status == 0 {
		status = http.StatusNoContent
	}
	if !validNoContentStatus(status) {
		return resolvedRouteResponse{}, ErrInvalidRouteResponse
	}
	response.Status = status
	response.headers = cloneResponseHeaders(response.headers)
	return resolvedRouteResponse{
		kind:   routeResponseNoContent,
		noBody: response,
	}, nil
}

func resolveRouteErrorResponse(response RouteError) (resolvedRouteResponse, error) {
	if response.Err == nil {
		return resolvedRouteResponse{}, ErrNilRouteError
	}
	return resolvedRouteResponse{
		kind: routeResponseRouteError,
		err:  response.Err,
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

func withLayoutValue(data LayoutData, id *layoutKeyID, value any) LayoutData {
	next := cloneLayoutData(data)
	if next.values == nil {
		next.values = make(map[*layoutKeyID]any)
	}
	next.values[id] = value
	return next
}

func cloneLayoutData(data LayoutData) LayoutData {
	if len(data.values) == 0 {
		return LayoutData{}
	}

	clone := make(map[*layoutKeyID]any, len(data.values))
	maps.Copy(clone, data.values)
	return LayoutData{values: clone}
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

func validNoContentStatus(status int) bool {
	switch status {
	case http.StatusNoContent, http.StatusResetContent, http.StatusNotModified:
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
