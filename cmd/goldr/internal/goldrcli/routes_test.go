package goldrcli

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRunRoutesPrintsRouteTable(t *testing.T) {
	root := t.TempDir()
	writeRouteListFixture(t, root)

	code, stdout, stderr := runGoldr(t, "routes", "list", "--app-root", root)

	if code != 0 {
		t.Fatalf("Run(routes list) exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	want := [][]string{
		{"KIND", "METHOD", "PATH", "PARAMS", "SOURCE", "OWNER", "DECL", "NAME", "TITLE", "LABELS", "NAV", "TRAIL_KEYS", "HELPER"},
		{"layout", "-", "/", "-", "layout.go", "-", "-", "-", "-", "-", "-", "-", "-"},
		{"page", "GET,HEAD", "/", "-", "route.go", "-", "local", "-", "-", "-", "-", "-", "urls.Root.Path()"},
		{"action", "POST", "/users/create", "-", "users/route.go:GoldrRoutePostCreate", "-", "local", "users.index", "Users", "app.nav=\"users\",app.permission=\"view\"", "-", "-", "urls.Users.Create.Path()"},
		{"fragment", "GET,HEAD", "/users/table", "-", "users/route.go", "-", "local", "users.index", "Users", "app.nav=\"users\",app.permission=\"view\"", "-", "-", "urls.Users.Table.Path()"},
		{"page", "GET,HEAD", "/users", "-", "users/route.go", "-", "local", "users.index", "Users", "app.nav=\"users\",app.permission=\"view\"", "-", "-", "urls.Users.Path()"},
		{"page", "GET,HEAD", "/users/{id}", "id", "users/by_id/route.go", "-", "local", "-", "-", "-", "-", "-", "urls.Users.ByID.Bind(id).Path()"},
	}
	requireRouteTableRows(t, stdout, want)
	requireMissingFile(t, filepath.Join(root, "app", "routes", "goldr_gen.go"))
	requireMissingFile(t, filepath.Join(root, "app", "urls", "goldr_gen.go"))
}

func writeRouteListFixture(t *testing.T, root string) {
	t.Helper()

	writeFile(t, root, "app/routes/layout.go", "package routes\n")
	writeFile(t, root, "app/routes/route.go", routeDeclarationSource("routes", "page", routeDeclarationOptions{Page: true}))
	writeFile(t, root, "app/routes/users/route.go", routeDeclarationSource("users", "page", routeDeclarationOptions{
		Page:      true,
		Fragments: []string{"table"},
		Actions:   []routeDeclarationAction{{Helper: "Action", Name: "create", Func: "postCreate"}},
		Name:      "users.index",
		Title:     "Users",
		Labels: []routeDeclarationLabel{
			{Key: "app.permission", Value: "view"},
			{Key: "app.nav", Value: "users"},
		},
	}))
	writeFile(t, root, "app/routes/users/by_id/route.go", routeDeclarationSource("by_id", "page", routeDeclarationOptions{Page: true}))
}

func TestRunRoutesPrintsJSON(t *testing.T) {
	root := t.TempDir()
	writeRouteListFixture(t, root)

	code, stdout, stderr := runGoldr(t, "routes", "list", "--app-root", root, "--json")

	if code != 0 {
		t.Fatalf("Run(routes list --json) exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if strings.Contains(stdout, "null") {
		t.Fatalf("stdout = %q, must not contain null arrays", stdout)
	}
	if strings.Contains(stdout, `"nav"`) {
		t.Fatalf("stdout = %q, must omit nav when route declarations have no nav metadata", stdout)
	}

	var rows []routeSurfaceJSONRow
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil {
		t.Fatalf("Unmarshal(routes list --json) error = %v; stdout = %q", err, stdout)
	}
	want := []routeSurfaceJSONRow{
		{Kind: "layout", Methods: []string{}, Path: "/", Params: []string{}, Source: "layout.go"},
		{Kind: "page", Methods: []string{"GET", "HEAD"}, Path: "/", Params: []string{}, Source: "route.go", Helper: "urls.Root.Path()", Declaration: &routeSurfaceJSONDeclaration{
			Source: "route.go",
			Kind:   "local",
			Labels: []routeSurfaceJSONLabel{},
			Page:   &routeSurfaceJSONPage{Handler: "page", Adapter: "GoldrRoutePage"},
		}},
		{Kind: "action", Methods: []string{"POST"}, Path: "/users/create", Params: []string{}, Source: "users/route.go:GoldrRoutePostCreate", Helper: "urls.Users.Create.Path()", Declaration: &routeSurfaceJSONDeclaration{
			Source: "users/route.go",
			Kind:   "local",
			Name:   "users.index",
			Title:  "Users",
			Labels: []routeSurfaceJSONLabel{
				{Key: "app.nav", Value: "users"},
				{Key: "app.permission", Value: "view"},
			},
			Action: &routeSurfaceJSONAction{Method: "POST", Name: "create", Segment: "create", Handler: "postCreate", Adapter: "GoldrRoutePostCreate"},
		}},
		{Kind: "fragment", Methods: []string{"GET", "HEAD"}, Path: "/users/table", Params: []string{}, Source: "users/route.go", Helper: "urls.Users.Table.Path()", Declaration: &routeSurfaceJSONDeclaration{
			Source: "users/route.go",
			Kind:   "local",
			Name:   "users.index",
			Title:  "Users",
			Labels: []routeSurfaceJSONLabel{
				{Key: "app.nav", Value: "users"},
				{Key: "app.permission", Value: "view"},
			},
			Fragment: &routeSurfaceJSONFragment{Name: "table", Segment: "table", Handler: "fragTable", Adapter: "GoldrRouteFragTable"},
		}},
		{Kind: "page", Methods: []string{"GET", "HEAD"}, Path: "/users", Params: []string{}, Source: "users/route.go", Helper: "urls.Users.Path()", Declaration: &routeSurfaceJSONDeclaration{
			Source: "users/route.go",
			Kind:   "local",
			Name:   "users.index",
			Title:  "Users",
			Labels: []routeSurfaceJSONLabel{
				{Key: "app.nav", Value: "users"},
				{Key: "app.permission", Value: "view"},
			},
			Page: &routeSurfaceJSONPage{Handler: "page", Adapter: "GoldrRoutePage"},
		}},
		{Kind: "page", Methods: []string{"GET", "HEAD"}, Path: "/users/{id}", Params: []string{"id"}, Source: "users/by_id/route.go", Helper: "urls.Users.ByID.Bind(id).Path()", Declaration: &routeSurfaceJSONDeclaration{
			Source: "users/by_id/route.go",
			Kind:   "local",
			Labels: []routeSurfaceJSONLabel{},
			Page:   &routeSurfaceJSONPage{Handler: "page", Adapter: "GoldrRoutePage"},
		}},
	}
	if len(rows) != len(want) {
		t.Fatalf("JSON rows = %#v, want %#v", rows, want)
	}
	for index := range want {
		if !reflect.DeepEqual(rows[index], want[index]) {
			t.Fatalf("JSON row %d = %#v, want %#v", index, rows[index], want[index])
		}
		if rows[index].Methods == nil {
			t.Fatalf("JSON row %d methods = nil, want empty array when empty", index)
		}
		if rows[index].Params == nil {
			t.Fatalf("JSON row %d params = nil, want empty array when empty", index)
		}
	}
	requireMissingFile(t, filepath.Join(root, "app", "routes", "goldr_gen.go"))
	requireMissingFile(t, filepath.Join(root, "app", "urls", "goldr_gen.go"))
}

func TestRunRoutesInspectIndexFragment(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "app/routes/users/status_options/route.go", routeDeclarationSource("status_options", "page", routeDeclarationOptions{
		IndexFragment: true,
	}))

	code, listOut, listErr := runGoldr(t, "routes", "list", "--app-root", root)
	if code != 0 {
		t.Fatalf("Run(routes list) exit code = %d, want 0; stderr = %q", code, listErr)
	}
	requireRouteTableRows(t, listOut, [][]string{
		{"KIND", "METHOD", "PATH", "PARAMS", "SOURCE", "OWNER", "DECL", "NAME", "TITLE", "LABELS", "NAV", "TRAIL_KEYS", "HELPER"},
		{"fragment", "GET,HEAD", "/users/status-options", "-", "users/status_options/route.go", "-", "local", "-", "-", "-", "-", "-", "urls.Users.StatusOptions.Path()"},
	})

	jsonOut := runGoldrDeterministic(t, "routes list --json", "routes", "list", "--app-root", root, "--json")
	var rows []routeSurfaceJSONRow
	if err := json.Unmarshal([]byte(jsonOut), &rows); err != nil {
		t.Fatalf("Unmarshal(routes list --json) error = %v; stdout = %q", err, jsonOut)
	}
	requireRouteJSONContainsRow(t, rows, routeSurfaceJSONRow{
		Kind:    "fragment",
		Methods: []string{"GET", "HEAD"},
		Path:    "/users/status-options",
		Params:  []string{},
		Source:  "users/status_options/route.go",
		Helper:  "urls.Users.StatusOptions.Path()",
		Declaration: &routeSurfaceJSONDeclaration{
			Source: "users/status_options/route.go",
			Kind:   "local",
			Labels: []routeSurfaceJSONLabel{},
			Fragment: &routeSurfaceJSONFragment{
				Name:    "index",
				Index:   true,
				Handler: "fragIndex",
				Adapter: "GoldrRouteFragIndex",
			},
		},
	})

	code, explainOut, explainErr := runGoldr(t, "routes", "explain", "--app-root", root, "/users/status-options")
	if code != 0 {
		t.Fatalf("Run(routes explain) exit code = %d, want 0; stderr = %q", code, explainErr)
	}
	for _, want := range []string{
		"/users/status-options  GET",
		"  fragment /users/status-options",
		"IMPLEMENTATION",
		"  fragment index -> /users/status-options",
		"  handler  fragIndex -> GoldrRouteFragIndex",
		"LAYOUT STACK",
		"  not layout-wrapped",
	} {
		if !strings.Contains(explainOut, want) {
			t.Fatalf("routes explain output = %q, want %q", explainOut, want)
		}
	}
}

func TestRunRoutesExplainNavigationDeclaration(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "app/routes/analytics/route.go", `package analytics

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"example.com/app/urls"
)

		var Route = goldr.RouteDef{
			Page: page,
			Nav:  goldr.RouteNav{Label: "Analytics"},
			Destinations: goldr.Destinations{
				"project": goldr.To(urls.Projects.ByID).TrailKey("workflow-a"),
			},
	}

func page(r *http.Request) goldr.PageRouteResponse {
	return goldr.Text{Body: "analytics"}
}
`)
	writeFile(t, root, "app/routes/projects/by_id/route.go", `package by_id

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

	var Route = goldr.RouteDef{
		Page: page,
		Nav:  goldr.RouteNav{Key: "project"},
	}

func page(r *http.Request) goldr.PageRouteResponse {
	return goldr.Text{Body: "project"}
}
`)

	code, explainOut, explainErr := runGoldr(t, "routes", "explain", "--app-root", root, "/analytics")
	if code != 0 {
		t.Fatalf("Run(routes explain) exit code = %d, want 0; stderr = %q", code, explainErr)
	}
	for _, want := range []string{
		"  nav      label=\"Analytics\"",
		"DESTINATIONS",
		"  project            urls.Analytics.Destinations.Project -> urls.Projects.ByID trail_key=workflow-a",
	} {
		if !strings.Contains(explainOut, want) {
			t.Fatalf("routes explain output = %q, want %q", explainOut, want)
		}
	}

	jsonOut := runGoldrDeterministic(t, "routes list --json", "routes", "list", "--app-root", root, "--json")
	var rows []routeSurfaceJSONRow
	if err := json.Unmarshal([]byte(jsonOut), &rows); err != nil {
		t.Fatalf("Unmarshal(routes list --json) error = %v; stdout = %q", err, jsonOut)
	}
	requireRouteJSONContainsRow(t, rows, routeSurfaceJSONRow{
		Kind:    "page",
		Methods: []string{"GET", "HEAD"},
		Path:    "/analytics",
		Params:  []string{},
		Source:  "analytics/route.go",
		Helper:  "urls.Analytics.Path()",
		Declaration: &routeSurfaceJSONDeclaration{
			Source: "analytics/route.go",
			Kind:   "local",
			Labels: []routeSurfaceJSONLabel{},
			Nav:    &routeSurfaceJSONNav{Label: "Analytics"},
			Destinations: []routeSurfaceJSONDestination{{
				Name:     "project",
				Helper:   "urls.Analytics.Destinations.Project",
				Target:   "urls.Projects.ByID",
				TrailKey: "workflow-a",
			}},
			Page: &routeSurfaceJSONPage{Handler: "page", Adapter: "GoldrRoutePage"},
		},
	})
	requireRouteJSONContainsRow(t, rows, routeSurfaceJSONRow{
		Kind:    "page",
		Methods: []string{"GET", "HEAD"},
		Path:    "/projects/{id}",
		Params:  []string{"id"},
		Source:  "projects/by_id/route.go",
		Helper:  "urls.Projects.ByID.Bind(id).Path()",
		Declaration: &routeSurfaceJSONDeclaration{
			Source:    "projects/by_id/route.go",
			Kind:      "local",
			Labels:    []routeSurfaceJSONLabel{},
			Nav:       &routeSurfaceJSONNav{Key: "project"},
			TrailKeys: []string{"workflow-a"},
			InboundDestinations: []routeSurfaceJSONInboundDestination{{
				Source:   "/analytics",
				Name:     "project",
				Helper:   "urls.Analytics.Destinations.Project",
				TrailKey: "workflow-a",
			}},
			Page: &routeSurfaceJSONPage{Handler: "page", Adapter: "GoldrRoutePage"},
		},
	})
}

