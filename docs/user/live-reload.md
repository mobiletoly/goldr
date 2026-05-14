# Live Reload

`goldr dev` runs a local live-reload loop for goldr applications.

It uses templ watch mode for the reload loop: templ generation, app restart,
the browser proxy, and browser reload events. Goldr configures the watch
patterns for goldr apps and adds the goldr-specific steps around that loop:
route generation and production-faithful asset fingerprinting.

Start it from an application module:

```bash
go tool goldr dev
```

Open the printed proxy URL. By default, the proxy listens at
`http://127.0.0.1:7331` and forwards to the app server at
`http://127.0.0.1:8080`.

## App Command

The app still owns its HTTP server. By default, `goldr dev` starts the app with:

```bash
go run .
```

Use `--cmd` if your app starts somewhere else:

```bash
go tool goldr dev --cmd "go run ./cmd/web"
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
  -> goldr assets dist
  -> restart app
  -> reload browser
```

Goldr does not replace templ's proxy or reload mechanism. It asks templ to
watch `.go`, `.templ`, and `assets/build` files, then creates a temporary
wrapper command for templ to run after generation. The wrapper runs goldr
generation, updates fingerprinted assets when `assets/build` exists, then
starts the app.

## Options

Defaults:

```text
--root .
--app-url http://127.0.0.1:8080
--proxy-addr 127.0.0.1:7331
--cmd "go run ."
```

Use `--root` when running from outside the app root:

```bash
go tool goldr dev --root examples/full_feature
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
`assets/build` changes, Goldr runs `goldr assets dist`, restarts the app, and
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
