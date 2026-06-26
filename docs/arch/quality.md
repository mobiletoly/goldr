# Quality Gates

Use `scripts/check-all.sh` as the normal repository validation command.

The gate checks:
- Go formatting without rewriting files
- module tidiness
- package loading
- tests
- `go vet`
- race-enabled tests
- ASCII text
- trailing whitespace
- golangci-lint when installed
- gopls hint diagnostics

The ASCII text check is intentionally broad. The current exceptions are:
- the README GitHub star callout, where only U+2B50 is allowed
- Unicode box-drawing output for `goldr routes layouts`, including its
  renderer, expected-output test, user documentation, and implementation spec

Do not extend those exceptions for ordinary prose, comments, identifiers, or
generated files.

The linter policy is strict on correctness and moderate on style.

The golangci-lint configuration is explicit. It enables only selected linters and does not inherit default linter sets.

Current linter groups:
- core correctness: `govet`, `staticcheck`, `unused`, `ineffassign`
- error handling: `errcheck`, `errorlint`, `nilerr`, `nilnesserr`, `nilnil`
- Go-native modernization: `exptostd`, `usestdlibvars`, `unconvert`
- HTTP and context safety: `bodyclose`, `canonicalheader`, `noctx`
- hygiene: `asciicheck`, `bidichk`, `copyloopvar`, `durationcheck`, `makezero`, `misspell`, `nolintlint`
- test quality: `thelper`, `tparallel`

Do not add broad style or complexity linters unless they protect a real goldr contract.

Avoid enabling noisy linters just to look strict.

Good linter additions catch likely defects, dependency drift, unsafe HTTP behavior, unchecked errors, hidden complexity, or non-Go-native code.

`errcheck` remains enabled, but ordinary CLI terminal writes in `cmd/goldr/internal/goldrcli` may ignore `fmt.Fprint*` errors.

Do not use that exception for file writes, generated output, HTTP responses, subprocess pipes, or framework-controlled side effects.

gopls hint diagnostics are part of the normal gate because they catch current Go modernization opportunities that golangci-lint may not report.

This includes type-safe standard library updates such as replacing eligible `errors.As` usages with `errors.AsType`.

Optional checks:

```bash
GOLDR_RUN_GOVULNCHECK=1 scripts/check-all.sh
```

Use optional checks before releases and when dependency or security-sensitive code changes.

Fast local iteration may skip expensive checks:

```bash
GOLDR_SKIP_RACE=1 scripts/check-all.sh
```

Do not use skipped checks as final validation for completed work.

The gopls hint gate may be skipped only for local investigation:

```bash
GOLDR_SKIP_GOPLS_HINTS=1 scripts/check-all.sh
```
