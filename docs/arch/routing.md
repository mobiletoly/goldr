# Routing Scanner

`cmd/goldr/internal/routing` owns route filesystem scanning and route manifest assembly.

The scanner and manifest provide route data for framework internals without
performing rendering, handler registration, or code generation.

It is not runtime request routing.

## Page And Layout Model

goldr borrows the familiar filesystem page and layout model, not the runtime
semantics of JavaScript routers.

Pages are route endpoints. Layouts are route-subtree wrappers.

Generated runtime wiring keeps this model Go-native: route packages expose
route declaration adapters or convention functions, layouts return
`templ.Component`, params live on the request, and rendering remains
server-side HTML.

## Scanner Boundary

The scanner reads a route root and returns a deterministic tree of:

- pages
- layouts
- fragments
- actions
- route declarations
- middleware

It records matching `.templ` files when present.

It does not:
- parse arbitrary Go files
- parse templ files
- render HTML
- register HTTP handlers
- generate code
- assign fragment URLs

It parses only Goldr convention files that carry route meaning:
`route.go`, `layout.go`, and `middleware.go`.

## Route Root

User-facing applications use:

```text
app/routes/
```

Reusable mounted route subtrees use:

```text
app/mounts/
```

`app/mounts` is not a live route root. It is scanned only when a real
`app/routes` route declares `goldr.KitRouteMount[K]`.

The internal scanner accepts any root path so callers can pass a concrete
directory.

## Naming Rules

Static route directories must match:

```text
^[a-z][a-z0-9_]*$
```

Dynamic route directories must use:

```text
by_<param>/
```

`<param>` must use the same lowercase Go-safe identifier rule.

Static source identifiers map to browser path segments by replacing `_` with
`-`. For example, `build_info/` serves `/build-info`.

The scanner validates the forms goldr supports:
- lowercase static route directories
- `by_<param>/` dynamic route directories
- `route.go`
- `layout.go` and `layout.templ`
- `middleware.go`
- ordinary helper Go files
- `.templ` files
- Go test files and templ-generated Go files, which are ignored

The scanner rejects:
- underscore-prefixed route directories
- dot-prefixed route directories
- `testdata` route directories
- malformed dynamic route directories such as `by_/`
- route directories with uppercase letters or hyphens
- dot-prefixed Go route files
- underscore-prefixed Go route files
- `page.go`, `frag_*.go`, and `actions.go` anywhere under the route root

The scanner should enforce goldr's allowed forms directly. It should not grow
special defensive branches for unsupported conventions from other frameworks.

Invalid directories are reported and skipped.

Invalid files are reported and skipped.

Go test files and templ-generated `*_templ.go` files are ignored by the
scanner. They must not become route units.

The scanner collects validation problems and returns one `ScanError`.

## Route Mapping

Examples:

```text
route.go with Page
/

users/route.go with Page
/users

settings/build_info/route.go with Page
/settings/build-info

users/by_id/route.go with Page
/users/{id}

orgs/by_org_id/users/by_user_id/route.go with Page
/orgs/{org_id}/users/{user_id}
```

Parameter names are preserved in route traversal order.

## Render Units

The scanner records templ-pair presence:

```text
route.go      page.templ optional
layout.go     layout.templ required
```

Missing templ pairs do not fail the scanner.

Layout render-unit pairing is enforced after manifest assembly by
`cmd/goldr/internal/renderunit`. Generated adapter readiness validates that route
declaration handler expression roots resolve to imports or route-package
declarations. The Go compiler type-checks the selected handler values passed to
`goldr.RouteDef`, `goldr.KitRouteDef`, or `goldr.KitRouteMount`.

Fragment templates are ordinary templ declarations used by handlers selected in
`route.go`. The fragment URL surface comes from `Route.Fragments`, not from a
`frag_*.go` file.

## Route Declaration Discovery

`route.go` declares a route directory's endpoint surface with one top-level
`Route` variable:

```go
var Route = goldr.RouteDef{
	Page: page,
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/table", table),
		goldr.FragmentRoute("/", statusOptions),
	},
	Actions: goldr.Actions{
		goldr.Action(http.MethodPost, "/create", postCreate),
	},
}
```

The declaration parser accepts static `goldr.RouteDef`, `goldr.KitRouteDef[K]`,
and `goldr.KitRouteMount[K]` composite literals. It parses the declaration and
imports syntactically; it does not type-check, evaluate code, inspect function
bodies, register handlers, or render templates.

`route.go` owns pages, fragments, and actions for its directory. Named
fragments append their declared segment to the route path. Index fragments use
the route directory path itself and cannot coexist with `Page` because both own
`GET,HEAD` for the same path. The scanner rejects `page.go`, `frag_*.go`, and
`actions.go` anywhere under the route root. Layouts, middleware, templates,
tests, templ-generated files, and ordinary helper files remain separate
route-package files.

Generated route-package adapters call the route declaration's selected handler
symbols. Adapter generation reconstructs only the imports used by those handler
expressions. Dot and blank imports are invalid. If an unaliased import's
package name differs from the final import path segment, the application must
use an explicit import alias in `route.go` so the generated adapter can import
the same selector name.

## Route Navigation

Route declarations may contribute one canonical route navigation step:

