# Content Pages

Use Goldr's optional content package for trusted, first-party HTML or Markdown
pages that do not need one generated route declaration per page. Examples
include product documentation, policies, help pages, and other editorial
content maintained with the application.

Do not confuse content pages with static asset serving. Content pages render
through Goldr's root layout. Images and other browser resources still belong
to the application's asset pipeline or another app-owned file handler.

## Trust Boundary

Content authors must have the same trust level as application template
authors. Goldr does not sanitize content and does not support public uploads or
any otherwise untrusted content source.

`body.html` passes through without HTML parsing, repair, normalization,
serialization, allowlisting, URL checks, or element-depth checks. Active or
malformed markup is author-owned. Goldr inserts the authored bytes through a
private raw boundary and keeps an `<article class="goldr-content">` wrapper,
but trusted markup can affect or escape that wrapper in the browser. The
wrapper is not a DOM containment boundary.

Markdown uses Goldmark's built-in GitHub-Flavored Markdown (GFM) support.
Every `body.md` supports tables, strikethrough, task lists, and bare URL
linking. Markdown headings receive automatic Goldmark IDs with document-local
duplicate suffixes such as `-1`. IDs use Goldmark normalization and are not
promised to match GitHub slugs byte for byte. Raw HTML headings are not
rewritten.

Raw block and inline HTML and potentially dangerous link or image destinations
are emitted as authored. Goldr does not parse or validate the rendered HTML and
adds no CSS for GFM output; applications own styling.

Never use this package as a sanitizer. An application-owned CSP is separate
defense in depth and does not replace author trust. Do not execute or click
dangerous destinations while testing their preservation.

## Install The Optional Package

Select the Goldr runtime and content package with one root module version:

```bash
go get github.com/mobiletoly/goldr@${GOLDR_VERSION}
```

Goldmark is a direct dependency in the root Goldr module graph, but only the
content package imports it. Applications that do not import the content
package do not compile or link the Markdown parser into their binaries.

## Directory Layout

Keep content outside `app/` so it is not mistaken for filesystem route source:

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
contain a page and child directories. A directory with children but no
recognized page files is an organizational container and does not resolve.

Each page requires `page.json` and exactly one body file:

```json
{"title":"Privacy","description":"How this website handles information."}
```

- `title` is a required nonblank string.
- `description` is an optional string.
- Unknown or duplicate keys, nulls, wrong types, malformed JSON, and trailing
  JSON values are errors.
- Metadata and body files must be valid UTF-8.
- A body must contain non-whitespace content.
- Metadata is limited to 16 KiB, source bodies to 512 KiB, and rendered
  Markdown to 2 MiB.
- Symlinks and special files are rejected. Ordinary unrecognized files are
  ignored and are never served as assets.

Directory names use lowercase letters and digits separated by single hyphens.
Content URLs use the same segments without a prefix. The root URL is not a
content entry. Escapes, dot segments, empty segments, backslashes, and trailing
slashes decline without normalization.

## Wire Editable External Content

Use `os.OpenRoot` to confine editable external content and keep it open for the
life of the server:

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
	Fallback: pages.Resolve,
})
```

`content.New` validates the complete tree before returning. External entries
are then read again for every request. Edits and new pages appear on the next
request without route generation or a server restart. A recognized entry made
invalid by a live edit returns an error until it is corrected. Multi-file saves
are not transactional and there is no last-known-good cache.

Configure shared request context in mux-level middleware outside the generated
handler. Route-tree middleware belongs to matched generated routes and does not
run for fallback content.

## Add A Check-Only Mode

`content.Check` performs the same filesystem and entry validation without
creating a resolver:

```go
if err := content.Check(root.FS()); err != nil {
	return fmt.Errorf("check content: %w", err)
}
```

Use it in an application flag or deployment gate, for example:

```bash
go run . -check-content
```

`Check` validates filesystem shape, entry shape, metadata, UTF-8, nonblank
bodies, input and rendered sizes, symlinks, special files, and I/O behavior. It
is not an HTML safety check.

## Preserve Route And Error Ownership

Generated static, parameterized, and mounted routes run before the fallback.
A generated `/about` route therefore wins over a `content/about` entry. A
matched route keeps ownership of its 404, error, and method mismatch. Content
fallback is attempted at most once only after an ordinary router miss.

`Pages.Resolve` handles valid `GET` and `HEAD` content requests. Missing pages,
organizational containers, unsupported methods, and invalid URL identifiers
decline. Existing but incomplete or operationally invalid entries return a
terminal `goldr.RouteError`. The generated `RouteError` hook should log the
source-local error and return an app-owned generic response.

Resolved content uses the ordinary root layout, request context, buffered
rendering, and `HEAD` behavior. A declined content request continues to the
configured `RouteNotFound` hook or default 404.

Goldr exposes one `HandlerOptions.Fallback`. Compose multiple sources
explicitly when needed:

```go
Fallback: func(r *http.Request) (goldr.PageRouteResponse, bool) {
	if response, handled := pages.Resolve(r); handled {
		return response, true
	}
	return otherPages.Resolve(r)
},
```

The first handled response wins, including a handled error. If every source
declines, final not-found handling runs once. Do not add a resolver registry or
chain helper.

## Use Embedded Content When Immutable

For content that changes only with a new binary:

```go
//go:embed content
var contentFiles embed.FS

sub, err := fs.Sub(contentFiles, "content")
if err != nil {
	return err
}
pages, err := content.New(content.Config{FS: sub})
```

Embedded changes require rebuilding and restarting the application. Do not add
an embedded content path to `--reload-path`.

## Develop And Validate

For editable external content, ask `goldr dev` to refresh the browser after a
change without generation or an application restart:

```bash
go tool goldr dev --reload-path content --cmd "go run . -dev"
```

The running application reads content again on the next request; the reload
path only triggers the browser refresh. Inspect the target application's flags
before copying `-dev` or another example command.

After a content-pages change, run the app's focused tests and ordinary Goldr
validation. Include its check-only mode when present:

```bash
go tool goldr generate --check
go tool goldr check
go test ./...
go run . -check-content
```

Verify at least one generated route still wins over content at the same URL,
one content page renders through the root layout, a missing URL reaches the
final 404, and operational content errors use the application's generic error
response. For external content, verify an edit and a newly added nested page
appear without route generation or a server restart.
