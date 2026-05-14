# CLI

The `goldr` command initializes, validates, generates, and inspects goldr app
route trees.

In an application module, prefer the app-local tool form:

```bash
go tool goldr <command>
```

The usage text still shows the underlying command name, such as
`goldr generate`.

Use `--root` when running a command from outside the application root:

```bash
go tool goldr check --root examples/full_feature
```

`--root` points to the application root. goldr reads `<root>/app/routes` and
writes generated files under `<root>/app/routes` and `<root>/app/urls`.

## Help And Version

Print help:

```bash
go tool goldr
go tool goldr help
go tool goldr --help
go tool goldr -h
```

Print the version:

```bash
go tool goldr version
go tool goldr --version
go tool goldr -version
```

During v0 development, local builds print:

```text
goldr dev
```

## Init

`goldr init` creates the minimal route skeleton for an existing Go module:

```bash
go tool goldr init
```

It creates:

```text
app/routes/page.go
app/routes/page.templ
app/routes/layout.go
app/routes/layout.templ
app/routes/goldr_gen.go
app/urls/goldr_gen.go
```

It fails if `app` already exists. It does not create `go.mod`, edit `go.mod`,
write `main.go`, run templ, or start a server.

Use `--root` from outside the application root:

```bash
go tool goldr init --root ./hello-goldr
```

For a manual first-app walkthrough, read [Getting Started](getting-started.md).

## Generate

`goldr generate` scans `app/routes` and writes goldr-owned generated files:

```bash
go tool goldr generate
```

Generated files:

```text
app/routes/goldr_gen.go
app/urls/goldr_gen.go
```

Use `--check` in CI or before committing generated files:

```bash
go tool goldr generate --check
go tool goldr generate --root examples/full_feature --check
```

Check mode compares generated output with files on disk, reports stale or
missing generated files, and exits non-zero without writing.

`goldr generate` does not run templ generation. Run templ separately when
`.templ` files change:

```bash
go tool templ generate
go tool goldr generate
```

## Dev

`goldr dev` runs a local live-reload loop using templ watch mode:

```bash
go tool goldr dev
```

It keeps route generation, templ generation, production-faithful asset
fingerprinting, app restart, and browser reload moving together. For the full
workflow, read [Live Reload](live-reload.md).

Use flags when the app command, app URL, or proxy address differs from the
defaults:

```bash
go tool goldr dev --cmd "go run ./cmd/web" --app-url http://127.0.0.1:3000 --proxy-addr 127.0.0.1:7331
```

## Check

`goldr check` validates the route tree and generated-file freshness without
writing files:

```bash
go tool goldr check
go tool goldr check --root examples/full_feature
```

It checks:

- `app/routes` naming and action conventions
- page, layout, and fragment `.go` / `.templ` pairs
- route dispatch generation readiness
- URL helper generation readiness
- freshness of `app/routes/goldr_gen.go` and `app/urls/goldr_gen.go`

It prints nothing and exits `0` when the app is clean. It reports diagnostics
to stderr and exits non-zero when validation fails.

Diagnostic categories:

| Code | Meaning |
| --- | --- |
| `GOLDR001` | App root, Go module, or `app/routes` resolution failed. |
| `GOLDR002` | Route tree, route naming, fragment naming, action convention, or route scan failed. |
| `GOLDR003` | A page, layout, or fragment is missing its matching `.templ` file. |
| `GOLDR004` | Generated route dispatch is not ready. |
| `GOLDR005` | Generated URL helpers are not ready. |
| `GOLDR006` | A goldr-owned generated file is missing or stale. |

Examples:

```text
app/routes/Users: GOLDR002 static route directories must use lowercase Go-safe names
app/routes/page.go: GOLDR003 page /: missing matching .templ file
GOLDR006 app/routes/goldr_gen.go is stale
```

`goldr check` does not run templ generation, tests, or the application server.

## Assets

`goldr assets` fingerprints final static files that your app already built.
It does not compile CSS, bundle JavaScript, upload to a CDN, or register static
handlers. For the full workflow, read [Assets](assets.md).

Build fingerprinted files from `assets/build` into `assets/dist`:

```bash
go tool goldr assets dist
go tool goldr assets dist --root examples/full_feature
```

Verify asset output without writing:

```bash
go tool goldr assets check
```

Remove stale goldr-managed fingerprinted files:

```bash
go tool goldr assets clean
```

List the current manifest:

```bash
go tool goldr assets list
go tool goldr assets list --json
```

`goldr check` stays route-focused. Run `go tool goldr assets check`
explicitly in CI when fingerprinted assets are part of the app.

## Routes List

`goldr routes list` prints the route surface goldr sees:

```bash
go tool goldr routes list
go tool goldr routes list --root examples/full_feature
```

Columns:

```text
KIND    METHOD    PATH    PARAMS    SOURCE    HELPER
```

Rows include pages, layouts, fragments, and actions. Pages and fragments show
`GET,HEAD`. Actions show their HTTP method. Layouts show `METHOD` and `HELPER`
as `-`.

Use `--json` for machine-readable output:

```bash
go tool goldr routes list --json
go tool goldr routes list --root examples/full_feature --json
```

JSON rows include `kind`, `methods`, `path`, `params`, `source`, and `helper`.

## Routes Layouts

`goldr routes layouts` prints the layout inheritance map:

```bash
go tool goldr routes layouts
go tool goldr routes layouts --root examples/full_feature
```

The output shows where layouts start, which pages inherit them, and which
fragments and actions are not layout-wrapped.

When stdout is a terminal, the command adds restrained styles. Piped or
redirected output stays plain, and styles are disabled when `NO_COLOR` is set
or `TERM=dumb`.

## Routes Explain

`goldr routes explain` explains one browser URL or HTMX request path:

```bash
go tool goldr routes explain /users/7
go tool goldr routes explain --root examples/full_feature http://127.0.0.1:8080/users/7
go tool goldr routes explain --root examples/full_feature --method POST /users/create
```

It accepts full URLs and absolute paths. Query strings and fragments are
ignored for matching. The default method is `GET`; pass `--method` when
debugging an action or method mismatch.

Matched page routes show the route pattern, source file, dynamic params, and
layout stack. Fragments and actions are reported as not layout-wrapped.

If a path exists but the method is wrong, the command exits non-zero and prints
the allowed methods. If no generated route matches the path, it exits non-zero
with a no-route diagnostic.

## Unknown Commands

Unknown commands print an error, print help, and exit with status code `2`.