```go
var Route = goldr.RouteDef{
	Page: Page,
	Nav: goldr.RouteNav{
		Label: "Customers",
	},
}
```

Dynamic labels use semantic keys that handlers resolve from app data:

```go
var Route = goldr.RouteDef{
	Page: Page,
	Nav:  goldr.RouteNav{Key: "customer"},
}
```

The declaration parser accepts only literal `goldr.RouteNav` values. `Label`
and `Key` are mutually exclusive. `Label` must not be empty after trimming.
`Key` values must use lower snake case ASCII. `RouteNav` supports only
canonical route-step metadata; alternate workflow keys are declared on inbound
destinations.

Manifest and runtime route data carry canonical nav metadata for each live
route declaration. Generated dispatch builds a canonical ordered plan from the
matched route's live ancestors that declare `Nav.Label` or `Nav.Key`, rejects
duplicate `Nav.Key` values in one canonical trail, binds canonical hrefs from
the matched request path values, and stores the request-scoped state before the
endpoint handler or middleware stack runs. Application code reads that state
with `goldr.Nav(r)`, resolves dynamic keys, and renders the resulting
`goldr.Navigation`.

Generated handler `HandlerOptions.BasePath` prefixes only canonical nav hrefs.
It does not affect route matching. App-provided hrefs passed to
`RequestNav.ResolveHref` are used as-is.

Alternate workflow selectors are derived from inbound destinations:

```go
var Route = goldr.RouteDef{
	Page: Page,
	Nav:  goldr.RouteNav{Label: "Report"},
}

var SourceRoute = goldr.RouteDef{
	Page: SourcePage,
	Destinations: goldr.Destinations{
		"model-report": goldr.To(urls.Admin.Models.ByModelID.Report).
			TrailKey("from-inventory"),
	},
}
```

Generated `app/urls` route nodes expose route-scoped `TrailKeys` fields only
when live inbound destinations select keys for that target. Constants are
generated as fields on the target route helper, not as global package
constants. Bound dynamic route helpers do not expose duplicate `TrailKeys`
fields.

Destination declarations are parsed from literal `goldr.Destinations` maps on
`RouteDef`, `KitRouteDef`, and selected `MountRoute` entries. Values must use
`goldr.To(generatedRouteNode)` and may call `.TrailKey("key")` with a string
literal. URL helper generation resolves each target against the live generated
route graph and rejects unknown targets, destination helper-name collisions, and
generated trail-key field-name collisions on each target route. Generated
destination helpers always expose `Href()`, bind target params with `Bind(...)`,
append `_goldr_nav_trail_key` only for destination-selected trail keys, and
preserve clean `Path()` output on normal route helpers. Goldr does not copy
request query parameters or expose query-forwarding destination helpers. Target
route query strings are app-owned URL composition.

Destination helpers that select a trail key also expose
`NavigationHref(goldr.Navigation)`. That helper appends `_goldr_nav_trail_key`
and preserves the current relative URL in `_goldr_return_to` for link-based
Back. The target honors `_goldr_return_to` only when route dispatch validated a
selected trail key for the matched route.

## Mounted Kit Route Subtrees

`KitRouteMount[K]` lets a live route owner mount a non-live Kit route subtree
from `app/mounts`:

```go
var Route = goldr.KitRouteMount[reports.Kit]{
	New:   newReportKit,
	Mount: "reports",
	Routes: goldr.MountRoutes{
		{Path: "/"},
		{Path: "/audit"},
	},
}
```

The mounted subtree uses `KitRouteDef[K]` without `New`:

```go
var Route = goldr.KitRouteDef[reports.Kit]{
	Page: reports.Kit.Page,
}
```

Live `KitRouteDef[K]` declarations under `app/routes` require `New`.
Mounted `KitRouteDef[K]` declarations under `app/mounts` must omit `New`
because the `KitRouteMount[K]` owner supplies the request-scoped kit
constructor. They must also omit `Destinations`; live owners declare
owner-specific destination edges. Mounted source routes may declare default
`Nav` metadata, and included live mounted routes inherit it when the owning
`MountRoute` entry does not provide `Nav`.

Mount expansion happens after the live route tree scan. The mount path must be
a clean relative slash path under `app/mounts`, and each component must be a
lowercase Go-safe route directory name. Underscores are still converted to
hyphens in final URL segments. The scanner reads the referenced subtree under
`app/mounts`, rebases routes, params, layouts, fragments, and actions under the
live owner path, and records both the mounted source and owner source in the
manifest. The expanded routes are ordinary final runtime routes for dispatch
and URL helper generation.

`KitRouteMount.Routes` is an optional explicit allowlist at the live owner.
When omitted, every mounted route declaration is included for that owner. When
present, entries are structured `goldr.MountRoute` values with mount-relative
browser route patterns such as `/`, `/audit`, or `/{id}`. Missing, duplicate,
or malformed entries are scan problems. Excluded mounted children are not added
to dispatch, live URL helpers, route inventory, or middleware composition for
that owner. `MountRoute.Nav` metadata overrides the mounted source route
default only for the included live mounted child selected by that owner. A
child-only selection does not synthesize mount-root nav metadata.

