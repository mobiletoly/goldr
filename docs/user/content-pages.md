# Content Pages

Build websites and application sections from Markdown and HTML files. Goldr
renders content pages through your application's layouts, so they share the
same structure and presentation as your Go-backed pages. Add new pages without
writing a handler for each one.

Store content as external files to update it without rebuilding or restarting
the application, or embed it in your Go binary for deployment. Use content
pages for documentation, guides, articles, product information, and other
content you maintain with your application.

Content authors must be trusted like template authors. Goldr preserves authored
HTML, including HTML inside Markdown, and does not sanitize it. Public uploads
and other untrusted content are unsupported.

## Add Your First Content Page

Start with a working Goldr app, such as the one from
[Getting Started](getting-started.md). This walkthrough adds
`/guides/getting-started` to that app. The optional `content` package is part of
the Goldr module you already installed; it does not need a separate version.

Stop the running server while you add the files and update `main.go`. Run the
commands below from the application root, beside `go.mod`.

### Write The Page

Keep content outside `app/`. Create its directory:

```bash
mkdir -p content/guides/getting-started
```

Create `content/guides/getting-started/page.json`:

```json
{
  "title": "Getting started",
  "description": "Your first steps with our application."
}
```

The title and optional description become page metadata for your layout.
The Getting Started layout uses the title in the browser tab; add a description
meta tag to your layout if you want to render the description too.

Create `content/guides/getting-started/body.md`:

```markdown
# Getting started

Welcome to our product guides.

## First steps

1. Explore the application.
2. Try the example workflow.

[Return to the home page](/)
```

The content directory determines the URL. This page will appear at
`/guides/getting-started`, with no `content` prefix and no file extension.

### Connect Content To Your Server

Replace the Getting Started app's `main.go` with:

```go
package main

import (
	"log"
	"net/http"
	"os"

	"example.com/hello-goldr/app/routes"
	"github.com/mobiletoly/goldr/content"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	root, err := os.OpenRoot("content")
	if err != nil {
		return err
	}
	defer root.Close()

	pages, err := content.New(content.Config{FS: root.FS()})
	if err != nil {
		return err
	}

	handler := routes.HandlerWithOptions(routes.HandlerOptions{
		AdditionalPageSource: pages.Resolve,
	})
	mux := http.NewServeMux()
	mux.Handle("/", handler)

	log.Println("listening on http://127.0.0.1:8080")
	return http.ListenAndServe("127.0.0.1:8080", mux)
}
```

Use your own module path for the `app/routes` import. In an existing server,
keep your middleware, static handlers, and other handler options when adding
`AdditionalPageSource`.

`os.OpenRoot` opens the content directory and confines filesystem access to it.
The `run` function keeps it open while the server runs. `content.New` checks the
content tree at startup, and `pages.Resolve` loads a content page when no
application route matches the request.

Application routes keep priority. The new content page uses the existing root
layout, just like the home and user pages.

### Open The Page

Tidy the module and start the server:

```bash
go mod tidy
go run .
```