func TestRunRoutesExplainMountedNavigationDeclaration(t *testing.T) {
	root := t.TempDir()
	writeTemplToolModule(t, root, "example.com/mountednavapp")
	writeFile(t, root, "app/routes/admin/reports/route.go", `package reports

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	sharedreports "example.com/mountednavapp/app/mounts/reports"
	"example.com/mountednavapp/app/urls"
)

var Route = goldr.KitRouteMount[sharedreports.Kit]{
	New: newKit,
	Mount: "reports",
	Routes: goldr.MountRoutes{
		{
				Path: "/",
				Destinations: goldr.Destinations{
					"audit": goldr.To(urls.Admin.Reports.Audit).TrailKey("admin-reports"),
				},
		},
			{
				Path: "/audit",
			},
		},
	}

func newKit(_ *http.Request) sharedreports.Kit {
	return sharedreports.Kit{}
}
`)
	writeFile(t, root, "app/mounts/reports/route.go", `package reports

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

type Kit struct{}

var Route = goldr.KitRouteDef[Kit]{
	Page: Kit.Page,
}

func (kit Kit) Page(_ *http.Request) goldr.PageRouteResponse {
	return goldr.Text{Body: "ok"}
}
`)
	writeFile(t, root, "app/mounts/reports/audit/route.go", `package audit

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	sharedreports "example.com/mountednavapp/app/mounts/reports"
)

	var Route = goldr.KitRouteDef[sharedreports.Kit]{
		Nav:  goldr.RouteNav{Label: "Audit"},
		Page: sharedreports.Kit.Audit,
	}

func (kit sharedreports.Kit) Audit(_ *http.Request) goldr.PageRouteResponse {
	return goldr.Text{Body: "ok"}
}
`)

	code, rootOut, rootErr := runGoldr(t, "routes", "explain", "--app-root", root, "/admin/reports")
	if code != 0 {
		t.Fatalf("Run(routes explain root) exit code = %d, want 0; stderr = %q", code, rootErr)
	}
	for _, want := range []string{
		"DESTINATIONS",
		"  audit              urls.Admin.Reports.Destinations.Audit -> urls.Admin.Reports.Audit trail_key=admin-reports",
	} {
		if !strings.Contains(rootOut, want) {
			t.Fatalf("routes explain root output = %q, want %q", rootOut, want)
		}
	}

	code, auditOut, auditErr := runGoldr(t, "routes", "explain", "--app-root", root, "/admin/reports/audit")
	if code != 0 {
		t.Fatalf("Run(routes explain audit) exit code = %d, want 0; stderr = %q", code, auditErr)
	}
	for _, want := range []string{
		"/admin/reports/audit  GET",
		"  nav      label=\"Audit\"",
		"  trailkey admin-reports",
		"INBOUND DESTINATIONS",
		"  audit              urls.Admin.Reports.Destinations.Audit -> /admin/reports trail_key=admin-reports",
		"app/mounts/reports/audit/route.go",
	} {
		if !strings.Contains(auditOut, want) {
			t.Fatalf("routes explain audit output = %q, want %q", auditOut, want)
		}
	}
}

