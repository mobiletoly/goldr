# Client Islands

Goldr applications can embed a React, Svelte, or other stateful client
component when one part of a page genuinely needs a rich client interaction.
Keep that component bounded. Goldr still owns the page, layout, filesystem
routes, navigation, and surrounding HTML.

## Ownership Boundary

Render an explicit, empty island root from a colocated templ file:

```templ
<div
    data-client-island="project-editor"
    data-project-name={ project.Name }
    data-save-url={ urls.Save.Path() }
></div>
```

The client framework may create and mutate descendants of that element. HTMX
and other app-owned scripts must not swap inside the framework-owned subtree.
Goldr and HTMX may replace an ancestor during page navigation after the client
framework has been unmounted.

The `data-*` marker and initial values are application conventions. Goldr does
not scan them, register components, generate JavaScript, or own client state.

## Mount And Unmount

Mount islands during initial loading and whenever HTMX processes newly inserted
HTML. Keep mounting idempotent and retain the framework's teardown handle.

These examples use HTMX 4.0 beta lifecycle names:

- `htmx.onLoad` discovers newly processed island roots.
- `htmx:before:cleanup` unmounts a registered island before HTMX removes it.
- `htmx.process(root)` runs after React or Svelte renders ordinary links that
  should participate in HTMX behavior.

React applications retain the value returned by `createRoot` and call
`root.unmount()`. Svelte 5 applications retain the value returned by `mount`
and pass it to `unmount`.

The lifecycle bridge belongs to the application. Keeping the small integration
visible makes ownership and cleanup behavior inspectable and avoids a hidden
Goldr client runtime.

## Saving State

A client island does not require JSON. Choose the simplest HTTP interaction
that fits the workflow:

- Use an ordinary form and redirect when Save completes the editing workflow.
- Use `fetch` with `FormData` or JSON for inline validation, autosave, or
  structured client state.
- Use an ordinary Goldr or `net/http` action in either case.

For unsafe `fetch` requests, render `csrf.Meta` in the layout, read
`meta[name="csrf-token"]`, and send the value in `X-CSRF-Token`. Keep the
signed CSRF cookie HttpOnly.

The paired examples use a bounded JSON `HTTPAction` because they demonstrate an
inline Save with field errors. This is application response policy, not a new
Goldr response type.

## Navigation

Render ordinary anchors inside an island. Do not add React Router or SvelteKit
routing beside Goldr filesystem routes.

When an application wants boosted navigation under HTMX 4, opt into inheritance
explicitly in visible layout markup:

```templ
<body hx-boost:inherited="true">
```

After the client framework renders an anchor, process the island root so HTMX
can attach the boosted behavior. A normal browser navigation remains the
fallback.

Applications with unsaved-change prompts or navigation guards own that policy.
Goldr does not synchronize island state with server-rendered navigation.

## What This Does Not Add

Client islands do not add:

- SPA routing or a client application shell
- hydration or server-rendered React/Svelte markup
- framework-owned browser state in Goldr
- a component registry or generated JavaScript
- a requirement that Goldr actions return JSON
- a Goldr-owned frontend compiler or bundler

If React or Svelte owns the entire page and routing system, Goldr is being used
as an HTTP backend rather than as an HTML-first frontend framework.

## Runnable Examples

- `examples/react_island` uses React 19, TypeScript, and Vite.
- `examples/svelte_island` uses Svelte 5, TypeScript, and Vite.

Both examples deliberately implement the small lifecycle bridge independently.
Run `scripts/check-client-islands.sh` from a Goldr checkout to reproduce their
frontend assets and qualify Save, boosted navigation, cleanup, and remounting
in Playwright Chromium. Normal GitHub Actions checks only their committed Go
and Goldr output.
