# Routing Scanner

`internal/routing` owns route filesystem scanning and route manifest assembly.

The scanner and manifest provide route data for framework internals without
performing rendering, handler registration, or code generation.

It is not runtime request routing.

## Page And Layout Model

goldr borrows the familiar filesystem page and layout model, not the runtime
semantics of JavaScript routers.

Pages are route endpoints. Layouts are route-subtree wrappers.

Generated runtime wiring keeps this model Go-native: route packages expose
plain `Page` and `Layout` functions returning `templ.Component`, params live on
the request, and rendering remains server-side HTML.

## Scanner Boundary

The scanner reads a route root and returns a deterministic tree of:

- pages
- layouts
- fragments
- actions

It records matching `.templ` files when present.

It does not:
- parse Go files other than `actions.go`
- parse templ files
- render HTML
- register HTTP handlers
- generate code
- assign fragment URLs

## Route Root

User-facing applications use:

```text
app/routes/
```

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
- `page.go` and `page.templ`
- `layout.go` and `layout.templ`
- `frag_*.go` and `frag_*.templ`
- `actions.go`

The scanner rejects:
- underscore-prefixed route directories
- dot-prefixed route directories
- `testdata` route directories
- malformed dynamic route directories such as `by_/`
- route directories with uppercase letters or hyphens
- dot-prefixed Go route files
- underscore-prefixed Go route files
- malformed fragment files such as `frag_.go`

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
page.go
/

users/page.go
/users

settings/build_info/page.go
/settings/build-info

users/by_id/page.go
/users/{id}

orgs/by_org_id/users/by_user_id/page.go
/orgs/{org_id}/users/{user_id}
```

Parameter names are preserved in route traversal order.

## Render Units

The scanner records templ-pair presence:

```text
page.go       page.templ
layout.go     layout.templ
frag_row.go   frag_row.templ
```

Missing templ pairs do not fail the scanner.

Strict render-unit pairing is enforced after manifest assembly by
`internal/renderunit`.

## Determinism

Scanner output is sorted.

Pages and layouts sort by route.

Fragments sort by route prefix and fragment name.

Actions sort by route, method, and function symbol.

No dependency, package-load, or filesystem timestamp ordering should affect scanner output.

## Action Discovery

`internal/actionscan` owns parsing `actions.go`.

It parses only files named:

```text
actions.go
```

The action parser inspects top-level function declarations only. It ignores
function bodies, does not type-check, and does not use reflection.

Supported action names use method prefixes:

```text
Post<Name>   -> POST
Put<Name>    -> PUT
Patch<Name>  -> PATCH
Delete<Name> -> DELETE
```

`Get<Name>` is rejected because generated pages and fragments own `GET` and
`HEAD`.

Each action function must have this shape:

```go
func PostCreate(w http.ResponseWriter, r *http.Request)
```

The suffix `Index` maps to the current route path. Other suffixes map to one
lowercase kebab-case child segment.

The scanner combines action metadata with the route directory path and params.
An `actions.go` file with no supported action functions adds no action rows.

## Ignored Files

Non-convention Go files in route directories are ignored by the scanner.

Examples:

```text
helpers.go
post_save.go
actions_helper.go
```

Only `actions.go` has action-routing meaning.

## Route Manifest

The manifest is the stable internal model built from scanner output.

`BuildManifest(tree)` converts a scanner `Tree` into a deterministic `Manifest`
containing:
- pages
- layouts
- fragments
- actions
- render-unit file pairs
- route params

The manifest preserves the scanner's route paths, route prefixes, fragment
names, action methods, action functions, Go files, templ files, and `HasTempl`
values.

Manifest params are cloned from scanner output. Callers must not rely on shared
slice ownership between scanner trees and manifests.

## Manifest Render Units

Each manifest page, layout, and fragment has one `RenderUnit`:

```text
GoFile
TemplFile
HasTempl
```

The manifest records whether a matching `.templ` file exists. It does not make
missing `.templ` files an error.

Strict render-unit pairing is outside routing.

## Manifest Determinism

Manifest output is sorted even if the input `Tree` slices are not sorted.

Pages sort by route.

Layouts sort by route prefix.

Fragments sort by route prefix and fragment name.

Actions sort by route, method, and function symbol.

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

`internal/renderunit` owns strict render-unit validation.

It consumes a `Manifest` and validates that every page, layout, and fragment
render unit has:
- `HasTempl` set
- a non-empty templ file path

Validation is file-pair based only. It does not parse `.templ` files, inspect
generated templ Go files, require component names, register HTTP handlers,
resolve layouts, or assign fragment URLs.

Missing render-unit pairs are collected into one validation error so callers
can report all missing `.templ` files together.

## templ Rendering

goldr uses templ for v0 render units.

`internal/renderunit` includes a thin internal helper for rendering a
`templ.Component` to an `io.Writer`.

This helper is not public goldr API. Public runtime routing, layout
resolution, and fragment rendering are generated by `internal/wiring`.

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

`internal/wiring` owns deterministic generated Go metadata.

It consumes a validated `Manifest` and emits source intended for:

```text
goldr_gen.go
```

The generated source is self-contained in the target package. It does not
import goldr internal packages.

Generated metadata includes:
- page routes, params, Go files, and templ files
- layout route prefixes, params, Go files, and templ files
- fragment names, route prefixes, params, Go files, and templ files
- action methods, routes, params, Go files, functions, suffixes, and URL
  segments

Generated metadata omits the route root so output does not depend on local
machine paths.

Runtime generation takes the target package name and route-root import path.
The import path is required when generated runtime code must call nested route
packages.

Generated metadata is emitted beside generated runtime wiring. The metadata
does not define templ component names. Runtime wiring derives page, layout, and
fragment function calls from goldr's filesystem conventions and calls action
functions by their scanned symbols.

Page metadata is not scanner metadata. It is runtime data returned by page
functions and passed to layout functions by generated dispatch.

## Generated URL Helper Wiring

`internal/wiring` also owns deterministic generated URL helper source for:

```text
app/urls/goldr_gen.go
```

The helper package is separate from `app/routes` so route packages and
templ-generated Go can import `app/urls` without creating an import cycle.

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
urls.Users.FragTable.Path()
urls.BySlug(slug).Path()
urls.Users.ByID(id).Path()
urls.Users.ByID(id).Profile.Path()
```