Mounted middleware is rejected. Middleware remains owned by the live
`app/routes` tree. Mounted layouts are allowed and are rebased under the mount
path; real ancestry layouts wrap mounted layouts, and a real layout at the same
final prefix is ordered outside the mounted layout.

Mounted route subtrees may contain cohesive shared implementation that is not
exposed by every owner. The live `app/routes` owner remains the source of truth
for which subset is public.

Generated mount-relative helpers under `app/mounts/<mount>/goldr_gen.go` stay
path helpers. They do not expose owner-specific trail constants or destination
state. Shared mounted code may resolve live owner nav keys with `goldr.Nav(r)`
when the live mounted route inherited or overrode matching `Nav` keys. When
alternate trail shape or app query state is owner-specific, pass trail builders
or URL callbacks through the kit value `K`; the mounted implementation chooses
the app-owned keys and the live owner composes the destination-aware href.

## Determinism

Scanner output is sorted.

Pages and layouts sort by route.

Fragments sort by route prefix and fragment segment.

Actions sort by route, method, and function symbol.

Middleware sorts by route prefix and Go file path.

No dependency, package-load, or filesystem timestamp ordering should affect scanner output.

## Action Discovery

Actions are declared in `route.go` with `Route.Actions`.

```go
var Route = goldr.RouteDef{
	Actions: goldr.Actions{
		goldr.Action(http.MethodPost, "/create", postCreate),
		goldr.Action(http.MethodDelete, "/archive", deleteArchive),
	},
}
```

Default action handlers return `goldr.RouteResponse`:

```go
func postCreate(r *http.Request) goldr.RouteResponse
```

The declaration parser inspects the static `Route.Actions` entries and records
the HTTP method, URL segment, handler expression, handler shape, imports
needed by generated adapters, and route-directory params. It does not inspect
function bodies, type-check, use reflection, or infer actions from function
names.

The empty action segment maps to the current route path. Other segments map to
one lowercase kebab-case child path segment.

Files named `actions.go` are invalid under the route root. Helper files may
still contain action handler functions as long as the route surface is declared
from `route.go`.

## Middleware Discovery

`cmd/goldr/internal/middlewarescan` owns parsing `middleware.go`.

It parses only files named:

```text
middleware.go
```

The middleware parser looks for one top-level function declaration with an
unaliased `net/http` import and exact `http.Handler` spelling:

```go
func Middleware(next http.Handler) http.Handler
```

Middleware is ordinary application-owned `net/http` middleware. Goldr does not
own CSRF validation policy, auth, role checks, rate limits, session policy, or
adapter behavior through this convention.

The scanner combines middleware metadata with the route directory path and
params. Middleware inheritance is resolved later from source-directory
ancestry, not from runtime URL prefix matching.

## Ignored Files

Non-convention Go files in route directories are ignored by the scanner.

Examples:

```text
helpers.go
post_save.go
actions_helper.go
```

Only `route.go`, `layout.go`, and `middleware.go` have scanner meaning beyond
directory validation.

## Route Manifest

The manifest is the stable internal model built from scanner output.

`BuildManifest(tree)` converts a scanner `Tree` into a deterministic `Manifest`
containing:
- pages
- layouts
- fragments
- actions
- middleware
- render-unit file pairs
- route params

The manifest preserves the scanner's route paths, route prefixes, fragment
names, action methods, action functions, middleware route prefixes, Go files,
templ files, and `HasTempl` values.

Manifest params are cloned from scanner output. Callers must not rely on shared
slice ownership between scanner trees and manifests.

## Manifest Render Units

Each manifest page, layout, and first-class fragment row has one `RenderUnit`:

```text
GoFile
TemplFile
HasTempl
```

The manifest records whether a matching `.templ` file exists. It does not make
missing `.templ` files an error.

Layout render-unit pairing is outside routing. Page, fragment, and action
handler contracts selected by route declarations are checked by normal Go
compilation of generated adapters.

## Manifest Determinism

Manifest output is sorted even if the input `Tree` slices are not sorted.

Pages sort by route.

Layouts sort by route prefix.

Fragments sort by route prefix and fragment segment.

Actions sort by route, method, and function symbol.

Middleware sorts by route prefix and Go file path.

Tie breakers use Go file paths to keep ordering stable for malformed or
manually constructed input.

## Manifest Boundary

The manifest does not:
- read the filesystem
- parse Go files
- parse templ files
- generate Go code
- register HTTP handlers
- resolve layout stacks
- assign fragment URLs
- validate strict templ pairing

## Render-Unit Validation

`cmd/goldr/internal/renderunit` owns layout render-unit validation.

It consumes a `Manifest` and validates that every layout render unit has the
expected Go function signature:

```go
func Layout(*http.Request, goldr.LayoutContext) templ.Component
```

It also validates that the layout render unit has:
- `HasTempl` set
- a non-empty templ file path

Validation does not parse `.templ` files, inspect generated templ Go files,
require component names, validate route declaration handler types, register
HTTP handlers, resolve layouts, or assign fragment URLs.

Missing layout render-unit pairs are collected into one validation error so
callers can report all missing `.templ` files together.

## templ Rendering

goldr uses templ for v0 render units.

`cmd/goldr/internal/renderunit` includes a thin internal helper for rendering a
`templ.Component` to an `io.Writer`.

