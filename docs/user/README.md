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

- [CLI](cli.md) - app-local `go tool goldr` commands.
- [Routes](routes.md) - filesystem conventions and runtime behavior.
- [Mounted Kit Route Subtrees](mounted-routes.md) - reusable non-live
  `app/mounts` route surfaces mounted by real `app/routes` owners.
- [Navigation Trails](navigation.md) - app-owned contextual trails,
  breadcrumb-style rendering, and app-level Back links.
- [HTMX](htmx.md) - visible `hx-*` attributes and response headers.
- [Forms](forms.md) - form parsing, validation errors, and redisplay.
- [Error Handling](error-handling.md) - route errors, custom generated error
  hooks, full-page errors, and HTMX error fragments.
- [Assets](assets.md) - fingerprinted static files, cache headers, and
  app-owned asset tooling.
- [SSE](sse.md) - app-owned streams, event IDs, and named SSE swaps.
- [CSRF](csrf.md) - signed-cookie tokens for unsafe form and HTMX requests.
- [Composition](composition.md) - mux, middleware, static assets, and app-owned
  server behavior.
- [Application Dependencies](dependencies.md) - app-owned typed dependencies
  for generated route packages.
- [Live Reload](live-reload.md) - `goldr dev`, browser reload, assets, and
  Tailwind workflows.
- [Template Inspection](template-inspection.md) - local render-unit comments,
  visible browser overlays, and app-owned env-var wiring.

- [Goldr App Skill](../skills/goldr/SKILL.md) - installable skill package
  for agents working inside goldr applications.

## What To Build First

Use `getting-started.md` when you want to create the smallest working goldr app.

Use `examples/full_feature/` when you want to see a larger app that combines
pages, nested layouts, fragments, actions, forms, URL helpers, custom errors,
middleware, CSRF, and fingerprinted static assets.

From a goldr checkout:

```bash
(cd examples/full_feature && go run .)
```

Use `examples/chat/` when you want to see app-owned server-sent events with
HTMX, the `sse` protocol helper, and Goldr's named-event browser swap helper:

```bash
(cd examples/chat && go run .)
```

Inspect the example route tree:

```bash
(cd examples/full_feature && go tool goldr routes layouts)
(cd examples/full_feature && go tool goldr routes list)
```

## Current Scope

goldr is v0. These docs describe current supported behavior only. They do not
document planned features, migration history, or deprecated alternatives.
