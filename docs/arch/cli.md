# CLI Architecture

goldr uses `github.com/urfave/cli/v3` for CLI command parsing, help output, and command dispatch.

The CLI is structured in two layers:

- `cmd/goldr`
  Process entrypoint only. It wires process arguments, stdout, stderr, context, and exit code handling.

- `internal/goldrcli`
  Internal CLI construction and behavior. It owns the urfave command tree and command tests.

This keeps the framework public API small.

Do not expose CLI construction as a public package.

Top-level commands live in focused internal command packages and are assembled
by `internal/goldrcli`:

```text
internal/goldrcli/initcmd
internal/goldrcli/dev
internal/goldrcli/check
internal/goldrcli/generate
internal/goldrcli/assets
internal/goldrcli/routes
```

Command packages expose `Command() *cli.Command`; the root package should not
mix package-owned commands with root-local command factories. Shared app-root
and generated-file plumbing lives in `internal/goldrcli/project`. Shared templ
tool lookup and generation lives in `internal/goldrcli/templtool`. Do not add a
Go workspace or nested module for CLI organization.

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
app/routes/page.go
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
`internal/wiring` generator path as `goldr generate`. The starter page and
layout source files are the only handwritten template files owned by init.
