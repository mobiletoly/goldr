# goldr (Go Layout-Driven Router)

goldr is a server-first Go framework for building web/HTMX applications.

It gives Go projects a predictable filesystem route tree, page layouts,
fragments, action handlers, route-derived navigation trails, generated route
wiring, and generated URL helpers. It also includes small helpers for CSRF
tokens, HTMX headers, server-sent event wire formatting, named SSE browser
swaps, and final-file asset fingerprinting.
The application still owns its `net/http` server, middleware, static asset
handlers, auth, sessions, request parsing, validation, data access, and
deployment.

goldr is v0. APIs and conventions may change before v1.

In goldr, the filesystem is the route map:

```text
app/routes/
  layout.go          -> layout logic for / and below
  layout.templ       -> layout HTML
  route.go           -> GET /
  page.templ         -> page HTML
  users/
    layout.go        -> layout logic for /users and below
    layout.templ     -> users layout HTML
    route.go         -> GET /users, GET /users/table, POST /users/create
    page.templ       -> users page HTML
    by_id/
      route.go       -> GET /users/{id}
      page.templ     -> user detail HTML
    frag_table.templ -> fragment HTML
```

Route declarations define pages, fragments, and actions. Layouts wrap pages in
their route directory and below. Fragments are standalone HTMX partials, and
actions are ordinary Go handlers colocated with the route they mutate.

Reusable Kit route subtrees can live under `app/mounts` and be mounted by real
`app/routes` owners. The mounted subtree is not routable on its own; final
paths and URL helpers still come from the live route owner. Keep owner-only
children under the live owner and pass owner-only URLs through app data when
shared mounted templates need those links.

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
- route metadata can prepare breadcrumb-style UI and app-level Back links
- generated URL helpers remove hard-coded paths
- generated route wiring stays inspectable
- final static files can be fingerprinted without adopting an asset pipeline

goldr intentionally avoids SPA routing, hydration, virtual DOM, framework-owned
client state, and hidden runtime registration.

## Install

