# Coding Agents

Many coding agents read `AGENTS.md` or similar repository instruction files
before editing. If your application uses goldr, give the agent the framework
rules in the app repository, not only in chat.

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
- Static assets should not live under `app/routes`.
- A render unit is normally a `.go` file beside a matching `.templ` file.
- `page.go` defines a page route and returns `goldr.Page`.
- `layout.go` defines a layout and accepts `goldr.LayoutContext`.
- `frag_<name>.go` and `frag_<name>.templ` define HTMX fragments.
- `actions.go` defines ordinary HTTP mutation handlers such as `PostCreate`.
- Use `by_<name>/` directories for path parameters.
- Read path parameters with `r.PathValue("<name>")`.
- Do not hand-edit `app/routes/goldr_gen.go` or `app/urls/goldr_gen.go`.

## Normal Change Loop

After route or template changes, run:

```bash
go tool templ generate
go tool goldr generate
go tool goldr check
go test ./...
```

If this app has project-specific scripts, use those scripts instead of the raw
commands above.

## Static Assets

If the app uses goldr asset fingerprinting:

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
