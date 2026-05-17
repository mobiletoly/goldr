# goldr (Go Layout-Driven Router)

goldr is a server-first Go framework for building web/HTMX applications.

It gives Go projects a predictable filesystem route tree, page layouts,
fragments, action handlers, generated route wiring, and generated URL helpers.
It also includes small helpers for form parsing, CSRF tokens, HTMX headers,
server-sent event wire formatting, and final-file asset fingerprinting. The
application still owns its `net/http` server, middleware, static asset
handlers, auth, sessions, data access, and deployment.

goldr is v0. APIs and conventions may change before v1.

In goldr, the filesystem is the route map:

```text
app/routes/
  layout.go          -> layout logic for / and below
  layout.templ       -> layout HTML
  page.go            -> GET /
  page.templ         -> page HTML
  users/
    layout.go        -> layout logic for /users and below
    layout.templ     -> users layout HTML
    page.go          -> GET /users
    page.templ       -> users page HTML
    by_id/
      page.go        -> GET /users/{id}
      page.templ     -> user detail HTML
    frag_table.go    -> GET /users/frag-table
    frag_table.templ -> fragment HTML
    actions.go       -> POST /users/create via PostCreate
```

Pages define endpoints, layouts wrap pages in their route directory and below,
fragments are standalone HTMX partials, and actions are ordinary Go handlers
colocated with the route they mutate.

Static directory underscores become hyphens in browser URLs, so Go-safe source
names such as `build_info/` can serve stable paths such as `/build-info`.

## Why goldr

goldr is for developers who want modern web application structure without
turning the browser into the center of the system.

With goldr:

- routes are visible in the filesystem
- pages and layouts are rendered on the server
- HTMX attributes stay visible in templates
- forms and mutations use ordinary Go handlers
- CSRF protection stays explicit and middleware-friendly
- generated URL helpers remove hard-coded paths
- generated route wiring stays inspectable
- final static files can be fingerprinted without adopting an asset pipeline

goldr intentionally avoids SPA routing, hydration, virtual DOM, framework-owned
client state, and hidden runtime registration.

## Install