Static child segments are exported fields. Dynamic child segments are exported
methods such as `ByID(id string)` because they need path param values. Dynamic
params are escaped with `url.PathEscape` when the dynamic method is called, and
the escaped values are stored on the returned route node.

Generated runtime dispatch matches `r.URL.EscapedPath()` so escaped dynamic
segment values are not split as extra path segments. Captured params are
decoded with `url.PathUnescape` before generated dispatch attaches them with
`r.SetPathValue`.

Root-level dynamic segments have no parent route node, so they are emitted as
package functions such as `BySlug(slug string)`.

Generated URL helper source imports only standard library packages. It imports
`net/url` only when dynamic params exist. It does not import route packages,
goldr internals, templ, HTMX helpers, or application handlers.

Generated URL helpers do not register routes, match requests, inspect handler
bodies, wrap HTMX attributes, carry HTTP methods, or perform runtime route
lookup. They are generated string helpers.

If two unique paths produce the same generated helper path, or if a route
segment collides with generated names such as `Path`, generation fails with a
clear error rather than adding aliases or method suffixes.

URL helper collision checks are owned by URL helper tree construction and by
route-surface inspection, because both expose helper expressions. Generated
route dispatch does not fail on URL-helper-only collisions; dispatch readiness
and URL-helper readiness are separate generated-file contracts.

## Route Surface Inspection

`internal/wiring` owns the read-only route surface used by `goldr routes list`.

The route surface consumes a `Manifest` and returns table rows for:

- layouts
- pages
- fragments
- actions

It reuses the same runtime path helpers as generated dispatch and generated URL
helpers:

- page paths from the manifest
- fragment paths using the generated `frag-<name>` browser route rule
- action paths from scanned action metadata
- runtime path deduplication and validation
- URL helper tree construction and collision checks

This keeps `goldr routes list` as an inspection view over the existing route
model. It must not grow a second router, route registry, matcher, or annotation
system.

