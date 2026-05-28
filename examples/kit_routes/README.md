# Kit Route Mount Example

This example shows a reusable report route subtree mounted under two
filesystem-owned routes:

```text
/admin/reports
/admin/reports/table
/admin/reports/audit
/user/reports
/user/reports/table
```

Both route directories declare their own `route.go` file with
`goldr.KitRouteMount[reports.Kit]`. The reusable route surface lives under
`app/mounts/reports`, while each live route owns its URL and adapts the request
into a route-specific report kit. The mounted subtree owns the route surface,
ordinary Go methods, and templ components.

The mounted subtree owns the page and the shared `table` fragment. Each mount
supplies request-scoped data and binds the mounted package's generated
`GoldrMountURLs` from its real route owner. The admin-only audit child lives in
the shared mounted subtree, but only the admin owner includes it with
`KitRouteMount.Routes`. The user owner does not expose it, so
`/user/reports/audit` is absent.

Run it with:

```sh
go run .
```

Inspect the route bindings with:

```sh
go tool goldr routes list
go tool goldr routes list --mount reports
go tool goldr routes explain /admin/reports
go tool goldr routes explain /admin/reports/table
go tool goldr routes explain /admin/reports/audit
go tool goldr routes explain /user/reports
```
