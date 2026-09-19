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

**Build interactive web applications with Go, HTML, and HTMX.**

Goldr (Go Layout-Driven Router) is a server-first Go web framework. Keep page
handlers, templates, and actions together, compose layouts through directories,
and let Goldr generate your routing and URL helpers. Write HTML with
[templ](https://github.com/a-h/templ) and keep browser interactions visible in
ordinary HTMX attributes.

Your application uses ordinary Go functions and a standard `net/http` server.

[Try it](#try-it) | [Getting Started](docs/user/getting-started.md) |
[Documentation](docs/user/README.md) | [Examples](examples)

Goldr is v0. APIs and conventions may change before v1.

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

For About, Privacy, and other informational pages, write Markdown or HTML
under `content/`. With Goldr's optional [content package](docs/user/content-pages.md)
configured, these pages use your application's layouts without a separate Go
handler for each page.

## A Page And An HTMX Update

Here is a small users page in a module named `example.com/hello-goldr`.
In `app/routes/users/route.go`, declare the page and a table fragment:

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

Goldr also generates the `app/urls` package from your declared route tree.
Its helpers give you compiler-checked references to pages, fragments, and
actions instead of copied URL strings. Here, `urls.Users.Table.Path()` returns
the URL for the `/table` fragment declared above.

Import that package in `app/routes/users/page.templ` and use the helper in an
HTMX button:

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

In `app/routes/users/frag_table.templ`, render the table:

```templ
package users

templ TableView(names []string) {
	<table>
		<tbody>
			for _, name := range names {
				<tr><td>{ name }</td></tr>
			}
		</tbody>
	</table>
}
```

With HTMX loaded by the shared layout, clicking **Load users** calls the Go
handler and replaces the contents of `#users-table` with its HTML response.
Goldr renders the fragment without page layouts.

Use the same generated helpers for links, forms, and redirects:

```go
urls.Users.Path()                // /users
urls.Users.Table.Path()          // /users/table
urls.Users.ByID.Bind("42").Path() // /users/42
```

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
  to refresh the browser when Markdown or HTML changes.
- **Trace a URL to its code.** `go tool goldr routes list` shows your endpoints;
  `routes explain` and `routes layouts` show their handlers and layout stacks.
- **Find the template behind a page region.** The optional
  [visual inspector](docs/user/template-inspection.md) outlines layouts,
  pages, and fragments in the browser and shows their source paths.
- **Check generated output.** `go tool goldr check` verifies routes, templ
  output, and managed assets without rewriting files. Use it locally and in CI.
- **Package browser assets.** Goldr fingerprints your built CSS and JavaScript
  and generates asset URLs and an embedded filesystem for your static handler.

You choose your database, authentication, middleware, and deployment. Mount
Goldr's generated handler alongside your other `net/http` handlers and keep
using the Go libraries you need.

## Explore Further

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
- [Coding Agents](docs/user/coding-agents.md) - install the Goldr App skill
  and guide your coding agent.
- [All Documentation](docs/user/README.md) - browse the complete guide and reference.

## License

Goldr is licensed under the [Apache License 2.0](LICENSE).

If Goldr helps you build, consider starring the repository so more Go developers
can find it.
