# User Documentation

This documentation is for developers building applications with goldr.

goldr is the Go Layout-Driven Router: a server-first framework where the
filesystem is the route map, layouts compose by directory, templates render
HTML, and HTMX stays visible in the markup.

## Documentation

- [Getting Started](getting-started.md) - build a minimal app by hand, then
  see where `go tool goldr init` fits.
- [Concepts](concepts.md) - pages, layouts, fragments, actions, render units,
  generated handlers, and URL helpers.
- [CLI](cli.md) - app-local `go tool goldr` commands.
- [Routes](routes.md) - filesystem conventions and runtime behavior.
- [Mounted Kit Route Subtrees](mounted-routes.md) - reusable non-live
  `app/mounts` route surfaces mounted by real `app/routes` owners.
- [Navigation Trails](navigation.md) - app-owned contextual trails,
  breadcrumb-style rendering, and app-level Back links.
- [Client Islands](client-islands.md) - bounded React or Svelte components,
  HTMX lifecycle cleanup, navigation, and app-owned frontend builds.
- [HTMX](htmx.md) - visible `hx-*` attributes and response headers.
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
- [Coding Agents](coding-agents.md) - repository guidance for agents working
  on Goldr applications.

## Current Scope

goldr is v0. These docs describe current supported behavior only. They do not
document planned features, migration history, or deprecated alternatives.
