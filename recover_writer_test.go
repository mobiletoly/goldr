package goldr

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mobiletoly/goldr/sse"
)

func TestRecoverWriterCommitment(t *testing.T) {
	for _, test := range []struct {
		name        string
		write       func(http.ResponseWriter)
		committed   bool
		status      int
		wantHeaders []int
	}{
		{"informational", func(w http.ResponseWriter) { w.WriteHeader(http.StatusEarlyHints) }, false, 0, []int{http.StatusEarlyHints, http.StatusInternalServerError}},
		{"switch protocols", func(w http.ResponseWriter) { w.WriteHeader(http.StatusSwitchingProtocols) }, true, http.StatusSwitchingProtocols, []int{http.StatusSwitchingProtocols}},
		{"explicit status", func(w http.ResponseWriter) { w.WriteHeader(http.StatusAccepted); w.WriteHeader(http.StatusForbidden) }, true, http.StatusAccepted, []int{http.StatusAccepted, http.StatusForbidden}},
		{"body", func(w http.ResponseWriter) { _, _ = w.Write([]byte("partial")) }, true, http.StatusOK, nil},
		{"empty body write", func(w http.ResponseWriter) { _, _ = w.Write(nil) }, true, http.StatusOK, nil},
		{"flush", func(w http.ResponseWriter) { w.(http.Flusher).Flush() }, true, http.StatusOK, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			base := &recoveryTestWriter{header: make(http.Header)}
			writer := struct {
				*recoveryTestWriter
				http.Flusher
			}{base, httptest.NewRecorder()}
			calls := 0
			var info PanicInfo
			caught := catchWriterPanic(func() {
				Recover(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					test.write(w)
					panic("private-writer-panic")
				}), func(_ *http.Request, got PanicInfo) { calls++; info = got }).ServeHTTP(writer, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
			})
			if calls != 1 || info.Committed != test.committed || info.Status != test.status || !slices.Equal(base.statuses, test.wantHeaders) {
				t.Fatalf("calls=%d info=%+v headers=%v", calls, info, base.statuses)
			}
			if test.committed && caught != http.ErrAbortHandler || !test.committed && caught != nil { //nolint:errorlint // Assert the exact abort sentinel, matching net/http.
				t.Fatalf("unexpected panic=%v", caught)
			}
			if test.committed && strings.Contains(base.body.String(), "Internal server error") {
				t.Fatal("fallback appended to committed response")
			}
		})
	}
}

func TestRecoverWriterPreservesConditionalCapabilities(t *testing.T) {
	base := &recoveryTestWriter{header: make(http.Header)}
	flusher := httptest.NewRecorder()
	hijacker := &recoveryTestHijacker{}
	pusher := &recoveryTestPusher{}
	readerFrom := &recoveryTestReaderFrom{}
	for _, test := range []struct {
		name                          string
		writer                        http.ResponseWriter
		flush, hijack, push, readFrom bool
	}{
		{name: "plain", writer: base},
		{name: "flush", writer: struct {
			http.ResponseWriter
			http.Flusher
		}{base, flusher}, flush: true},
		{name: "hijack", writer: struct {
			http.ResponseWriter
			http.Hijacker
		}{base, hijacker}, hijack: true},
		{name: "push", writer: struct {
			http.ResponseWriter
			http.Pusher
		}{base, pusher}, push: true},
		{name: "reader", writer: struct {
			http.ResponseWriter
			io.ReaderFrom
		}{base, readerFrom}, readFrom: true},
		{name: "all", writer: struct {
			http.ResponseWriter
			http.Flusher
			http.Hijacker
			http.Pusher
			io.ReaderFrom
		}{base, flusher, hijacker, pusher, readerFrom}, flush: true, hijack: true, push: true, readFrom: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			Recover(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, flush := w.(http.Flusher)
				_, flushError := w.(interface{ FlushError() error })
				_, hijack := w.(http.Hijacker)
				_, push := w.(http.Pusher)
				_, readFrom := w.(io.ReaderFrom)
				if flush != test.flush || flushError != test.flush || hijack != test.hijack || push != test.push || readFrom != test.readFrom {
					t.Fatalf("capabilities: flush=%v flushError=%v hijack=%v push=%v readFrom=%v", flush, flushError, hijack, push, readFrom)
				}
				if _, ok := sse.Start(w); ok != test.flush {
					t.Fatal("SSE support changed")
				}
			}), func(*http.Request, PanicInfo) { t.Fatal("ordinary capability check panicked") }).ServeHTTP(test.writer, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
		})
	}
}

