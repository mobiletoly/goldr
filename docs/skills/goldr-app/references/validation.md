# Validation Playbook For Goldr Apps

Use this reference before finalizing non-trivial Goldr app changes.

## Start With Current App Shape

Before editing, inspect:

```bash
find app/routes -maxdepth 4 -type f | sort 2>/dev/null
rg -n "routes.Handler|HandlerWithOptions|browser.Handler|assets.FS|goldr dev|GOLDR_" .
```

Use `rg` or project scripts instead of assuming the app follows an example
exactly.

If `app/routes` or Goldr tooling is missing because the task is to add Goldr to
the project, read `project-setup.md` before making source edits.

## Generated Output

Do not patch generated files directly. After app-owned route or template
changes, regenerate:

```bash
go tool goldr generate
```

Then check:

```bash
go tool goldr check
```

If the app has `assets/build`, `go tool goldr generate` also refreshes
Goldr-managed asset output.

## Tests

Run the smallest relevant test first, then broaden as needed:

```bash
go test ./path/to/package
go test ./...
```

If the app has wrappers such as `make test`, `just test`, or `go test` scripts,
prefer the project wrapper when it is clearly the maintained path.

## Browser Checks

For browser-visible app changes, run the app and verify the affected route in a
browser. Use `go tool goldr dev` when live reload matters; otherwise use the
app's normal local command.

If using `goldr dev`, open the proxy URL printed by `goldr dev`, not the app
server URL.

Stop any server you start.

## Template Inspection Checks

Use template inspection only when render ownership is unclear. It can show
which app layout, page, or fragment produced a visible region.

Do not assert against inspection comments or overlay DOM in application tests
unless the application explicitly owns that behavior for local tooling.

## Dirty Worktree Discipline

Preserve unrelated dirty files. If generated files are already dirty, inspect
whether they are related to your source edit before regenerating. Do not revert
user changes unless explicitly asked.

## Final Sanity Scan

Before finishing:

```bash
git diff --check
git status --short
```

Report:

- files changed
- validation run
- any tests not run
- any server started and stopped
- any unrelated dirty files noticed
