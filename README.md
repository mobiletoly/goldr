# goldr (Go Layout-Driven Router)

goldr is a server-first Go framework for building web/HTMX applications.

It gives Go projects a predictable filesystem route tree, page layouts,
fragments, action handlers, generated route wiring, and generated URL helpers.
The application still owns its `net/http` server, middleware, static assets,
auth, data access, and deployment.

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
    frag_table.go    -> GET /users/frag_table
    frag_table.templ -> fragment HTML
    actions.go       -> POST /users/create via PostCreate
```

Pages define endpoints, layouts wrap pages in their route directory and below,
fragments are standalone HTMX partials, and actions are ordinary Go handlers
colocated with the route they mutate.

## Why goldr

goldr is for developers who want modern web application structure without
turning the browser into the center of the system.

With goldr:

- routes are visible in the filesystem
- pages and layouts are rendered on the server
- HTMX attributes stay visible in templates
- forms and mutations use ordinary Go handlers
- generated URL helpers remove hard-coded paths
- generated route wiring stays inspectable

goldr intentionally avoids SPA routing, hydration, virtual DOM, framework-owned
client state, and hidden runtime registration.

## Install

Install the CLI:

```bash
go install github.com/mobiletoly/goldr/cmd/goldr@latest
```

goldr applications use Go and templ. The current module targets Go 1.26.

## Quick Start

Create a new module:

```bash
mkdir hello-goldr
cd hello-goldr
go mod init example.com/hello-goldr
go get github.com/mobiletoly/goldr github.com/a-h/templ@v0.3.1020
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
			<script src="https://unpkg.com/htmx.org@2.0.4" defer></script>
		</head>
		<body>
			<main>
				@child
			</main>
		</body>
	</html>
}
```

Generate templ output, generate goldr route wiring, validate, and run:

```bash
go run github.com/a-h/templ/cmd/templ@v0.3.1020 generate
goldr generate
goldr check
go run .
```

Open:

```text
http://127.0.0.1:8080
```

After route or template edits, the normal loop is:

```bash
go run github.com/a-h/templ/cmd/templ@v0.3.1020 generate
goldr generate
goldr check
go run .
```

## Optional Scaffold

`goldr init` can create the minimal route skeleton for an existing Go module:

```bash
goldr init
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

Handlers that need HTMX response headers can use the small `hx` package:

```go
func PostCreate(w http.ResponseWriter, r *http.Request) {
	hx.Retarget(w, "#users-table")
	hx.Reswap(w, "outerHTML")
	hx.Trigger(w, "user:created")

	_ = UsersTable().Render(r.Context(), w)
}
```

## Try the Full Example

From a goldr checkout, run the full-feature example:

```bash
go run ./examples/full_feature
```

Inspect the route surface:

```bash
go run ./cmd/goldr routes list --root examples/full_feature
go run ./cmd/goldr routes layouts --root examples/full_feature
go run ./cmd/goldr check --root examples/full_feature
```

The example shows pages, nested layouts, fragments, actions, forms, generated
URL helpers, custom errors, middleware, and static assets in one small app.

## Documentation

- [User documentation](docs/user/README.md)
- [Getting Started](docs/user/getting-started.md)
- [Routes](docs/user/routes.md)
- [CLI](docs/user/cli.md)
- [HTMX](docs/user/htmx.md)
- [Forms](docs/user/forms.md)
- [Composition](docs/user/composition.md)

Contributors should also read [AGENTS.md](AGENTS.md) and the
[code review pattern catalog](docs/arch/code-review.md).

## License

goldr is licensed under the [Apache License 2.0](LICENSE.txt).
