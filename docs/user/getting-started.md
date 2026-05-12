# Getting Started

This guide builds the smallest useful goldr app manually. Manual setup comes
first so the project shape is visible. `go tool goldr init` is available as a
shortcut after the manual path.

## Install

Create a module and add goldr, templ, and app-local CLI tools:

```bash
mkdir hello-goldr
cd hello-goldr
go mod init example.com/hello-goldr
go get github.com/mobiletoly/goldr github.com/a-h/templ@v0.3.1020
go get -tool github.com/mobiletoly/goldr/cmd/goldr@latest
go get -tool github.com/a-h/templ/cmd/templ@v0.3.1020
```

Run goldr and templ with `go tool goldr` and `go tool templ`. This keeps the
tool versions pinned in the application module.

## Add The HTTP Server

Create `main.go`:

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

The application owns the server and mux. goldr generates `routes.Handler()`
from files under `app/routes`.

## Add The First Page

Create the route directory:

```bash
mkdir -p app/routes
```

Create `app/routes/page.go`:

```go
package routes

import (
	"net/http"
	"time"

	"github.com/mobiletoly/goldr"
)

func Page(_ *http.Request) goldr.Page {
	return goldr.Page{
		Component: PageView(time.Now()),
		Metadata: goldr.PageMetadata{
			Title: "Hello goldr",
		},
	}
}
```

Create `app/routes/page.templ`:

```templ
package routes

import "time"

templ PageView(now time.Time) {
	<section>
		<h1>Hello goldr</h1>
		<p>Edit app/routes/page.templ to start building.</p>
		<p>Rendered at { now.Format(time.RFC3339) }</p>
	</section>
}
```

`page.go` handles the request-facing page function. `page.templ` renders the
HTML. Pass ordinary Go values from `Page` into the templ component when the
view needs request data, loaded records, validation state, or computed values.

## Add The Root Layout

Create `app/routes/layout.go`:

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

Create `app/routes/layout.templ`:

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

The root layout wraps the root page and pages below it. Fragments and actions
are not layout-wrapped.

## Generate And Run

Generate templ output, generate goldr route wiring, validate, and run:

```bash
go tool templ generate
go tool goldr generate
go tool goldr check
go run .
```

Open:

```text
http://127.0.0.1:8080
```

After route or template edits, use the same loop:

```bash
go tool templ generate
go tool goldr generate
go tool goldr check
go run .
```

## Optional Scaffold

`goldr init` creates the minimal route skeleton for an existing Go module:

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

It does not create `go.mod`, edit `go.mod`, write `main.go`, run templ, or
start a server.

Use `--root` when running from outside the application root:

```bash
go tool goldr init --root ./hello-goldr
```

`--root` points to the application root. goldr still uses
`<root>/app/routes` and `<root>/app/urls`.

## Coding Agents

If you use a coding agent in a goldr app, add goldr-specific instructions to
the app's `AGENTS.md`. See [Coding Agents](coding-agents.md) for a copyable
block that explains the route tree, generated files, HTMX conventions, assets,
and validation commands.