func TestRunRoutesPrintsKitDeclarationJSON(t *testing.T) {
	root := t.TempDir()
	writeKitRouteFixture(t, root)

	stdout := runGoldrDeterministic(t, "routes list --json", "routes", "list", "--app-root", root, "--json")

	var rows []routeSurfaceJSONRow
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil {
		t.Fatalf("Unmarshal(routes list --json) error = %v; stdout = %q", err, stdout)
	}
	requireRouteJSONContainsRow(t, rows, routeSurfaceJSONRow{
		Kind:    "action",
		Methods: []string{"POST"},
		Path:    "/reports/export",
		Params:  []string{},
		Source:  "reports/route.go:GoldrRoutePostExport",
		Helper:  "urls.Reports.Export.Path()",
		Declaration: &routeSurfaceJSONDeclaration{
			Source: "reports/route.go",
			Kind:   "kit",
			Name:   "reports.index",
			Title:  "Reports",
			Labels: []routeSurfaceJSONLabel{{Key: "app.nav", Value: "reports"}},
			Kit: &routeSurfaceJSONKit{
				KitType: "Kit",
				New:     "New",
			},
			Action: &routeSurfaceJSONAction{
				Method:  "POST",
				Name:    "export",
				Segment: "export",
				Handler: "Kit.PostExport",
				Adapter: "GoldrRoutePostExport",
			},
		},
	})
}

