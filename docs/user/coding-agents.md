# Coding Agents

Many coding agents read `AGENTS.md` or similar repository instruction files
before editing. If your application uses goldr, give the agent the framework
rules in the app repository, not only in chat.

If your agent supports installable skills, you can also install the
[Goldr App skill](../skills/goldr/SKILL.md). The skill is self-contained
and gives agents an operational workflow for editing Goldr applications.

Copy this into your application's `AGENTS.md` and adjust the command list to
match your project scripts:

````md
# Goldr Application Rules

This project uses goldr.

Goldr is server-first, HTML-first, HTMX-native, and Go-native.

## Framework Boundaries

- Keep the application server-first and HTML-first.
- Do not add SPA routing, hydration, virtual DOM, or client-state frameworks.
- Do not hide HTMX behind custom framework components.
- Keep `hx-*` attributes visible in `.templ` files.
- The application owns `net/http` setup, middleware, auth, sessions,
  persistence, validation, static asset serving, cache policy, and deployment.
- Prefer ordinary Go, `net/http`, templ templates, and explicit handlers.
- Prefer existing project patterns before adding new abstractions.

## Goldr File Conventions

- Routes live under `app/routes`.
- Non-live reusable Kit route subtrees live under `app/mounts` and must be
  mounted by real `app/routes` owners.
- Use `KitRouteMount.Routes` when one live owner exposes only part of a
  mounted subtree. A child-only selection keeps the owner mount-base URL helper
  for binding mounted helpers, but does not make the root URL live.
- Static assets should not live under `app/routes`.
- A render unit is normally a `.go` file beside a matching `.templ` file.
- `route.go` declares route pages, fragments, and actions with
  `goldr.RouteDef`, `goldr.KitRouteDef`, or `goldr.KitRouteMount`.
- `layout.go` defines a layout and accepts `goldr.LayoutContext`.
- Fragment declarations return `goldr.FragmentRouteResponse`.
- Action declarations use ordinary HTTP mutation handlers.
- Do not put `page.go`, `frag_<name>.go`, or `actions.go` under `app/routes`.
- Use `by_<name>/` directories for path parameters.
- Read path parameters with `r.PathValue("<name>")`.
- If an HTMX action or fragment only supports one page workflow, usually nest
  it under that page route instead of making it a top-level sibling route.
- Nested action or fragment directories do not need standalone pages.
- Keep one-route templates directly in the route directory. Use `internal`
  packages, shared packages, or Kit routes only for real reuse across sibling
  routes or route trees.
- Choose route directory names for generated helper readability. Prefer
  `user_events/prepare -> UserEvents.Prepare` over redundant names such as
  `prepare_user_event`.
- Do not hand-edit Goldr-owned `goldr_gen.go` files,
  `app/urls/goldr_gen.go`, or generated mount helper files under
  `app/mounts`.

## Normal Change Loop

After route or template changes, run:

```bash
go tool goldr generate
go tool goldr check
go tool goldr routes list
go test ./...
```

`goldr generate` runs templ generation when `.templ` files exist before
writing goldr-owned generated files. When `assets/build` exists, it also
refreshes Goldr-managed asset outputs. `goldr check` runs templ check mode when
`.templ` files exist and validates generated output without rewriting files.
Use `goldr routes list --app-root <dir>` when the Goldr app root is nested, and
use the `HELPER` column to confirm route ownership after route refactors.

If this app has project-specific scripts, use those scripts instead of the raw
commands above.

## Static Assets

If the app uses Goldr asset fingerprinting, `go tool goldr generate` handles
the normal asset refresh after the app-owned tool writes `assets/build`. Use
asset-only commands when you need a narrower check or cleanup:

```bash
go tool goldr assets dist
go tool goldr assets check
```

Goldr fingerprints final files from `assets/build` into `assets/dist`. It does
not compile Tailwind, bundle JavaScript, minify files, optimize images, upload
to a CDN, or register static handlers.

## Before Making Broad Changes

- Inspect the existing `app/routes` tree before adding routes.
- Keep page, layout, fragment, and action behavior local to the route directory.
- Keep generated files current.
- Do not introduce a second routing system.
- Do not move application-owned server concerns into goldr framework code.
````

If the application already has an `AGENTS.md`, merge the goldr section into the
existing file instead of replacing project-specific rules.
