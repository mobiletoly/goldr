# Full-Feature Example

This is the runnable goldr v0 large-app composition example. It keeps the app
small, but shows how pages, layouts, fragments, actions, forms, URL helpers,
custom errors, middleware, and static assets fit together in one Go/HTMX app.

Run:

```bash
go run ./examples/full_feature
```

Then open the printed localhost URL in a browser.

Useful paths:

```text
/
/settings
/users
/users/42
/users/frag-table
/users/save-preview
/users/create
```

Use a custom address when needed:

```bash
go run ./examples/full_feature -addr 127.0.0.1:0
```

Inspect the route surface from the repository root:

```bash
go run ./cmd/goldr routes list --root examples/full_feature
go run ./cmd/goldr routes layouts --root examples/full_feature
go run ./cmd/goldr routes list --root examples/full_feature --json
go run ./cmd/goldr assets list --root examples/full_feature
```

Regenerate goldr-owned route and URL files from the repository root:

```bash
go run ./cmd/goldr generate --root examples/full_feature
```

Check that generated files are current without writing:

```bash
go run ./cmd/goldr generate --root examples/full_feature --check
```

Check the route tree, render-unit pairs, and generated-file freshness:

```bash
go run ./cmd/goldr check --root examples/full_feature
```

The example demonstrates root, settings, nested static, and dynamic page
runtime dispatch, page metadata passed explicitly to layouts, root-to-leaf
layout wrapping, a standalone users table fragment for HTMX swaps, generated
URL helpers from `app/urls`, generated action routes that set HTMX response
headers, a minimal add-contact form with server-side redisplay errors, and a
route-rendered custom 404 page. The app shell uses page metadata for document
title, description, canonical path, and active navigation. The `/users` and
`/users/42` pages share the users section shell from `users/layout.templ`;
`/users/frag-table` renders only the fragment partial.

The example fingerprints final static files from
`examples/full_feature/assets/build/` into `assets/dist/` with
`goldr assets dist`. The generated `assets` package provides `assets.Path` for
the root layout and `assets.FS()` for the app-owned `/assets/` handler. The
example includes browser-ready CSS and JavaScript under `assets/build`; Goldr
fingerprints both without owning a CSS or JavaScript pipeline. The asset
handler is registered before generated routes and sets immutable cache headers
in application code. Generated route dispatch is wrapped with a tiny app-owned
middleware that sets `X-Content-Type-Options: nosniff`.

For the broader asset workflow, read `docs/user/assets.md`.

Post to `/users/save-preview` to see `HX-Trigger`, `HX-Retarget`, and
`HX-Reswap` response headers from `users.PostSavePreview` in
`app/routes/users/actions.go`.

Post to `/users/create` with URL-encoded `name` and `status` fields to see
form parsing, field-error redisplay, and successful HTMX replacement from
`users.PostCreate` in `app/routes/users/actions.go`.
