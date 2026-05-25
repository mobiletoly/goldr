# Validation Playbook For Goldr Apps

Use this reference before finalizing non-trivial Goldr app changes.

## Start With Current App Shape

Before editing, inspect:

```bash
find app/routes -maxdepth 4 -type f | sort 2>/dev/null
rg -n "routes.Handler|HandlerWithOptions|browser.Handler|assets.FS|goldr dev|GOLDR_" .
```

If the Goldr app is nested, identify the app root first and pass it to Goldr
commands:

```bash
go tool goldr check --app-root <app-root>
go tool goldr routes list --app-root <app-root>
```

Use placeholders such as `<app-root>` only after inspecting the target app's
actual app root, command package, and scripts.

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

## Route Inspection

When route ownership, generated paths, URL helpers, layouts, middleware, or Kit
bindings changed, inspect the generated route surface:

```bash
go tool goldr routes list
go tool goldr routes list --json
go tool goldr routes layouts
go tool goldr routes explain /users
go tool goldr routes explain --method POST /users/create
```

Use `--app-root <path>` when needed. For Kit routes, check that `routes list`
or `routes explain` shows `kit` declaration rows and that the route source is
the filesystem route directory's `route.go`, not the shared implementation
package.

For mounted Kit route subtrees, also inspect the mounted surface directly:

```bash
go tool goldr routes list --mount reports
go tool goldr routes list --json
```

Mounted rows should show the mounted source under `app/mounts/<mount>` and the
live owner under `app/routes`. The live owner still controls final URL paths
and app URL helpers; the mounted subtree source only owns the reusable route
shape and mount-relative helper set.

For route refactors, compare the before and after `PATH` and `HELPER` values.
Route-local workflows should produce helpers that read like ownership, such as
`urls.Users.Prepare.Path()` or `urls.Notifications.PendingEvents.Send.Path()`,
not helpers that repeat parent context such as
`urls.Notifications.SendPendingEvents.Path()`.

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