Goldr applications use Go and [templ](https://github.com/a-h/templ). During v0,
templ is Goldr's render contract: route functions return templ components, and
`.templ` files own HTML rendering. Goldr owns the filesystem route model,
generated route wiring, URL helpers, and validation around that workflow.

The current module targets Go 1.26.

Add goldr, templ, and app-local CLI tools to your module:

```bash
go get github.com/mobiletoly/goldr github.com/a-h/templ@v0.3.1020
go get -tool github.com/mobiletoly/goldr/cmd/goldr@latest
go get -tool github.com/a-h/templ/cmd/templ@v0.3.1020
```

Then run goldr and templ with `go tool goldr` and `go tool templ`. This keeps
the commands versioned with the application. If you prefer a global convenience
binary, `go install github.com/mobiletoly/goldr/cmd/goldr@latest` also works.

## Quick Start

Create a new module:

```bash
mkdir hello-goldr
cd hello-goldr
go mod init example.com/hello-goldr
go get github.com/mobiletoly/goldr github.com/a-h/templ@v0.3.1020
go get -tool github.com/mobiletoly/goldr/cmd/goldr@latest
go get -tool github.com/a-h/templ/cmd/templ@v0.3.1020
```

Add `main.go`:

```go
package main

import (
	"log"
	"net/http"

	"example.com/hello-goldr/app/routes"
)

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", routes.Handler())

	log.Println("listening on http://127.0.0.1:8080")
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", mux))
}
```

Create the route directory:

```bash
mkdir -p app/routes
```

Add `app/routes/page.go`:

```go
package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(_ *http.Request) goldr.Page {
	return goldr.Page{
		Component: PageView(),
		Metadata: goldr.PageMetadata{
			Title: "Hello goldr",
		},
	}
}
```

Add `app/routes/page.templ`:

```templ
package routes

templ PageView() {
	<section>
		<h1>Hello goldr</h1>
		<p>Edit app/routes/page.templ to start building.</p>
	</section>
}
```

Add `app/routes/layout.go`:

```go
package routes

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

const defaultTitle = "Hello goldr"

func Layout(_ *http.Request, ctx goldr.LayoutContext) templ.Component {
	return LayoutView(ctx.Metadata, ctx.Child)
}

func pageTitle(metadata goldr.PageMetadata) string {
	if metadata.Title != "" {
		return metadata.Title
	}
	return defaultTitle
}
```

Add `app/routes/layout.templ`:

```templ
package routes

import "github.com/mobiletoly/goldr"

templ LayoutView(metadata goldr.PageMetadata, child templ.Component) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1"/>
			<title>{ pageTitle(metadata) }</title>
			<script src="https://cdn.jsdelivr.net/npm/htmx.org@4.0.0-beta3" integrity="sha384-bq4nTap5u8w4XlVP8JHkDioQVZBI5wUx5PxNwlbCq27H5QJ+q0CSeJcTYU+PLdCp" crossorigin="anonymous" defer></script>
		</head>
		<body>
			<main>
				@child
			</main>
		</body>
	</html>
}
```

Generate templ output and goldr route wiring, validate, and run:

```bash
go tool goldr generate
go tool goldr check
go run .
```

`goldr check` verifies Goldr-owned generated files, templ-generated files, and
Goldr-managed asset outputs are current. It does not write them.

Open:

```text
http://127.0.0.1:8080
```

After route or template edits, the normal loop is:

```bash
go tool goldr generate
go tool goldr check
go run .
```

For live reload during development:

```bash
go tool goldr dev
```

Read [Live Reload](docs/user/live-reload.md) for assets and Tailwind workflows.

## Optional Scaffold

`goldr init` can create the minimal route skeleton for an existing Go module:

```bash
go tool goldr init
```

It creates:

```text
app/routes/page.go
app/routes/page.templ
app/routes/layout.go
app/routes/layout.templ
app/routes/goldr_gen.go
app/urls/goldr_gen.go
```

`goldr init` does not create `go.mod`, edit `go.mod`, write `main.go`, run
templ, or start a server.

## How goldr Apps Are Shaped

goldr applications use a filesystem route tree rooted at `app/routes/`:

```text
app/
  routes/
    page.go
    page.templ
    layout.go
    layout.templ
    users/
      page.go
      page.templ
      layout.go
      layout.templ
      frag_table.go
      frag_table.templ
      actions.go
      by_id/
        page.go
        page.templ
  urls/
    goldr_gen.go
```

The conventions are Go-native:

- `page.go` defines a route page
- `layout.go` wraps pages in that directory and below
- `frag_*.go` defines an independently renderable HTMX fragment
- `actions.go` defines colocated mutation handlers such as `PostCreate`
- `by_id/` maps to a dynamic `{id}` route segment
- `build_info/` maps to a static `/build-info` browser segment

goldr generates route dispatch in `app/routes/goldr_gen.go` and URL helpers in
`app/urls/goldr_gen.go`.

## HTMX Stays Visible

goldr does not hide HTMX behind framework components. Templates keep ordinary
HTML and `hx-*` attributes:

```templ
package users

import "example.com/hello-goldr/app/urls"

templ DirectoryView() {
	<button
		hx-get={ urls.Users.FragTable.Path() }
		hx-target="#users-table"
		hx-swap="outerHTML"
	>
		Load users
	</button>
}
```

Handlers that need HTMX response headers can use the small `hx` package after
`goldr.Render` has buffered the templ response:

```go
func PostCreate(w http.ResponseWriter, r *http.Request) {
	response, err := goldr.Render(r, UsersTable())
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	hx.Retarget(w, "#users-table")
	hx.Reswap(w, "outerHTML")
	hx.Trigger(w, "user:created")
	_ = response.Write(w, r)
}
```

Use `response.WriteStatus(w, r, status)` for rendered HTML that needs a
non-200 status. Set HTMX and other response headers before either write method.

For app-owned server-sent event streams, use the small `sse` package for
event-stream headers, comments, event fields, templ-rendered HTML data, and
flushing. The application still owns stream URLs, mux registration,
subscribers, replay policy, and HTMX attributes.

## Try the Full Example

From a goldr checkout, run the full-feature example:

```bash
go run ./examples/full_feature
```

Inspect the route surface:

```bash
go run ./cmd/goldr routes list --root examples/full_feature
go run ./cmd/goldr routes layouts --root examples/full_feature
go run ./cmd/goldr assets list --root examples/full_feature
go run ./cmd/goldr check --root examples/full_feature
```

The example shows pages, nested layouts, fragments, actions, forms, generated
URL helpers, custom errors, middleware, and fingerprinted static assets in one
small app.

For a focused realtime example using server-sent events, run:

```bash
go run ./examples/chat
```

The chat example shows ordinary actions for input, app-owned in-memory
persistence, and an app-owned SSE stream that uses `sse` to push rendered HTML
to HTMX.

## Documentation

- [User documentation](docs/user/README.md)
- [Getting Started](docs/user/getting-started.md)
- [Routes](docs/user/routes.md)
- [CLI](docs/user/cli.md)
- [Live Reload](docs/user/live-reload.md)
- [Assets](docs/user/assets.md)
- [Coding Agents](docs/user/coding-agents.md)
- [HTMX](docs/user/htmx.md)
- [Forms](docs/user/forms.md)
- [Composition](docs/user/composition.md)

## License

goldr is licensed under the [Apache License 2.0](LICENSE.txt).
