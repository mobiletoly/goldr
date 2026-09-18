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
- `/missing` for the custom final 404 page

Validate the content tree without starting a server:

```bash
go run . -check-content
```

For external content reload without a Go restart or route regeneration:

```bash
go tool goldr dev --reload-path content --cmd "go run . -dev"
```

The application configures one `HandlerOptions.Fallback`, keeps content and
assets outside `app/`, serves fingerprinted assets through its own mux, and
uses the ordinary root layout and error hooks for content responses.

The content files are trusted application input. Goldr does not validate or
sanitize HTML. Markdown permits raw HTML and potentially dangerous link and
image destinations. Content authors must be trusted like template authors;
public or otherwise untrusted content is unsupported. `-check-content` checks
filesystem shape, metadata, UTF-8, sizes, and other operational rules, not HTML
safety. An application-owned CSP is separate defense in depth and does not
replace author trust.
