// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
)

// PanicFrame identifies a source location without a path, address or argument values.
type PanicFrame struct {
	Function string
	File     string
	Line     int
}

// PanicInfo describes an unexpected serving-goroutine panic without its value or type.
// Frames contains at most 16 frames and 4 KiB of JSON-encoded frame-array data.
// Status is zero until a final response has been committed.
type PanicInfo struct {
	Frames          []PanicFrame
	FramesTruncated bool
	Committed       bool
	Status          int
	Hijacked        bool
}

// Recover wraps next with opt-in, whole-handler HTTP panic recovery. Both arguments
// are required. It reports an unexpected panic synchronously using the request passed
// to this wrapper; downstream WithContext copies are not implicitly visible.
//
// Before commitment Recover writes a fixed plain-text 500; after commitment it aborts
// the exchange without appending a response. An exact http.ErrAbortHandler passes
// through without reporting. Reporter panics are contained without recursive reporting.
// Detached goroutines and fatal runtime failures are outside this boundary.
func Recover(next http.Handler, report func(*http.Request, PanicInfo)) http.Handler {
	if next == nil || report == nil {
		panic("goldr: recovery requires a handler and reporter")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := &recoveryWriter{ResponseWriter: w}
		defer func() {
			failure := recover()
			if failure == nil {
				return
			}
			if failure == http.ErrAbortHandler { //nolint:errorlint // Only net/http's exact sentinel bypasses reporting.
				panic(http.ErrAbortHandler)
			}
			defer func() {
				if recover() != nil {
					panic(http.ErrAbortHandler)
				}
			}()
			info := recoveryPanicInfo()
			info.Committed, info.Status, info.Hijacked = state.committed, state.status, state.connection != nil
			reportRecoveryPanic(report, r, info)
			if state.connection != nil {
				_ = state.connection.Close()
			}
			if state.committed || state.connection != nil {
				panic(http.ErrAbortHandler)
			}
			clear(w.Header())
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.WriteHeader(http.StatusInternalServerError)
			if r.Method != http.MethodHead {
				_, _ = w.Write([]byte("Internal server error.\n"))
			}
		}()
		next.ServeHTTP(recoveryResponseWriter(state), r)
	})
}

func reportRecoveryPanic(report func(*http.Request, PanicInfo), r *http.Request, info PanicInfo) {
	defer func() { _ = recover() }()
	report(r, info)
}

func recoveryPanicInfo() PanicInfo {
	var pcs [64]uintptr
	count := runtime.Callers(2, pcs[:])
	_, recoveryFile, _, _ := runtime.Caller(0)
	writerFile := filepath.Join(filepath.Dir(recoveryFile), "recover_writer.go")
	frames := runtime.CallersFrames(pcs[:count])
	info := PanicInfo{Frames: []PanicFrame{}, FramesTruncated: count == len(pcs)}
	for {
		frame, more := frames.Next()
		if frame.Line > 0 && !strings.HasPrefix(frame.Function, "runtime.") &&
			frame.File != recoveryFile && filepath.Clean(frame.File) != writerFile {
			if len(info.Frames) == 16 {
				info.FramesTruncated = true
				break
			}
			function, functionCut := recoveryFrameText(frame.Function, 256)
			file, fileCut := recoveryFrameText(filepath.Base(frame.File), 128)
			candidate := append(info.Frames, PanicFrame{Function: function, File: file, Line: frame.Line})
			encoded, _ := json.Marshal(candidate)
			if len(encoded) > 4096 {
				info.FramesTruncated = true
				break
			}
			info.Frames = candidate
			info.FramesTruncated = info.FramesTruncated || functionCut || fileCut
		}
		if !more {
			break
		}
	}
	return info
}

func recoveryFrameText(text string, maximum int) (string, bool) {
	var result strings.Builder
	changed := false
	for _, character := range text {
		if result.Len() == maximum {
			changed = true
			break
		}
		if character < ' ' || character > '~' {
			character, changed = '?', true
		}
		result.WriteByte(byte(character))
	}
	return result.String(), changed
}
