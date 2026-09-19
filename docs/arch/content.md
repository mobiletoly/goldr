# Content Architecture

Goldr content support is split at the generated router miss boundary.

The generated route package owns selection and response writing. It invokes at
most one configured `HandlerOptions.AdditionalPageSource` after ordinary route selection
misses and before final not-found handling. Static, parameterized, and mounted
matches, matched method mismatches, endpoint responses, endpoint errors, and
explicit invalid-path rejections never enter the source branch.

The optional `github.com/mobiletoly/goldr/content` package lives in the root
Goldr module and owns filesystem loading, metadata decoding, operational body
validation, Markdown conversion, trusted raw insertion, and page construction.
The generator has no content import, file-format knowledge, resolver registry,
or automatic source discovery.

```text
request
  -> generated route selection
     -> match: existing endpoint ownership
     -> miss: select eligible static middleware and layout ancestry
        -> middleware: application additional page source
           -> handled: existing page writer with selected layouts
           -> decline: existing RouteNotFound hook or default 404
```

`handled=true` is terminal. It may carry a page, redirect, text response, or
`goldr.RouteError`. Writer validation and rendering failures use the existing
additional-page route error path. A custom error-hook failure remains terminal and
does not recurse. `handled=false` ignores the response and runs final not-found
handling once.

Generated routing captures `r.URL.EscapedPath()` before middleware and matches
eligible static `app/routes` prefixes without cleaning or decoding. It composes
matching middleware root to leaf and layouts outer to inner. Layout-only and
middleware-only directories participate. Dynamic ancestry and mounted layouts
are excluded; mounted middleware is invalid. The selected middleware wraps the
source, handled response writing, final 404, and route-error handling. A changed
request passed by middleware reaches the source and renderers without
reselecting ancestry.

Handled pages use the selected layout stack and preserve metadata and layout
data. Redirects and text bypass layouts. Additional-page errors use only the
eligible live `app/routes` root layout, excluding mounted-root layouts. Content
pages do not acquire generated navigation identity or dynamic path values.
Outer application middleware continues to provide broader application policy.
Route inventory and URL helpers remain derived from generated routes. Content
entries and middleware-only directories do not acquire route identity.

## Source Composition

Applications can compose sources explicitly in the single
`HandlerOptions.AdditionalPageSource` callback:

```go
AdditionalPageSource: func(r *http.Request) (goldr.PageRouteResponse, bool) {
	if response, handled := pages.Resolve(r); handled {
		return response, true
	}
	return otherPages.Resolve(r)
},
```

The first handled response wins, including a handled error. If every source
declines, final not-found handling runs once. Goldr does not provide a source
registry or chain helper.

## Validation Flow

`New` calls `Check` before returning private resolver state. Tree validation
walks directories in lexical order and validates entry structure, regular-file
requirements, metadata, body UTF-8 and nonblank content, and size limits. Request
loading reuses entry validation and checks every traversed directory and
recognized file.

External content is read on every request. This deliberately permits live
updates without an index, cache, watcher, snapshot, or generation step. Reads
across multiple files are not transactional. Supported external deployments
use a confined `os.OpenRoot(...).FS()` rooted at trusted, checked release files.
Hostile writable filesystems are outside the trust model.

Goldmark is a direct dependency in the root Goldr module graph. Only the
content package imports it, so applications that do not import content do not
compile or link the Markdown parser into their binaries. The separately
versioned CLI module remains isolated from both the root runtime and Goldmark.

Runtime and content releases use the root `vX.Y.Z` tag together. The CLI keeps
its `cmd/goldr/vX.Y.Z` tag. There is no separate `content/vX.Y.Z` release tag.

Content authors are trusted like application template authors. `body.html`
bytes cross a private `templ.Raw` boundary without parsing, repair,
normalization, serialization, allowlisting, URL checks, or element-depth
checks. `body.md` uses one Goldmark parser with `extension.GFMParser` and
`parser.WithAutoHeadingID()`, then one HTML renderer with
`html.WithUnsafe()` and `extension.GFMHTMLRenderer`. This enables GFM tables,
strikethrough, task lists, and bare URL linking, and emits Goldmark-generated
document-local heading IDs. Duplicate heading IDs receive Goldmark numeric
suffixes beginning at `-1`; Goldr does not implement or promise GitHub's exact
slug algorithm. Raw HTML headings are not rewritten.

The unsafe renderer leaves raw HTML and potentially dangerous destinations
authored in Markdown unfiltered. Rendered HTML is not parsed or validated.
Rendered-size validation runs after GFM output and heading attributes are
emitted. Both formats retain the article wrapper, but authored HTML can affect
or escape it in the browser, so it is not a DOM containment boundary.

`Check` enforces operational filesystem, metadata, UTF-8, nonblank-body, size,
symlink, special-file, and I/O rules. It is not an HTML safety check. Public or
otherwise untrusted content is unsupported, and there is no sanitizer, policy
callback, or strict mode. Application CSP remains separately owned and does not
replace author trust.

Applications own source order, filesystem lifetime, mux composition, cache
headers, assets, CSP, logging, check-only commands, deployment, and rollback.
The public module surface remains `Config`, `Pages`, `New`, `Check`, and
`Resolve`.
