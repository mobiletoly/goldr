package sse

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/a-h/templ"
)

func TestStartSetsEventStreamHeaders(t *testing.T) {
	recorder := httptest.NewRecorder()

	stream, ok := Start(recorder)

	if !ok {
		t.Fatalf("Start() ok = false, want true")
	}
	if stream == nil {
		t.Fatalf("Start() stream = nil, want stream")
	}
	tests := []struct {
		header string
		want   string
	}{
		{header: "Content-Type", want: "text/event-stream"},
		{header: "Cache-Control", want: "no-cache"},
		{header: "X-Accel-Buffering", want: "no"},
	}
	for _, test := range tests {
		if got := recorder.Header().Get(test.header); got != test.want {
			t.Fatalf("%s = %q, want %q", test.header, got, test.want)
		}
	}
}

func TestStartRejectsWriterWithoutFlusher(t *testing.T) {
	writer := headerOnlyWriter{header: make(http.Header)}

	stream, ok := Start(writer)

	if ok {
		t.Fatalf("Start() ok = true, want false")
	}
	if stream != nil {
		t.Fatalf("Start() stream = %#v, want nil", stream)
	}
	if len(writer.header) != 0 {
		t.Fatalf("headers = %#v, want empty", writer.header)
	}
}

func TestStartUsesWrappedWriterFlusher(t *testing.T) {
	recorder := httptest.NewRecorder()
	writer := unwrapWriter{ResponseWriter: recorder}

	stream, ok := Start(writer)
	if !ok {
		t.Fatalf("Start() ok = false, want true")
	}
	if stream == nil {
		t.Fatalf("Start() stream = nil, want stream")
	}
	if recorder.Flushed {
		t.Fatalf("Start() flushed writer, want no flush before Flush")
	}

	stream.Flush()

	if !recorder.Flushed {
		t.Fatalf("Flush() did not flush wrapped writer")
	}
}

func TestCommentWritesSSECommentFrame(t *testing.T) {
	recorder := httptest.NewRecorder()
	stream, ok := Start(recorder)
	if !ok {
		t.Fatalf("Start() ok = false, want true")
	}

	if err := stream.Comment("connected\r\nagain"); err != nil {
		t.Fatalf("Comment() error = %v, want nil", err)
	}

	if got := recorder.Body.String(); got != ": connected\n: again\n\n" {
		t.Fatalf("body = %q, want comment frame", got)
	}
}

func TestEventWritesFieldsAndMultilineData(t *testing.T) {
	recorder := httptest.NewRecorder()
	stream, ok := Start(recorder)
	if !ok {
		t.Fatalf("Start() ok = false, want true")
	}

	err := stream.Event(Event{
		ID:    "42",
		Name:  "chat:message",
		Retry: 2 * time.Second,
		Data:  "hello\r\nworld",
	})

	if err != nil {
		t.Fatalf("Event() error = %v, want nil", err)
	}
	want := "id: 42\nevent: chat:message\nretry: 2000\ndata: hello\ndata: world\n\n"
	if got := recorder.Body.String(); got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestEventWritesEmptyDataField(t *testing.T) {
	recorder := httptest.NewRecorder()
	stream, ok := Start(recorder)
	if !ok {
		t.Fatalf("Start() ok = false, want true")
	}

	if err := stream.Event(Event{}); err != nil {
		t.Fatalf("Event() error = %v, want nil", err)
	}

	if got := recorder.Body.String(); got != "data: \n\n" {
		t.Fatalf("body = %q, want empty data event", got)
	}
}

func TestEventRejectsFieldNewlines(t *testing.T) {
	tests := []struct {
		name  string
		event Event
	}{
		{name: "id", event: Event{ID: "1\n2"}},
		{name: "event", event: Event{Name: "chat\rmessage"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			stream, ok := Start(recorder)
			if !ok {
				t.Fatalf("Start() ok = false, want true")
			}

			if err := stream.Event(test.event); err == nil {
				t.Fatalf("Event() error = nil, want error")
			}
		})
	}
}

func TestComponentRendersTemplHTMLAsEventData(t *testing.T) {
	recorder := httptest.NewRecorder()
	stream, ok := Start(recorder)
	if !ok {
		t.Fatalf("Start() ok = false, want true")
	}
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

	err := stream.Component(request, ComponentEvent{
		ID:        "7",
		Component: stringComponent("<article>Hello</article>"),
	})

	if err != nil {
		t.Fatalf("Component() error = %v, want nil", err)
	}
	want := "id: 7\ndata: <article>Hello</article>\n\n"
	if got := recorder.Body.String(); got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestComponentReturnsRenderError(t *testing.T) {
	recorder := httptest.NewRecorder()
	stream, ok := Start(recorder)
	if !ok {
		t.Fatalf("Start() ok = false, want true")
	}
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	componentErr := errors.New("render failed")

	err := stream.Component(request, ComponentEvent{
		Component: templ.ComponentFunc(func(_ context.Context, _ io.Writer) error {
			return componentErr
		}),
	})

	if !errors.Is(err, componentErr) {
		t.Fatalf("Component() error = %v, want render error", err)
	}
}

func TestComponentRejectsNilRequestAndComponent(t *testing.T) {
	recorder := httptest.NewRecorder()
	stream, ok := Start(recorder)
	if !ok {
		t.Fatalf("Start() ok = false, want true")
	}
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

	if err := stream.Component(nil, ComponentEvent{Component: stringComponent("ok")}); err == nil {
		t.Fatalf("Component(nil request) error = nil, want error")
	}
	if err := stream.Component(request, ComponentEvent{}); err == nil {
		t.Fatalf("Component(nil component) error = nil, want error")
	}
}

func TestFlushFlushesWriter(t *testing.T) {
	writer := flushWriter{header: make(http.Header)}
	stream, ok := Start(&writer)
	if !ok {
		t.Fatalf("Start() ok = false, want true")
	}

	stream.Flush()

	if writer.flushes != 1 {
		t.Fatalf("flushes = %d, want 1", writer.flushes)
	}
}

func TestLastEventIDReadsHeader(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	request.Header.Set(HeaderLastEventID, "42")

	if got := LastEventID(request); got != "42" {
		t.Fatalf("LastEventID() = %q, want 42", got)
	}
}

func stringComponent(value string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, value)
		return err
	})
}

type headerOnlyWriter struct {
	header http.Header
}

func (w headerOnlyWriter) Header() http.Header {
	return w.header
}

func (w headerOnlyWriter) Write([]byte) (int, error) {
	return 0, errors.New("unexpected write")
}

func (w headerOnlyWriter) WriteHeader(int) {}

type unwrapWriter struct {
	http.ResponseWriter
}

func (w unwrapWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

type flushWriter struct {
	header  http.Header
	flushes int
}

func (w *flushWriter) Header() http.Header {
	return w.header
}

func (w *flushWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (w *flushWriter) WriteHeader(int) {}

func (w *flushWriter) Flush() {
	w.flushes++
}
