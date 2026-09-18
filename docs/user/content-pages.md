# Content Pages

The optional `github.com/mobiletoly/goldr/content` package loads trusted,
first-party HTML and Markdown pages from an `fs.FS`. Generated application
routes keep priority. Only an ordinary router miss reaches the configured
additional page source.

Select the Goldr runtime and content package with one root module version:

```bash
go get github.com/mobiletoly/goldr@${GOLDR_VERSION}
```

Goldmark is a direct dependency in the root Goldr module graph, but only the
optional content package imports it. Applications that do not import the
content package do not compile or link the Markdown parser into their binaries.

## Directory Layout

Keep content and assets outside `app/`:

```text
app/routes/
assets/
content/
  about/
    page.json
    body.md
  privacy/
    page.json
    body.html
    p1/
      page.json
      body.md
```

These entries map to `/about`, `/privacy`, and `/privacy/p1`. A directory may
contain both a page and child directories. A directory with children but no
recognized page files is an organizational container and does not resolve.

Each page has `page.json` and exactly one of `body.html` or `body.md`:

```json
{"title":"Privacy","description":"How this website handles information."}
```

`title` is a required nonblank string. `description` is an optional string.
Unknown keys, duplicate keys, nulls, wrong types, malformed JSON, and trailing
JSON values are errors. Metadata and body files must contain valid UTF-8, and
the body must contain non-whitespace content.

Directory names use lowercase letters and digits separated by single hyphens.
Content URLs use the same segments without a prefix. The root URL is not a
content entry, and recognized page files at the filesystem root are invalid.
Escapes, dot segments, empty segments, backslashes, and trailing slashes decline
without being normalized.

## Bootstrap

Use `os.OpenRoot` for editable external content and keep it open for the life
of the server:

```go
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
```

`content.New` validates the complete tree before returning. `content.Check`
performs the same validation without creating a resolver and is suitable for a
check-only application mode or deployment gate. These checks cover filesystem
shape, metadata, UTF-8, sizes, symlinks, special files, and I/O rules. They are
not HTML safety checks.

`Resolve` handles valid `GET` and `HEAD` content requests. Missing pages,
containers, unsupported methods, and invalid URL identifiers decline. Existing
but incomplete or invalid entries return a terminal `goldr.RouteError` so the
normal generated `RouteError` hook can log and present a generic error.

## Route And Error Ownership

Generated static, parameterized, and mounted routes run before the additional
page source. A matched route keeps ownership of its 404, error, and method
mismatch. The source is not response interception and is not invoked from an
error hook or an explicit invalid-path rejection.

On an ordinary miss, generated routing selects static `app/routes` ancestry
from the original `r.URL.EscapedPath()`. Matching root-to-leaf middleware wraps
source resolution, response writing, final not-found handling, and route-error
handling. Matching layouts render handled pages in outer-to-inner order and
receive the page metadata and layout data unchanged. Layout-only and
middleware-only static directories participate after regeneration.

Goldr does not clean or decode the path, infer dynamic ownership, or bind path
values for an additional page. Layouts and middleware at or below a dynamic
directory are excluded, as are layouts from `app/mounts`. Mounted middleware is
invalid. A real static owner at the same URL prefix still participates. A
middleware change to the request reaches the source and renderers but does not
reselect ancestry.

An eligible static middleware such as `app/routes/admin/middleware.go` may
authenticate or add request context before resolving an additional page below
`/admin`. Keep policy that must cover nil-source misses, invalid-path rejection,
and handlers outside this branch in mux-level middleware.

If content declines, the configured `RouteNotFound` hook or default 404 remains
the final response inside the selected middleware chain. If content returns an
error with `handled=true`, resolution is terminal and uses `RouteError`
handling. Additional-page errors render through only the live
`app/routes/layout.go` root layout when present; success pages may use the full
selected static layout stack. Existing final not-found rendering keeps its
current root and mounted-root behavior.

If no endpoint is declared, generation still emits `Handler` and
`HandlerWithOptions`; a configured source can therefore serve a content-only
application with eligible static layouts and middleware. Adding or changing a
layout or middleware file requires `goldr generate`. Editing external content
does not.

Route inventory remains generated-route inventory. Content pages and
middleware-only directories do not gain route rows, route identity, or URL
helpers; layout inventory may show the application-owned static layouts that
can wrap them.

## Source Chaining

Compose sources explicitly in the single application callback:

```go
AdditionalPageSource: func(r *http.Request) (goldr.PageRouteResponse, bool) {
	if response, handled := pages.Resolve(r); handled {
		return response, true
	}
	return otherPages.Resolve(r)
},
```

The first handled response wins, including a handled error. If every source
declines, final not-found handling runs once. Goldr does not provide a resolver
registry or chain helper.

## Body Policy

Content authors have the same trust level as application template authors.
Goldr does not sanitize content and does not support public uploads or any
otherwise untrusted content source.

`body.html` passes through without HTML parsing, repair, normalization,
serialization, allowlisting, URL checks, or element-depth checks. Active or
malformed markup is author-owned. Goldr inserts the authored bytes through a
private raw boundary and keeps an `<article class="goldr-content">` wrapper,
but trusted markup can affect or escape that wrapper in the browser. The
wrapper is not a DOM containment promise.

Markdown uses Goldmark's built-in GitHub-Flavored Markdown (GFM) support.
Tables, strikethrough, task lists, and bare URL linking are available for every
`body.md`. Markdown headings receive automatic Goldmark `id` attributes. IDs
use Goldmark normalization and document-local duplicate suffixes such as
`-1`; Goldr does not promise byte-for-byte GitHub slug compatibility. Raw HTML
headings are not rewritten.

The unsafe renderer emits raw block and inline HTML and potentially dangerous
link and image destinations as authored. Goldr does not parse or validate the
rendered HTML, and it adds no CSS for GFM's semantic elements. Applications own
their styling.

Metadata is limited to 16 KiB and source bodies to 512 KiB. Rendered Markdown
is limited to 2 MiB. Bodies must be valid UTF-8 and contain non-whitespace
content. Symlinks and special files are rejected. Ordinary unrecognized files
are ignored and are never served as assets.

## External And Embedded Content

External entries are validated at startup and read again for every request.
Edits and new pages appear on the next request without route regeneration or a
server restart. An invalid live edit fails until corrected. Multi-file saves
are not transactional and there is no last-known-good cache.

For immutable embedded content:

```go
//go:embed content
var contentFiles embed.FS

sub, err := fs.Sub(contentFiles, "content")
if err != nil {
	return err
}
pages, err := content.New(content.Config{FS: sub})
```

Embedded changes require rebuilding and restarting the application.

Use existing reload-only watching for external files:

```bash
go tool goldr dev --reload-path content --cmd "go run . -dev"
```

Cache headers, CSP, logging, public error text, deployment, and rollback remain
application-owned. A CSP is separate defense in depth and does not replace the
requirement to trust content authors. Validate an immutable release directory before activation.
Content sources are not static assets; link images and other browser resources
through the application's normal [asset pipeline](assets.md).

See `examples/content_pages` for a complete external-content application with
HTML, Markdown, nested pages, route precedence, assets, custom errors, and a
check-only mode.