func TestRunRoutesListFiltersMountedRoutes(t *testing.T) {
	root := tempMountedRouteApp(t)

	code, stdout, stderr := runGoldr(t, "routes", "list", "--app-root", root, "--mount", "reports")

	if code != 0 {
		t.Fatalf("Run(routes list --mount) exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	requireRouteTableRows(t, stdout, [][]string{
		{"KIND", "METHOD", "PATH", "PARAMS", "SOURCE", "OWNER", "DECL", "NAME", "TITLE", "LABELS", "NAV", "TRAIL_KEYS", "HELPER", "STATUS"},
		{"fragment", "GET,HEAD", "/admin/reports/table", "-", "../mounts/reports/route.go", "admin/reports/route.go", "mounted-kit", "-", "-", "-", "-", "-", "urls.Admin.Reports.Table.Path()", "included"},
		{"page", "GET,HEAD", "/admin/reports", "-", "../mounts/reports/route.go", "admin/reports/route.go", "mounted-kit", "-", "-", "-", "-", "-", "urls.Admin.Reports.Path()", "included"},
	})
}

func TestRunRoutesListMountFilterPrintsEmptyInventory(t *testing.T) {
	root := tempMountedRouteApp(t)

	code, stdout, stderr := runGoldr(t, "routes", "list", "--app-root", root, "--mount", "missing")

	if code != 0 {
		t.Fatalf("Run(routes list --mount missing) exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	requireRouteTableRows(t, stdout, [][]string{
		{"KIND", "METHOD", "PATH", "PARAMS", "SOURCE", "OWNER", "DECL", "NAME", "TITLE", "LABELS", "NAV", "TRAIL_KEYS", "HELPER"},
	})
}

func TestRunRoutesListMountFilterShowsExcludedMountedRoutes(t *testing.T) {
	root := tempSelectiveMountedRouteApp(t)

	code, stdout, stderr := runGoldr(t, "routes", "list", "--app-root", root, "--mount", "reports")

	if code != 0 {
		t.Fatalf("Run(routes list --mount selective) exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	requireRouteTableRows(t, stdout, [][]string{
		{"KIND", "METHOD", "PATH", "PARAMS", "SOURCE", "OWNER", "DECL", "NAME", "TITLE", "LABELS", "NAV", "TRAIL_KEYS", "HELPER", "STATUS"},
		{"route", "-", "/admin/reports/audit", "-", "../mounts/reports/audit/route.go", "admin/reports/route.go", "mounted-kit", "-", "-", "-", "-", "-", "-", "excluded"},
		{"page", "GET,HEAD", "/admin/reports", "-", "../mounts/reports/route.go", "admin/reports/route.go", "mounted-kit", "-", "-", "-", "-", "-", "urls.Admin.Reports.Path()", "included"},
	})
}

