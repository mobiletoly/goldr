# Getting Started

Build a small application with a home page, a dynamic user page, and an HTMX
button that loads the server time without navigating away. Start with Goldr's
scaffold, see how its files fit together, then extend the running application.

You need Go 1.26.0 or newer and a terminal. Use the latest patch release in
your supported Go line. This guide uses a POSIX shell, such as bash or zsh,
and assumes you can write basic Go and HTML; it introduces templ and HTMX as
you use them.

## 1. Create Your Application

Create an empty Go module:

```bash
mkdir hello-goldr
cd hello-goldr
go mod init example.com/hello-goldr
```

Run the remaining commands from this directory. If you choose a different
module path, use it in place of `example.com/hello-goldr` in the imports below.

Install Goldr, [templ](https://github.com/a-h/templ), and their command-line
tools:

```bash
GOLDR_VERSION=latest
TEMPL_VERSION=v0.3.1020

go get github.com/mobiletoly/goldr@${GOLDR_VERSION} github.com/a-h/templ@${TEMPL_VERSION}
go get -tool github.com/mobiletoly/goldr/cmd/goldr@${GOLDR_VERSION}
go get -tool github.com/a-h/templ/cmd/templ@${TEMPL_VERSION}
```

Keep the Goldr library and CLI on the same version. Go records the tools in
`go.mod`, so `go tool goldr` and `go tool templ` use this application's versions.

Create the starter:

```bash
go tool goldr init
```

The scaffold gives you four editable files:

```text
app/routes/
  route.go       Home page declaration and handler
  page.templ     Home page HTML
  layout.go      Shared layout logic
  layout.templ   Shared HTML document
```

It also creates initial routing code, URL helpers, and inspection support.
Files named `goldr_gen.go` belong to Goldr; `*_templ.go` files will come from
templ. Regenerate these files rather than editing them.

`init` creates the route skeleton once, in a module without an `app` directory.
You will add the HTTP server in step 3.

## 2. Understand The Starter

The following four files already exist. Read them in your editor; you do not
need to create them again.

### The Page Declaration And Handler

`app/routes/route.go`:

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

The directory determines the URL: `app/routes` owns `/`. The static `Route`
declaration selects `page` as its handler for `GET` and `HEAD`. The handler
returns a templ component plus the page title.

Before compilation, `go tool goldr generate` reads these declarations without
running your application and generates ordinary Go routing code and URL
helpers. You can inspect that wiring, and the Go compiler checks calls to your
handlers. The running server uses the generated code without discovering
routes or loading handlers dynamically.

### The Page HTML

`app/routes/page.templ`:

```templ
package routes

templ PageView() {
	<section>
		<h1>Hello Goldr</h1>
		<p>Edit app/routes/page.templ to start building.</p>
	</section>
}
```

A templ component is HTML with typed Go inputs. `PageView()` defines the
component that the page handler calls. Goldr's generation command also runs
templ to turn this template into Go code.

### The Shared Layout Logic

`app/routes/layout.go`:

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

The root layout wraps the home page and descendant pages. Goldr supplies the
matched page as `ctx.Child` and its metadata as `ctx.Metadata`. This function
passes both to the layout template. `pageTitle` uses the page's title, falling
back to `Hello Goldr` when no title was supplied.

### The Shared HTML Document

`app/routes/layout.templ`:

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
			<script src="https://cdn.jsdelivr.net/npm/htmx.org@4.0.0" integrity="sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc" crossorigin="anonymous" defer></script>
		</head>
		<body>
			<main>
				@child
			</main>
		</body>
	</html>
}
```

`@child` renders the matched page inside `<main>`. Each page can supply its own
content and title while sharing this document shell.

The scaffold also loads HTMX for the interaction you will add later.

## 3. Run And Edit Your First Page

Goldr generates `routes.Handler()`, an ordinary `http.Handler`. Your application
chooses how to serve it. Create `main.go` in the project root:

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

Generate the initial template and route output, tidy the module, and start the
development server:

```bash
go tool goldr generate
go mod tidy
go tool goldr dev
```

Open [http://127.0.0.1:7331](http://127.0.0.1:7331). You should see **Hello
Goldr**. Port 7331 is the development proxy, which adds browser reload; keep
using that URL while following the tutorial.

Change the heading in `app/routes/page.templ` to `My first Goldr page` and save.
The browser reloads with your new heading. `goldr dev` handles template and
route generation, application restarts when needed, and browser reload.

## 4. Add A Dynamic User Page

Stop the development server with Ctrl+C while you add the new route directory
and its two files. From `hello-goldr`, create the directory:

```bash
mkdir -p app/routes/users/by_id
```

`by_id` declares a dynamic segment, so this directory maps to `/users/{id}`.
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

Go's `r.PathValue("id")` reads the matched URL parameter. The handler passes
it to the page template and uses it in the browser title.

Create `app/routes/users/by_id/page.templ`:

```templ
package by_id

