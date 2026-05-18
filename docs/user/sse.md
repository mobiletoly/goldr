# Server-Sent Events

Goldr provides two small pieces for realtime server-rendered HTML:

- `sse` writes valid event-stream responses and renders templ components into
  SSE `data:` lines.
- `browser` serves the optional `goldr-sse-event.js` helper for declarative
  named-event swaps in htmx.

Applications still own stream routes, subscriber state, replay policy,
persistence, authorization, and script inclusion. Goldr does not generate SSE
routes or hide HTMX attributes behind Go helpers.

## Stream Handler

Use the `sse` package inside an app-owned route handler:

```go
package chat

import (
	"net/http"
	"strconv"

	"github.com/mobiletoly/goldr/sse"
)

func Events(w http.ResponseWriter, r *http.Request) {
	stream, ok := sse.Start(w)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	stream.Comment("connected")
	stream.Flush()

	message := Message{ID: 1, Body: "Hello"}
	_ = stream.Component(r, sse.ComponentEvent{
		ID:        strconv.FormatInt(message.ID, 10),
		Name:      "chat-message",
		Component: MessageView(message),
	})
	stream.Flush()
}
```

`ComponentEvent.ID` writes the SSE `id:` field. Browsers send the latest event
ID back as `Last-Event-ID` when reconnecting, and applications decide how to
replay or resume from that value.

```go
lastID := sse.LastEventID(r)
```

## Unnamed Events

If the server omits the SSE `event:` field, htmx 4 swaps the message directly
through its SSE extension:

```go
_ = stream.Component(r, sse.ComponentEvent{
	ID:        strconv.FormatInt(message.ID, 10),
	Component: MessageView(message),
})
```

Use unnamed messages for a one-purpose HTML stream where every message should
target the same element and the event type is not part of the protocol.

## Named Event Swaps

Named SSE events are better when the event type is part of the stream contract:

```text
event: chat-message
id: 123
data: <article>...</article>
```

By default, htmx 4 dispatches named SSE events as DOM events instead of swapping
them as HTML. To swap a selected named event, mount Goldr's browser helper in
the application mux and load it explicitly in the layout.

```go
package main

import (
	"net/http"

	"myapp/app/routes"

	"github.com/mobiletoly/goldr/browser"
)

func handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/goldr/", http.StripPrefix("/goldr/", browser.Handler()))
	mux.Handle("/", routes.Handler())
	return mux
}
```

```html
<script src="/goldr/goldr-sse-event.js" defer></script>
```

Templates keep the HTMX SSE connection and the named event visible:

```templ
<div
	id="messages"
	hx-sse:connect="/chat/events"
	goldr-sse-event="chat-message"
	hx-swap="beforeend scroll:bottom"
>
	<!-- streamed message HTML lands here -->
</div>
```

The helper leaves unnamed messages and non-matching named events alone. This
keeps one stream able to carry other semantic events without accidentally
swapping them into the wrong target.

## Helper Caching

`browser.Handler()` serves `goldr-sse-event.js` from a stable URL with:

- `Cache-Control: no-cache`
- `Content-Type: text/javascript; charset=utf-8`
- a content-derived `ETag`
- `304 Not Modified` for matching `If-None-Match`

Because the URL is stable, keep revalidation enabled. Do not put immutable cache
headers in front of `/goldr/goldr-sse-event.js`. Immutable cache policy belongs
on fingerprinted app assets, not this helper.

## Runnable Example

`examples/chat/` demonstrates app-owned SSE streams with htmx 4
`hx-sse:connect`, ordinary generated actions for posted messages, in-memory
server-side persistence, `event: chat-message`, and `goldr-sse-event`.

Run it from the repository root:

```bash
go run ./examples/chat
```