This helper is not public goldr API. Public runtime routing, layout
resolution, and fragment rendering are generated by `cmd/goldr/internal/wiring`.

## templ Dependency Boundary

templ is Goldr's v0 render contract, not an incidental implementation detail.

It is valid for v0 public route contracts, generated route wiring, examples,
tests, and docs to mention `github.com/a-h/templ` and `templ.Component`.

Goldr owns:

- filesystem route conventions
- page, layout, and fragment render-unit pairing
- generated route dispatch
- generated URL helpers
- check and dev-tool integration around the templ workflow
- Goldr diagnostics for missing pairs, stale generated files, and invalid route
  surface

templ owns:

- `.templ` parsing
- HTML escaping and component rendering semantics
- generated `*_templ.go` files
- `go tool templ generate`
- templ watch and proxy behavior used by `goldr dev`

Goldr does not hide templ behind a `goldr.Component` facade or a render-engine
abstraction during v0. A different render backend or custom Goldr template
language would need a separate spec with concrete evidence that templ-backed
checks and helpers cannot solve the problem.

## Generated Metadata Wiring

`cmd/goldr/internal/wiring` owns deterministic generated Go metadata.

It consumes a validated `Manifest` and emits source intended for:

```text
goldr_gen.go
```

The generated source is self-contained in the target package. It does not
import goldr internal packages.

Generated metadata includes:
- page routes, params, Go files, and templ files
- layout route prefixes, params, Go files, and templ files
- fragment segments, route prefixes, params, Go files, and templ files
- action methods, routes, params, Go files, functions, suffixes, and URL
  segments
- route-local allowed navigation trail keys for page, fragment, and action
  endpoints

Generated metadata omits the route root so output does not depend on local
machine paths.

Runtime generation takes the target package name and route-root import path.
The import path is required when generated runtime code must call nested route
packages.

Generated metadata is emitted beside generated runtime wiring. The metadata
does not define templ component names. Runtime wiring derives page, layout, and
fragment function calls from goldr's filesystem conventions, calls action
functions by their scanned symbols, and calls generated `GoldrRoute*` adapters
for route declarations.

Page metadata is not scanner metadata. It is runtime data returned by page
functions and passed to layout functions by generated dispatch.

## Generated URL Helper Wiring

`cmd/goldr/internal/wiring` also owns deterministic generated URL helper source for:

```text
app/urls/goldr_gen.go
app/mounts/<mount>/goldr_gen.go for referenced Kit mount subtrees
```

The helper package is separate from `app/routes` so route packages and
templ-generated Go can import `app/urls` without creating an import cycle.
Mounted helper files are generated into the referenced `app/mounts` package so
shared mounted handlers and templates can accept owner-bound helpers without
importing `app/urls`.

URL helpers are generated from the same runtime paths used by generated
dispatch:
- page paths
- fragment paths
- action paths

Helpers are deduplicated by unique path before naming. Same-path routes with
different HTTP methods share one helper because HTTP methods remain visible in
callers such as HTMX attributes.

The generated API uses route-node namespaces ending in `.Path()`:

```go
urls.Root.Path()
urls.Users.Path()
urls.Users.Create.Path()
urls.Users.Table.Path()
urls.BySlug.Bind(slug).Path()
urls.Users.ByID.Bind(id).Path()
urls.Users.ByID.Bind(id).Profile.Path()
```

Static child segments are exported fields. Dynamic child segments are exported
route nodes with `.Bind(value)` methods. Dynamic params are escaped with
`url.PathEscape` when `.Bind(value)` is called, and the escaped values are
stored on the returned bound route node.

Generated dynamic route nodes work with `goldr.BindFromRequest`, which uses
`GoldrRouteParams()` to choose the last param for that node and returns
`(route, ok)` after binding `r.PathValue("<param>")`. Nil requests, nodes with
no params, and empty path values return `ok == false`. Nested dynamic helpers
still bind one node at a time because parent params are carried on the receiver
after the parent node is bound.

Generated runtime dispatch matches `r.URL.EscapedPath()` so escaped dynamic
segment values are not split as extra path segments. Captured params are
decoded with `url.PathUnescape` before generated dispatch attaches them with
`r.SetPathValue`.

Root-level dynamic segments are emitted as package route nodes, so callers use
the same `urls.BySlug.Bind(slug).Path()` shape as nested dynamic routes.

Generated URL helpers also expose `WithBasePath(basePath string)` for
applications mounted below a URL prefix. It returns an exported `MountedRoutes`
route set with the same top-level helper surface as the package globals:

```go
mounted := urls.WithBasePath("/webapp")
mounted.Users.ByID.Bind(id).Path()
mounted.BySlug.Bind(slug).Path()
```

The generated helper normalizes `""` and `"/"` to no prefix, adds a missing
leading slash, and removes trailing slashes. It does not escape or clean the
base path. Mounted helpers only affect generated strings; generated dispatch
continues to match unmounted paths after the application strips its mount
prefix.

