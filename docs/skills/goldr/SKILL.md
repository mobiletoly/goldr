---
name: goldr-app
description: Create, bootstrap, edit, debug, review, or extend applications that use the Goldr Go web framework. Use when Codex needs to add Goldr to an app, work on Goldr routes, layouts, pages, fragments, actions, templ templates, HTMX behavior, forms, CSRF, assets, SSE, generated output, or local development workflows.
---

# Goldr App Development

Use this skill inside downstream application repositories that use Goldr, or
when adding Goldr to an application.

Goldr is server-first, HTML-first, HTMX-native, filesystem-routed, and
Go-native. Keep application behavior explicit and inspectable. Do not introduce
SPA routing, hydration, virtual DOM, framework-owned browser state, hidden
client synchronization, or a second routing system.

This package is self-contained. Do not assume the target project has Goldr's
repository, Goldr's repository docs, or network access available. Do not clone
or edit the Goldr framework repository while working in a downstream app unless
the user explicitly asks for framework development.

## Package Map

- `github.com/mobiletoly/goldr`: route declarations, Kit-backed route
  declarations, route responses, pages, fragments, layouts, generated handler
  options, template inspection, and HTML writing helpers.
- `github.com/mobiletoly/goldr/hx`: HTMX request checks and response headers.
- `github.com/mobiletoly/goldr/csrf`: signed-cookie CSRF token helpers.
- `github.com/mobiletoly/goldr/browser`: optional browser helper scripts.
- `github.com/mobiletoly/goldr/sse`: server-sent event wire helpers.

## Default Workflow

1. Decide the current app state:
   - existing Goldr app
   - Go module that needs Goldr added
   - new app/module requested by the user
2. For setup, missing dependencies, missing tools, or a new app skeleton, read
   `references/project-setup.md` before editing.
3. Identify the Goldr app root and inspect the current structure before
   editing:
   - `app/routes` under the app root
   - `app/mounts` when routes use `goldr.KitRouteMount`
   - app server setup
   - project scripts and existing tests
   - existing route paths and helpers with `go tool goldr routes list` when
     refactoring routes
4. Edit app-owned source first:
   - route `.go` files
   - route `.templ` files
   - app server setup, middleware, deps, assets, and tests
5. For route refactors, run `go tool goldr routes list` again after generation
   to check that paths and generated helper names match the intended ownership.
6. Do not hand-edit generated files. Regenerate them.
7. Run focused validation, then broader validation when the change crosses
   package or browser-visible boundaries.
8. Stop any dev server you start.

Common commands, when the app does not provide wrapper scripts:

```bash
go tool goldr generate
go tool goldr check
go tool goldr routes list
go test ./...
```

Prefer project-specific commands when the application defines them.

## Read References By Task

Load only the references needed for the current request:

- For adding Goldr to a project, creating a minimal app, installing app-local
  tools, or understanding app ownership boundaries: read
  `references/project-setup.md`.
- For route trees, pages, layouts, params, URL helpers, generated handlers, or
  custom error hooks: read `references/routes.md`.
- For navigation trails, breadcrumb-style rendering, or app-level Back links:
  read `references/navigation.md`.
- For shared page, fragment, or action implementations mounted by multiple
  filesystem-owned routes: read `references/shared-kit-routes.md`.
- For fragments, visible HTMX attributes, action responses, `hx` headers, or
  embedded fragment wrappers: read `references/htmx-fragments-actions.md`.
- For request parsing boundaries, multipart uploads, CSRF, or app dependency
  wiring: read
  `references/forms-csrf-dependencies.md`.
- For fingerprinted assets, `goldr dev`, browser helpers, or SSE streams: read
  `references/assets-dev-sse.md`.
- For local render-unit debugging with comments or overlays: read
  `references/template-inspection.md`.
- Before finalizing non-trivial app changes: read `references/validation.md`.

## Non-Negotiables

- Keep `hx-*` attributes visible in `.templ` files.
- Keep app server concerns app-owned: mux, middleware, auth, sessions,
  persistence, validation, static serving, cache policy, and deployment.
- Keep Goldr render behavior local to the filesystem route tree.
- Keep pages, layouts, fragments, and actions close to their route directory.
- Prefer nested route-local action or fragment directories when an HTMX
  endpoint exists only to support one page workflow, even when that child route
  has no standalone page.
- Keep templates used by only one route directly in that route directory. Use
  route-local `internal` packages or Kit routes only for real shared
  implementation reuse across sibling routes or route trees.
- Choose route directory names for the generated helper shape. Avoid repeating
  parent context in child names.
- Keep route surface in `route.go`.
- Use Kit routes only for real shared implementation reuse. Each URL still
  belongs to a filesystem route directory with its own `route.go`.
- Use `app/mounts` only for non-live Kit route subtrees mounted by real
  `app/routes` owners. Do not put app policy middleware in `app/mounts`.
- Use `KitRouteMount.Routes` when a live owner exposes only part of a mounted
  subtree. Excluded children must be absent routes, not middleware-only
  rejections. A child-only selection still has an owner mount-base URL helper
  for binding `NewGoldrMountURLs`; the mount root does not dispatch unless `/`
  is selected.
- Do not assume a local checkout of the Goldr framework exists in the target
  project.
- Use generated URL helpers instead of hard-coded app route paths when helpers
  exist.
- Bind app URL helpers with `urls.WithBasePath(...)` once when the app is
  served below a URL prefix, and pass the mounted route set through app-owned
  helpers instead of rebuilding prefixed strings.
- Bind mounted subtree helpers from the live owner helper object, for example
  `reports.NewGoldrMountURLs(urls.Admin.Reports)`. Do not pass a raw path
  string. Mount helpers include mounted source routes; check app-owned owner
  state before rendering links to children that only some owners select.
- Inside a mounted implementation, use the bound mount helper for links, HTMX
  fragments, and actions that live inside the same mounted subtree, such as
  `kit.URLs.ResetPassword.Path()`. Do not pass separate raw URL strings for
  same-mount child routes; use owner callbacks only for destinations outside
  the mounted subtree or routes with no generated helper.
- Prefer generated helper paths for HTMX response headers too, such as
  `hx.PushURL(w, urls.Users.Path())`, unless the target is intentionally
  external or not represented by Goldr routes.
- Do not patch generated `goldr_gen.go`, `*_templ.go`, asset manifests, or
  generated URL helpers by hand.
- Do not depend on template-inspection comments or overlay elements in
  production behavior or tests.

## Generated Files

Treat these as generated output:

```text
app/routes/goldr_gen.go
app/routes/**/goldr_gen.go
app/urls/goldr_gen.go
app/mounts/<mount>/goldr_gen.go
app/internal/goldrinspect/goldr_gen.go
**/*_templ.go
assets/goldr_assets_gen.go
assets/dist/*
assets/.goldr/*
```

If generated files are stale, run the appropriate generator instead of editing
them directly.

## Stop Conditions

Stop and inspect before continuing when:

- the app does not appear to follow Goldr route conventions
- a generated file differs from app-owned source expectations
- a route, layout, or fragment is not visible in `app/routes` where expected
- an app-owned env var, flag, or script is ambiguous
- browser-visible behavior changed but has not been checked in a browser
- the requested change would require SPA, hydration, hidden client state, or a
  second routing system
- failures appear unrelated to the current edit

Prefer a narrow, source-grounded app fix over broad rewrites.
