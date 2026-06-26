# goldr (Go Layout-Driven Router)

goldr is a server-first, HTML-first, HTMX-native Go framework for building web
applications that stay easy to see, run, and change as they grow.

Goldr keeps the productive parts of modern web apps without moving the center
of gravity out of Go: the filesystem is the route map, `.templ` files own
HTML, HTMX stays visible in markup, handlers stay ordinary Go, and generated
wiring handles the repeatable route work around them.

You get a full Go+HTMX application workflow: route-local pages, nested layouts,
HTMX fragments, mutation actions, generated URL helpers, live reload,
fingerprinted and embedded static resources, route inspection commands, and a
browser visual inspector that can outline the rendered page. The app still
owns its `net/http` server, middleware, static handlers, auth, sessions,
request parsing, validation, data access, asset tools, and deployment.

goldr is v0. APIs and conventions may change before v1.

## What goldr gives you

Goldr is useful when a Go app needs real web-app structure but should still
feel like a Go app.

### Build

- A route tree you can read: `app/routes` is the URL map, using Go-safe
  directory names and colocated source.
- Route-local workflows: one `route.go` declares the page, HTMX fragments, and
  POST actions for that part of the app.
- Nested layouts: route directories compose shells naturally without a second
  routing or template tree.
- Generated URL helpers: templates, redirects, HTMX attributes, and response
  headers use generated paths instead of copied strings.
- Visible HTMX: browser behavior stays in normal `hx-*` attributes instead of
  disappearing behind proprietary components or client state.

### Iterate

- `goldr dev` keeps the local loop moving: templ generation, Goldr route
  generation, asset fingerprinting, app restart, and browser reload.
- `goldr generate` refreshes route wiring, URL helpers, templ output, and
  fingerprinted assets from one command.
- `goldr check` verifies generated routes, templ output, and managed assets
  without writing files.
- Inspection commands show what Goldr generated: paths, helpers, layout stacks,
  direct HTMX references, and asset manifests.

### Ship

- Put final browser-ready files in `assets/build`; Goldr writes fingerprinted
  files to `assets/dist`, generates logical paths such as
  `assets.Path("app.css")`, and exposes an embedded `assets.FS()` for the
  app-owned static handler.
- `goldr check`, `go tool goldr assets check`, and
  `go tool goldr assets list` make the packaged resource state visible before
  you ship.
- Goldr does not compile CSS, bundle JavaScript, register handlers, deploy the
  app, or choose your CDN policy.

### Debug

- `routes list`, `routes explain`, and `routes layouts` make the route tree
  inspectable from the command line.
- `routes refs` inventories direct HTMX references in `.templ` files.
- The visual inspector can draw browser overlays for the layouts, pages,
  fragments, and labeled components that produced each page region.

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

A route directory is the unit of local web behavior. Its `route.go` declares
the page, HTMX fragments, and actions for that part of the app:

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

goldr turns the filesystem and route declarations into generated dispatch and
route-shaped URL helpers. The source route remains ordinary Go, while
templates can link to generated helpers instead of hard-coded paths:

```go
urls.Users.Path()
urls.Users.Table.Path()
urls.Users.Create.Path()
urls.Users.ByID.Bind(id).Path()
```

HTMX stays visible at the call site. A template can use the generated fragment
path directly in the `hx-get` attribute:

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

Layouts wrap pages in their route directory and below. Fragments are
standalone HTMX partials, and actions are ordinary Go handlers colocated with
the route they mutate. The result is a Go-native structure where page loading,
partial refreshes, form posts, redirects, validation failures, and layout state
all live close to the workflow they support.

Static directory underscores become hyphens in browser URLs, so Go-safe source
names such as `build_info/` can serve stable paths such as `/build-info`.

## Why goldr

Most Go+HTMX applications start simple, then quietly collect the same support
layer: route dispatch, layout stacking, route-safe URLs, stale-generated-output
checks, asset fingerprints, and debugging commands. Without a shared framework,
each app tends to invent a private routing convention and copy path strings
through handlers, templates, redirects, and tests.

Goldr standardizes that layer while keeping the application explicit. One
route directory can contain the whole local workflow:

- the page handler that loads data
- the `.templ` file that renders HTML
- the layout state needed by parent shells
- the HTMX fragments used by that page
- the POST actions that mutate that route's data
- the generated URL helpers used by templates and redirects
- the metadata used for titles, navigation trails, and app-level Back links

