# CLI

The `goldr` command initializes, validates, generates, and inspects goldr app
route trees.

In an application module, prefer the app-local tool form:

```bash
go tool goldr <command>
```

The usage text still shows the underlying command name, such as
`goldr generate`.

Use `--app-root` when running a command from outside the application root:

```bash
go tool goldr check --app-root ../my-app
```

`--app-root` points to the application root. goldr reads `<root>/app/routes`,
expands any mounted route surfaces from `<root>/app/mounts`, and writes
generated files under `<root>/app/routes`, `<root>/app/urls`, and referenced
mount roots.

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
app/routes/route.go
app/routes/page.templ
app/routes/layout.go
app/routes/layout.templ
app/routes/goldr_gen.go
app/internal/goldrinspect/goldr_gen.go
app/urls/goldr_gen.go
```

It fails if `app` already exists. It does not create `go.mod`, edit `go.mod`,
write `main.go`, run templ, or start a server.

Use `--app-root` from outside the application root:

```bash
go tool goldr init --app-root ./hello-goldr
```

For a manual first-app walkthrough, read [Getting Started](getting-started.md).

## Generate

`goldr generate` runs templ generation when `.templ` files exist, scans
`app/routes`, expands mounted surfaces from `app/mounts`, writes route and URL
helper files, and refreshes fingerprinted assets when `assets/build` exists:

```bash
go tool goldr generate
```

Generated files:

```text
app/routes/goldr_gen.go
app/routes/**/goldr_gen.go when route packages need generated helpers
app/internal/goldrinspect/goldr_gen.go
app/urls/goldr_gen.go
app/mounts/<mount>/goldr_gen.go for referenced Kit mount subtrees
assets/goldr_assets_gen.go when assets/build exists
```

Use `--check` in CI or before committing generated files:

```bash
go tool goldr generate --check
go tool goldr generate --app-root ../my-app --check
```

Check mode runs templ check mode when `.templ` files exist, compares
goldr-generated output with files on disk, checks Goldr-managed asset output
when `assets/build` exists, reports stale or missing generated files, and exits
non-zero without writing.

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

`--cmd` runs from the Goldr app root by default. Use `--cmd-dir` when the app
root is nested but the server command should run from another directory.
Relative `--cmd-dir` paths are resolved from the directory where you invoked
`goldr dev`:

```bash
go tool goldr dev --app-root internal/adapters/webapp --cmd-dir . --cmd './scripts/run-goldr-dev-app.sh'
```

## Check

`goldr check` validates the route tree and generated-file freshness without
writing files. When Goldr-managed asset output exists, it also validates asset
freshness:

```bash
go tool goldr check
go tool goldr check --app-root ../my-app
```

It checks:

- `app/routes` naming and action conventions
- mounted `app/mounts` route-surface validity when mounts are referenced
- page handler signatures
- layout and fragment `.go` / `.templ` pairs
- route dispatch generation readiness
- URL helper generation readiness
- freshness of templ-generated files when `.templ` files exist
- freshness of Goldr-owned `goldr_gen.go` files and `app/urls/goldr_gen.go`
- freshness of `assets/dist`, `assets/goldr_assets_gen.go`, and
  `assets/.goldr/assets.json` when Goldr-managed asset output exists

It prints nothing and exits `0` when the app is clean. It reports diagnostics
to stderr and exits non-zero when validation fails. It requires the app-local
templ tool added with `go get -tool github.com/a-h/templ/cmd/templ@v0.3.1020`
only when `.templ` files exist.

Diagnostic categories:

| Code | Meaning |
| --- | --- |
| `GOLDR001` | App root, Go module, or `app/routes` resolution failed. |
| `GOLDR002` | Route tree, route naming, fragment naming, action convention, or route scan failed. |
| `GOLDR003` | A page handler signature is invalid, or a layout or fragment is missing its matching `.templ` file. |
| `GOLDR004` | Generated route dispatch is not ready. |
| `GOLDR005` | Generated URL helpers are not ready. |
| `GOLDR006` | A goldr-owned generated file is missing or stale. |
| `GOLDR007` | templ is unavailable, or a templ-generated file is missing or stale. |
| `GOLDR008` | Goldr-managed asset output is missing or stale. |

Examples:

```text
app/routes/Users: GOLDR002 static route directories must use lowercase Go-safe names
app/routes/page.go: GOLDR002 route surface belongs in route.go
GOLDR006 app/routes/goldr_gen.go is stale
GOLDR007 templ generated files are not up to date; run go tool goldr generate
GOLDR008 Goldr-managed assets are not current; run go tool goldr generate
```

`goldr check` runs templ check mode when `.templ` files exist and asset check
mode when asset output is present. It does not write templ output, write asset
output, run tests, or start the application server.

For nested Goldr app roots, prefer `goldr generate --app-root <dir>` and
`goldr check --app-root <dir>` over raw `go tool templ generate` from a
different directory. Templ records source paths in generated `*_templ.go` files
relative to the generation root, so raw repo-root templ generation can churn
`FileName` metadata for nested Goldr routes.

## Assets

`goldr assets` fingerprints final static files that your app already built.
It does not compile CSS, bundle JavaScript, upload to a CDN, or register static
handlers. For the full workflow, read [Assets](assets.md).

`goldr generate` refreshes fingerprinted assets automatically when
`assets/build` exists. Use `assets dist` when you want the asset-only step:

```bash
go tool goldr assets dist
go tool goldr assets dist --app-root ../my-app
```

Verify asset output without writing:

```bash
go tool goldr assets check
```

Remove stale Goldr-managed fingerprinted files without rebuilding current
outputs:

```bash
go tool goldr assets clean
```

List the current manifest:

```bash
go tool goldr assets list
go tool goldr assets list --json
```

`goldr check` includes asset freshness when Goldr-managed asset output exists.
Run `go tool goldr assets check` when you want the asset-only check.

## Routes List

`goldr routes list` prints the route surface goldr sees. During route
refactors, use it before and after generation to inspect browser paths and
generated helper names together:

```bash
go tool goldr routes list
go tool goldr routes list --app-root ../my-app
go tool goldr routes list --mount reports
```

Columns:

```text
KIND    METHOD    PATH    PARAMS    SOURCE    OWNER    DECL    NAME    TITLE    LABELS    HELPER
```

When `--mount` is used and mounted-route selection status is available, the
table also includes `STATUS` with `included` or `excluded`.

Rows include pages, layouts, fragments, and actions. Pages and fragments show
`GET,HEAD`. Actions show their HTTP method. Layouts show `METHOD`, `OWNER`,
and `HELPER` as `-`.

The `HELPER` column is useful when naming nested route directories. A
route-local workflow such as `notifications/pending_events/send` should produce
a helper like:

```text
urls.Notifications.PendingEvents.Send.Path()
```

If the helper repeats parent context, such as
`urls.Notifications.SendPendingEvents.Path()`, the route tree may still be too
flat for the workflow ownership you intended.

For route declaration rows, `DECL` is `local` for `goldr.RouteDef` routes,
`kit` for `goldr.KitRouteDef[K]` routes, and `mounted-kit` for routes expanded
from `app/mounts`. `NAME`, `TITLE`, and `LABELS` come from the static
declaration metadata. Labels are shown as sorted `key="value"` pairs. Goldr
displays this metadata opaquely; it does not treat labels as auth, navigation,
tenant, portal, or policy configuration. `HELPER` stays path-derived and does
not use declaration names or labels. For mounted rows, `OWNER` is the live
mount owner under `app/routes`.

Use `--mount <path>` to filter the existing inventory to one mounted subtree.
The filter includes selected live endpoints and explicitly excluded mounted
children, which is useful for checking which live owners expose a shared mount.

Use `--json` for machine-readable output:

```bash
go tool goldr routes list --json
go tool goldr routes list --app-root ../my-app --json
go tool goldr routes list --mount reports --json
```

JSON rows include `kind`, `methods`, `path`, `params`, `source`, and `helper`.
Mounted selection rows include `status` when selection status is present.
Declaration-backed endpoint rows also include a `declaration` object with
`source`, `kind`, `name`, `title`, sorted `labels`, optional `nav_trails`,
optional `destinations`, and the page, fragment, or action implementation
evidence Goldr parsed from `route.go`. Mounted route declarations include a
`mount` object with the mount path and live owner.
Fragment and
action declaration evidence includes `index` when the endpoint uses the route
directory path itself. Kit declarations also include static kit type and
constructor. Layout rows omit `declaration`. JSON arrays are emitted as empty
arrays when no values exist.

## Routes Layouts

`goldr routes layouts` prints the layout inheritance map:

```bash
go tool goldr routes layouts
go tool goldr routes layouts --app-root ../my-app
```

The output shows where layouts start, which pages inherit them, which actions
can use the same stack with `goldr.WriteRouteResponse`, and which fragments are
not layout-wrapped.

When stdout is a terminal, the command adds restrained styles. Piped or
redirected output stays plain, and styles are disabled when `NO_COLOR` is set
or `TERM=dumb`.

## Routes Explain

`goldr routes explain` explains one browser URL or HTMX request path:

```bash
go tool goldr routes explain /users/7
go tool goldr routes explain --app-root ../my-app http://127.0.0.1:8080/users/7
go tool goldr routes explain --app-root ../my-app --method POST /users/create
```

It accepts full URLs and absolute paths. Query strings and fragments are
ignored for matching. The default method is `GET`; pass `--method` when
debugging an action or method mismatch.

Matched page routes show the route pattern, source file, dynamic params, and
layout stack. Matched actions show the layout stack available to
`goldr.WriteRouteResponse`. Fragments are reported as not layout-wrapped.
For routes declared in `route.go`, matched output also shows `DECLARATION` and
`IMPLEMENTATION` sections with the static declaration metadata and generated
adapter evidence. These sections are read-only inspection output; they do not
execute handlers or change route matching.

When a matched declaration has allowed navigation trail keys, `routes explain`
prints a `trails` row. When it declares destinations, it also prints a
`DESTINATIONS` section with the destination name, generated helper path, target
route helper, and selected nav trail key when present.

If a path exists but the method is wrong, the command exits non-zero and prints
the allowed methods. If no generated route matches the path, it exits non-zero
with a no-route diagnostic.

## Unknown Commands

Unknown commands print an error, print help, and exit with status code `2`.