Referenced Kit mount subtrees also get mount-relative helper files. These files
expose `NewGoldrMountURLs(route interface{ Path() string })` and a
`GoldrMountURLs` route set. The mount owner passes the final live route helper,
usually from `app/urls`, and the mounted package uses helpers such as
`mountURLs.Path()` and
`mountURLs.Table.Path()`. `Path()` returns the normalized mount path itself
instead of forcing a trailing slash. Child and dynamic helpers follow the same
path-derived naming and escaping rules as `app/urls`.
These mount-relative helper files include every route declaration from the
mounted source subtree, including owner-specific children selected by only some
live owners. They are subtree path helpers. They do not make excluded children
part of an owner's live dispatch, normal route inventory, or `app/urls` helper
surface. A child-only mounted selection still gets an `app/urls` helper for the
owner mount base so `NewGoldrMountURLs` can be bound from the live owner, but
that synthetic helper does not register the mount root as a live handler.

Generated URL helper source imports only standard library packages. It imports
`net/url` only when dynamic params exist. It does not import route packages,
goldr internals, templ, HTMX helpers, or application handlers.

Generated URL helpers do not register routes, match requests, inspect handler
bodies, wrap HTMX attributes, carry HTTP methods, or perform runtime route
lookup. They are generated string helpers.

If two unique paths produce the same generated helper path, or if a route
segment collides with generated names such as `Path`, `WithBasePath`, or
`MountedRoutes`,
generation fails with a clear error rather than adding aliases or method
suffixes.

URL helper collision checks are owned by URL helper tree construction and by
route-surface inspection, because both expose helper expressions. Generated
route dispatch does not fail on URL-helper-only collisions; dispatch readiness
and URL-helper readiness are separate generated-file contracts.

## Route Surface Inspection

`cmd/goldr/internal/wiring` owns the read-only route surface used by `goldr routes list`.

The route surface consumes a `Manifest` and returns table rows for:

- layouts
- pages
- fragments
- actions

It reuses the same runtime path helpers as generated dispatch and generated URL
helpers:

- page paths from the manifest
- fragment paths using the generated `<name>` browser route rule
- action paths from scanned action metadata
- runtime path deduplication and validation
- URL helper tree construction and collision checks

This keeps `goldr routes list` as an inspection view over the existing route
model. It must not grow a second router, route registry, matcher, or annotation
system.

Route-surface rows include kind, methods, path, params, route-relative source,
optional route declaration metadata, and URL helper expression. Layout rows do
not have HTTP methods, declaration metadata, or URL helpers. Page and fragment
rows carry `GET` and `HEAD`. Action rows carry the scanned HTTP method and
include the generated action adapter in the source column.

Route declaration metadata is derived from the already-scanned
`ManifestRouteDeclaration`. It identifies whether the route came from
`goldr.RouteDef`, `goldr.KitRouteDef[K]`, or an expanded mounted Kit route,
shows static name, title, and sorted labels, and records the parsed page,
fragment, action, kit, mounted source, and mount owner expressions. The route
surface must not execute app code, infer policy from labels, or make
declaration names affect URL helper generation.

`goldr routes list` renders that route surface either as the default text table
or as JSON when `--json` is passed. `--mount <path>` filters the same route
surface to rows expanded from one mounted subtree. JSON output is CLI-owned
formatting over the same internal route-surface rows. Declaration-backed
endpoint rows include a structured `declaration` object; layout rows omit it.
It is not a persisted manifest and must not be loaded by generated runtime
dispatch.

`goldr routes layouts` renders a text-only layout map over the same manifest
and runtime route helpers. `cmd/goldr/internal/wiring` owns the read-only layout-map model:
route-tree nodes, layouts, pages, fragments, actions, and page layout stacks.
`cmd/goldr/internal/goldrcli` owns Unicode tree rendering, automatic terminal styling, and
cwd-relative source-path formatting. Terminal styles are display-only and must
not affect plain piped output, route discovery, route ordering, source paths, or
the layout-map model. The layout map is an inspection view; it must not
introduce a second router, route registry, matcher, generated runtime contract,
or JSON schema.

The route surface does not validate render-unit health. Missing layout or
fragment template pairs remain visible in `goldr routes`; `cmd/goldr/internal/renderunit`
and `goldr check` own health validation.

Rows sort root first, then by generated runtime path priority, kind order,
method order, and source path.

## Route Explain Inspection

`cmd/goldr/internal/wiring` owns the read-only route explanation model used by
`goldr routes explain`.

Route explanation consumes the same `Manifest` and runtime route helpers used by
generated dispatch, route-surface inspection, layout maps, and URL helper
generation. It must not introduce a second router or registry.

The matcher uses escaped URL paths, static-before-dynamic route priority,
generated fragment URLs, scanned action methods, page layout stacks, and the
same method ownership as generated dispatch:

- pages and fragments support `GET` and `HEAD`
- actions support their scanned method
- matched paths with unsupported methods report allowed methods
- dynamic params are decoded with `url.PathUnescape`

`cmd/goldr/internal/goldrcli` owns command parsing, full-URL path extraction, human text
rendering, automatic terminal styling, and cwd-relative source paths.

`goldr routes explain` is an inspection view. It does not call a running server,
write files, validate generated-file freshness, persist a manifest, define JSON
output, or change generated runtime dispatch.

