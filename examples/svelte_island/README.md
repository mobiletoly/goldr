# Svelte Island Example

This example mounts one Svelte 5 editor inside a Goldr page. Goldr owns the
page shell, filesystem routes, Save endpoint, CSRF, navigation, and assets.
Svelte owns only the descendants of the explicit editor root.

Build the app-owned frontend and refresh Goldr output:

```bash
npm ci
npm run check
npm run build
go tool templ generate
go tool goldr generate
```

Run the example:

```bash
go run .
```

Then open the printed URL. The editor saves inline through JSON, while Cancel
is an ordinary link enhanced by HTMX. The app-owned bridge unmounts Svelte
before HTMX replaces the page and mounts a fresh editor after Back navigation.

Check committed Go and Goldr output:

```bash
go tool goldr generate --check
go tool goldr check
go test ./...
```

From the repository root, `scripts/check-client-islands.sh` also rebuilds the
frontend into temporary output and runs the Playwright lifecycle scenario.
