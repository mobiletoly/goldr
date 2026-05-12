# Assets

Goldr fingerprints final static files and generates a small Go manifest package
for templates and app-owned static handlers.

Goldr does not compile CSS, bundle JavaScript, minify files, upload to a CDN,
register handlers, or inject assets into layouts. Use whatever asset tools your
app needs first, then let Goldr fingerprint the files the browser will load.

## Directory Layout

Use `assets/build` for final browser-ready files:

```text
assets/
  src/                  optional app-owned source files
    app.css
  build/                final files goldr reads
    app.css
  dist/                 fingerprinted files goldr writes
    app.733637bd.css
  .goldr/
    assets.json         goldr-managed cleanup state
  goldr_assets_gen.go   generated Go manifest package
```

Only `assets/build` is input to Goldr. `assets/src` is optional and belongs to
your app or your asset tool.

## Build Fingerprinted Assets

After your app-owned asset tool writes final files into `assets/build`, run:

```bash
go tool goldr assets dist
```

From another directory, pass the app root:

```bash
go tool goldr assets dist --root examples/full_feature
```

Goldr copies each file into `assets/dist` with a content hash in the filename
and writes `assets/goldr_assets_gen.go`.

Example:

```text
assets/build/app.css -> assets/dist/app.733637bd.css
```

The logical asset name remains `app.css`.

## Use Assets In Templates

Import the generated `assets` package in the template that references the
asset:

```templ
package routes

import "myapp/assets"

templ Layout(child templ.Component) {
	<html>
		<head>
			<link rel="stylesheet" href={ assets.Path("app.css") }/>
		</head>
		<body>
			@child
		</body>
	</html>
}
```

`assets.Path("app.css")` returns the fingerprinted URL path, such as
`/assets/app.733637bd.css`. Unknown asset names panic, which is useful for
template bugs that should fail loudly during development.

Use `assets.Lookup(name)` when application code needs a non-panic check.

## Serve Assets

The application still owns the HTTP server and mux. Register the asset handler
before generated routes:

```go
package main

import (
	"net/http"

	"myapp/app/routes"
	"myapp/assets"
)

func handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/assets/", staticCache(
		http.StripPrefix("/assets/", http.FileServer(http.FS(assets.FS()))),
	))
	mux.Handle("/", routes.Handler())
	return mux
}
```

`assets.FS()` returns the generated embedded filesystem rooted at
`assets/dist`.

## Cache Headers

Fingerprinting makes long-lived cache headers safe for static assets:

```go
func staticCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		next.ServeHTTP(w, r)
	})
}
```

Apply immutable cache headers only to fingerprinted files under `/assets/`.
Do not apply them globally to pages, private fragments, dashboards, form
responses, or action handlers. Those responses usually need application-owned
private or no-store cache policy.

## Check, List, And Clean

Use `check` in CI or before committing:

```bash
go tool goldr assets check
```

`check` verifies that `assets/dist`, `assets/goldr_assets_gen.go`, and
`assets/.goldr/assets.json` match the current files in `assets/build`. It does
not write files.

List the current manifest:

```bash
go tool goldr assets list
go tool goldr assets list --json
```

Remove stale Goldr-managed dist files:

```bash
go tool goldr assets clean
```

`clean` is fail-closed. It deletes only stale files proven by
`assets/.goldr/assets.json`; it does not delete arbitrary files from
`assets/dist`.

## Tailwind Example

Goldr does not run Tailwind. Tailwind is just one possible app-owned step that
can write final CSS into `assets/build`.

Example source CSS:

```css
@import "tailwindcss";
```

Example direct CLI flow:

```bash
npx @tailwindcss/cli -i ./assets/src/app.css -o ./assets/build/app.css
go tool goldr assets dist
```

For local development, run the Tailwind CLI in watch mode if you want it to
keep `assets/build/app.css` current, and run `go tool goldr assets dist` when
you want fresh fingerprinted output.

Tailwind also publishes a standalone CLI for projects that do not want Node or
npm in the app. In either case, Goldr only sees the final CSS file in
`assets/build`.

## CI Flow

A typical CI sequence is:

```bash
# app-owned asset build step, if needed
npx @tailwindcss/cli -i ./assets/src/app.css -o ./assets/build/app.css

go tool goldr assets check
go tool goldr generate --check
go tool goldr check
go test ./...
```

If you use a different asset tool, replace only the first command. Keep
`go tool goldr assets check` after the tool that writes `assets/build`.

## What Goldr Does Not Do

Goldr assets intentionally avoids:

- Tailwind, Sass, Less, or TypeScript compilation
- JavaScript bundling
- minification
- image optimization
- source maps
- package manager integration
- CDN upload or invalidation
- object storage integration
- dev servers and hot reload
- automatic layout injection

The app owns those choices. Goldr only makes production-safe cached asset paths
boring.