For declaration-backed matches, route explanation carries the same static
declaration metadata as route-surface rows plus implementation evidence for
the matched page, fragment, or action adapter. The CLI renders this as
`DECLARATION` and `IMPLEMENTATION` sections before the layout stack. These
sections are display-only and must not become a second route registry or policy
engine.

`goldr routes refs` is a template-source inspection view. It parses `.templ`
files with templ's parser through Goldr's internal `templscan` wrapper, then
uses `htmxrefs` to report direct HTMX request attributes and resolve only
obvious route-surface references. It does not render pages, execute app code,
trace templ component composition, or infer inherited HTMX attributes. This
keeps route references inspectable without turning HTMX behavior into hidden
Goldr metadata.

## CLI Generation

`goldr generate` is the supported CLI entrypoint for writing goldr-owned
generated files from a real application route tree.

The command treats `--app-root` as the application root and uses the fixed
conventions:

```text
<root>/app/routes
<root>/app/urls
```

It writes:

```text
<root>/app/routes/goldr_gen.go
<root>/app/routes/**/goldr_gen.go when route packages need generated helpers
<root>/app/internal/goldrinspect/goldr_gen.go
<root>/app/urls/goldr_gen.go
<root>/app/mounts/<mount>/goldr_gen.go for referenced Kit mount subtrees
```

Route dispatch output is generated as `package routes`. Package-local route
helpers use the package name of the route directory they are generated into.
URL helper output is always generated as `package urls`.
Mounted URL helper output uses the package name of the referenced mount root.

Nested route packages require a real Go import path. The command derives the
route-root import path by running `go env GOMOD` with `cwd=<root>`, parsing the
module directive from that `go.mod`, and joining the module path with the
module-relative `app/routes` path.

For example, running:

```bash
(cd examples/full_feature && go tool goldr generate)
```

inside the full-feature example module derives:

```text
github.com/mobiletoly/goldr/examples/full_feature/app/routes
```

`goldr generate --check` runs templ check mode when `.templ` files are present,
then generates Goldr-owned files in memory and compares them to disk without
writing. Missing or stale generated files fail the check.

Generated route dispatch starts with a deterministic route-surface comment.
The comment lists layout, page, fragment, and action rows with kind, methods,
path, and source. Layout rows use `-` for methods because layouts are generated
wiring surface, not standalone routes. URL helper expressions are reserved for
`goldr routes list` and `app/urls/goldr_gen.go`, not the route dispatch header.
The comment is for reviewability only; runtime code does not read it.

Generated route dispatch also emits a short contract comment immediately above
each call into app-owned route code. These call-site comments name only the
expected `app/routes/...` file. The comments are for compiler-error locality
and generated-source inspectability only; runtime code does not read them.

## Middleware And Static Assets

Generated route dispatch is an ordinary `http.Handler`.

Application code owns the outer HTTP composition:

- mux construction
- middleware ordering
- auth, sessions, CSRF, logging, recovery, and security headers
- static asset handlers
- static asset cache headers
- static asset errors

Static asset handlers should be registered on a more specific path such as
`/assets/` before generated routes are registered at `/`.

Goldr also supports route-tree endpoint middleware through `middleware.go`
files under `app/routes`. Generated dispatch wraps matched pages, actions, and
fragments with inherited middleware from root to leaf. This is compile-time
filesystem composition over ordinary `net/http` middleware, not a runtime
registry, global wrapper, or URL prefix matcher.

Layouts are not middleware endpoints. Layout rendering happens inside the
already wrapped page request or inside an action request when the action
returns a page response.

Generated 404 and 405 responses do not run route-tree middleware. Custom error
hooks still apply only inside generated route dispatch.

Generated error hooks apply only inside generated route dispatch. They do not
customize errors returned by application-owned static asset handlers or other
handlers mounted beside generated routes.

The `goldr assets` command fingerprints final static files and generates an
asset manifest package, but applications still own static handler registration
and cache policy. Goldr should not add framework-owned middleware policy, a
broad asset pipeline, deployment integration, or automatic asset injection
without a separate spec.

## Runtime Page Routing

Generated wiring owns runtime page dispatch.

When a manifest contains at least one runtime route, `goldr_gen.go` includes:

```go
func Handler() http.Handler
func HandlerWithOptions(options HandlerOptions) http.Handler
```

`Handler()` delegates to `HandlerWithOptions(HandlerOptions{})`.

Generated dispatch splits `r.URL.EscapedPath()` into path segments once per
request, then routes through private generated segment functions. Static
segments are matched before the single dynamic fallback at a directory level.
This keeps the generated source ordinary Go while avoiding one flat path chain
for larger route trees.

The generated segment functions are deterministic and route-tree shaped:

- `/` is handled as a root fast path
- trailing slash paths are rejected as unmatched generated routes
- static child segments dispatch through `switch` cases
- dynamic child segments dispatch only after static children do not match
- leaf paths dispatch by HTTP method before returning `405`

The generated handler calls each matched route package's page function:

```go
func Page(r *http.Request) goldr.PageRouteResponse
```

For route declarations, generated dispatch calls the route package's generated
`GoldrRoutePage` adapter instead. The adapter calls the handler selected by
`route.go`.

