# Mounted Kit Route Subtrees

Mounted Kit route subtrees let several live route owners reuse the same
filesystem-shaped route surface without giving the shared package live URLs.

Use this when repeated `KitRouteDef` child routes are the noise, not when a
single route needs one shared handler. For a single route, use
[`KitRouteDef`](routes.md#shared-kit-routes).

## Directory Shape

Live URLs still belong to `app/routes`:

```text
app/routes/admin/reports/route.go
app/routes/user/reports/route.go
```

Reusable route subtrees live under `app/mounts`:

```text
app/mounts/reports/
  route.go
  page.templ
  fragments.templ
```

Goldr never treats `app/mounts` as a live route tree. A mount subtree becomes
routable only when a real `app/routes` owner mounts it.

## Mount Owner

A real route directory mounts a subtree with `goldr.KitRouteMount[K]`:

```go
package reports

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	sharedreports "myapp/app/mounts/reports"
)

var Route = goldr.KitRouteMount[sharedreports.Kit]{
	New:   newReportKit,
	Mount: "reports",
}

func newReportKit(r *http.Request) sharedreports.Kit {
	return sharedreports.New(reportData(r))
}
```

`Mount` is a clean relative slash path under `app/mounts`. Each component must
be a lowercase Go-safe route directory name, so underscores are allowed and
still become hyphens in final URL paths. It is not a Go import path and must
not start with `/` or contain `..`.

The mount owner supplies `New func(*http.Request) K` as a local identifier in
the owner route package. Goldr calls it once per matched request and passes the
fresh kit value to the mounted page, fragment, or action handler.

Do not use an inline function literal for `New`. Goldr parses `route.go`
statically for route inspection and generated adapters; a named local
constructor keeps the mounted route declaration readable and supported.

## Mount Surface

Files under `app/mounts` use `goldr.KitRouteDef[K]` without `New`:

```go
package reports

import (
	"github.com/mobiletoly/goldr"
)

var Route = goldr.KitRouteDef[Kit]{
	Title: "Reports",
	Page:  goldr.KitPage(Kit.Page),
	Fragments: goldr.KitFragments[Kit]{
		goldr.KitFragment("table", Kit.Table),
	},
}
```

The mounted package can also own the kit type, handlers, and templ components
used by that reusable subtree. Child mount routes use the same `KitRouteDef`
shape when the shared subtree has real child URLs.

Keep only routes that are valid for every mount owner under `app/mounts`. If an
admin owner needs an extra child such as `/admin/reports/audit`, put that child
under the live `app/routes/admin/reports` owner. If shared mounted templates
need to link to that owner-only child, pass the URL through the owner-created
kit or page data.

`RouteDef` is invalid under `app/mounts`. `KitRouteDef.New` is also invalid
there because the live `KitRouteMount` owner supplies the request-scoped kit
constructor. Under `app/routes`, `KitRouteDef` requires `New`.

For live `KitRouteDef` declarations under `app/routes`, `New` may be a named
local function or method selector with type `func(*http.Request) K`. Inline
function literals are not supported.

## Generated Paths And Helpers

If both owners mount `reports`, Goldr generates live routes and helpers under
each owner:

```text
/admin/reports
/admin/reports/table

/user/reports
/user/reports/table
```

```go
urls.Admin.Reports.Path()
urls.Admin.Reports.Table.Path()

urls.User.Reports.Path()
urls.User.Reports.Table.Path()
```

Goldr also generates mount-relative helpers inside the referenced mounted
package:

```text
app/mounts/reports/goldr_gen.go
```

The live owner binds those helpers with the final live route helper:

```go
func reportData(_ *http.Request) reports.ReportData {
	return reports.ReportData{
		URLs: reports.NewGoldrMountURLs(urls.Admin.Reports),
	}
}
```

Mounted handlers and templates can then link within their own subtree without
knowing which owner mounted them:

```templ
<button hx-get={ report.URLs.Table.Path() } hx-target="#report-table-slot" hx-swap="innerHTML">
```

`NewGoldrMountURLs` accepts any route helper with `Path()`, including helpers
returned by `urls.WithBasePath`. It normalizes the helper path before storing
it. `Path()` returns the mount path itself, while child helpers append their
route segments. These helpers do not make `app/mounts` live; the final URL owner
is still the real `app/routes` mount owner.

## Layouts And Middleware

Mounted pages inherit real route ancestry layouts from the mount location.
Mounted subtrees may also define their own `layout.go` and `layout.templ`.
Those layouts are rebased under the mount path. When a real layout and a
mounted layout have the same final prefix, the real layout is outer and the
mounted layout is inner.

Middleware in `app/mounts` is not supported. Put middleware in the real
`app/routes` owner tree so request policy remains owned by live routes.

## Inspection And Collisions

Use route inspection after changing mounts:

```bash
go tool goldr routes list
go tool goldr routes list --mount reports
go tool goldr routes list --json
go tool goldr routes explain /admin/reports/table
```

Text output shows the mounted final path, helper, declaration kind
`mounted-kit`, mounted source file, and mount owner. JSON output includes the
same owner as structured mount evidence:

```json
"mount": {
  "path": "reports",
  "owner": "admin/reports/route.go"
}
```

`goldr check` and `goldr generate --check` fail when mounted and local routes
produce ambiguous runtime paths or when a mount path is missing or invalid.