func TestRunRoutesListMountFilterJSON(t *testing.T) {
	root := tempMountedRouteApp(t)

	stdout := runGoldrDeterministic(t, "routes list --mount --json", "routes", "list", "--app-root", root, "--mount", "reports", "--json")

	var rows []routeSurfaceJSONRow
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil {
		t.Fatalf("Unmarshal(routes list --mount --json) error = %v; stdout = %q", err, stdout)
	}
	if len(rows) != 2 {
		t.Fatalf("JSON rows = %#v, want two mounted rows", rows)
	}
	requireRouteJSONContainsRow(t, rows, routeSurfaceJSONRow{
		Kind:    "page",
		Methods: []string{"GET", "HEAD"},
		Path:    "/admin/reports",
		Params:  []string{},
		Source:  "../mounts/reports/route.go",
		Helper:  "urls.Admin.Reports.Path()",
		Status:  "included",
		Declaration: &routeSurfaceJSONDeclaration{
			Source: "../mounts/reports/route.go",
			Kind:   "mounted-kit",
			Labels: []routeSurfaceJSONLabel{},
			Mount:  &routeSurfaceJSONMount{Path: "reports", Owner: "admin/reports/route.go"},
			Kit:    &routeSurfaceJSONKit{KitType: "sharedreports.Kit", New: "newKit"},
			Page:   &routeSurfaceJSONPage{Handler: "Kit.Page", Adapter: "GoldrRouteMountReportsPage"},
		},
	})
}

func TestRunRoutesEscapesControlCharactersInDeclarationText(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "app/routes/route.go", `package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

var Route = goldr.RouteDef{
	Name:  "home\nroot",
	Title: "Home\nBad",
	Page:  page,
	Meta: goldr.RouteMeta{
		Labels: map[string]string{
			"app\nnav": "home",
		},
	},
}

func page(_ *http.Request) goldr.PageRouteResponse {
	return goldr.Text{Body: "ok"}
}
`)

	code, listOut, listErr := runGoldr(t, "routes", "list", "--app-root", root)
	if code != 0 {
		t.Fatalf("Run(routes list) exit code = %d, want 0; stderr = %q", code, listErr)
	}
	if listErr != "" {
		t.Fatalf("routes list stderr = %q, want empty", listErr)
	}
	requireRouteTableRows(t, listOut, [][]string{
		{"KIND", "METHOD", "PATH", "PARAMS", "SOURCE", "OWNER", "DECL", "NAME", "TITLE", "LABELS", "NAV", "TRAIL_KEYS", "HELPER"},
		{"page", "GET,HEAD", "/", "-", "route.go", "-", "local", `"home\nroot"`, `"Home\nBad"`, `"app\nnav"="home"`, "-", "-", "urls.Root.Path()"},
	})

	code, explainOut, explainErr := runGoldr(t, "routes", "explain", "--app-root", root, "/")
	if code != 0 {
		t.Fatalf("Run(routes explain) exit code = %d, want 0; stderr = %q", code, explainErr)
	}
	if explainErr != "" {
		t.Fatalf("routes explain stderr = %q, want empty", explainErr)
	}
	for _, want := range []string{
		"  name     \"home\\nroot\"",
		"  title    \"Home\\nBad\"",
		"  labels   \"app\\nnav\"=\"home\"",
	} {
		if !strings.Contains(explainOut, want) {
			t.Fatalf("routes explain stdout = %q, want %q", explainOut, want)
		}
	}
	if strings.Contains(explainOut, "Home\nBad") {
		t.Fatalf("routes explain stdout = %q, contains unescaped title newline", explainOut)
	}
}

func writeKitRouteFixture(t *testing.T, root string) {
	t.Helper()

	writeFile(t, root, "app/routes/reports/route.go", `package reports

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

type Context struct{}
type Kit struct{}

func New(r *http.Request) Kit {
	_ = routeContext(r)
	return Kit{}
}

func routeContext(*http.Request) Context { return Context{} }

var Route = goldr.KitRouteDef[Kit]{
	Name:  "reports.index",
	Title: "Reports",
	New:   New,
	Page:  Kit.Page,
	Fragments: goldr.KitFragments[Kit]{
		goldr.KitFragmentRoute("/panel", Kit.Panel),
	},
	Actions: goldr.KitActions[Kit]{
		goldr.KitAction(http.MethodPost, "/export", Kit.PostExport),
	},
	Meta: goldr.RouteMeta{
		Labels: map[string]string{
			"app.nav": "reports",
		},
	},
}
`)
}

func TestRunRoutesFullFeatureOutputIsDeterministic(t *testing.T) {
	root := fullFeatureRoot(t)
	stdout := runGoldrDeterministic(t, "routes list", "routes", "list", "--app-root", root)

	rows := routeTableRows(t, stdout)
	for _, want := range [][]string{
		{"layout", "-", "/", "-", "layout.go", "-", "-", "-", "-", "-", "-", "-", "-"},
		{"page", "GET,HEAD", "/", "-", "route.go", "-", "local", "-", "-", "-", "-", "-", "urls.Root.Path()"},
		{"page", "GET,HEAD", "/settings", "-", "settings/route.go", "-", "local", "-", "-", "-", "-", "-", "urls.Settings.Path()"},
		{"layout", "-", "/users", "-", "users/layout.go", "-", "-", "-", "-", "-", "-", "-", "-"},
		{"page", "GET,HEAD", "/users/{id}", "id", "users/by_id/route.go", "-", "local", "-", "-", "-", "-", "-", "urls.Users.ByID.Bind(id).Path()"},
		{"action", "POST", "/users/create", "-", "users/route.go:GoldrRoutePostCreate", "-", "local", "-", "-", "-", "-", "-", "urls.Users.Create.Path()"},
		{"action", "POST", "/users/save-preview", "-", "users/route.go:GoldrRoutePostSavePreview", "-", "local", "-", "-", "-", "-", "-", "urls.Users.SavePreview.Path()"},
	} {
		requireRouteTableContainsRow(t, rows, want)
	}
}