Route-surface rows include kind, methods, path, params, route-relative source,
and URL helper expression. Layout rows do not have HTTP methods or URL helpers.
Page and fragment rows carry `GET` and `HEAD`. Action rows carry the scanned
HTTP method and include the action function in the source column.

`goldr routes list` renders that route surface either as the default text table
or as JSON when `--json` is passed. JSON output is CLI-owned formatting over
the same internal route-surface rows. It is not a persisted manifest and must
not be loaded by generated runtime dispatch.

`goldr routes layouts` renders a text-only layout map over the same manifest
and runtime route helpers. `internal/wiring` owns the read-only layout-map model:
route-tree nodes, layouts, pages, fragments, actions, and page layout stacks.
`internal/goldrcli` owns Unicode tree rendering, automatic terminal styling, and
cwd-relative source-path formatting. Terminal styles are display-only and must
not affect plain piped output, route discovery, route ordering, source paths, or
the layout-map model. The layout map is an inspection view; it must not
introduce a second router, route registry, matcher, generated runtime contract,
or JSON schema.

The route surface does not validate strict `.go` / `.templ` pairing. Missing
render-unit pairs remain visible in `goldr routes`; `internal/renderunit` and
`goldr check` own strict health validation.

Rows sort root first, then by generated runtime path priority, kind order,
method order, and source path.

## Route Explain Inspection

`internal/wiring` owns the read-only route explanation model used by
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

`internal/goldrcli` owns command parsing, full-URL path extraction, human text
rendering, automatic terminal styling, and cwd-relative source paths.

`goldr routes explain` is an inspection view. It does not call a running server,
write files, validate generated-file freshness, persist a manifest, define JSON
output, or change generated runtime dispatch.

## CLI Generation

`goldr generate` is the supported CLI entrypoint for writing goldr-owned
generated files from a real application route tree.

The command treats `--root` as the application root and uses the fixed
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
```

Route dispatch output is generated as `package routes`. Package-local route
helpers use the package name of the route directory they are generated into.
URL helper output is always generated as `package urls`.

Nested route packages require a real Go import path. The command derives the
route-root import path by running `go env GOMOD` with `cwd=<root>`, parsing the
module directive from that `go.mod`, and joining the module path with the
module-relative `app/routes` path.

For example, running:

```bash
goldr generate --root examples/full_feature
```

inside the goldr repository derives:

```text
github.com/mobiletoly/goldr/examples/full_feature/app/routes
```

`goldr generate --check` generates Goldr-owned files in memory and compares
them to disk without writing. Missing or stale generated files fail the check.

The command does not run `templ generate`; templ-generated Go remains owned by
templ.

Generated route dispatch starts with a deterministic route-surface comment.
The comment lists layout, page, fragment, and action rows with kind, methods,
path, source, and URL helper expression. Layout rows use `-` for methods and
helper because layouts are generated wiring surface, not standalone routes.
The comment is for reviewability only; runtime code does not read it.

Generated route dispatch also emits a short contract comment immediately above
each call into app-owned route code. These call-site comments name the route
kind and path, the expected `app/routes/...` file, and the expected function
signature for page, fragment, layout, and action functions. The comments are
for compiler-error locality and generated-source inspectability only; runtime
code does not read them.

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

Generated error hooks apply only inside generated route dispatch. They do not
customize errors returned by application-owned static asset handlers or other
handlers mounted beside generated routes.

The `goldr assets` command fingerprints final static files and generates an
asset manifest package, but applications still own static handler registration
and cache policy. Goldr should not add a middleware stack, broad asset
pipeline, deployment integration, or automatic asset injection without a
separate spec.

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
func Page(r *http.Request) goldr.RouteResponse
```

The returned `goldr.RouteResponse` is resolved by generated dispatch. Generated
dispatch handles normal rendered pages, fragments, redirects, plain text status
responses, and internal error responses. Page, fragment, redirect, and text
responses may carry explicit headers with `WithHeader` and `AddHeader`;
generated dispatch applies those headers before writing status and body. Nil
rendered components and invalid route
responses are internal server errors. The framework metadata surface is
intentionally small: `Title` and `Description` are passed through to layouts,
while canonical links, navigation state, and other shell policy remain
application-owned.

