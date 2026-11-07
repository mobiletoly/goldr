package goldr_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr"
)

func TestRecoverBeforeCommit(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		t.Run(method, func(t *testing.T) {
			request := httptest.NewRequestWithContext(t.Context(), method, "/private-request?token=private-query", nil)
			writer := httptest.NewRecorder()
			calls := 0
			handler := goldr.Recover(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Set-Cookie", "private-cookie")
				w.Header().Set("Location", "/private-redirect")
				w.Header().Set("Hx-Redirect", "/private-htmx")
				panic("private-panic")
			}), func(r *http.Request, info goldr.PanicInfo) {
				calls++
				if r != request || info.Committed || info.Status != 0 || info.Hijacked {
					t.Fatalf("unexpected callback state: %+v", info)
				}
				assertRecoveryFrames(t, info)
				encoded, err := json.Marshal(info)
				if err != nil || strings.Contains(string(encoded), "private-") {
					t.Fatalf("unsafe report: %s, %v", encoded, err)
				}
			})
			handler.ServeHTTP(writer, request)
			if calls != 1 || writer.Code != http.StatusInternalServerError {
				t.Fatalf("calls=%d status=%d", calls, writer.Code)
			}
			for _, key := range []string{"Set-Cookie", "Location", "Hx-Redirect"} {
				if writer.Header().Get(key) != "" {
					t.Fatalf("pending %s survived", key)
				}
			}
			for key, want := range map[string]string{"Content-Type": "text/plain; charset=utf-8", "X-Content-Type-Options": "nosniff", "Cache-Control": "no-store", "Referrer-Policy": "no-referrer"} {
				if got := writer.Header().Get(key); got != want {
					t.Fatalf("%s=%q, want %q", key, got, want)
				}
			}
			want := "Internal server error.\n"
			if method == http.MethodHead {
				want = ""
			}
			if writer.Body.String() != want {
				t.Fatalf("body=%q", writer.Body.String())
			}
		})
	}
}

func TestRecoverPreservesOrdinaryHandlerAndOriginalContext(t *testing.T) {
	type contextKey struct{}
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	request = request.WithContext(context.WithValue(request.Context(), contextKey{}, "outer"))
	writer := httptest.NewRecorder()
	goldr.Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r != request {
			t.Fatal("request replaced")
		}
		w.Header().Set("X-Example", "kept")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("ordinary"))
	}), func(*http.Request, goldr.PanicInfo) { t.Fatal("ordinary request reported") }).ServeHTTP(writer, request)
	if writer.Code != http.StatusAccepted || writer.Body.String() != "ordinary" || writer.Header().Get("X-Example") != "kept" {
		t.Fatal("ordinary response changed")
	}
	goldr.Recover(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), contextKey{}, "inner"))
		_ = r
		panic("private-context-panic")
	}), func(r *http.Request, _ goldr.PanicInfo) {
		if r.Context().Value(contextKey{}) != "outer" {
			t.Fatal("callback did not receive wrapper request context")
		}
	}).ServeHTTP(httptest.NewRecorder(), request)
}

func TestRecoverContainsReporterPanicAndPassesAbortSentinel(t *testing.T) {
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	writer := httptest.NewRecorder()
	calls := 0
	goldr.Recover(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("private-handler") }), func(*http.Request, goldr.PanicInfo) {
		calls++
		panic("private-reporter")
	}).ServeHTTP(writer, request)
	if calls != 1 || writer.Code != http.StatusInternalServerError || writer.Body.String() != "Internal server error.\n" {
		t.Fatal("reporter panic changed safe fallback")
	}
	writer = httptest.NewRecorder()
	abort := captureRecoveryPanic(func() {
		goldr.Recover(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }), func(*http.Request, goldr.PanicInfo) {
			t.Fatal("abort sentinel reported")
		}).ServeHTTP(writer, request)
	})
	if abort != http.ErrAbortHandler || writer.Body.Len() != 0 { //nolint:errorlint // Assert the exact abort sentinel, matching net/http.
		t.Fatal("abort sentinel changed")
	}
}