Goldr applications use Go and [templ](https://github.com/a-h/templ). During v0,
templ is Goldr's HTML render contract: page functions return
`goldr.PageRouteResponse`, fragment functions return
`goldr.FragmentRouteResponse`, actions return `goldr.RouteResponse`, and
`.templ` files own HTML rendering. Goldr
owns the filesystem route model, generated route wiring, URL helpers, and
validation around that workflow.

Use Go 1.26 or newer.

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

Add `app/routes/route.go`:

```go
package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

var Route = goldr.RouteDef{
	Page: page,
}

func page(_ *http.Request) goldr.PageRouteResponse {
	return goldr.NewPage(
		PageView(),
		goldr.PageMetadata{
			Title: "Hello goldr",
		},
	)
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

When `assets/build` exists, `goldr generate` also refreshes fingerprinted
assets. `goldr check` verifies Goldr-owned generated files, templ-generated
files, and Goldr-managed asset outputs are current. It does not write them.

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
app/routes/route.go
app/routes/page.templ
app/routes/layout.go
app/routes/layout.templ
app/routes/goldr_gen.go
app/internal/goldrinspect/goldr_gen.go
app/urls/goldr_gen.go
```

`goldr init` does not create `go.mod`, edit `go.mod`, write `main.go`, run
templ, or start a server.

## How goldr Apps Are Shaped

goldr applications use a filesystem route tree rooted at `app/routes/`:

```text
app/
  routes/
    route.go
    page.templ
    layout.go
    layout.templ
    users/
      route.go
      page.templ
      layout.go
      layout.templ
      frag_table.templ
      by_id/
        route.go
        page.templ
  urls/
    goldr_gen.go
```

The conventions are Go-native:

- `route.go` declares a route page, fragments, and actions
- `layout.go` wraps pages in that directory and below
- fragment declarations in `route.go` define independently renderable HTMX fragments, including optional index fragments at the route path
- action declarations in `route.go` define colocated mutation handlers
- `by_id/` maps to a dynamic `{id}` route segment
- `build_info/` maps to a static `/build-info` browser segment

goldr generates route dispatch in `app/routes/goldr_gen.go` and URL helpers in
`app/urls/goldr_gen.go`.

When an HTMX action or fragment only supports one page workflow, nest it under
that page route instead of creating a flat sibling route:

```text
users/
  route.go
  page.templ
  prepare/
    route.go
    action_handlers.go
    result.templ
  save/
    route.go
    action_handlers.go
```

The nested action or fragment routes do not need standalone pages. Keep
templates used only by one route directly in that route directory, and choose
directory names for clear generated helpers such as
`urls.Users.Prepare.Path()` and `urls.Users.Save.Path()`.

## HTMX Stays Visible

goldr does not hide HTMX behind framework components. Templates keep ordinary
HTML and `hx-*` attributes:

```templ
package users

import "example.com/hello-goldr/app/urls"

templ DirectoryView() {
	<button
		hx-get={ urls.Users.Table.Path() }
		hx-target="#users-table-slot"
		hx-swap="innerHTML"
	>
		Load users
	</button>
	<div id="users-table-slot">
		@renderFragTable(FragTableView())
	</div>
}
```

Use a page-owned slot as the HTMX replacement boundary when refreshing an
embedded fragment. The slot swaps with `innerHTML`, and the fragment root stays
inside the slot.

Handlers that need HTMX response headers can return the rendered response with
those headers attached:

```go
func PostCreate(r *http.Request) goldr.RouteResponse {
	return goldr.NewFragment(FragTableView()).
		WithHeader(hx.HeaderRetarget, "#users-table-slot").
		WithHeader(hx.HeaderReswap, "innerHTML").
		WithHeader(hx.HeaderTrigger, "user:created")
}
```

Use `goldr.HTTPAction` only when the action needs direct
`http.ResponseWriter` control, such as streaming, installing
`http.MaxBytesReader`, or calling an API that requires the writer.

For app-owned server-sent event streams, use the small `sse` package for
event-stream headers, comments, event fields, templ-rendered HTML data, and
flushing. The application still owns stream URLs, mux registration,
subscribers, replay policy, and HTMX attributes.

Unnamed SSE messages use htmx 4's native swap behavior. For semantic named
events such as `event: chat-message`, mount the `browser` helper and add
`goldr-sse-event` to the target element:

```go
import (
	"net/http"

	"github.com/mobiletoly/goldr/browser"
)

mux.Handle("/goldr/", http.StripPrefix("/goldr/", browser.Handler()))
```

```html
<script src="/goldr/goldr-sse-event.js" defer></script>
```

The helper file name is also available as `browser.SSEEventHelperPath`.

```html
<div
  hx-sse:connect="/chat/events"
  goldr-sse-event="chat-message"
  hx-swap="beforeend">
</div>
```

## Try the Full Example

From a goldr checkout, run the full-feature example:

```bash
(cd examples/full_feature && go run .)
```

Inspect the route surface:

```bash
(cd examples/full_feature && go tool goldr routes list)
(cd examples/full_feature && go tool goldr routes layouts)
(cd examples/full_feature && go tool goldr assets list)
(cd examples/full_feature && go tool goldr check)
```

Use `routes list` during route refactors to inspect path and helper names
together. The example shows pages, nested layouts, fragments, actions, forms,
generated URL helpers, custom errors, middleware, request parsing, and
fingerprinted static assets in one small app.

For a focused example of mounting one shared Kit route subtree under multiple
filesystem-owned routes, run:

```bash
(cd examples/kit_routes && go run .)
```

The kit-routes example mounts the same report subtree under `/admin/reports`
and `/user/reports` with route-local `KitRouteMount` constructors, plus an
admin-only child route that stays under the live admin owner.

For a focused realtime example using server-sent events, run:

```bash
(cd examples/chat && go run .)
```

The chat example shows ordinary actions for input, app-owned in-memory
persistence, and an app-owned SSE stream that uses `sse` and the browser helper
to push named rendered HTML events to HTMX.

## Documentation

- [User documentation](docs/user/README.md)
- [Getting Started](docs/user/getting-started.md)
- [Routes](docs/user/routes.md)
- [Mounted Kit Route Subtrees](docs/user/mounted-routes.md)
- [CLI](docs/user/cli.md)
- [Live Reload](docs/user/live-reload.md)
- [Template Inspection](docs/user/template-inspection.md)
- [Assets](docs/user/assets.md)
- [Coding Agents](docs/user/coding-agents.md)
- [Installable Goldr App Skill](docs/skills/goldr/SKILL.md)
- [HTMX](docs/user/htmx.md)
- [SSE](docs/user/sse.md)
- [Composition](docs/user/composition.md)

## License

goldr is licensed under the [Apache License 2.0](LICENSE.txt).