Generated dispatch calls `goldr.ResolveRouteResponse(response)`. A non-nil
error from that function is an invalid Goldr route response contract, such as a
zero-value page, a nil render component, an empty redirect location, a redirect
status outside `301`, `302`, `303`, `307`, and `308`, a bodyless page status
such as `204` or `205`, or `goldr.ServerError{Err: nil}`. Rendered page statuses
must be final body-carrying statuses: `2xx` except `204` and `205`, plus `4xx`
and `5xx`. `goldr.ServerError{Err: err}` is separate from that validation path:
it is a valid route response, and `err` is the application error passed to the
generated internal server error handler.

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
URL: /users/frag-table

RoutePrefix: /users/{id}
Name: row
URL: /users/{id}/frag-row
```

The generated handler calls each matched route package's fragment function:

```go
func Frag<Name>(r *http.Request) goldr.RouteResponse
```

Fragment function names are derived from the fragment name:

```text
table      -> FragTable
user_row   -> FragUserRow
```

Fragment browser path segments use the `frag-` prefix plus the fragment source
name with `_` replaced by `-`.

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
Content-Type: text/html; charset=utf-8
```

Generated dispatch rejects ambiguous runtime patterns, including page-fragment
URL collisions and same-shape dynamic fragment routes such as
`/users/{id}/frag-row` and `/users/{slug}/frag-row`.

## Runtime Action Routing

Generated wiring dispatches action routes through the same
`Handler() http.Handler` used for pages and fragments.

Actions are plain `net/http` handlers:

```go
func PostCreate(w http.ResponseWriter, r *http.Request)
```

The generated handler calls action functions directly. Actions are not rendered
through templ and are not wrapped in layouts.

Action handlers own response status, headers, body, redirects, HTMX response
headers, and form redisplay.

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
	NotFound            http.HandlerFunc
	MethodNotAllowed    http.HandlerFunc
	InternalServerError func(http.ResponseWriter, *http.Request, error)
}

type HandlerOptions struct {
	ErrorHandlers    ErrorHandlers
	InspectTemplates bool
}
```

`HandlerWithOptions` accepts this value and applies the hooks only to generated
route dispatch. Nil fields use the default generated behavior independently.

The generated not-found helper handles unmatched generated routes and invalid
dynamic path unescape failures.

The generated method-not-allowed helper runs after dispatch sets the `Allow`
header for the matched path.

The generated internal-server-error helper handles nil page, layout, or
fragment components and templ render failures. Custom hooks receive
`goldr.ErrNilComponent` for nil render units or the underlying templ render
error.

`InspectTemplates` is a development inspection option. When it is true,
generated dispatch wraps page, layout, and fragment render boundaries in paired
HTML comments with app-relative `app/routes/...` source paths. It does not
emit comments for redirects, plain text responses, error responses, or default
handlers. The default `Handler()` output has no inspector comments.

Generated applications also include `app/internal/goldrinspect/goldr_gen.go`.
The route dispatcher enables inspector mode by attaching generated context to
the request, then page, layout, and direct fragment routes call the generated
`goldrinspect.Wrap` helper. The helper is app-internal generated code, not a
public `goldr` package API.

Embedded fragment boundaries are opt-in at the template call site. For every
first-class fragment render unit outside the root route package, Goldr writes a
package-local `goldr_gen.go` containing a helper such as:

```go
func renderFragTable(component templ.Component) templ.Component
```

The helper name is derived from the fragment file identity:
`frag_table.go` / `frag_table.templ` maps to `renderFragTable`. Root-package
fragment helpers are emitted into the existing `app/routes/goldr_gen.go`, so
each route package has at most one Goldr-generated `goldr_gen.go` file.

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
declarations in one `frag_*.templ` file are internal details of that fragment
file; separately inspectable fragments require separate `frag_*.go` /
`frag_*.templ` render units.

Generated helper names are reserved by convention. Goldr does not preflight
collisions; normal Go compilation reports redeclarations in the affected route
package.

Action handlers are called directly and remain responsible for their own error
responses. Static assets are application-owned and are outside this generated
error hook boundary.