Open [http://127.0.0.1:8080/guides/getting-started](http://127.0.0.1:8080/guides/getting-started).
You should see the heading **Getting started**, a numbered list, and a link
back to the home page. The browser tab uses the title from `page.json`.

The handler setup is complete. Adding another content page now requires only
its content files, with no new Go handler or route generation.

## Edit And Preview Content

Change a sentence in `body.md`, save, and refresh the browser. Goldr reads
external content on each request, so the change appears without restarting the
server. New content pages work the same way once both files are in place.

For automatic browser refresh, stop the server with Ctrl+C and run:

```bash
go tool goldr dev --reload-path content
```

Open [http://127.0.0.1:7331/guides/getting-started](http://127.0.0.1:7331/guides/getting-started).
Save another edit to see the browser refresh. Changes under this reload path
refresh the browser without regenerating routes or restarting the application.
See [Live Reload](live-reload.md) for more development options.

## Add More Pages

Each page directory contains `page.json` and one body file: `body.md` for
Markdown or `body.html` for HTML. You can mix formats across a content tree:

```text
content/
  guides/
    page.json                /guides
    body.md
    getting-started/
      page.json              /guides/getting-started
      body.md
    installation/
      page.json              /guides/installation
      body.html
  articles/
    first-release/
      page.json              /articles/first-release
      body.md
```

A directory can have its own page and contain child pages. Without its own
page files, it only groups children: `articles/` above does not create an
`/articles` page. Add its metadata and body if you want an index page there.

For an HTML page, create `content/guides/installation/page.json`:

```json
{"title": "Installation"}
```

Then create `content/guides/installation/body.html`:

```html
<h1>Installation</h1>
<p>Follow these steps to set up your application.</p>
```

Write the page body; your Goldr layout provides the surrounding HTML document.
Do not include both `body.md` and `body.html` in the same page directory.

### Link To Content Pages

Use ordinary URL paths in your templates, Markdown, or HTML:

```html
<a href="/guides/getting-started">Getting started</a>
```

Content pages do not receive generated `urls` helpers or entries in
`goldr routes list`. Those describe the routes declared under `app/routes`.
Use a Go-backed route when you need a request handler and its generated helpers.

### Use Markdown Features

Markdown supports GitHub-Flavored Markdown features, including tables, task
lists, strikethrough, and bare URL links. Goldr also adds IDs to Markdown
headings: the `First steps` heading above is linkable as
`/guides/getting-started#first-steps`.

Goldmark generates these IDs and adds numeric suffixes to duplicates. Exact
IDs can differ from GitHub's; inspect the rendered heading when linking to
punctuation-heavy or repeated headings. Raw HTML headings retain the IDs you
write yourself.

## Use Layouts, Styles, And Images

Content pages use the root layout and any matching layouts in static route
directories. For a shared guides navigation area, add:

```text
app/routes/
  layout.go                  Site-wide layout
  layout.templ
  guides/
    layout.go                Layout for /guides and its content pages
    layout.templ
```

Stop the development server before adding the new layout files. Create
`app/routes/guides/layout.go`:

```go
package guides

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(_ *http.Request, ctx goldr.LayoutContext) templ.Component {
	return LayoutView(ctx.Child)
}
```

Create `app/routes/guides/layout.templ`:

```templ
package guides

templ LayoutView(child templ.Component) {
	<nav aria-label="Guides">
		<a href="/guides/getting-started">Getting started</a>
		<a href="/guides/installation">Installation</a>
	</nav>
	<section>
		@child
	</section>
}
```

This directory needs no `route.go` because it only supplies a layout. Generate
its wiring, then restart the development server:

```bash
go tool goldr generate
go tool goldr dev --reload-path content
```

Open `/guides/getting-started` again. The root layout wraps the guides layout,
which wraps the content. Changes to layout or middleware files require route
generation; edits to external content do not.

Static route-directory middleware also applies to matching content pages.
Layouts and middleware under dynamic directories such as `by_id`, and layouts
from mounted route subtrees, do not apply to content pages. Put authentication
that must cover the whole application in mux-level middleware. See
[Routes](routes.md) and [Composition](composition.md) for middleware setup.

### Style The Content Body

Goldr places the rendered body inside `<article class="goldr-content">`.
It does not supply content styles. For a quick preview, add this inside the
`<head>` of your root `layout.templ`:

```html
<style>
  .goldr-content {
    max-width: 70ch;
    margin-inline: auto;
    line-height: 1.6;
  }
  .goldr-content img {
    max-width: 100%;
    height: auto;
  }
</style>
```

You can move these rules into your application's stylesheet as it grows.
Serve images through your normal static asset handler or image host, then use
those public URLs in Markdown or HTML. Files beside `body.md` are not served
as assets. See [Assets](assets.md) for Goldr's asset workflow.

## Choose External Or Embedded Content

| Task | External files | Embedded files |
| --- | --- | --- |
| Update content | Save the files; the next request reads them | Rebuild and restart the application |
| Deploy | Ship the content directory with the application | Include the content in the Go binary |
| Preview edits | Use `--reload-path content` | Rebuild to see changes |

The walkthrough uses external files. Run it from the application root so
`os.OpenRoot("content")` finds the directory, or change that path for your
deployment. Keep the content directory available while the server runs.

To embed the same content, update `main.go`:

1. Remove the `os` import and add `"embed"` and `"io/fs"`.
2. Add this declaration at package scope, outside any function:

```go
//go:embed content
var contentFiles embed.FS
```

3. Replace the filesystem setup at the start of `run`, through the
   `content.New` error check, with:

```go
files, err := fs.Sub(contentFiles, "content")
if err != nil {
	return err
}
pages, err := content.New(content.Config{FS: files})
if err != nil {
	return err
}
```

Keep the handler and server setup below it unchanged. The embed path is
relative to `main.go`; `fs.Sub` removes the outer `content/` prefix so the
loader sees `guides/` at its root.

Stop any running server, build, and run:

```bash
go build -o hello-goldr .
./hello-goldr
```

The content pages now travel with the binary. Rebuild and restart after content
edits; `--reload-path` alone cannot update embedded files.

## Rules And Troubleshooting

### Files And URLs

- Each page needs a nonblank string `title` in `page.json`; `description` is
  optional. Unknown or duplicate keys, nulls, wrong types, malformed JSON,
  and trailing JSON values are errors.
- Metadata and body files must be valid UTF-8. Bodies must contain more than
  whitespace. Metadata is limited to 16 KiB, source bodies to 512 KiB, and
  rendered Markdown to 2 MiB.
- Directory names use lowercase letters and digits separated by single
  hyphens, such as `getting-started`. Use exact paths without a trailing slash.
  Escaped path segments, dot segments, empty segments, and backslashes do not
  resolve; Goldr does not normalize them into content URLs.
- Keep the home page in `app/routes`: the content root does not represent `/`
  and cannot contain page files directly. Other pages can all come from content;
  a generated handler can serve them even without declared page endpoints.
- Symlinks and special files are rejected. Other ordinary files are ignored.

### A Page Does Not Appear

Check the URL against the content directory, confirm both page files exist,
and check the server's working directory. A directory that only groups children
does not have a page of its own.

Check for an application route at the same URL. Generated static, dynamic,
and mounted routes take priority over content, including when the matched
handler returns a 404 or error. Content does not replace those responses.

Content handles `GET` and `HEAD`. A missing page, unsupported method, or invalid
content URL leaves the request to the normal not-found handler. An existing
but incomplete or invalid content page produces a route error rather than a
404. Use [Error Handling](error-handling.md) to log the error and customize the
response. Content errors use the root error layout, not a nested content layout.

### Check Content Before Serving It

`content.New` validates the complete tree at startup. A later invalid edit
fails requests for that page until you correct it; Goldr does not retain a
last-known-good copy. Saving metadata and body files separately is not an
atomic update. For deployments, validate a complete content directory before
making it available to the server.

For an application-owned check command, call `content.Check` with the same
filesystem you would pass to `content.New`. It performs the same validation
without constructing a page resolver. The [content pages example](../../examples/content_pages)
shows a complete `-check-content` mode; that flag belongs to the example app.

These checks cover files, metadata, encoding, and size limits. They do not
sanitize HTML or check whether authored scripts and links are safe. Raw HTML
passes through in both formats, and the article wrapper is a styling hook,
not a containment boundary. Your application owns cache headers, CSP, logging,
public error messages, deployment, and rollback; a CSP does not replace trust
in content authors.

## Further Reading

- [Content pages example](../../examples/content_pages) - HTML, Markdown,
  nested layouts, middleware, assets, and a check-only command.
- [Content Architecture](../arch/content.md) - resolver behavior, source
  composition, and detailed layout and error rules.