templ PageView(id string) {
	<section>
		<h1>Hello User { id }</h1>
	</section>
}
```

Save both files, then start the development server again so it generates the
new route and watches its files:

```bash
go tool goldr dev
```

Open
[http://127.0.0.1:7331/users/3](http://127.0.0.1:7331/users/3). You should see
**Hello User 3** and the browser title **User 3**. Try changing `3` to `5` in
the URL.

This page has no local layout. It shares the root layout you inspected in
step 2, including its HTML document and HTMX script. Keep `goldr dev` running
for the remaining edits. When changing several files, you may briefly see
missing-symbol errors until all the files are saved and generation finishes.

## 5. Link To The Page With Generated URLs

Goldr generates the `app/urls` package from your route declarations. Its Go
helpers give you compiler-checked references to pages, fragments, and actions
without copying URL strings. The new dynamic route has this helper:

```go
urls.Users.ByID.Bind("3").Path() // /users/3
```

`Bind` supplies the parameter; `Path` builds the URL with that value escaped
for the path. If you remove or rename a route, regeneration updates the
helpers and the compiler catches references to helpers that no longer exist.

Replace `app/routes/page.templ` with:

```templ
package routes

import "example.com/hello-goldr/app/urls"

templ PageView() {
	<section>
		<h1>Choose a user</h1>
		<ul>
			for _, id := range []string{"1", "2", "3", "4", "5"} {
				<li>
					<a href={ urls.Users.ByID.Bind(id).Path() }>Open User { id }</a>
				</li>
			}
		</ul>
	</section>
}
```

The Go loop renders five ordinary links. In `app/routes/route.go`, change
`Title: "Hello Goldr"` to `Title: "Choose a user"` so the browser title matches.

Add a return link by replacing `app/routes/users/by_id/page.templ` with:

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

`urls.Root.Path()` refers to the home page. Open
[http://127.0.0.1:7331](http://127.0.0.1:7331), follow **Open User 3**, then
follow **Choose another user** to return. Both pages share the same layout,
and both links use generated helpers.

## 6. Update Part Of The Page With HTMX

Add a button to the user page that fetches the current server time. The page
stays open while HTMX replaces one region with HTML from a Go handler.

Replace `app/routes/users/by_id/route.go` with:

```go
package by_id

import (
	"net/http"
	"time"

	"github.com/mobiletoly/goldr"
)

var Route = goldr.RouteDef{
	Page: page,
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/time", serverTime),
	},
}

func page(r *http.Request) goldr.PageRouteResponse {
	id := r.PathValue("id")
	return goldr.NewPage(
		PageView(id),
		goldr.PageMetadata{Title: "User " + id},
	)
}

func serverTime(_ *http.Request) goldr.FragmentRouteResponse {
	return goldr.NewFragment(TimeView(time.Now().Format(time.TimeOnly)))
}
```

The fragment declaration adds `GET /users/{id}/time` beside the page. Unlike
a page response, a fragment renders its HTML without the surrounding layouts.

Create `app/routes/users/by_id/frag_time.templ`:

```templ
package by_id

templ TimeView(value string) {
	<p>Server time: { value }</p>
}
```

Goldr generates a `Time` helper beneath the bound user route. Replace
`app/routes/users/by_id/page.templ` with:

```templ
package by_id

import "example.com/hello-goldr/app/urls"

templ PageView(id string) {
	<section>
		<h1>Hello User { id }</h1>
		<button
			hx-get={ urls.Users.ByID.Bind(id).Time.Path() }
			hx-target="#server-time"
			hx-swap="innerHTML"
		>
			Show server time
		</button>
		<div id="server-time" aria-live="polite"></div>
		<p><a href={ urls.Root.Path() }>Choose another user</a></p>
	</section>
}
```

The three HTMX attributes describe the interaction:

- `hx-get` requests the generated fragment URL, such as `/users/3/time`.
- `hx-target` selects the element to update.
- `hx-swap="innerHTML"` replaces that element's contents with the response.

After saving all three files, open
[http://127.0.0.1:7331/users/3](http://127.0.0.1:7331/users/3) and click
**Show server time**. Wait a second and click again: the time changes, while
the heading, return link, and browser URL stay in place.

The shared layout already loads HTMX. You wrote the handler and HTML;
HTMX makes the request and swaps the response into the page.

## 7. Check And Build Your Application

Stop `goldr dev` with Ctrl+C in its terminal. Refresh the generated output,
check it, and build a standalone executable:

```bash
go tool goldr generate
go tool goldr check
go test ./...
go build -o hello-goldr .
```

`goldr check` verifies generated output without rewriting it and prints
nothing on success. `go test` compiles the packages and runs any tests you add.

Run the executable:

```bash
./hello-goldr
```

Open [http://127.0.0.1:8080](http://127.0.0.1:8080). This is the application
server directly, without the development proxy. The pages and HTMX interaction
work here too. Stop it with Ctrl+C when you finish.

To see how a URL maps to your code, run:

```bash
go tool goldr routes explain /users/3
```

The output identifies the page's source, the `id` parameter, and the inherited
root layout. Route inspection works without running the server.

## Next Steps

- [Content Pages](content-pages.md) - add About, Privacy, and other Markdown
  or HTML pages using your application's layouts.
- [HTMX](htmx.md) and [CSRF](csrf.md) - add forms and mutation actions.
- [Routes](routes.md) - explore nested layouts, fragments, and middleware.
- [Live Reload](live-reload.md) and [Assets](assets.md) - add CSS and configure
  your development workflow.
- [CLI](cli.md) - inspect routes and learn the available commands.
- [All Documentation](README.md) - browse the full guide and reference.
