# Getting Started

Build a small Goldr application from an empty directory. The home page will
show five user links. Each link will open a dynamic page such as `/users/3`
and render `Hello User 3`.

You will create each application-owned file by hand so you can see how the
server, route declarations, templates, layout, and generated URL helpers fit
together. Goldr can scaffold a starter with `go tool goldr init`, but do not
run that command while following this tutorial. The final section explains how
to use it for your next application.

Goldr and templ will still create their generated files. Do not write
`goldr_gen.go` or `*_templ.go` files by hand.

## Before You Start

Install Go 1.26 or newer, then confirm your Go installation:

```bash
go version
```

You will build this route tree:

```text
app/routes/
  layout.go
  layout.templ
  route.go
  page.templ
  users/
    by_id/
      route.go
      page.templ
```

The root directory owns `/`. The `users/by_id` directory owns
`/users/{id}`.

## 1. Create The Project

Create a directory and initialize a Go module:

```bash
mkdir hello-goldr
cd hello-goldr
go mod init example.com/hello-goldr
```

The module path becomes the import prefix for packages in this application.
If you choose another module path, replace `example.com/hello-goldr` in the Go
and templ files below.

## 2. Install Goldr And Its Tools

Keep the Goldr runtime and CLI on the same version. Add Goldr, templ, and both
app-local tools:

```bash
GOLDR_VERSION=v0.1.3
TEMPL_VERSION=v0.3.1020

go get github.com/mobiletoly/goldr@${GOLDR_VERSION} github.com/a-h/templ@${TEMPL_VERSION}
go get -tool github.com/mobiletoly/goldr/cmd/goldr@${GOLDR_VERSION}
go get -tool github.com/a-h/templ/cmd/templ@${TEMPL_VERSION}
```

Confirm that Go can resolve both tools:

```bash
go tool -n goldr
go tool -n templ
```

Each command prints the path to its executable. Running tools through
`go tool` keeps their versions in `go.mod` with the application.

## 3. Create The Route Directories

Create the root route directory and the dynamic user route directory:

```bash
mkdir -p app/routes/users/by_id
```

The `by_` prefix declares a dynamic segment. Goldr turns `by_id` into `{id}`
when it generates the route table.

## 4. Add The HTTP Server

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

Your application owns the HTTP server and mux. Goldr will generate
`routes.Handler()` after you add the route source files.

## 5. Add The Home Page

Create `app/routes/route.go`:

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
		goldr.PageMetadata{Title: "Choose a user"},
	)
}
```

`Route` is static generation input. Because this declaration lives at the root
of `app/routes`, it declares the page for `/`. `Page: page` names the ordinary
Go function that handles `GET` and `HEAD` requests for that path.

Create `app/routes/page.templ`:

```templ
package routes

import (
	"strconv"

	"example.com/hello-goldr/app/urls"
)

templ PageView() {
	<section>
		<h1>Choose a user</h1>
		<ul>
			for id := 1; id <= 5; id++ {
				<li>
					<a href={ urls.Users.ByID.Bind(strconv.Itoa(id)).Path() }>
						Open User { strconv.Itoa(id) }
					</a>
				</li>
			}
		</ul>
	</section>
}
```

templ accepts normal Go control flow. The loop renders five links. Goldr will
generate `urls.Users.ByID`, and `Bind` will insert and path-escape each ID.
The generated helper keeps the link tied to the filesystem route instead of a
copied `"/users/"` string.

## 6. Add The Root Layout

Create `app/routes/layout.go`:

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

The root layout wraps the home page and every descendant page. Goldr puts the
matched page component in `ctx.Child` and its metadata in `ctx.Metadata`.
`@child` renders that page inside the document.

HTMX stays visible as a normal script element. This tutorial uses standard
links, so the application still works if the browser cannot load that script.

## 7. Add The Dynamic User Page

Create `app/routes/users/by_id/route.go`:

```go
package by_id

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

var Route = goldr.RouteDef{
	Page: page,
}

func page(r *http.Request) goldr.PageRouteResponse {
	id := r.PathValue("id")
	return goldr.NewPage(
		PageView(id),
		goldr.PageMetadata{Title: "User " + id},
	)
}
```

Goldr maps `users/by_id` to `/users/{id}`. Go's `r.PathValue("id")` reads the
decoded value that matched `{id}` and passes it to the template.

Create `app/routes/users/by_id/page.templ`:

```templ
package by_id

import "example.com/hello-goldr/app/urls"