The returned `goldr.PageRouteResponse` is written through root-package route
writers. Generated page dispatch calls `goldr.WritePageRouteResponse` with a
route-local renderer that wraps the page component with inspector markers and
applies the selected layout stack. Generated fragment dispatch calls
`goldr.WriteFragmentRouteResponse` after wrapping actual fragment components
with inspector markers.

The root package validates and writes normal rendered pages, fragments,
redirects, plain text status responses, and internal error responses. Page,
fragment, redirect, and text responses may carry explicit headers with
`WithHeader` and `AddHeader`; the writer applies those headers before writing
status and body. Nil rendered components and invalid route responses are
internal server errors. The framework metadata surface is intentionally small:
`Title` and `Description` are passed through to layouts, while canonical links,
navigation state, and other shell policy remain application-owned.

An invalid Goldr route response contract includes a zero-value page, a nil
render component, an empty redirect location, a redirect status outside `301`,
`302`, `303`, `307`, and `308`, a bodyless page status such as `204` or `205`,
or `goldr.RouteError{Err: nil}`. Rendered page statuses must be final
body-carrying statuses: `2xx` except `204` and `205`, plus `4xx` and `5xx`.
`goldr.RouteError{Err: err}` is separate from that validation path: it is a
valid route response, and `err` is the application error passed to the generated
route error handler.

When the manifest contains matching layouts, the generated handler wraps the page
component by calling:

```go
func Layout(r *http.Request, ctx goldr.LayoutContext) templ.Component
```

The selected layout stack is root-to-leaf by route prefix. Runtime composition
applies the deepest layout first and the root layout last so the root layout
renders outermost.

The generated handler builds one `goldr.LayoutContext` per rendered page or
status component, copies the page metadata into it, and updates `ctx.Child`
before each layout call. Root and nested layouts receive the same metadata
value. Redirects, plain text status responses, and error responses bypass the
layout chain. The framework does not store metadata on `context.Context`, use a
registry, collect head items from components, or merge head output at runtime.

Nested page and layout packages are imported by the generated root route package
using deterministic aliases derived from route-relative directories.

Dynamic route params are attached to the request before page and layout
functions run. Application code reads them with `r.PathValue`.

The generated handler supports `GET` and `HEAD` for matched page routes.

It returns:
- `404` for unmatched paths
- `405` with `Allow: GET, HEAD` for unsupported methods on matched paths
- plain `500` by default for nil page components, nil layout components, or
  templ render errors

Successful page responses use:

```text
Content-Type: text/html; charset=utf-8
```

Generated dispatch rejects ambiguous page patterns, including duplicate page
routes and same-shape dynamic sibling routes such as `/users/{id}` and
`/users/{slug}`.

Trailing-slash normalization and static registration remain outside this
boundary.

## Runtime Fragment Routing

Generated wiring owns runtime fragment dispatch through the same
`Handler() http.Handler` used for pages.

A manifest fragment route prefix and name map to a public URL:

```text
RoutePrefix: /users
Name: table
URL: /users/table

RoutePrefix: /users/{id}
Name: row
URL: /users/{id}/row
```

The generated handler calls each matched route package's fragment function:

```go
func Frag<Name>(r *http.Request) goldr.FragmentRouteResponse
```

For route declarations, generated dispatch calls a generated
`GoldrRouteFrag<Name>` adapter instead. The adapter calls the fragment handler
selected by `route.go`.

Fragment function names are derived from the fragment segment, with `Index`
reserved for index fragments:

```text
table      -> FragTable
user_row   -> FragUserRow
index      -> FragIndex
```

Fragment browser path segments use the declared fragment segment directly. For
legacy manifest fixtures without a declaration segment, the fragment source
name is used with `_` replaced by `-`.

Index fragments do not append a segment. An index fragment declared in
`app/routes/users/status_options/route.go` serves `/users/status-options` and
uses the generated adapter `GoldrRouteFragIndex`.

Dynamic route params are attached to the request before fragment functions run.
Application code reads them with `r.PathValue`.

Fragments render standalone partial HTML. They are not wrapped by page layout
stacks.

The generated handler supports `GET` and `HEAD` for matched fragment routes.

It returns:
- `404` for unmatched paths
- `405` with `Allow: GET, HEAD` for unsupported methods on matched paths
- redirects, plain text responses, and server errors from the returned route
  response
- plain `500` by default for nil fragment components, invalid fragment route
  responses, or templ render errors

Successful fragment responses use:

```text
Cache-Control: no-store
Content-Type: text/html; charset=utf-8
```

The default cache policy is attached to `goldr.NewFragment` responses during
route-response resolution. An application-owned `Cache-Control` header on the
fragment response takes precedence.

Generated dispatch rejects ambiguous runtime patterns, including page-fragment
URL collisions and same-shape dynamic fragment routes such as
`/users/{id}/row` and `/users/{slug}/row`.

## Runtime Action Routing

Generated wiring dispatches action routes through the same
`Handler() http.Handler` used for pages and fragments.

Default actions return `goldr.RouteResponse`:

```go
func PostCreate(r *http.Request) goldr.RouteResponse
```

The generated handler calls action functions and writes returned route
responses with `goldr.WriteRouteResponse`. For route declarations, generated
dispatch calls a generated `GoldrRoute<Method><Name>` adapter, and that adapter
calls the action handler selected by `route.go`.

