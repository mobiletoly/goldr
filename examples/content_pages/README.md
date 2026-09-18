# Content Pages Example

This example combines generated Goldr routes with optional first-party content
loaded from the external `content/` directory.

Run it:

```bash
go run .
```

Then open:

- `/` for the generated home route
- `/about` for a generated route that takes priority over `content/about`
- `/privacy` for an HTML content page
- `/privacy/p1` and `/privacy/p2` for nested Markdown content pages
- `/privacy/missing` for the custom final 404 inside privacy middleware
- `/missing` for the custom final 404 page

Validate the content tree without starting a server:

```bash
go run . -check-content
```

For external content reload without a Go restart or route regeneration:

```bash
go tool goldr dev --reload-path content --cmd "go run . -dev"
```

The application configures one `HandlerOptions.AdditionalPageSource`, keeps
content and assets outside `app/`, and serves fingerprinted assets through its
own mux. The static `app/routes/privacy` directory contains only layout and
middleware declarations: both apply automatically to additional pages below
`/privacy` after generation. Dynamic and mounted ancestry are excluded. The
privacy middleware also wraps a declined `/privacy/missing` request, while the
existing final 404 renderer retains its normal ownership.

External content edits and new content entries appear on the next request and
do not require route regeneration. Adding or changing route-tree layout or
middleware files does require `go tool goldr generate`.

The nested Markdown example uses GitHub-Flavored Markdown (GFM) tables, task
lists, strikethrough, and bare URL linking. Markdown headings receive automatic
Goldmark IDs, so the opening `Privacy part one` heading is linkable as
`#privacy-part-one`.

The content files are trusted application input. Goldr does not validate or
sanitize HTML. Markdown permits raw HTML and potentially dangerous link and
image destinations. Content authors must be trusted like template authors;
public or otherwise untrusted content is unsupported. `-check-content` checks
filesystem shape, metadata, UTF-8, sizes, and other operational rules, not HTML
safety. An application-owned CSP is separate defense in depth and does not
replace author trust.

For a live-error check, temporarily replace a content body with whitespace and
request that page. The response is `500`, the privacy middleware header remains
present, and the route error page does not render through the privacy layout.
Restore the body after the check.
