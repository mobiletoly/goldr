// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

// Package sse writes server-sent event responses from application-owned stream
// handlers.
//
// The package owns only SSE wire formatting: response headers, comment frames,
// event fields, multiline data, templ component rendering, and flushing. The
// application still owns stream routes, subscriber state, replay policy,
// persistence, and HTMX attributes.
package sse

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/a-h/templ"
)

const (
	// HeaderLastEventID is the SSE reconnect header sent by browsers.
	HeaderLastEventID = "Last-Event-ID"
)

var (
	errNilRequest   = errors.New("sse component event: nil request")
	errNilComponent = errors.New("sse component event: nil component")
)

// Event is one server-sent event.
type Event struct {
	// ID is the optional SSE event id field.
	ID string
	// Name is the optional SSE event name field.
	Name string
	// Retry is the optional browser reconnect delay.
	Retry time.Duration
	// Data is the SSE event data payload.
	Data string
}

// ComponentEvent is one server-sent event with templ-rendered HTML data.
type ComponentEvent struct {
	// ID is the optional SSE event id field.
	ID string
	// Name is the optional SSE event name field.
	Name string
	// Retry is the optional browser reconnect delay.
	Retry time.Duration
	// Component is rendered as the SSE event data payload.
	Component templ.Component
}

// Stream writes server-sent events to a response.
type Stream struct {
	writer     http.ResponseWriter
	controller *http.ResponseController
}

// Start prepares an event-stream response.
//
// Start returns nil, false when the response writer does not support flushing.
// It does not write an error response; callers should choose the application
// error response.
func Start(w http.ResponseWriter) (*Stream, bool) {
	if !supportsFlush(w) {
		return nil, false
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	return &Stream{writer: w, controller: http.NewResponseController(w)}, true
}

// LastEventID returns the browser's last received SSE event ID.
func LastEventID(r *http.Request) string {
	return r.Header.Get(HeaderLastEventID)
}

// Comment writes an SSE comment frame.
func (s *Stream) Comment(text string) error {
	for _, line := range splitLines(text) {
		if _, err := fmt.Fprintf(s.writer, ": %s\n", line); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(s.writer, "\n")
	return err
}

// Event writes one SSE event.
func (s *Stream) Event(event Event) error {
	if err := validateFieldValue("id", event.ID); err != nil {
		return err
	}
	if err := validateFieldValue("event", event.Name); err != nil {
		return err
	}

	if event.ID != "" {
		if _, err := fmt.Fprintf(s.writer, "id: %s\n", event.ID); err != nil {
			return err
		}
	}
	if event.Name != "" {
		if _, err := fmt.Fprintf(s.writer, "event: %s\n", event.Name); err != nil {
			return err
		}
	}
	if event.Retry > 0 {
		if _, err := fmt.Fprintf(s.writer, "retry: %d\n", event.Retry.Milliseconds()); err != nil {
			return err
		}
	}
	for _, line := range splitLines(event.Data) {
		if _, err := fmt.Fprintf(s.writer, "data: %s\n", line); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(s.writer, "\n")
	return err
}

// Component renders a templ component and writes it as one SSE event.
func (s *Stream) Component(r *http.Request, event ComponentEvent) error {
	if r == nil {
		return errNilRequest
	}
	if event.Component == nil {
		return errNilComponent
	}

	var body bytes.Buffer
	if err := event.Component.Render(r.Context(), &body); err != nil {
		return err
	}

	return s.Event(Event{
		ID:    event.ID,
		Name:  event.Name,
		Retry: event.Retry,
		Data:  body.String(),
	})
}

// Flush flushes any buffered event-stream data.
func (s *Stream) Flush() {
	_ = s.controller.Flush()
}

type responseWriterUnwrapper interface {
	Unwrap() http.ResponseWriter
}

type flushErrorer interface {
	FlushError() error
}

func supportsFlush(w http.ResponseWriter) bool {
	for w != nil {
		switch writer := w.(type) {
		case flushErrorer:
			return true
		case http.Flusher:
			return true
		case responseWriterUnwrapper:
			w = writer.Unwrap()
		default:
			return false
		}
	}
	return false
}

func validateFieldValue(name string, value string) error {
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("sse event %s contains a newline", name)
	}
	return nil
}

func splitLines(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.Split(value, "\n")
}
