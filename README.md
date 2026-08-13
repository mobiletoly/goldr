# Goldr (Go Layout-Driven Router)

[![CI](https://github.com/mobiletoly/goldr/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/mobiletoly/goldr/actions/workflows/ci.yml)
[![Latest Release](https://img.shields.io/github/v/release/mobiletoly/goldr?sort=semver)](https://github.com/mobiletoly/goldr/releases/latest)
[![License](https://img.shields.io/github/license/mobiletoly/goldr?logo=apache&label=License)](LICENSE)

**If Goldr saves you time, please consider starring ⭐ the repository - it helps
more developers find it.**

Goldr is a server-first, HTML-first, HTMX-native Go framework for web
applications that stay easy to inspect and change as they grow.

The filesystem is the route map, `.templ` files own HTML, HTMX stays visible in
markup, and handlers remain ordinary Go. Goldr generates the repetitive route
dispatch and URL helpers around that source. Your application still owns its
`net/http` server, middleware, auth, sessions, validation, data access, static
handlers, asset tools, and deployment.

Goldr is v0. APIs and conventions may change before v1.

## Why Goldr

Go and HTMX make a small application easy to start. As the application grows,
teams rebuild the same support layer: filesystem routes, nested
layouts, route-safe URLs, generated-output checks, asset fingerprints, live
reload, and inspection commands.

Goldr provides that layer without moving the application out of Go. A route
directory owns the page, layout state, HTMX fragments, mutation actions, and
templates for one local workflow. You can inspect the application surface from
the directory tree instead of chasing runtime registration, copied path
strings, or hidden client state.

## Try A Complete App First

From a Goldr checkout, run the full-feature example:

```bash
(cd examples/full_feature && go run .)
```

Then inspect its route surface and generated assets:

```bash
(cd examples/full_feature && go tool goldr routes list)
(cd examples/full_feature && go tool goldr routes layouts)
(cd examples/full_feature && go tool goldr routes refs)
(cd examples/full_feature && go tool goldr assets list)
(cd examples/full_feature && go tool goldr check)
```

The example includes pages, nested layouts, HTMX fragments, POST actions,
forms, generated URL helpers, custom errors, middleware, request parsing,
CSRF, route-rendered error pages, and fingerprinted static assets.

## Quick Start

Use Go 1.26 or newer. Goldr applications use
[templ](https://github.com/a-h/templ) for HTML rendering and keep both CLI
tools pinned in the application module.

Create a module and install Goldr, templ, and their app-local tools:

```bash
mkdir hello-goldr
cd hello-goldr
go mod init example.com/hello-goldr

GOLDR_VERSION=v0.1.3
TEMPL_VERSION=v0.3.1020

go get github.com/mobiletoly/goldr@${GOLDR_VERSION} github.com/a-h/templ@${TEMPL_VERSION}
go get -tool github.com/mobiletoly/goldr/cmd/goldr@${GOLDR_VERSION}
go get -tool github.com/a-h/templ/cmd/templ@${TEMPL_VERSION}
go tool -n goldr
go tool -n templ
```

Use the same Goldr version for the runtime library and the `cmd/goldr` tool.
Running tools through `go tool` keeps their versions with the application.

Create the starter route tree:

```bash
go tool goldr init
```

The command creates:

```text
app/
  routes/
    route.go              app-owned route declaration and page handler
    page.templ            app-owned page HTML
    layout.go             app-owned layout logic
    layout.templ          app-owned document HTML
    goldr_gen.go          generated route dispatch
  urls/
    goldr_gen.go          generated URL helpers
  internal/goldrinspect/
    goldr_gen.go          generated inspection support
```

Edit the four app-owned files. Regenerate the three Goldr-owned files instead
of editing them.

The root `route.go` declares one page:

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
			Title: "Hello Goldr",
		},
	)
}
```

`Route` is generation input. Goldr reads this package-level static declaration
without running the application. Because the declaration lives at the root of
`app/routes`, it owns `/`. `Page: page` selects the ordinary Go function that
handles `GET` and `HEAD` requests for that path.

The handler returns a page response containing a templ component and metadata.
`page.templ` owns the component's HTML:

```templ
package routes

templ PageView() {
	<section>
		<h1>Hello Goldr</h1>
		<p>Edit app/routes/page.templ to start building.</p>
	</section>
}
```

The root layout wraps this page and every descendant page. `ctx.Child` is the
matched page or inner layout component, and `ctx.Metadata` comes from the
matched page response:

```go
package routes

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

const defaultTitle = "Hello Goldr"

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
			<script src="https://cdn.jsdelivr.net/npm/htmx.org@4.0.0-beta4" integrity="sha384-aWZK1NtOs/aWb/+YZdTM8q2JkWEshlMc9mgZ189numT9bwFhyAyYEoO4nO/2dTXt" crossorigin="anonymous" defer></script>
		</head>
		<body>
			<main>
				@child
			</main>
		</body>
	</html>
}
```

Add `main.go`. The application owns the server and mounts Goldr's generated
handler like any other `http.Handler`:

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

The request path is direct:

```text
GET /
  -> generated routes.Handler()
  -> page()
  -> goldr.NewPage(PageView(), metadata)
  -> Layout(..., ctx.Child)
  -> layout.templ renders @child
  -> HTML response
```

Generate templ output and Goldr route wiring, tidy the module, inspect the
route, validate generated files, and run the application:

```bash
go tool goldr generate
go mod tidy
go tool goldr routes list
go tool goldr check
go run .
```

The route list connects the URL to its source and generated helper:

```text
KIND    METHOD    PATH  PARAMS  SOURCE     OWNER  DECL   NAME  TITLE  LABELS  NAV  TRAIL_KEYS  HELPER
layout  -         /     -       layout.go  -      -      -     -      -       -    -           -
page    GET,HEAD  /     -       route.go   -      local  -     -      -       -    -           urls.Root.Path()
```

Open `http://127.0.0.1:8080`.

After route or template edits, run `go tool goldr generate` before
`go tool goldr check`. For live reload, use:

```bash
go tool goldr dev
```

Read [Getting Started](docs/user/getting-started.md) to build a two-page app by
hand and inspect its dynamic route. Read
[Live Reload](docs/user/live-reload.md) for asset and Tailwind workflows.

## How Goldr Apps Are Shaped

The filesystem is the route map:

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
    frag_table.templ -> fragment HTML
    by_id/
      route.go       -> GET /users/{id}
      page.templ     -> user detail HTML
```

Each route directory declares its page, fragments, and actions in one static
`Route` value:

```go
var Route = goldr.RouteDef{
	Page: page,
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/table", table),
	},
	Actions: goldr.Actions{
		goldr.Action(http.MethodPost, "/create", postCreate),
	},
}
```

The directory and declaration generate route-shaped URL helpers:

```go
urls.Users.Path()
urls.Users.Table.Path()
urls.Users.Create.Path()
urls.Users.ByID.Bind(id).Path()
```

HTMX remains visible at the call site. Templates use the generated fragment
path in ordinary `hx-*` attributes:

```templ
package users

import "example.com/hello-goldr/app/urls"

templ UsersView() {
	<button
		hx-get={ urls.Users.Table.Path() }
		hx-target="#users-table"
		hx-swap="innerHTML"
	>
		Refresh users
	</button>
	<div id="users-table"></div>
}
```

Goldr uses Go-safe filesystem names. `by_id/` maps to `{id}`, while a static
directory such as `build_info/` maps to `/build-info`. Layouts wrap pages in
their directory and below. Fragments render standalone partials, and actions
return explicit route responses.

Goldr writes ordinary Go dispatch to `app/routes/goldr_gen.go` and URL helpers
to `app/urls/goldr_gen.go`; it does not assemble a route registry at runtime.
Read [Routes](docs/user/routes.md) for dynamic segments, route-local workflows,
mounted routes, and the complete route contract.

## What Goldr Gives You

### Build

- A route tree you can read: `app/routes` is the URL map, with Go-safe names and
  colocated source.
- Route-local pages, fragments, and actions declared in `route.go`.
- Nested layouts that compose without a second routing or template tree.
- Generated paths for links, forms, redirects, HTMX attributes, and response
  headers.
- Visible browser behavior through normal `hx-*` attributes.

### Iterate

- `goldr dev` runs templ generation, Goldr route generation, asset
  fingerprinting, app restart, and browser reload.
- `goldr generate` refreshes route wiring, URL helpers, templ output, and
  fingerprinted assets.
- `goldr check` verifies generated routes, templ output, and managed assets
  without writing files.

### Ship

- Put final browser-ready files in `assets/build`; Goldr writes fingerprinted
  files to `assets/dist`, generates paths such as `assets.Path("app.css")`, and
  exposes an embedded `assets.FS()` for your static handler.
- Use `goldr check`, `go tool goldr assets check`, and
  `go tool goldr assets list` to inspect packaged resource state.
- Keep CSS compilation, JavaScript bundling, static serving, cache policy, and
  deployment in application-owned tools and code.

### Debug

- `routes list`, `routes explain`, and `routes layouts` expose route paths,
  handlers, and layout stacks.
- `routes refs` inventories direct HTMX references in `.templ` files.
- The visual inspector outlines the layouts, pages, fragments, and labeled
  components that produced each page region.

## More Examples

For a bounded rich-client escape hatch, see `examples/react_island` and
`examples/svelte_island`. Goldr owns pages and navigation while React or Svelte
owns one explicit editor subtree. Read
[Client Islands](docs/user/client-islands.md) for the lifecycle contract.

Run the Kit route example to see one shared implementation subtree mounted
under `/admin/reports` and `/user/reports`:

```bash
(cd examples/kit_routes && go run .)
```

Run the chat example for app-owned realtime behavior with server-sent events:

```bash
(cd examples/chat && go run .)
```

## Documentation

- [User Documentation](docs/user/README.md) - the complete documentation index.
- [Getting Started](docs/user/getting-started.md) - build a two-page app by
  hand and inspect its dynamic route.
- [Concepts](docs/user/concepts.md) - pages, layouts, fragments, actions,
  generated handlers, and URL helpers.
- [CLI](docs/user/cli.md) - app-local `go tool goldr` commands.
- [Routes](docs/user/routes.md) - filesystem conventions and runtime behavior.
- [Mounted Kit Route Subtrees](docs/user/mounted-routes.md) - reusable non-live
  route surfaces mounted by filesystem-owned routes.
- [Navigation Trails](docs/user/navigation.md) - contextual trails,
  breadcrumb-style rendering, and app-level Back links.
- [Client Islands](docs/user/client-islands.md) - bounded React or Svelte
  components inside Goldr pages.
- [HTMX](docs/user/htmx.md) - visible `hx-*` attributes and response headers.
- [Error Handling](docs/user/error-handling.md) - route errors, custom hooks,
  full-page errors, and HTMX error fragments.
- [Assets](docs/user/assets.md) - fingerprinted files, cache headers, and
  app-owned asset tooling.
- [SSE](docs/user/sse.md) - app-owned streams, event IDs, and named SSE swaps.
- [CSRF](docs/user/csrf.md) - signed-cookie tokens for unsafe requests.
- [Composition](docs/user/composition.md) - mux, middleware, static assets, and
  app-owned server behavior.
- [Application Dependencies](docs/user/dependencies.md) - app-owned typed
  dependencies for generated route packages.
- [Live Reload](docs/user/live-reload.md) - `goldr dev`, browser reload, assets,
  and Tailwind workflows.
- [Template Inspection](docs/user/template-inspection.md) - render-unit comments
  and visible browser overlays.
- [Coding Agents](docs/user/coding-agents.md) - guidance for agents working on
  Goldr applications.

### Agent Tooling

- [Installable Goldr App Skill](docs/skills/goldr/SKILL.md)

## License

Goldr is licensed under the [Apache License 2.0](LICENSE).
