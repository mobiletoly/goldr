# CLI Architecture

goldr uses `github.com/urfave/cli/v3` for CLI command parsing, help output, and command dispatch.

The CLI is structured in two layers:

- `cmd/goldr`
  Process entrypoint only. It wires process arguments, stdout, stderr, context, and exit code handling.

- `internal/goldrcli`
  Internal CLI construction and behavior. It owns the urfave command tree and command tests.

This keeps the framework public API small.

Do not expose CLI construction as a public package.

Future commands should be added to `internal/goldrcli` unless a later spec creates a more specific internal package boundary.

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

`goldr init` treats `--root` as the application root. It resolves the root
directory, derives the module-aware route import path with the same Go module
logic used by `goldr generate`, and fails before writing if `<root>/app`
already exists as a file, directory, or symlink.

The generated route and URL helper files are produced through the same
`internal/wiring` generator path as `goldr generate`. The starter page and
layout source files are the only handwritten template files owned by init.