func TestRecoverWriterDelegatesControllerAndPush(t *testing.T) {
	base := &recoveryTestWriter{header: make(http.Header)}
	pusher := &recoveryTestPusher{}
	writer := struct {
		*recoveryTestWriter
		http.Pusher
	}{base, pusher}
	Recover(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		controller := http.NewResponseController(w)
		deadline := time.Unix(1, 0)
		if err := controller.SetReadDeadline(deadline); err != nil {
			t.Fatal(err)
		}
		if err := controller.SetWriteDeadline(deadline); err != nil {
			t.Fatal(err)
		}
		if err := controller.EnableFullDuplex(); err != nil {
			t.Fatal(err)
		}
		options := &http.PushOptions{Method: http.MethodGet}
		if err := w.(http.Pusher).Push("/asset", options); err != nil {
			t.Fatal(err)
		}
		if base.readDeadline != deadline || base.writeDeadline != deadline || !base.fullDuplex || pusher.target != "/asset" || pusher.options != options {
			t.Fatal("controller or push delegation changed")
		}
	}), func(*http.Request, PanicInfo) { t.Fatal("delegation panicked") }).ServeHTTP(writer, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
}

func TestRecoverWriterTracksUnwrappedFlushAndFlushError(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failure"}[fail], func(t *testing.T) {
			base := &recoveryTestWriter{header: make(http.Header)}
			flush := &recoveryTestFlushError{ResponseWriter: base}
			if fail {
				flush.failure = http.ErrNotSupported
			}
			writer := recoveryTestUnwrapper{flush}
			caught := catchWriterPanic(func() {
				Recover(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					if err := http.NewResponseController(w).Flush(); !errors.Is(err, flush.failure) {
						t.Fatalf("flush error=%v", err)
					}
					panic("private-flush")
				}), func(_ *http.Request, info PanicInfo) {
					if info.Committed == fail {
						t.Fatalf("flush commitment=%v", info.Committed)
					}
				}).ServeHTTP(writer, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
			})
			if fail && caught != nil || !fail && caught != http.ErrAbortHandler { //nolint:errorlint // Assert the exact abort sentinel, matching net/http.
				t.Fatalf("panic=%v", caught)
			}
		})
	}
}

func TestRecoverWriterReaderFromTracksPartialAndEmptyCopy(t *testing.T) {
	for _, partial := range []bool{false, true} {
		base := &recoveryTestWriter{header: make(http.Header)}
		writer := struct {
			http.ResponseWriter
			io.ReaderFrom
		}{base, &recoveryTestReaderFrom{}}
		caught := catchWriterPanic(func() {
			Recover(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if partial {
					_, _ = w.(io.ReaderFrom).ReadFrom(&recoveryTestPanicReader{})
				} else {
					n, err := w.(io.ReaderFrom).ReadFrom(strings.NewReader(""))
					if n != 0 || err != nil {
						t.Fatalf("empty copy=%d %v", n, err)
					}
					panic("private-empty-copy")
				}
			}), func(_ *http.Request, info PanicInfo) {
				if info.Committed != partial {
					t.Fatalf("copy commitment=%v", info.Committed)
				}
			}).ServeHTTP(writer, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
		})
		if partial && (caught != http.ErrAbortHandler || base.body.String() != "partial") || !partial && caught != nil { //nolint:errorlint // Assert the exact abort sentinel, matching net/http.
			t.Fatalf("copy panic=%v body=%q", caught, base.body.String())
		}
	}
}

func TestRecoverWriterFailedFlushDoesNotAppendFallback(t *testing.T) {
	base := &recoveryTestWriter{header: make(http.Header)}
	writer := &recoveryTestFlushError{ResponseWriter: base, failure: io.ErrClosedPipe, commit: true}
	caught := catchWriterPanic(func() {
		Recover(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if err := http.NewResponseController(w).Flush(); !errors.Is(err, io.ErrClosedPipe) {
				t.Fatalf("flush error=%v", err)
			}
			panic("private-after-failed-flush")
		}), func(_ *http.Request, info PanicInfo) {
			if !info.Committed || info.Status != http.StatusOK {
				t.Errorf("failed flush lost commitment: %+v", info)
			}
		}).ServeHTTP(writer, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	})
	if caught != http.ErrAbortHandler || base.body.Len() != 0 || !slices.Equal(base.statuses, []int{http.StatusOK}) { //nolint:errorlint // Assert the exact abort sentinel, matching net/http.
		t.Fatalf("failed flush changed response: panic=%v statuses=%v body=%q", caught, base.statuses, base.body.String())
	}
}

func TestRecoverWriterClosesHijackedConnection(t *testing.T) {
	connection, peer := net.Pipe()
	defer func() { _ = peer.Close() }()
	defer func() { _ = connection.Close() }()
	if err := peer.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	base := &recoveryTestWriter{header: make(http.Header)}
	hijacker := &recoveryTestHijacker{connection: connection}
	writer := struct {
		http.ResponseWriter
		http.Hijacker
	}{base, hijacker}
	caught := catchWriterPanic(func() {
		Recover(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			got, _, err := http.NewResponseController(w).Hijack()
			if err != nil || got != connection {
				t.Fatalf("hijack error=%v", err)
			}
			panic("private-hijack")
		}), func(_ *http.Request, info PanicInfo) {
			if !info.Hijacked {
				t.Fatal("hijack missing")
			}
		}).ServeHTTP(writer, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	})
	if caught != http.ErrAbortHandler || base.body.Len() != 0 || len(base.statuses) != 0 { //nolint:errorlint // Assert the exact abort sentinel, matching net/http.
		t.Fatal("hijacked fallback changed")
	}
	if _, err := peer.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Fatalf("connection not closed: %v", err)
	}
}