That means a developer can look at the filesystem and understand the
application surface. They do not need to chase runtime route registration,
stringly typed paths, hidden client-side state, or a custom folder convention
that only exists in project lore.

The tradeoff is deliberate. Goldr gives Go+HTMX apps a route tree, generated
wiring, asset fingerprints, local dev loop, and inspection tools. It does not
own your server, data layer, asset compiler, JavaScript architecture, client
state, hydration, deployment, or runtime registration system.

## Install

goldr applications use Go and [templ](https://github.com/a-h/templ). During v0,
templ is goldr's HTML render contract: page functions return
`goldr.PageRouteResponse`, fragment functions return
`goldr.FragmentRouteResponse`, actions return `goldr.RouteResponse`, and
`.templ` files own HTML rendering. goldr owns the filesystem route model,
generated route wiring, URL helpers, route validation, and inspection metadata
around that workflow.

Use Go 1.26 or newer.

Add goldr, templ, and app-local CLI tools to your module:

```bash
GOLDR_VERSION=v0.1.3
go get github.com/mobiletoly/goldr@${GOLDR_VERSION} github.com/a-h/templ@v0.3.1020
go get -tool github.com/mobiletoly/goldr/cmd/goldr@${GOLDR_VERSION}
go get -tool github.com/a-h/templ/cmd/templ@v0.3.1020
go tool -n goldr
```

Use the same Goldr version for the runtime library and the `cmd/goldr` tool.
`go tool -n goldr` confirms that the app-local CLI tool is in the module
graph. Then run goldr and templ with `go tool goldr` and `go tool templ`. This
keeps the commands versioned with the application. If you prefer a global
convenience binary, `go install github.com/mobiletoly/goldr/cmd/goldr@v0.1.3`
also works.

## Try A Complete App First

From a goldr checkout, run the full-feature example:

```bash
(cd examples/full_feature && go run .)
```

Then inspect the route surface and generated assets:

```bash
(cd examples/full_feature && go tool goldr routes list)
(cd examples/full_feature && go tool goldr routes layouts)
(cd examples/full_feature && go tool goldr routes refs)
(cd examples/full_feature && go tool goldr assets list)
(cd examples/full_feature && go tool goldr check)
```

The example shows pages, nested layouts, HTMX fragments, POST actions, forms,
generated URL helpers, custom errors, middleware, request parsing, CSRF,
route-rendered error pages, and fingerprinted static assets in one small app.

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
- fragment declarations in `route.go` define independently renderable HTMX
  fragments, including optional index fragments at the route path
- action declarations in `route.go` define colocated mutation handlers
- `by_id/` maps to a dynamic `{id}` route segment
- `build_info/` maps to a static `/build-info` browser segment

goldr generates route dispatch in `app/routes/goldr_gen.go` and URL helpers in
`app/urls/goldr_gen.go`.

This generated code is meant to be inspected. It is the executable route truth
produced from the filesystem and `route.go` declarations, not a hidden
registry built at runtime.

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

## Quick Start

Create a new module:

```bash
mkdir hello-goldr
cd hello-goldr
go mod init example.com/hello-goldr
go get github.com/mobiletoly/goldr@$0.1.3 github.com/a-h/templ@v0.3.1020
go get -tool github.com/mobiletoly/goldr/cmd/goldr@$0.1.3
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

const appTitle = "Hello goldr"

func Layout(_ *http.Request, ctx goldr.LayoutContext) templ.Component {
	return LayoutView(ctx.Metadata, ctx.Child)
}

// Each matched page can set metadata.Title; fall back to the app title when it does not.
func pageTitle(metadata goldr.PageMetadata) string {
	// Propagated from children pages, if available
	if metadata.Title != "" {
		return metadata.Title
	}
	return appTitle
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

## HTMX Stays Visible

goldr does not hide HTMX behind framework components. Templates keep ordinary
HTML and `hx-*` attributes. The framework helps with routing, response shape,
headers, and helpers, but the browser behavior remains visible at the call
site:

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

For deeper HTMX response patterns, read [HTMX](docs/user/htmx.md).

## More Examples

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

The chat example shows where realtime/SSE support fits when an app needs it,
without making SSE part of the first-read path.

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