templ PageView(id string) {
	<section>
		<h1>Hello User { id }</h1>
		<p><a href={ urls.Root.Path() }>Choose another user</a></p>
	</section>
}
```

The Back link uses the generated root helper. This page has no local layout,
so it inherits `app/routes/layout.go` from its parent route directory.

## 8. Generate The Application

You have now written every application-owned file used by this tutorial:

```text
main.go
app/routes/layout.go
app/routes/layout.templ
app/routes/page.templ
app/routes/route.go
app/routes/users/by_id/page.templ
app/routes/users/by_id/route.go
```

Generate templ output, route dispatch, URL helpers, and inspection support,
then tidy the module:

```bash
go tool goldr generate
go mod tidy
```

Goldr and templ add these generated files:

```text
app/internal/goldrinspect/goldr_gen.go
app/routes/goldr_gen.go
app/routes/layout_templ.go
app/routes/page_templ.go
app/routes/users/by_id/goldr_gen.go
app/routes/users/by_id/page_templ.go
app/urls/goldr_gen.go
```

Regenerate after changing a route declaration or `.templ` file. Do not edit
these generated files.

## 9. Validate The Application

Check generated-file freshness, compile every package through `go test`, and
build the application:

```bash
go tool goldr check
go test ./...
go build ./...
```

`goldr check` prints nothing when the route tree and generated files are
current. It validates without rewriting files.

## 10. Run The Application

Start the server:

```bash
go run .
```

Open `http://127.0.0.1:8080`. The home page shows:

```text
Open User 1
Open User 2
Open User 3
Open User 4
Open User 5
```

Open each link. `/users/3`, for example, renders `Hello User 3`. Use the
`Choose another user` link to return to `/`.

Stop the server with `Ctrl+C` when you finish checking the pages.

## 11. Inspect The Application

Goldr can show the route surface without running the server:

```bash
go tool goldr routes list
```

The completed application reports:

```text
KIND    METHOD    PATH         PARAMS  SOURCE                OWNER  DECL   NAME  TITLE  LABELS  NAV  TRAIL_KEYS  HELPER
layout  -         /            -       layout.go             -      -      -     -      -       -    -           -
page    GET,HEAD  /            -       route.go              -      local  -     -      -       -    -           urls.Root.Path()
page    GET,HEAD  /users/{id}  id      users/by_id/route.go  -      local  -     -      -       -    -           urls.Users.ByID.Bind(id).Path()
```

Explain one concrete URL:

```bash
go tool goldr routes explain /users/3
```

```text
/users/3  GET

MATCH
  page     /users/{id}
  source   app/routes/users/by_id/route.go
  params   id = "3"

DECLARATION
  kind     local
  source   app/routes/users/by_id/route.go
  name     -
  title    -
  labels   -

IMPLEMENTATION
  page     page -> GoldrRoutePage

LAYOUT STACK
  / app/routes/layout.go
```

The explanation connects the browser URL to its source, decoded parameter,
generated adapter, and inherited layout.

Inspect the full layout tree:

```bash
go tool goldr routes layouts
```

The layout map places both pages below the root layout. It also shows the
`users/by_id` directory and its `id` parameter in the route tree.

These commands read the same filesystem declarations that Goldr uses for
generation. They do not need a runtime route registry or a running server.

## 12. Use The Development Loop

Run the live-reload server while changing routes or templates:

```bash
go tool goldr dev
```

Open the proxy URL printed by `goldr dev`. The command watches templ files,
regenerates Goldr output, restarts the application, and reloads the browser.
Read [Live Reload](live-reload.md) for asset and Tailwind workflows.

## Use The Scaffold Next Time

You created the application-owned files by hand in this tutorial. For a new
Go module, Goldr can create the minimal root route and layout skeleton:

```bash
go tool goldr init
```

Do not run `goldr init` in the application you just built because `app`
already exists. In a new module, the command creates the four root route and
layout source files plus initial generated output. It does not create
`go.mod`, write `main.go`, run the application, or add the dynamic user route.

## Next Steps

- Read [Concepts](concepts.md) for pages, fragments, actions, layouts, and
  generated handlers.
- Read [HTMX](htmx.md) to add visible `hx-*` interactions and partial HTML
  responses.
- Read [Routes](routes.md) for nested parameters, actions, fragments,
  middleware, and URL helper behavior.
- Read [CLI](cli.md) for all generation, inspection, asset, and development
  commands.
