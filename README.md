<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/assets/goldr-logo-dark.svg">
    <img src="docs/assets/goldr-logo-light.svg" alt="Goldr gold crystal logo" width="112">
  </picture>
</p>

# Goldr

[![CI](https://github.com/mobiletoly/goldr/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/mobiletoly/goldr/actions/workflows/ci.yml)
[![Latest Release](https://img.shields.io/github/v/release/mobiletoly/goldr?sort=semver)](https://github.com/mobiletoly/goldr/releases/latest)
[![License](https://img.shields.io/github/license/mobiletoly/goldr?logo=apache&label=License)](LICENSE)

**Build customer portals, admin tools, and interactive websites with Go, HTML, and HTMX.**

Goldr (Go Layout-Driven Router) is a server-first Go web framework. Keep page
handlers, templates, and actions together, compose layouts through directories,
and let Goldr generate your routing and URL helpers. Write HTML with
[templ](https://github.com/a-h/templ) and keep browser interactions visible in
ordinary HTMX attributes.

Your application uses ordinary Go functions and a standard `net/http` server.

[Capabilities](#what-goldr-provides) | [Try it](#try-it) | [Getting Started](docs/user/getting-started.md) |
[Documentation](docs/user/README.md) | [Examples](examples)

Goldr is v0. APIs and conventions may change before v1.

## Why Goldr

A portal needs more than a collection of request handlers. You need shared
page shells, forms that update part of a screen, links between related records,
and consistent behavior across customer and admin areas. As those areas grow,
you also need to find the code behind a URL and understand which layouts and
middleware apply to it.

Goldr brings those pieces into one Go workflow: filesystem routes, nested
layouts, explicit page and fragment responses, generated URL helpers, and
tools for developing and inspecting the application. You write the business
logic and HTML; Goldr generates the dispatch and composition code around them.
You can keep your existing Go libraries and mount the generated handler in a
standard `net/http` server.

## See Your Application In The Directory Tree

Find a page's request logic and HTML in the same directory:

```text
app/routes/
  layout.go            Shared layout logic
  layout.templ         Shared HTML shell
  route.go             /
  page.templ
  users/
    layout.go          Layout for /users and its pages
    layout.templ
    route.go           /users, /users/table
    page.templ
    frag_table.templ
    by_id/
      route.go         /users/{id}
      page.templ

content/
  about/
    page.json          Page title and description
    body.md            /about
```

The route directory determines the URL. Declare pages, HTMX fragments, and
mutation actions in `route.go`; keep their templates beside it. Layouts wrap
pages in their directory and below, so the user detail page shares both the
site shell and the users layout. `by_id/` declares the `{id}` URL parameter
using a Go-safe directory name.

Before your application runs, `go tool goldr generate` scans `app/routes` and
reads its route and layout declarations without executing your application.
It generates ordinary Go dispatch and URL helpers: you can inspect the wiring,
and the Go compiler checks calls to your handlers. The running server uses that
code, with no runtime route discovery or dynamic handler loading.

## What Goldr Provides

### Shared Shells And Portal Sections

Use nested layouts for the site shell, an admin sidebar, and a section's tabs.
Pages supply metadata and typed layout data, so a child page can select an
active tab or provide a toolbar without duplicating the surrounding HTML.
Keep each workflow's handlers, templates, and actions in its route directory.
See [Routes And Layouts](docs/user/routes.md).

For sections you need in several places, mount a shared Kit route subtree
under different live URLs. A reports implementation can serve both
`/admin/reports` and `/user/reports`, with each owner supplying its dependencies
and selecting which child routes to expose. See
[Mounted Kit Route Subtrees](docs/user/mounted-routes.md).

### Forms And Partial Page Updates

Declare pages, fragments, and mutation actions together. Use HTMX to filter a
table, load a dialog, or submit a form; return the HTML needed for that update
from Go. Goldr provides explicit page, fragment, redirect, and text responses,
plus HTMX response-header helpers. Your handlers choose validation rules and
render field errors beside the inputs.

Custom error hooks let you provide full error pages or HTMX error fragments
that fit the surrounding workflow. See [HTMX](docs/user/htmx.md) and
[Error Handling](docs/user/error-handling.md).

### Links And Contextual Navigation

Generated URL helpers cover pages, fragments, and actions, including dynamic
parameters. Use the same helpers in links, forms, redirects, and `hx-*`
attributes, and let the Go compiler catch references to removed route helpers.

Goldr also prepares navigation data for breadcrumbs and contextual Back links.
A customer record reached through a regional report can retain that workflow's
trail. You supply labels from application data and render the navigation with
your own HTML. See [Navigation Trails](docs/user/navigation.md).

### Middleware And Request Protection

Place ordinary Go middleware at the root or within a route section to apply
authentication checks, attach a principal, or supply request context to its
handlers. Use mux-level middleware for policy shared with other HTTP handlers.
Goldr's optional CSRF package provides signed-cookie tokens, form and metadata
helpers, and validation for unsafe requests.

You choose authentication, sessions, permissions, and data access to suit your
application. See [Composition](docs/user/composition.md),
[CSRF](docs/user/csrf.md), and [Application Dependencies](docs/user/dependencies.md).

### Markdown And HTML Content Pages

Add help articles, policies, and product documentation through the optional
content package. Each page has metadata and a Markdown or HTML body, and uses
the eligible static route-tree layouts and middleware. Generated application
routes keep priority.

With external content, authors can edit or add pages without a new Go handler,
route generation, or server restart. Markdown supports GFM tables, task lists,
and heading IDs. Content authors must be trusted like application template
authors. See [Content Pages](docs/user/content-pages.md).

### Live Updates And Rich Editors

Use the optional SSE helpers to send rendered HTML from an application-owned
stream, with event IDs and named-event swaps for HTMX. The
[chat example](examples/chat) demonstrates the flow; see
[Server-Sent Events](docs/user/sse.md) for the supported helpers.

For an editor that needs its own client state, embed a bounded React or Svelte
island inside a Goldr page. The examples show mounting and cleanup around HTMX
navigation while Goldr owns the surrounding routes and layouts. Your
application owns the client integration and build. See
[Client Islands](docs/user/client-islands.md).

## A Page And An HTMX Update

In a module named `example.com/hello-goldr`, a users page and its table fragment
share `app/routes/users/route.go`:

```go
package users

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

var Route = goldr.RouteDef{
	Page: page,
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/table", table),
	},
}

func page(_ *http.Request) goldr.PageRouteResponse {
	return goldr.NewPage(PageView(), goldr.PageMetadata{Title: "Users"})
}

func table(_ *http.Request) goldr.FragmentRouteResponse {
	return goldr.NewFragment(TableView([]string{"Ada", "Linus"}))
}
```

The `Route` declaration is input to the generator. Here it defines
`GET /users` and `GET /users/table`.
The generated handler mounts on your application's mux with
`mux.Handle("/", routes.Handler())`.

The page template uses the generated `app/urls` helper in an ordinary HTMX
button:

```templ
package users

import "example.com/hello-goldr/app/urls"

templ PageView() {
	<h1>Users</h1>
	<button
		hx-get={ urls.Users.Table.Path() }
		hx-target="#users-table"
		hx-swap="innerHTML"
	>
		Load users
	</button>
	<div id="users-table"></div>
}
```

With HTMX loaded by the shared layout, clicking **Load users** calls the Go
handler and replaces the contents of `#users-table` with its HTML response.
`TableView` is a colocated templ component that renders the supplied names as
table rows. Goldr renders the fragment without page layouts. The
[Getting Started article](docs/user/getting-started.md) walks through a complete
application, including the templates and server setup.

## Try It

With Git and Go 1.26.0 or newer installed, run the complete example. Use the
latest patch release in your supported Go line; the example pins Goldr and
templ as app-local tools.

```bash
git clone https://github.com/mobiletoly/goldr.git
cd goldr/examples/full_feature
go tool goldr dev
```

Open [http://127.0.0.1:7331/users](http://127.0.0.1:7331/users). Add a contact
or filter the table to see HTMX updates. Change the heading in
`app/routes/users/page.templ` and save; the browser reloads with your updated
heading. Press Ctrl+C to stop.

The example includes nested layouts, forms, validation, CSRF, and fingerprinted
assets. Browse its [source](examples/full_feature) to see how they fit together.
For Markdown and HTML pages, see the separate
[content pages example](examples/content_pages).

To create your own application, follow [Getting Started](docs/user/getting-started.md).
It explains the scaffolded code, gets your first page running, then adds a
dynamic route and an HTMX interaction.

## Tools For Everyday Development

- **Edit and see the result.** `go tool goldr dev` handles generation, app
  restart, and browser reload. For external content, add `--reload-path content`
  to refresh the browser when Markdown or HTML changes. See
  [Live Reload](docs/user/live-reload.md).
- **Trace a URL to its code.** `go tool goldr routes list` shows your endpoints;
  `routes explain` and `routes layouts` show their handlers and layout stacks.
  `routes refs` inventories direct HTMX references in templates. See the
  [CLI Reference](docs/user/cli.md).
- **Find the template behind a page region.** The optional
  [visual inspector](docs/user/template-inspection.md) outlines layouts,
  pages, and fragments in the browser and shows their source paths.
- **Check generated output.** `go tool goldr check` verifies routes, templ
  output, and managed assets without rewriting files. Use it locally and in CI.
- **Package browser assets.** Goldr fingerprints your built CSS and JavaScript
  and generates asset URLs and an embedded filesystem for your static handler.
  Keep your preferred CSS and JavaScript build tools. See [Assets](docs/user/assets.md).
- **Give coding agents project context.** The installable
  [Goldr App skill](docs/user/coding-agents.md) covers route conventions,
  framework APIs, and validation workflows.

You choose your database, authentication, middleware, and deployment. Mount
Goldr's generated handler alongside your other `net/http` handlers and keep
using the Go libraries you need.

## Explore Further

- [All Documentation](docs/user/README.md) - browse the complete guide and reference.
- [Coding Agents](docs/user/coding-agents.md) - install the Goldr App skill
  and guide your coding agent.
- [Getting Started](docs/user/getting-started.md) - build your first application.
- [Concepts](docs/user/concepts.md) - understand pages, layouts, fragments,
  and actions.
- [Routes](docs/user/routes.md) and [HTMX](docs/user/htmx.md) - build route-local
  workflows.
- [Content Pages](docs/user/content-pages.md) - add Markdown and HTML pages.
- [Live Reload](docs/user/live-reload.md) and [Assets](docs/user/assets.md) -
  configure your development and asset tools.
- [Examples](examples) - explore forms, SSE chat, shared route subtrees, and
  React or Svelte islands.

## License

Goldr is licensed under the [Apache License 2.0](LICENSE).

If Goldr helps you build, consider starring the repository so more Go developers
can find it.