For page responses, generated dispatch attaches the matched route page renderer
before calling the action. `WriteRouteResponse` uses that renderer for page
responses, while redirect, text, fragment, no-content, and server-error
responses are handled by the root writer directly. Actions without a matched
layout stack still use the same API; the renderer returns the page component
without adding layouts.

Low-level writer actions are explicit. Route declarations use
`HTTPAction` or `KitHTTPAction` when an action needs direct `http.ResponseWriter`
control.

Dynamic route params are attached to the request before action handlers run.
Application code reads them with `r.PathValue`.

Pages, fragments, and actions may share one path when methods differ. The
generated path branch dispatches by method before returning a `405`.

For matched paths with unsupported methods, generated dispatch sets `Allow` to
the path's supported methods.

Generated dispatch rejects exact method-and-path collisions. It also rejects
same-shape dynamic runtime collisions so generation does not silently drop or
replace routes.

Collision ownership is intentionally split:

- generated route dispatch rejects runtime ambiguity, including exact
  method/path collisions, page-fragment URL collisions, and same-shape dynamic
  page, fragment, or action routes
- generated URL helpers reject helper API ambiguity, including collisions with
  generated `Root` or `Path` names, normalized static child names,
  static/dynamic helper names, and repeated dynamic param argument names
- `goldr routes` validates URL-helper collisions because it prints helper
  expressions
- `goldr check` reports route-dispatch readiness and URL-helper readiness as
  separate diagnostic categories

## Runtime Error Hooks

Generated route packages expose:

```go
type ErrorHandlers struct {
	RouteNotFound         func(*http.Request) goldr.RouteResponse
	RouteMethodNotAllowed func(*http.Request) goldr.RouteResponse
	RouteError            func(*http.Request, error) goldr.RouteResponse
}

type HandlerOptions struct {
	ErrorHandlers       ErrorHandlers
	TemplateInspection goldr.TemplateInspectionMode
}
```

`HandlerWithOptions` accepts this value and applies the hooks only to generated
route dispatch. Nil fields use the default generated behavior independently.

Generated error hooks return route responses rather than writing to
`http.ResponseWriter`. Not-found and method-not-allowed page responses render
through the root layout stack when one exists. Internal-server-error page
responses render through the matched route layout stack because generated
dispatch already knows which endpoint failed. Fragment, text, redirect, and
no-content error responses are written as returned. If writing a custom error
response fails, generated dispatch writes a plain `500` and does not call the
custom hooks recursively.

The generated not-found helper handles unmatched generated routes and invalid
dynamic path unescape failures.

The generated method-not-allowed helper runs after dispatch sets the `Allow`
header for the matched path.

The generated internal-server-error helper handles nil page, layout, or
fragment components, invalid route responses, and templ render failures. Custom
hooks receive `goldr.ErrNilComponent` for nil render units or the underlying
templ render error.

`TemplateInspection` is a development inspection option. The zero value is
`goldr.TemplateInspectionOff`, so `Handler()` output has no inspector comments
or overlay scripts. `goldr.TemplateInspectionComments` wraps page, layout, and
fragment render boundaries in paired HTML comments with app-relative
`app/routes/...` source paths. `goldr.TemplateInspectionOverlay` emits the
same comments and enables `goldr.TemplateInspector()` to render Goldr's browser
overlay helper script from an explicit app layout. Redirect, plain text, error,
and default-handler responses do not emit inspector body markers.

Generated applications also include `app/internal/goldrinspect/goldr_gen.go`.
The route dispatcher enables inspector mode by attaching the public Goldr
inspection mode to request context, then page, layout, and direct fragment
routes call the generated `goldrinspect.Wrap` helper. The wrapping helper is
app-internal generated code, not a public `goldr` package API.

Embedded fragment boundaries are opt-in at the template call site. For every
first-class fragment declared outside the root route package, Goldr writes a
package-local `goldr_gen.go` containing a helper such as:

```go
func renderFragTable(component templ.Component) templ.Component
```

The helper name is derived from the fragment segment in `route.go`: a fragment
segment `table` maps to `renderFragTable`. Root-package fragment helpers are
emitted into the existing `app/routes/goldr_gen.go`, so each route package has
at most one Goldr-generated `goldr_gen.go` file.

Templates can use the helper when an embedded fragment should be visible to the
inspector:

```templ
<div id="users-table-slot">
	@renderFragTable(FragTableView(contacts))
</div>
```

When HTMX updates an inspected embedded fragment, the application should target
the page-owned slot with `innerHTML`, not the fragment root with `outerHTML`.
Inspector comments wrap the rendered fragment as sibling nodes. The slot keeps
the marker comments and fragment root inside one replacement boundary.

Calling `@FragTableView(contacts)` directly remains valid and renders the same
HTML, but no embedded fragment inspector boundary is emitted. Multiple templ
declarations in one template file are ordinary application implementation
details; separately inspectable fragments require separate `goldr.FragmentRoute`
declarations in `route.go`.

Generated helper names are reserved by convention. Goldr does not preflight
collisions; normal Go compilation reports redeclarations in the affected route
package.

Action handlers are called directly and remain responsible for their own error
responses. Static assets are application-owned and are outside this generated
error hook boundary.