func TestRecoverConstructionRequiresHandlerAndReporter(t *testing.T) {
	for _, test := range []struct {
		handler http.Handler
		report  func(*http.Request, goldr.PanicInfo)
	}{
		{report: func(*http.Request, goldr.PanicInfo) {}},
		{handler: http.NotFoundHandler()},
	} {
		got := captureRecoveryPanic(func() { goldr.Recover(test.handler, test.report) })
		if got != "goldr: recovery requires a handler and reporter" {
			t.Fatalf("construction panic=%v", got)
		}
	}
}

func TestRecoverWrappedAbortIsUnexpected(t *testing.T) {
	writer := httptest.NewRecorder()
	calls := 0
	goldr.Recover(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(fmt.Errorf("private-wrapped-panic: %w", http.ErrAbortHandler))
	}), func(_ *http.Request, info goldr.PanicInfo) {
		calls++
		assertRecoveryFrames(t, info)
	}).ServeHTTP(writer, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	if calls != 1 || writer.Code != http.StatusInternalServerError || writer.Body.String() != "Internal server error.\n" {
		t.Fatalf("wrapped abort changed recovery: calls=%d status=%d body=%q", calls, writer.Code, writer.Body.String())
	}
}

func TestRecoverSafeFramesDistinguishSitesAndBoundRecursion(t *testing.T) {
	for _, test := range []struct {
		handler http.HandlerFunc
		want    string
	}{{firstRecoveryPanicSite, "firstRecoveryPanicSite"}, {secondRecoveryPanicSite, "secondRecoveryPanicSite"}} {
		goldr.Recover(test.handler, func(_ *http.Request, info goldr.PanicInfo) {
			assertRecoveryFrames(t, info)
			if !strings.HasSuffix(info.Frames[0].Function, test.want) {
				t.Fatalf("origin missing: %+v", info.Frames)
			}
		}).ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	}
	goldr.Recover(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { recursiveRecoveryPanic(64) }), func(_ *http.Request, info goldr.PanicInfo) {
		assertRecoveryFrames(t, info)
		if !info.FramesTruncated {
			t.Fatal("deep stack was not marked truncated")
		}
	}).ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
}

func firstRecoveryPanicSite(http.ResponseWriter, *http.Request)  { panic("private-first") }
func secondRecoveryPanicSite(http.ResponseWriter, *http.Request) { panic("private-second") }
func recursiveRecoveryPanic(depth int) {
	if depth == 0 {
		panic("private-recursive")
	}
	recursiveRecoveryPanic(depth - 1)
}

func assertRecoveryFrames(t *testing.T, info goldr.PanicInfo) {
	t.Helper()
	if len(info.Frames) == 0 || len(info.Frames) > 16 {
		t.Fatalf("frame count=%d", len(info.Frames))
	}
	encoded, err := json.Marshal(info.Frames)
	if err != nil || len(encoded) > 4096 {
		t.Fatalf("frame bytes=%d error=%v", len(encoded), err)
	}
	for _, frame := range info.Frames {
		if frame.Line <= 0 || frame.File != filepath.Base(frame.File) || strings.ContainsAny(frame.File, `/\`) || len(frame.File) > 128 || len(frame.Function) > 256 {
			t.Fatalf("unsafe frame: %+v", frame)
		}
		if strings.HasPrefix(frame.Function, "runtime.") || strings.HasPrefix(frame.Function, "github.com/mobiletoly/goldr.Recover") {
			t.Fatalf("recovery plumbing retained: %+v", frame)
		}
		for _, text := range []string{frame.File, frame.Function} {
			for _, character := range text {
				if character < 32 || character > 126 {
					t.Fatalf("non-printable frame: %+v", frame)
				}
			}
		}
	}
}

func captureRecoveryPanic(run func()) (caught any) {
	defer func() { caught = recover() }()
	run()
	return nil
}
