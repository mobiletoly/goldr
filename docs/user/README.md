# User Documentation

This documentation is for developers building applications with goldr.

goldr is the Go Layout-Driven Router: a server-first framework where the
filesystem is the route map, layouts compose by directory, templates render
HTML, and HTMX stays visible in the markup.

## Start Here

Read these first:

1. [Getting Started](getting-started.md) - build a minimal app by hand, then
   see where `go tool goldr init` fits.
2. [Concepts](concepts.md) - learn pages, layouts, fragments, actions, render
   units, generated handlers, and URL helpers.

Then use the references:

- [Routes](routes.md) - filesystem conventions and runtime behavior.
- [CLI](cli.md) - app-local `go tool goldr` commands.
- [Live Reload](live-reload.md) - `goldr dev`, browser reload, assets, and
  Tailwind workflows.
- [Assets](assets.md) - fingerprinted static files, cache headers, and
  app-owned asset tooling.
- [Coding Agents](coding-agents.md) - copyable `AGENTS.md` guidance for coding
  agents working inside goldr applications.
- [HTMX](htmx.md) - visible `hx-*` attributes and response headers.
- [SSE](sse.md) - app-owned streams, event IDs, and named SSE swaps.
- [Forms](forms.md) - form parsing, validation errors, and redisplay.
- [CSRF](csrf.md) - signed-cookie tokens for unsafe form and HTMX requests.
- [Composition](composition.md) - mux, middleware, static assets, and app-owned
  server behavior.
- [Application Dependencies](dependencies.md) - app-owned typed dependencies
  for generated route packages.

## What To Build First

Use `getting-started.md` when you want to create the smallest working goldr app.

Use `examples/full_feature/` when you want to see a larger app that combines
pages, nested layouts, fragments, actions, forms, URL helpers, custom errors,
middleware, CSRF, and fingerprinted static assets.

From the repository root:

```bash
go run ./examples/full_feature
```

Use `examples/chat/` when you want to see app-owned server-sent events with
HTMX, the `sse` protocol helper, and Goldr's named-event browser swap helper:

```bash
go run ./examples/chat
```

Inspect the example route tree:

```bash
go run ./cmd/goldr routes layouts --root examples/full_feature
go run ./cmd/goldr routes list --root examples/full_feature
```

## Current Scope

goldr is v0. These docs describe current supported behavior only. They do not
document planned features, migration history, or deprecated alternatives.
