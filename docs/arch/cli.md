# CLI Architecture

goldr uses `github.com/urfave/cli/v3` for CLI command parsing, help output, and command dispatch.

The CLI is a nested Go tool module:

```text
cmd/goldr/go.mod
```

The module path is `github.com/mobiletoly/goldr/cmd/goldr`, so downstream
applications can declare `tool github.com/mobiletoly/goldr/cmd/goldr` and run
the command as `go tool goldr`.

The CLI is structured in two layers:

- `cmd/goldr`
  Process entrypoint only. It wires process arguments, stdout, stderr, context, and exit code handling.

- `cmd/goldr/internal/goldrcli`
  Internal CLI construction and behavior. It owns the urfave command tree and command tests.

This keeps the framework public API small.

Do not expose CLI construction as a public package.

Top-level commands live in focused internal command packages and are assembled
by `cmd/goldr/internal/goldrcli`:

```text
cmd/goldr/internal/goldrcli/initcmd
cmd/goldr/internal/goldrcli/dev
cmd/goldr/internal/goldrcli/check
cmd/goldr/internal/goldrcli/generate
cmd/goldr/internal/goldrcli/assets
cmd/goldr/internal/goldrcli/routes
```

Command packages expose `Command() *cli.Command`; the root package should not
mix package-owned commands with root-local command factories. Shared app-root
and generated-file plumbing lives in `cmd/goldr/internal/goldrcli/project`.
Shared templ tool lookup and generation lives in
`cmd/goldr/internal/goldrcli/templtool`.

The CLI must keep using standard process behavior:
- successful commands exit `0`
- usage or command errors use nonzero exit codes
- user-facing output goes to stdout
- errors and usage failures go to stderr

The current version value is `dev` until a later release/versioning spec defines build-time version injection.

## Starter App Initialization

`goldr init` initializes goldr's app structure inside an existing Go module.

It owns only the `app/` starter tree and goldr-generated files:

```text
app/routes/route.go
app/routes/page.templ
app/routes/layout.go
app/routes/layout.templ
app/routes/goldr_gen.go
app/internal/goldrinspect/goldr_gen.go
app/urls/goldr_gen.go
```

It must not create a project directory, create `go.mod`, edit `go.mod`, write
`main.go`, run templ generation, run tests, or start an application server.
Those bootstrap steps belong in user documentation and ordinary Go tooling.

`goldr init` treats `--app-root` as the application root. It resolves the root
directory, derives the module-aware route import path with the same Go module
logic used by `goldr generate`, and fails before writing if `<root>/app`
already exists as a file, directory, or symlink.

The generated route and URL helper files are produced through the same
`cmd/goldr/internal/wiring` generator path as `goldr generate`. The starter
page and layout source files are the only handwritten template files owned by
init.

## Route Inspection Commands

`goldr routes list`, `goldr routes layouts`, and `goldr routes explain` are
read-only inspection commands. They scan the application route tree, build the
same manifest and wiring models used by generation, and render CLI output from
those models.

Route inspection commands must not write generated files, execute application
handlers, register runtime routes, use reflection over a running application,
or persist a second route registry. They are views over filesystem-owned route
declarations, layouts, fragments, actions, generated adapter names, and URL
helper readiness.

`goldr routes list` owns human table output and JSON formatting for the route
surface. Declaration metadata shown by the command is static parse evidence:
declaration kind, source, name, title, sorted opaque labels, kit expressions,
and the parsed page, fragment, or action implementation. URL helper names stay
derived from the route path.

`goldr routes explain` owns full-URL path extraction, method selection, and
human rendering for one matched route. For `route.go` matches it renders
declaration and implementation sections before the layout stack. It does not
define JSON output.

`goldr check` remains the readiness gate. It validates route scanning, route
declaration parsing, runtime route generation, URL helper generation,
generated-file freshness, templ output, and managed assets through existing
diagnostic codes. It does not call route inspection commands or interpret
route declaration metadata as policy.