func TestRunRoutesFullFeatureJSONOutputIsDeterministic(t *testing.T) {
	root := fullFeatureRoot(t)
	stdout := runGoldrDeterministic(t, "routes list --json", "routes", "list", "--app-root", root, "--json")

	var rows []routeSurfaceJSONRow
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil {
		t.Fatalf("Unmarshal(routes list --json) error = %v; stdout = %q", err, stdout)
	}
	requireRouteJSONContainsRow(t, rows, routeSurfaceJSONRow{
		Kind:    "layout",
		Methods: []string{},
		Path:    "/",
		Params:  []string{},
		Source:  "layout.go",
	})
	requireRouteJSONContainsRow(t, rows, routeSurfaceJSONRow{
		Kind:    "page",
		Methods: []string{"GET", "HEAD"},
		Path:    "/users/{id}",
		Params:  []string{"id"},
		Source:  "users/by_id/route.go",
		Helper:  "urls.Users.ByID.Bind(id).Path()",
	})
	requireRouteJSONContainsRow(t, rows, routeSurfaceJSONRow{
		Kind:    "action",
		Methods: []string{"POST"},
		Path:    "/users/save-preview",
		Params:  []string{},
		Source:  "users/route.go:GoldrRoutePostSavePreview",
		Helper:  "urls.Users.SavePreview.Path()",
	})
}

func TestRunRoutesFullFeatureLayoutMapOutputIsDeterministic(t *testing.T) {
	root := fullFeatureRoot(t)
	stdout := runGoldrDeterministic(t, "routes layouts", "routes", "layouts", "--app-root", root)

	want := fullFeatureLayoutMapOutput(t)
	if stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
}

func TestRunRoutesExplainFullURLDynamicPage(t *testing.T) {
	root := fullFeatureRoot(t)
	source := func(name string) string {
		return fullFeatureRouteSourcePath(name)
	}

	code, stdout, stderr := runGoldr(t, "routes", "explain", "--app-root", root, "http://127.0.0.1:8080/users/a%2Fb?tab=profile#details")

	if code != 0 {
		t.Fatalf("Run(routes explain) exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	want := strings.Join([]string{
		"/users/a%2Fb  GET",
		"",
		"MATCH",
		"  page     /users/{id}",
		"  source   " + source("users/by_id/route.go"),
		"  params   id = \"a/b\"",
		"",
		"DECLARATION",
		"  kind     local",
		"  source   " + source("users/by_id/route.go"),
		"  name     -",
		"  title    -",
		"  labels   -",
		"",
		"IMPLEMENTATION",
		"  page     Page -> GoldrRoutePage",
		"",
		"LAYOUT STACK",
		"  /      " + source("layout.go"),
		"  /users " + source("users/layout.go"),
	}, "\n") + "\n"
	if stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
}

func TestRunRoutesExplainHonorsRootFlag(t *testing.T) {
	root := fullFeatureRoot(t)

	code, stdout, stderr := runGoldr(t, "routes", "explain", "--app-root", root, "/users/7")

	if code != 0 {
		t.Fatalf("Run(routes explain --app-root) exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "  page     /users/{id}") {
		t.Fatalf("stdout = %q, want users by_id route", stdout)
	}
}

func TestRunRoutesExplainActionShowsLayoutStack(t *testing.T) {
	root := fullFeatureRoot(t)
	source := func(name string) string {
		return fullFeatureRouteSourcePath(name)
	}

	code, stdout, stderr := runGoldr(t, "routes", "explain", "--app-root", root, "--method", "POST", "/users/create")

	if code != 0 {
		t.Fatalf("Run(routes explain action) exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"/users/create  POST",
		"  action   /users/create",
		"  source   " + source("users/route.go") + " (GoldrRoutePostCreate)",
		"LAYOUT STACK",
		"  /      " + source("layout.go"),
		"  /users " + source("users/layout.go"),
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout = %q, want %q", stdout, want)
		}
	}
}

func TestRunRoutesExplainFragmentShowsDeclaration(t *testing.T) {
	root := fullFeatureRoot(t)
	source := func(name string) string {
		return fullFeatureRouteSourcePath(name)
	}

	code, stdout, stderr := runGoldr(t, "routes", "explain", "--app-root", root, "/users/table")

	if code != 0 {
		t.Fatalf("Run(routes explain fragment) exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"/users/table  GET",
		"  fragment /users/table",
		"  source   " + source("users/route.go"),
		"DECLARATION",
		"  kind     local",
		"IMPLEMENTATION",
		"  fragment table -> /users/table",
		"  handler  FragTable -> GoldrRouteFragTable",
		"LAYOUT STACK",
		"  not layout-wrapped",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout = %q, want %q", stdout, want)
		}
	}
}

func TestRunRoutesExplainKitShowsDeclaration(t *testing.T) {
	root := t.TempDir()
	writeKitRouteFixture(t, root)

	code, stdout, stderr := runGoldr(t, "routes", "explain", "--app-root", root, "/reports")

	if code != 0 {
		t.Fatalf("Run(routes explain kit) exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"/reports  GET",
		"  page     /reports",
		"DECLARATION",
		"  kind     kit",
		"  name     reports.index",
		"  title    Reports",
		"  labels   app.nav=\"reports\"",
		"  kit      Kit",
		"  new      New",
		"IMPLEMENTATION",
		"  page     Kit.Page -> GoldrRoutePage",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout = %q, want %q", stdout, want)
		}
	}
}

func TestRunRoutesExplainReportsFailures(t *testing.T) {
	root := fullFeatureRoot(t)

	tests := []struct {
		name  string
		args  []string
		wants []string
	}{
		{
			name:  "method mismatch",
			args:  []string{"routes", "explain", "--app-root", root, "--method", "DELETE", "/users/7"},
			wants: []string{"goldr routes explain:", "DELETE /users/7", "method not allowed", "allowed: GET,HEAD"},
		},
		{
			name:  "no match",
			args:  []string{"routes", "explain", "--app-root", root, "/missing"},
			wants: []string{"goldr routes explain:", "GET /missing", "no route matches path"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runGoldr(t, tt.args...)

			if code != 1 {
				t.Fatalf("Run(%s) exit code = %d, want 1", tt.name, code)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			for _, want := range tt.wants {
				if !strings.Contains(stderr, want) {
					t.Fatalf("stderr = %q, want %q", stderr, want)
				}
			}
		})
	}
}

func TestRunRoutesOldModeFlagsFail(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "json", args: []string{"routes", "--json"}, want: "json"},
		{name: "layouts", args: []string{"routes", "--layouts"}, want: "layouts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runGoldr(t, tt.args...)
			if code == 0 {
				t.Fatalf("Run(%s) exit code = 0, want failure", strings.Join(tt.args, " "))
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if !strings.Contains(stderr, tt.want) {
				t.Fatalf("stderr = %q, want %q", stderr, tt.want)
			}
		})
	}
}

