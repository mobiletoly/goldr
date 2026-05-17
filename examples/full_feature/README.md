# Full-Feature Example

This is the runnable goldr v0 large-app composition example. It keeps the app
small, but shows how pages, layouts, fragments, actions, forms, URL helpers,
custom errors, middleware, CSRF, and static assets fit together in one
Go/HTMX app.

Run:

```bash
go run ./examples/full_feature
```

Then open the printed localhost URL in a browser.

Useful paths:

```text
/
/settings
/protected-resource-demo
/sign-in
/admin
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
headers, a multipart add-contact form with server-side redisplay errors, a
protected-resource demo with a CSRF-protected sign-out action, a protected page
that returns redirects and forbidden status responses from its page handler,
and a route-rendered custom 404 page. The app shell uses page metadata for
document title, description, canonical path, and active navigation. The `/users`
and `/users/42` pages share the users section shell from `users/layout.templ`;
`/users/frag-table` renders only the fragment partial.

Generated route dispatch is wrapped with app-owned middleware for security
headers and a signed-cookie CSRF guard. The users form renders a visible hidden
CSRF field, and unsafe actions validate the submitted token before mutating
example state.

Open `/protected-resource-demo` to sign in as a demo admin or member, sign out,
and open the protected admin page. Opening `/admin` without a demo role returns
a page-level `303 See Other` redirect to `/sign-in?next=/admin`; signing in as
member returns to `/admin` and shows a `403 Forbidden` page rendered through the
normal layout, while signing in as admin renders the protected page. Choosing
Unknown Credentials on `/sign-in` keeps the user on the sign-in page with a
visible error. Open `/admin?demo_error=1` to see a page delegate an unexpected
application error through `goldr.Error(err)` and the generated internal error
path.

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

In a browser, the `/users` page receives the signed `goldr_csrf` cookie and
renders the matching hidden token before HTMX submits unsafe requests. Manual
POST clients must do the same setup first: load `/users`, preserve the
`goldr_csrf` cookie, and reuse the rendered CSRF token.

Post to `/users/save-preview` with the `goldr_csrf` cookie and matching
`X-CSRF-Token` header to see `HX-Trigger`, `HX-Retarget`, and `HX-Reswap`
response headers from `users.PostSavePreview` in
`app/routes/users/actions.go`.

Post to `/users/create` with the `goldr_csrf` cookie, multipart `name`,
`status`, optional `avatar`, and matching `csrf_token` fields to see
`hx-encoding="multipart/form-data"`, CSRF validation, form parsing, app-owned
request-size limiting, field-error redisplay with `422`, optional upload
filename display, and successful HTMX replacement from `users.PostCreate` in
`app/routes/users/actions.go`.
