// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
)

type recoveryWriter struct {
	http.ResponseWriter
	committed  bool
	status     int
	connection net.Conn
}

func (w *recoveryWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *recoveryWriter) WriteHeader(status int) {
	w.ResponseWriter.WriteHeader(status)
	if !w.committed && (status == http.StatusSwitchingProtocols || status >= 200) {
		w.committed, w.status = true, status
	}
}

func (w *recoveryWriter) Write(body []byte) (int, error) {
	if !w.committed {
		w.committed, w.status = true, http.StatusOK
	}
	return w.ResponseWriter.Write(body)
}

type recoveryFlusher struct{ state *recoveryWriter }

func (w recoveryFlusher) Flush() { _ = w.FlushError() }
func (w recoveryFlusher) FlushError() error {
	err := http.NewResponseController(w.state.ResponseWriter).Flush()
	// A supported flush can write the implicit status before returning a transport error.
	if !errors.Is(err, http.ErrNotSupported) && !w.state.committed {
		w.state.committed, w.state.status = true, http.StatusOK
	}
	return err
}

type recoveryHijacker struct{ state *recoveryWriter }

func (w recoveryHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	connection, buffer, err := http.NewResponseController(w.state.ResponseWriter).Hijack()
	if err == nil {
		w.state.connection = connection
	}
	return connection, buffer, err
}

type recoveryPusher struct{ state *recoveryWriter }

func (w recoveryPusher) Push(target string, options *http.PushOptions) error {
	return w.state.ResponseWriter.(http.Pusher).Push(target, options)
}

type recoveryReaderFrom struct{ state *recoveryWriter }

func (w recoveryReaderFrom) ReadFrom(reader io.Reader) (int64, error) {
	// Route each write through the tracker, including a panic after partial copying.
	return io.Copy(struct{ io.Writer }{w.state}, reader)
}

func recoveryResponseWriter(state *recoveryWriter) http.ResponseWriter {
	capabilities := 0
	for current := state.ResponseWriter; current != nil; {
		if _, ok := current.(interface{ FlushError() error }); ok {
			capabilities |= 1
		} else if _, ok := current.(http.Flusher); ok {
			capabilities |= 1
		}
		if _, ok := current.(http.Hijacker); ok {
			capabilities |= 2
		}
		unwrapper, ok := current.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			break
		}
		current = unwrapper.Unwrap()
	}
	if _, ok := state.ResponseWriter.(http.Pusher); ok {
		capabilities |= 4
	}
	if _, ok := state.ResponseWriter.(io.ReaderFrom); ok {
		capabilities |= 8
	}
	switch capabilities {
	case 1:
		return struct {
			*recoveryWriter
			recoveryFlusher
		}{state, recoveryFlusher{state}}
	case 2:
		return struct {
			*recoveryWriter
			recoveryHijacker
		}{state, recoveryHijacker{state}}
	case 3:
		return struct {
			*recoveryWriter
			recoveryFlusher
			recoveryHijacker
		}{state, recoveryFlusher{state}, recoveryHijacker{state}}
	case 4:
		return struct {
			*recoveryWriter
			recoveryPusher
		}{state, recoveryPusher{state}}
	case 5:
		return struct {
			*recoveryWriter
			recoveryFlusher
			recoveryPusher
		}{state, recoveryFlusher{state}, recoveryPusher{state}}
	case 6:
		return struct {
			*recoveryWriter
			recoveryHijacker
			recoveryPusher
		}{state, recoveryHijacker{state}, recoveryPusher{state}}
	case 7:
		return struct {
			*recoveryWriter
			recoveryFlusher
			recoveryHijacker
			recoveryPusher
		}{state, recoveryFlusher{state}, recoveryHijacker{state}, recoveryPusher{state}}
	case 8:
		return struct {
			*recoveryWriter
			recoveryReaderFrom
		}{state, recoveryReaderFrom{state}}
	case 9:
		return struct {
			*recoveryWriter
			recoveryFlusher
			recoveryReaderFrom
		}{state, recoveryFlusher{state}, recoveryReaderFrom{state}}
	case 10:
		return struct {
			*recoveryWriter
			recoveryHijacker
			recoveryReaderFrom
		}{state, recoveryHijacker{state}, recoveryReaderFrom{state}}
	case 11:
		return struct {
			*recoveryWriter
			recoveryFlusher
			recoveryHijacker
			recoveryReaderFrom
		}{state, recoveryFlusher{state}, recoveryHijacker{state}, recoveryReaderFrom{state}}
	case 12:
		return struct {
			*recoveryWriter
			recoveryPusher
			recoveryReaderFrom
		}{state, recoveryPusher{state}, recoveryReaderFrom{state}}
	case 13:
		return struct {
			*recoveryWriter
			recoveryFlusher
			recoveryPusher
			recoveryReaderFrom
		}{state, recoveryFlusher{state}, recoveryPusher{state}, recoveryReaderFrom{state}}
	case 14:
		return struct {
			*recoveryWriter
			recoveryHijacker
			recoveryPusher
			recoveryReaderFrom
		}{state, recoveryHijacker{state}, recoveryPusher{state}, recoveryReaderFrom{state}}
	case 15:
		return struct {
			*recoveryWriter
			recoveryFlusher
			recoveryHijacker
			recoveryPusher
			recoveryReaderFrom
		}{state, recoveryFlusher{state}, recoveryHijacker{state}, recoveryPusher{state}, recoveryReaderFrom{state}}
	default:
		return state
	}
}
