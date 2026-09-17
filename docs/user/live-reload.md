# Live Reload

`goldr dev` runs a local live-reload loop for Goldr applications.

It uses templ watch mode for the reload loop: templ generation, app restart,
the browser proxy, and browser reload events. Goldr configures the watch
patterns for Goldr apps and adds the Goldr-specific steps around that loop:
route generation and production-faithful asset fingerprinting.

Start it from an application module:

```bash
go tool goldr dev
```

Open the printed proxy URL. By default, the proxy listens at
`http://127.0.0.1:7331` and forwards to the app server at
`http://127.0.0.1:8080`.

`goldr dev` prints the proxy URL in a banner after generation and before the
app starts or restarts. Open the banner URL, not the app server URL.

## App Command

The app still owns its HTTP server. By default, `goldr dev` starts the app with:

```bash
go run .
```

Use `--cmd` if your app starts somewhere else:

```bash
go tool goldr dev --cmd "go run ./cmd/web"
```

By default, `--cmd` runs from `--app-root`. Use `--cmd-dir` when your Goldr app
root is nested but the executable, config files, or local scripts live in
another directory:

```bash
go tool goldr dev \
  --app-root internal/adapters/webapp \
  --cmd-dir . \
  --cmd './scripts/run-goldr-dev-app.sh'
```

## How It Works

`goldr dev` keeps development close to production behavior:

```text
.go or .templ change
  -> templ generate
  -> goldr generate
  -> restart app
  -> reload browser

assets/build change
  -> goldr generate
  -> restart app
  -> reload browser

--reload-path change
  -> notify templ proxy
  -> reload browser
  -> keep app process running
```

Goldr does not replace templ's proxy or reload mechanism. It asks templ to
watch `.go`, `.templ`, and `assets/build` files, then creates a temporary
wrapper command for templ to run after generation. The wrapper runs
`goldr generate`, which updates route files and fingerprinted assets when
`assets/build` exists, then starts the app.

## Reload-Only Content

Use repeatable `--reload-path` flags for files that the running application
reads on each request:

```bash
go tool goldr dev \
  --reload-path content/pages \
  --reload-path ../shared/legal \
  --cmd "go run . -dev"
```

Each path must already be a regular file or directory when `goldr dev` starts.
Relative paths resolve from the directory where the command is invoked, not
from `--app-root` or `--cmd-dir`. Directories are watched recursively. Exact
files are watched through their parent directory, so ordinary saves, atomic
replacement, removal, and recreation are detected. Newly created descendant
directories are also watched for directory targets. An exact-file target does
not recursively watch sibling directories under its parent.

Goldr coalesces a burst of create, write, remove, and rename events for 100 ms,
then posts one reload notification to templ's existing proxy. It does not run
templ generation, `goldr generate`, the app wrapper, or restart the Go process.
This preserves in-memory application state while the next request reads the
changed content.

Inside `--app-root`, templ's existing inputs remain authoritative. Changes to
`.go`, `.templ`, and files under `assets/build` use the normal generation and
restart flow and do not receive a second reload-only notification. Generated
outputs such as `goldr_gen.go`, `*_templ.go`, `assets/dist`, and
`assets/.goldr` are also excluded from reload-only notifications. Outside
`--app-root`, configured paths are reload-only regardless of extension. The
flag does not add source rebuild roots.

Do not use `--reload-path` for content embedded in the executable. Embedded
bytes change only when the application is rebuilt and restarted. Keep content
external when it must be editable and served by the current process.

A missing, inaccessible, or non-regular/non-directory path fails before templ
or the app starts. Removing or renaming a configured directory root stops the
development session with an error; removing and recreating an exact configured
file or a descendant remains supported. If one proxy notification cannot be
delivered after a short retry window, Goldr prints a warning and keeps
watching.

Templ injects the browser reload script through its proxy. The application is
responsible for a development Content Security Policy that permits that
same-origin script and event connection.

## Options

Defaults:

```text
--app-root .
--cmd-dir <app root>
--reload-path <none; repeat for each reload-only file or directory>
--app-url http://127.0.0.1:8080
--proxy-addr 127.0.0.1:7331
--cmd "go run ."
```

Use `--app-root` when running from outside the app root:

```bash
go tool goldr dev --app-root ../my-app
```

Use `--cmd-dir` when the command should run from a different directory than the
Goldr app root:

```bash
go tool goldr dev --app-root internal/adapters/webapp --cmd-dir .
```

Relative `--cmd-dir` paths are resolved from the directory where you invoked
`goldr dev`.

Relative `--reload-path` values use the same invocation-directory rule:

```bash
go tool goldr dev --reload-path content/pages --reload-path ../shared/legal
```

Use `--app-url` when the app listens on another address:

```bash
go tool goldr dev --app-url http://127.0.0.1:3000
```

Use `--proxy-addr` when the default proxy port is busy:

```bash
go tool goldr dev --proxy-addr 127.0.0.1:7332
```

## Assets

`goldr dev` uses the same asset path as production.

Templates keep using the generated asset package:

```templ
<link rel="stylesheet" href={ assets.Path("app.css") }/>
```

The app keeps serving fingerprinted files:

```go
mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets.FS()))))
```

The development flow is:

```text
assets/src        optional source files owned by the app
assets/build      final browser-ready files written by app-owned tools
assets/dist       fingerprinted files written by goldr assets dist
```

`goldr dev` watches `assets/build`, not `assets/src`. When a file under
`assets/build` changes, Goldr runs `goldr generate`, restarts the app, and
reloads the browser.

## Regular Assets Workflow

If your CSS, images, or JavaScript files are already browser-ready, write them
directly under `assets/build`:

```text
assets/build/app.css
assets/build/logo.svg
assets/build/app.js
```

Run:

```bash
go tool goldr dev
```

When you edit a file under `assets/build`, Goldr refreshes `assets/dist` and
reloads the browser.

## Tailwind Workflow

Goldr does not run Tailwind. Run Tailwind separately so it writes final CSS into
`assets/build`.

Use two terminals:

```bash
npx @tailwindcss/cli -i ./assets/src/app.css -o ./assets/build/app.css --watch
```

```bash
go tool goldr dev
```

Tailwind watches `assets/src/app.css` and writes `assets/build/app.css`. Goldr
watches `assets/build/app.css`, fingerprints it into `assets/dist`, restarts
the app, and reloads the browser.

## What Goldr Does Not Do

`goldr dev` does not:

- compile Tailwind, Sass, Less, or TypeScript
- bundle JavaScript
- minify files
- serve `assets/build` directly
- inject assets into layouts
- add a browser runtime beyond templ's reload script

The app owns asset tools and the HTTP server. Goldr keeps route generation,
templ generation, fingerprinted assets, app restart, and browser reload moving
together during development.

## Nested App Roots

When a Goldr app root is nested inside a larger Go module, run Goldr commands
with `--app-root` instead of running raw templ generation from a different
directory:

```bash
go tool goldr generate --app-root internal/adapters/webapp
go tool goldr check --app-root internal/adapters/webapp
```

Templ records source paths in generated `*_templ.go` files relative to the
generation root. Running raw `go tool templ generate` from the repository root
can change those paths for nested Goldr routes and make generated files look
stale. `goldr generate --app-root` keeps templ generation, Goldr route
generation, and Goldr checks on the same app-root contract.
