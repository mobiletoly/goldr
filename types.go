// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import (
	"errors"
	"net/http"

	"github.com/a-h/templ"
)

var (
	// ErrInvalidPageResponse reports a page response that cannot be written.
	ErrInvalidPageResponse = errors.New("invalid page response")
	// ErrNilPageError reports goldr.Error(nil).
	ErrNilPageError = errors.New("page error response: nil error")
)

type pageKind uint8

const (
	pageKindInvalid pageKind = iota
	pageKindRender
	pageKindRedirect
	pageKindStatus
	pageKindTextStatus
	pageKindError
)

// PageResponseKind identifies the response behavior for a PageResponse.
type PageResponseKind uint8

const (
	// PageResponseInvalid is never returned by valid Page responses.
	PageResponseInvalid PageResponseKind = iota
	// PageResponseRender renders a templ component through matching layouts.
	PageResponseRender
	// PageResponseRedirect writes a redirect without rendering.
	PageResponseRedirect
	// PageResponseText writes a plain text status response without rendering.
	PageResponseText
	// PageResponseError delegates an application error to Goldr error handling.
	PageResponseError
)

// Page is the response value returned by Goldr page functions.
//
// Construct Page values with RenderPage, Redirect, Status, TextStatus, or
// Error.
type Page struct {
	kind             pageKind
	component        templ.Component
	metadata         PageMetadata
	status           int
	redirectLocation string
	textBody         string
	err              error
}

// PageResponse is the generated-dispatch-facing view of a Page.
//
// Application code should create Page values with RenderPage, Redirect, Status,
// TextStatus, or Error. Goldr-generated dispatch calls Page.Response and uses
// the returned PageResponse to write the HTTP response.
type PageResponse struct {
	// Kind identifies which response behavior generated dispatch should use.
	Kind PageResponseKind
	// Component is rendered for PageResponseRender.
	Component templ.Component
	// Metadata is passed to layouts for PageResponseRender.
	Metadata PageMetadata
	// Status is the HTTP status written by render, redirect, and text responses.
	Status int
	// Location is the redirect target for PageResponseRedirect.
	Location string
	// Body is the plain text response body for PageResponseText.
	Body string
	// Error is the application error for PageResponseError.
	Error error
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

// RenderPage returns a normal rendered page response.
func RenderPage(component templ.Component, metadata PageMetadata) Page {
	return Page{
		kind:      pageKindRender,
		component: component,
		metadata:  metadata,
		status:    http.StatusOK,
	}
}

// Redirect returns a page response that redirects without rendering.
//
// Redirect accepts status codes 301, 302, 303, 307, and 308.
func Redirect(location string, status int) Page {
	return Page{
		kind:             pageKindRedirect,
		status:           status,
		redirectLocation: location,
	}
}

// Status returns a rendered page response with an explicit HTTP status.
//
// Status accepts final body-carrying non-redirect statuses: 2xx except 204 and
// 205, and 4xx-5xx.
func Status(status int, component templ.Component, metadata PageMetadata) Page {
	return Page{
		kind:      pageKindStatus,
		component: component,
		metadata:  metadata,
		status:    status,
	}
}

// TextStatus returns a plain text page response with an explicit HTTP status.
//
// TextStatus accepts final body-carrying non-redirect statuses: 2xx except 204
// and 205, and 4xx-5xx.
func TextStatus(status int, body string) Page {
	return Page{
		kind:     pageKindTextStatus,
		status:   status,
		textBody: body,
	}
}

// Error returns a page response handled by Goldr's internal server error path.
func Error(err error) Page {
	return Page{
		kind: pageKindError,
		err:  err,
	}
}

// Response returns the generated-dispatch-facing response for page.
//
// A non-nil error reports an invalid Page contract, such as a nil render
// component, invalid status, empty redirect location, or zero-value Page.
func (page Page) Response() (PageResponse, error) {
	switch page.kind {
	case pageKindRender, pageKindStatus:
		if err := validateRenderResponse(page.component, page.status); err != nil {
			return PageResponse{}, err
		}
		return PageResponse{
			Kind:      PageResponseRender,
			Component: page.component,
			Metadata:  page.metadata,
			Status:    page.status,
		}, nil
	case pageKindRedirect:
		if err := validateRedirectResponse(page.redirectLocation, page.status); err != nil {
			return PageResponse{}, err
		}
		return PageResponse{
			Kind:     PageResponseRedirect,
			Status:   page.status,
			Location: page.redirectLocation,
		}, nil
	case pageKindTextStatus:
		if err := validateTextResponse(page.status); err != nil {
			return PageResponse{}, err
		}
		return PageResponse{
			Kind:   PageResponseText,
			Status: page.status,
			Body:   page.textBody,
		}, nil
	case pageKindError:
		if page.err == nil {
			return PageResponse{}, ErrNilPageError
		}
		return PageResponse{
			Kind:  PageResponseError,
			Error: page.err,
		}, nil
	default:
		return PageResponse{}, ErrInvalidPageResponse
	}
}

func validateRenderResponse(component templ.Component, status int) error {
	if component == nil {
		return ErrNilComponent
	}
	if !validPageStatus(status) {
		return ErrInvalidPageResponse
	}
	return nil
}

func validateRedirectResponse(location string, status int) error {
	if location == "" {
		return ErrInvalidPageResponse
	}
	if !validRedirectStatus(status) {
		return ErrInvalidPageResponse
	}
	return nil
}

func validateTextResponse(status int) error {
	if !validPageStatus(status) {
		return ErrInvalidPageResponse
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
