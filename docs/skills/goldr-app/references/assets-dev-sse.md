# Assets, Dev Server, Browser Helpers, And SSE

Use this reference when editing static assets, live reload setup, Goldr browser
helpers, or server-sent event streams.

## Assets

Goldr fingerprints final browser-ready files. It does not compile CSS, bundle
JavaScript, minify files, optimize images, upload to a CDN, register static
handlers, or inject assets into layouts.

Use this shape:

```text
assets/
  src/                  optional app-owned source files
  build/                final browser-ready files Goldr reads
  dist/                 fingerprinted files Goldr writes
  .goldr/               Goldr-managed cleanup state
  goldr_assets_gen.go   generated manifest package
```

Only `assets/build` is input to Goldr. App-owned tools write final files there.

After assets are built:

```bash
go tool goldr generate
```

or narrowly:

```bash
go tool goldr assets dist
go tool goldr assets check
go tool goldr assets list
```

Use generated paths in templates:

```templ
import "myapp/assets"

<link rel="stylesheet" href={ assets.Path("app.css") }/>
<script src={ assets.Path("app.js") } defer></script>
```

Serve generated assets from app server setup:

```go
mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets.FS()))))
```

Apply immutable cache headers only to fingerprinted `/assets/` files, not to
pages, fragments, forms, or action responses.

## goldr dev

`go tool goldr dev` runs a local live-reload loop. The app still owns its HTTP
server.

Default shape:

```text
--root .
--app-url http://127.0.0.1:8080
--proxy-addr 127.0.0.1:7331
--cmd "go run ."
```

Open the proxy URL printed by `goldr dev`, not the app server URL.

Common usage:

```bash
go tool goldr dev
go tool goldr dev --cmd "go run ./cmd/web"
go tool goldr dev --app-url http://127.0.0.1:3000
go tool goldr dev --proxy-addr 127.0.0.1:7332
```

Goldr watches `.go`, `.templ`, and `assets/build`. If a separate asset tool is
needed, run it separately so it writes final files into `assets/build`.

Stop any dev server you start.

## Browser Helpers

Goldr's `browser` package serves optional helper scripts. The application must
mount it explicitly:

```go
mux.Handle("/goldr/", http.StripPrefix("/goldr/", browser.Handler()))
```

Goldr does not inject browser helper scripts automatically.

Do not put immutable cache headers in front of `/goldr/...` helpers. They are
stable URLs with revalidation. Immutable cache belongs on fingerprinted app
assets.

## Server-Sent Events

Goldr's `sse` package writes valid event-stream responses and can render templ
components into SSE `data:` lines. Applications own stream URLs, subscribers,
replay policy, persistence, authorization, and script inclusion.

```go
func Events(w http.ResponseWriter, r *http.Request) {
	stream, ok := sse.Start(w)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	stream.Comment("connected")
	stream.Flush()

	_ = stream.Component(r, sse.ComponentEvent{
		ID:        "123",
		Name:      "chat-message",
		Component: MessageView(message),
	})
	stream.Flush()
}
```

Browsers send the latest event ID back as `Last-Event-ID` when reconnecting:

```go
lastID := sse.LastEventID(r)
```

For named SSE event swaps with HTMX, mount `browser.Handler()`, load the helper
explicitly, and keep the stream connection visible in templates:

```html
<script src="/goldr/goldr-sse-event.js" defer></script>
```

```templ
<div
	id="messages"
	hx-sse:connect="/chat/events"
	goldr-sse-event="chat-message"
	hx-swap="beforeend scroll:bottom"
>
</div>
```