func TestRecoverFrameTextBoundsAndSanitizes(t *testing.T) {
	got, changed := recoveryFrameText(strings.Repeat("a", 257), 256)
	if len(got) != 256 || !changed {
		t.Fatal("symbol limit lost")
	}
	got, changed = recoveryFrameText("safe\n\x00\u2603", 128)
	if got != "safe???" || !changed {
		t.Fatalf("unsafe frame text=%q", got)
	}
}

func TestRecoverWriterPanicFramesExcludeAdapters(t *testing.T) {
	for _, test := range []struct {
		name   string
		invoke func(http.ResponseWriter)
	}{
		{"Write", func(w http.ResponseWriter) { _, _ = w.Write([]byte("private-body")) }},
		{"FlushError", func(w http.ResponseWriter) { _ = http.NewResponseController(w).Flush() }},
		{"Hijack", func(w http.ResponseWriter) { _, _, _ = http.NewResponseController(w).Hijack() }},
		{"Push", func(w http.ResponseWriter) { _ = w.(http.Pusher).Push("/private-target", nil) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			writer := recoveryTestPanickingWriter{httptest.NewRecorder()}
			calls := 0
			_ = catchWriterPanic(func() {
				Recover(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					test.invoke(w)
				}), func(_ *http.Request, info PanicInfo) {
					calls++
					if len(info.Frames) == 0 || info.Frames[0].File != "recover_writer_test.go" ||
						!strings.HasSuffix(info.Frames[0].Function, ".recoveryTestPanickingWriter."+test.name) {
						t.Errorf("application panic site missing: %+v", info.Frames)
					}
					for _, frame := range info.Frames {
						if frame.File == "recover.go" || frame.File == "recover_writer.go" {
							t.Errorf("recovery adapter frame retained: %+v", frame)
						}
					}
				}).ServeHTTP(writer, httptest.NewRequestWithContext(t.Context(), http.MethodHead, "/", nil))
			})
			if calls != 1 {
				t.Fatalf("panic reports=%d, want one", calls)
			}
		})
	}
}

type recoveryTestPanickingWriter struct{ http.ResponseWriter }

func (recoveryTestPanickingWriter) Write([]byte) (int, error) { panic("private-write-panic") }
func (recoveryTestPanickingWriter) FlushError() error         { panic("private-flush-panic") }
func (recoveryTestPanickingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	panic("private-hijack-panic")
}
func (recoveryTestPanickingWriter) Push(string, *http.PushOptions) error {
	panic("private-push-panic")
}

type recoveryTestWriter struct {
	header                      http.Header
	statuses                    []int
	body                        bytes.Buffer
	readDeadline, writeDeadline time.Time
	fullDuplex                  bool
}

func (w *recoveryTestWriter) Header() http.Header            { return w.header }
func (w *recoveryTestWriter) WriteHeader(status int)         { w.statuses = append(w.statuses, status) }
func (w *recoveryTestWriter) Write(body []byte) (int, error) { return w.body.Write(body) }
func (w *recoveryTestWriter) SetReadDeadline(deadline time.Time) error {
	w.readDeadline = deadline
	return nil
}
func (w *recoveryTestWriter) SetWriteDeadline(deadline time.Time) error {
	w.writeDeadline = deadline
	return nil
}
func (w *recoveryTestWriter) EnableFullDuplex() error { w.fullDuplex = true; return nil }

type recoveryTestHijacker struct{ connection net.Conn }

func (w *recoveryTestHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.connection, nil, nil
}

type recoveryTestPusher struct {
	target  string
	options *http.PushOptions
}

func (w *recoveryTestPusher) Push(target string, options *http.PushOptions) error {
	w.target, w.options = target, options
	return nil
}

type recoveryTestReaderFrom struct{}

func (*recoveryTestReaderFrom) ReadFrom(reader io.Reader) (int64, error) {
	return io.Copy(io.Discard, reader)
}

type recoveryTestFlushError struct {
	http.ResponseWriter
	failure error
	commit  bool
}

func (w *recoveryTestFlushError) FlushError() error {
	if w.commit {
		w.WriteHeader(http.StatusOK)
	}
	return w.failure
}

type recoveryTestUnwrapper struct{ http.ResponseWriter }

func (w recoveryTestUnwrapper) Unwrap() http.ResponseWriter { return w.ResponseWriter }

type recoveryTestPanicReader struct{ read bool }

func (r *recoveryTestPanicReader) Read(body []byte) (int, error) {
	if r.read {
		panic("private-reader")
	}
	r.read = true
	return copy(body, "partial"), nil
}
func catchWriterPanic(run func()) (caught any) {
	defer func() { caught = recover() }()
	run()
	return nil
}