func TestRunRoutesReportsInvalidRouteNames(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "app/routes/Users/page.go", "package Users\n")

	requireCommandArgsFailureContains(t, []string{"routes", "list", "--app-root", root}, "goldr routes list:", "app/routes/Users", "static route directories must use lowercase Go-safe names")
}

func TestRunRoutesReportsURLHelperGenerationErrors(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "app/routes/users/route.go", routeDeclarationSource("users", "page", routeDeclarationOptions{
		Actions: []routeDeclarationAction{{Helper: "Action", Name: "path", Func: "postPath"}},
	}))

	requireCommandArgsFailureContains(t, []string{"routes", "list", "--app-root", root}, "goldr routes list:", "ambiguous URL helper", "Path method")
}

func TestRunRoutesJSONReportsErrorsToStderr(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "app/routes/Users/page.go", "package Users\n")

	code, stdout, stderr := runGoldr(t, "routes", "list", "--app-root", root, "--json")
	if code != 1 {
		t.Fatalf("Run(routes list --json) exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	for _, want := range []string{"goldr routes list:", "app/routes/Users", "static route directories must use lowercase Go-safe names"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr = %q, want %q", stderr, want)
		}
	}
}

func requireRouteTableRows(t *testing.T, stdout string, want [][]string) {
	t.Helper()

	got := routeTableRows(t, stdout)
	if len(got) != len(want) {
		t.Fatalf("route table rows = %#v, want %#v", got, want)
	}
	for index := range want {
		if strings.Join(got[index], "\x00") != strings.Join(want[index], "\x00") {
			t.Fatalf("route table row %d = %#v, want %#v", index, got[index], want[index])
		}
	}
}

func routeTableRows(t *testing.T, stdout string) [][]string {
	t.Helper()

	trimmed := strings.TrimSuffix(stdout, "\n")
	if trimmed == "" {
		t.Fatalf("stdout is empty, want route table")
	}
	lines := strings.Split(trimmed, "\n")
	rows := make([][]string, 0, len(lines))
	for _, line := range lines {
		rows = append(rows, strings.Fields(line))
	}
	return rows
}

func requireRouteTableContainsRow(t *testing.T, rows [][]string, want []string) {
	t.Helper()

	wantText := strings.Join(want, "\x00")
	for _, row := range rows {
		if strings.Join(row, "\x00") == wantText {
			return
		}
	}
	t.Fatalf("route table rows = %#v, want row %#v", rows, want)
}

type routeSurfaceJSONRow struct {
	Kind        string                       `json:"kind"`
	Methods     []string                     `json:"methods"`
	Path        string                       `json:"path"`
	Params      []string                     `json:"params"`
	Source      string                       `json:"source"`
	Helper      string                       `json:"helper"`
	Status      string                       `json:"status,omitempty"`
	Declaration *routeSurfaceJSONDeclaration `json:"declaration,omitempty"`
}

type routeSurfaceJSONDeclaration struct {
	Source              string                               `json:"source"`
	Kind                string                               `json:"kind"`
	Name                string                               `json:"name"`
	Title               string                               `json:"title"`
	Labels              []routeSurfaceJSONLabel              `json:"labels"`
	Nav                 *routeSurfaceJSONNav                 `json:"nav,omitempty"`
	TrailKeys           []string                             `json:"trail_keys,omitempty"`
	Destinations        []routeSurfaceJSONDestination        `json:"destinations,omitempty"`
	InboundDestinations []routeSurfaceJSONInboundDestination `json:"inbound_destinations,omitempty"`
	Mount               *routeSurfaceJSONMount               `json:"mount,omitempty"`
	Kit                 *routeSurfaceJSONKit                 `json:"kit,omitempty"`
	Page                *routeSurfaceJSONPage                `json:"page,omitempty"`
	Fragment            *routeSurfaceJSONFragment            `json:"fragment,omitempty"`
	Action              *routeSurfaceJSONAction              `json:"action,omitempty"`
}

