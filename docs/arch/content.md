# Content Architecture

Goldr content support is split at the generated router miss boundary.

The generated route package owns selection and response writing. It invokes at
most one configured `HandlerOptions.Fallback` after ordinary route selection
misses and before final not-found handling. Static, parameterized, and mounted
matches, matched method mismatches, endpoint responses, endpoint errors, and
explicit invalid-path rejections never enter fallback.

The optional `github.com/mobiletoly/goldr/content` package lives in the root
Goldr module and owns filesystem loading, metadata decoding, operational body
validation, Markdown conversion, trusted raw insertion, and page construction.
The generator has no content import, file-format knowledge, resolver registry,
or automatic source discovery.

```text
request
  -> generated route selection
     -> match: existing endpoint ownership
     -> miss: application fallback
        -> handled: existing page writer and root layout/error machinery
        -> decline: existing RouteNotFound hook or default 404
```

`handled=true` is terminal. It may carry a page, redirect, text response, or
`goldr.RouteError`. Writer validation and rendering failures use the existing
root-level route error path. A custom error-hook failure remains terminal and
does not recurse. `handled=false` ignores the response and runs final not-found
handling once.

Content pages use only the root layout because no route subtree matched. They
do not acquire route-tree middleware or generated navigation identity. Outer
application middleware still sees the original request and provides any shared
context needed by the root layout.

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
checks. Markdown uses Goldmark CommonMark without optional extensions and with
`html.WithUnsafe()`, so raw HTML and potentially dangerous destinations are
emitted as authored. Rendered HTML is not parsed or validated. Both formats
retain the article wrapper, but authored HTML can affect or escape it in the
browser, so it is not a DOM containment boundary.

`Check` enforces operational filesystem, metadata, UTF-8, nonblank-body, size,
symlink, special-file, and I/O rules. It is not an HTML safety check. Public or
otherwise untrusted content is unsupported, and there is no sanitizer, policy
callback, or strict mode. Application CSP remains separately owned and does not
replace author trust.

Applications own source order, filesystem lifetime, mux composition, cache
headers, assets, CSP, logging, check-only commands, deployment, and rollback.
The public module surface remains `Config`, `Pages`, `New`, `Check`, and
`Resolve`.