type routeSurfaceJSONLabel struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type routeSurfaceJSONNav struct {
	Label string `json:"label,omitempty"`
	Key   string `json:"key,omitempty"`
}

type routeSurfaceJSONDestination struct {
	Name     string `json:"name"`
	Helper   string `json:"helper"`
	Target   string `json:"target"`
	TrailKey string `json:"trail_key,omitempty"`
}

type routeSurfaceJSONInboundDestination struct {
	Source   string `json:"source"`
	Name     string `json:"name"`
	Helper   string `json:"helper"`
	TrailKey string `json:"trail_key"`
}

type routeSurfaceJSONKit struct {
	KitType string `json:"kit_type"`
	New     string `json:"new"`
}

type routeSurfaceJSONMount struct {
	Path  string `json:"path"`
	Owner string `json:"owner"`
}

type routeSurfaceJSONPage struct {
	Handler string `json:"handler"`
	Adapter string `json:"adapter"`
}

type routeSurfaceJSONFragment struct {
	Name    string `json:"name"`
	Segment string `json:"segment"`
	Index   bool   `json:"index"`
	Handler string `json:"handler"`
	Adapter string `json:"adapter"`
}

type routeSurfaceJSONAction struct {
	Method  string `json:"method"`
	Name    string `json:"name"`
	Segment string `json:"segment"`
	Index   bool   `json:"index"`
	Handler string `json:"handler"`
	Adapter string `json:"adapter"`
}

func requireRouteJSONContainsRow(t *testing.T, rows []routeSurfaceJSONRow, want routeSurfaceJSONRow) {
	t.Helper()

	for _, row := range rows {
		if row.Kind == want.Kind &&
			strings.Join(row.Methods, "\x00") == strings.Join(want.Methods, "\x00") &&
			row.Path == want.Path &&
			strings.Join(row.Params, "\x00") == strings.Join(want.Params, "\x00") &&
			row.Source == want.Source &&
			row.Helper == want.Helper &&
			(want.Status == "" || row.Status == want.Status) &&
			(want.Declaration == nil || reflect.DeepEqual(row.Declaration, want.Declaration)) {
			return
		}
	}
	t.Fatalf("route JSON rows = %#v, want row %#v", rows, want)
}

func fullFeatureLayoutMapOutput(t *testing.T) string {
	t.Helper()

	source := func(name string) string {
		return fullFeatureRouteSourcePath(name)
	}
	rootPath := fullFeatureRoutesDisplayRoot()
	if !strings.HasSuffix(rootPath, "/") {
		rootPath += "/"
	}
	last := "\u2514\u2500 "
	mid := "\u251c\u2500 "
	pipe := "\u2502"
	lines := []string{
		"Layout map",
		"",
		rootPath,
		last + "/  layout: " + source("layout.go"),
		"   " + mid + "page: GET,HEAD /  " + source("route.go"),
		"   " + mid + "admin/",
		"   " + pipe + "  " + last + "page: GET,HEAD /admin  " + source("admin/route.go"),
		"   " + mid + "protected_resource_demo/",
		"   " + pipe + "  " + mid + "page: GET,HEAD /protected-resource-demo  " + source("protected_resource_demo/route.go"),
		"   " + pipe + "  " + mid + "action (layout-aware): POST /protected-resource-demo/reveal-secret  " + source("protected_resource_demo/route.go") + " (GoldrRoutePostRevealSecret)",
		"   " + pipe + "  " + last + "action (layout-aware): POST /protected-resource-demo/sign-out  " + source("protected_resource_demo/route.go") + " (GoldrRoutePostSignOut)",
		"   " + mid + "settings/",
		"   " + pipe + "  " + last + "page: GET,HEAD /settings  " + source("settings/route.go"),
		"   " + mid + "sign_in/",
		"   " + pipe + "  " + mid + "page: GET,HEAD /sign-in  " + source("sign_in/route.go"),
		"   " + pipe + "  " + last + "action (layout-aware): POST /sign-in  " + source("sign_in/route.go") + " (GoldrRoutePostIndex)",
		"   " + last + "users/  layout: " + source("users/layout.go"),
		"      " + mid + "page: GET,HEAD /users  " + source("users/route.go"),
		"      " + mid + "by_id/",
		"      " + pipe + "  " + last + "page: GET,HEAD /users/{id}  params: id  " + source("users/by_id/route.go"),
		"      " + mid + "status_options/",
		"      " + pipe + "  " + last + "fragment (not wrapped): GET,HEAD /users/status-options  " + source("users/status_options/route.go"),
		"      " + mid + "fragment (not wrapped): GET,HEAD /users/table  " + source("users/route.go"),
		"      " + mid + "action (layout-aware): POST /users/create  " + source("users/route.go") + " (GoldrRoutePostCreate)",
		"      " + last + "action (layout-aware): POST /users/save-preview  " + source("users/route.go") + " (GoldrRoutePostSavePreview)",
		"",
		"Rule:",
		"  pages inherit every layout above them",
		"  actions can use the same layout stack with goldr.WriteRouteResponse",
		"  fragments are not layout-wrapped",
	}
	return strings.Join(lines, "\n") + "\n"
}

func fullFeatureRouteSourcePath(source string) string {
	return filepath.ToSlash(filepath.Join("..", "..", "..", "..", "examples", "full_feature", "app", "routes", filepath.FromSlash(source)))
}

func fullFeatureRoutesDisplayRoot() string {
	return filepath.ToSlash(filepath.Join("..", "..", "..", "..", "examples", "full_feature", "app", "routes"))
}
