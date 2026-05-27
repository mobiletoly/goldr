package routing

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestScanRouteDeclarationsAndParams(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "route.go", routeGoPageSource("routes"))
	writeFile(t, root, "admin_v1/route.go", routeGoPageSource("admin_v1"))
	writeFile(t, root, "settings/build_info/route.go", routeGoPageSource("build_info"))
	writeFile(t, root, "settings/by_build_id/route.go", routeGoPageSource("by_build_id"))
	writeFile(t, root, "users/route.go", routeGoPageSource("users"))
	writeFile(t, root, "users/by_id/route.go", routeGoPageSource("by_id"))
	writeFile(t, root, "orgs/by_org_id/users/by_user_id/route.go", routeGoPageSource("by_user_id"))

	tree := scanOK(t, root)

	got := make(map[string][]string)
	for _, route := range tree.Routes {
		got[route.Route] = route.Params
	}

	want := map[string][]string{
		"/":                              nil,
		"/admin-v1":                      nil,
		"/orgs/{org_id}/users/{user_id}": {"org_id", "user_id"},
		"/settings/{build_id}":           {"build_id"},
		"/settings/build-info":           nil,
		"/users":                         nil,
		"/users/{id}":                    {"id"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("routes = %#v, want %#v", got, want)
	}
}

func TestScanLayouts(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root,
		"layout.go",
		"layout.templ",
		"settings/build_info/layout.go",
		"users/layout.go",
	)

	tree := scanOK(t, root)

	wantLayouts := []Layout{
		{RoutePrefix: "/", GoFile: "layout.go", TemplFile: "layout.templ", HasTempl: true},
		{RoutePrefix: "/settings/build-info", GoFile: "settings/build_info/layout.go"},
		{RoutePrefix: "/users", GoFile: "users/layout.go"},
	}
	if !reflect.DeepEqual(tree.Layouts, wantLayouts) {
		t.Fatalf("layouts = %#v, want %#v", tree.Layouts, wantLayouts)
	}
}

func TestScanRecordsMissingTemplPairsWithoutError(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root,
		"layout.go",
	)

	tree := scanOK(t, root)

	if tree.Layouts[0].HasTempl {
		t.Fatalf("layout HasTempl = true, want false")
	}
}

func TestScanIgnoresNonConventionGoFiles(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root,
		"users/helpers.go",
		"users/frag_row_test.go",
		"users/frag_table_templ.go",
		"users/page_templ.go",
	)
	writeFile(t, root, "users/route.go", routeGoPageSource("users"))

	tree := scanOK(t, root)

	if len(tree.Routes) != 1 || tree.Routes[0].Route != "/users" {
		t.Fatalf("routes = %#v, want one /users route declaration", tree.Routes)
	}
	if len(tree.Layouts) != 0 {
		t.Fatalf("layouts = %#v, want empty", tree.Layouts)
	}
	if len(tree.Fragments) != 0 {
		t.Fatalf("fragments = %#v, want empty", tree.Fragments)
	}
	if len(tree.Actions) != 0 {
		t.Fatalf("actions = %#v, want empty", tree.Actions)
	}
}

func TestScanRejectsOldRouteSurfaceFilesWithoutRouteDeclaration(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root,
		"page.go",
		"users/page.go",
		"users/frag_table.go",
		"users/actions.go",
	)

	_, err := Scan(root)
	var scanErr *ScanError
	if !errors.As(err, &scanErr) {
		t.Fatalf("Scan() error = %T, want *ScanError", err)
	}

	wantMessage := "route surface belongs in route.go"
	for _, wantPath := range []string{
		"page.go",
		"users/page.go",
		"users/frag_table.go",
		"users/actions.go",
	} {
		if !hasProblem(scanErr.Problems, wantPath, wantMessage) {
			t.Fatalf("problems = %#v, want %s: %q", scanErr.Problems, wantPath, wantMessage)
		}
	}
}

func TestScanAllowsLayoutsMiddlewareHelpersAndTemplates(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "route.go", routeGoPageSource("routes"))
	writeFile(t, root, "page.templ", "package routes\n")
	writeFile(t, root, "frag_table.templ", "package routes\n")
	writeFile(t, root, "layout.go", `package routes

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, ctx goldr.LayoutContext) templ.Component {
	return templ.NopComponent
}
`)
	writeFile(t, root, "middleware.go", `package routes

import "net/http"

func Middleware(next http.Handler) http.Handler {
	return next
}
`)
	writeFile(t, root, "helpers.go", "package routes\n")
	writeFile(t, root, "route_test.go", "package routes\n")
	writeFile(t, root, "page_templ.go", "package routes\n")

	tree := scanOK(t, root)

	if len(tree.Routes) != 1 || tree.Routes[0].GoFile != "route.go" {
		t.Fatalf("routes = %#v, want one route.go declaration", tree.Routes)
	}
	if len(tree.Layouts) != 1 || tree.Layouts[0].GoFile != "layout.go" {
		t.Fatalf("layouts = %#v, want layout.go", tree.Layouts)
	}
	if len(tree.Middlewares) != 1 || tree.Middlewares[0].GoFile != "middleware.go" {
		t.Fatalf("middlewares = %#v, want middleware.go", tree.Middlewares)
	}
}

func TestScanIgnoresGoSpecialDirectories(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "route.go", routeGoPageSource("routes"))
	writeFile(t, root, "users/route.go", routeGoPageSource("users"))
	writeFiles(t, root,
		"internal/page.go",
		"internal/layout.go",
		"internal/frag_row.go",
		"internal/users/page.go",
		"testdata/page.go",
		"users/internal/page.go",
		"vendor/page.go",
	)

	tree := scanOK(t, root)

	got := make([]string, 0, len(tree.Routes))
	for _, route := range tree.Routes {
		got = append(got, route.Route)
	}
	want := []string{"/", "/users"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("routes = %#v, want %#v", got, want)
	}
	if len(tree.Layouts) != 0 {
		t.Fatalf("layouts = %#v, want empty", tree.Layouts)
	}
	if len(tree.Fragments) != 0 {
		t.Fatalf("fragments = %#v, want empty", tree.Fragments)
	}
}

func TestScanWithMountsExpandsKitRouteDefs(t *testing.T) {
	root := t.TempDir()
	routesRoot := filepath.Join(root, "app/routes")
	mountsRoot := filepath.Join(root, "app/mounts")
	writeFile(t, routesRoot, "admin/reports/route.go", routeGoKitMountSource("reports", "reports"))
	writeFile(t, routesRoot, "admin/reports/layout.go", "package reports\n")
	writeFile(t, routesRoot, "admin/reports/layout.templ", "package reports\n")
	writeFile(t, mountsRoot, "reports/route.go", routeGoMountedKitPageSource("reports", "shared.Kit.Page"))
	writeFile(t, mountsRoot, "reports/layout.go", "package reports\n")
	writeFile(t, mountsRoot, "reports/layout.templ", "package reports\n")
	writeFile(t, mountsRoot, "reports/table/route.go", routeGoMountedKitActionsSource("table", "shared.Kit.Table",
		`goldr.KitAction(http.MethodPost, "/refresh", shared.Kit.Refresh)`,
	))

	tree, err := ScanWithMounts(routesRoot, mountsRoot)
	if err != nil {
		t.Fatalf("ScanWithMounts() error = %v, want nil", err)
	}

	if got, want := len(tree.Routes), 2; got != want {
		t.Fatalf("len(Routes) = %d, want %d: %#v", got, want, tree.Routes)
	}
	rootRoute := tree.Routes[0]
	if rootRoute.Route != "/admin/reports" || rootRoute.GoFile != "admin/reports/route.go" || rootRoute.Source != "../mounts/reports/route.go" || rootRoute.Adapter != "MountReports" {
		t.Fatalf("root mounted route = %#v, want rebased /admin/reports from app/mounts/reports", rootRoute)
	}
	if rootRoute.Kind != routeDeclarationKindKitMount || rootRoute.Kit == nil || rootRoute.Kit.New != "newKit" {
		t.Fatalf("root mounted kit = %#v, want mounted kit with owner New", rootRoute)
	}
	childRoute := tree.Routes[1]
	if childRoute.Route != "/admin/reports/table" || childRoute.Source != "../mounts/reports/table/route.go" || childRoute.Adapter != "MountReportsTable" {
		t.Fatalf("child mounted route = %#v, want rebased /admin/reports/table", childRoute)
	}
	if childRoute.MiddlewareGoFile != "admin/reports/table/route.go" {
		t.Fatalf("child MiddlewareGoFile = %q, want live child route path", childRoute.MiddlewareGoFile)
	}
	if got, want := len(childRoute.Actions), 1; got != want {
		t.Fatalf("len(child Actions) = %d, want %d", got, want)
	}
	wantLayouts := []Layout{
		{RoutePrefix: "/admin/reports", GoFile: "admin/reports/layout.go", TemplFile: "admin/reports/layout.templ", HasTempl: true},
		{RoutePrefix: "/admin/reports", GoFile: "../mounts/reports/layout.go", TemplFile: "../mounts/reports/layout.templ", HasTempl: true},
	}
	if !reflect.DeepEqual(tree.Layouts, wantLayouts) {
		t.Fatalf("layouts = %#v, want %#v", tree.Layouts, wantLayouts)
	}
}

func TestScanWithMountsRejectsInvalidMountSurfaces(t *testing.T) {
	root := t.TempDir()
	routesRoot := filepath.Join(root, "app/routes")
	mountsRoot := filepath.Join(root, "app/mounts")
	writeFile(t, routesRoot, "admin/reports/route.go", routeGoKitMountSource("reports", "reports"))
	writeFile(t, mountsRoot, "reports/route.go", routeGoPageSource("reports"))
	writeFile(t, mountsRoot, "reports/middleware.go", `package reports

import "net/http"

func Middleware(next http.Handler) http.Handler {
	return next
}
`)

	_, err := ScanWithMounts(routesRoot, mountsRoot)
	var scanErr *ScanError
	if !errors.As(err, &scanErr) {
		t.Fatalf("ScanWithMounts() error = %T, want *ScanError", err)
	}
	for _, want := range []Problem{
		{Path: "../mounts/reports/route.go", Message: "mounted route files must use goldr.KitRouteDef[K]"},
		{Path: "../mounts/reports/middleware.go", Message: "middleware.go is not supported in app/mounts"},
	} {
		if !hasProblem(scanErr.Problems, want.Path, want.Message) {
			t.Fatalf("problems = %#v, want %#v", scanErr.Problems, want)
		}
	}
}

func TestScanWithMountsRejectsMountedKitRouteDefNew(t *testing.T) {
	root := t.TempDir()
	routesRoot := filepath.Join(root, "app/routes")
	mountsRoot := filepath.Join(root, "app/mounts")
	writeFile(t, routesRoot, "admin/reports/route.go", routeGoKitMountSource("reports", "reports"))
	writeFile(t, mountsRoot, "reports/route.go", `package reports

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"example.com/app/shared"
)

var Route = goldr.KitRouteDef[shared.Kit]{
	New: newKit,
	Page: shared.Kit.Page,
}

func newKit(r *http.Request) shared.Kit {
	return shared.New()
}
`)

	_, err := ScanWithMounts(routesRoot, mountsRoot)
	var scanErr *ScanError
	if !errors.As(err, &scanErr) {
		t.Fatalf("ScanWithMounts() error = %T, want *ScanError", err)
	}
	want := Problem{
		Path:    "../mounts/reports/route.go",
		Message: "KitRouteDef.New is not supported under app/mounts; the KitRouteMount owner supplies New",
	}
	if !hasProblem(scanErr.Problems, want.Path, want.Message) {
		t.Fatalf("problems = %#v, want %#v", scanErr.Problems, want)
	}
}

func TestScanWithMountsKeepsOwnerAndMountedImportsSeparate(t *testing.T) {
	root := t.TempDir()
	routesRoot := filepath.Join(root, "app/routes")
	mountsRoot := filepath.Join(root, "app/mounts")
	writeFile(t, routesRoot, "admin/reports/route.go", `package reports

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	shared "example.com/app/owner/shared"
)

var Route = goldr.KitRouteMount[shared.Kit]{
	New: newKit,
	Mount: "reports",
}

func newKit(r *http.Request) shared.Kit {
	return shared.New()
}
`)
	writeFile(t, mountsRoot, "reports/route.go", `package reports

import (
	"github.com/mobiletoly/goldr"
	shared "example.com/app/mount/shared"
)

var Route = goldr.KitRouteDef[shared.Kit]{
	Page: shared.Kit.Page,
}
`)

	tree, err := ScanWithMounts(routesRoot, mountsRoot)
	if err != nil {
		t.Fatalf("ScanWithMounts() error = %v, want nil", err)
	}
	if len(tree.Routes) != 1 {
		t.Fatalf("routes = %#v, want one mounted route", tree.Routes)
	}
	if got := tree.Routes[0].Imports; !reflect.DeepEqual(got, []RouteImportDeclaration{
		{Name: "goldr", Path: "github.com/mobiletoly/goldr"},
		{Name: "shared", Path: "example.com/app/mount/shared", Explicit: true},
	}) {
		t.Fatalf("mounted route imports = %#v, want mounted route imports only", got)
	}
}

func TestScanWithMountsRejectsInvalidMountPaths(t *testing.T) {
	tests := []string{
		"",
		"/reports",
		"reports/",
		"reports/../other",
		"reports//daily",
		"reports/daily-report",
		"reports/by-id",
		"reports/ByID",
		"reports/.hidden",
		`reports\\daily`,
	}

	for _, mountPath := range tests {
		t.Run(mountPath, func(t *testing.T) {
			root := t.TempDir()
			routesRoot := filepath.Join(root, "app/routes")
			mountsRoot := filepath.Join(root, "app/mounts")
			writeFile(t, routesRoot, "admin/reports/route.go", routeGoKitMountSource("reports", mountPath))
			writeFile(t, mountsRoot, "reports/route.go", routeGoMountedKitPageSource("reports", "shared.Kit.Page"))

			_, err := ScanWithMounts(routesRoot, mountsRoot)
			var scanErr *ScanError
			if !errors.As(err, &scanErr) {
				t.Fatalf("ScanWithMounts() error = %T, want *ScanError", err)
			}
			want := "Mount must be a clean relative path under app/mounts using lowercase Go-safe slash components"
			if !hasProblem(scanErr.Problems, "admin/reports/route.go", want) {
				t.Fatalf("problems = %#v, want %q", scanErr.Problems, want)
			}
		})
	}
}

func TestScanActions(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "route.go", routeGoActionsSource("routes",
		`goldr.Action(http.MethodPost, "/", postIndex)`,
		`goldr.Action(http.MethodPut, "/search", putSearch)`,
	))
	writeFile(t, root, "users/route.go", routeGoActionsSource("users",
		`goldr.Action(http.MethodPost, "/create", postCreate)`,
		`goldr.Action(http.MethodPatch, "/save-preview", patchSavePreview)`,
	))
	writeFile(t, root, "users/by_id/route.go", routeGoActionsSource("by_id",
		`goldr.Action(http.MethodDelete, "/", deleteIndex)`,
		`goldr.Action(http.MethodPatch, "/profile", patchProfile)`,
	))

	tree := scanOK(t, root)

	actionsByRoute := make(map[string][]RouteActionDeclaration)
	for _, route := range tree.Routes {
		actionsByRoute[route.Route] = route.Actions
	}
	want := map[string][]RouteActionDeclaration{
		"/": {
			{Method: "POST", Index: true, SymbolName: "Index", Handler: "postIndex"},
			{Method: "PUT", Name: "search", Segment: "search", SymbolName: "Search", Handler: "putSearch"},
		},
		"/users": {
			{Method: "POST", Name: "create", Segment: "create", SymbolName: "Create", Handler: "postCreate"},
			{Method: "PATCH", Name: "save-preview", Segment: "save-preview", SymbolName: "SavePreview", Handler: "patchSavePreview"},
		},
		"/users/{id}": {
			{Method: "DELETE", Index: true, SymbolName: "Index", Handler: "deleteIndex"},
			{Method: "PATCH", Name: "profile", Segment: "profile", SymbolName: "Profile", Handler: "patchProfile"},
		},
	}
	if !reflect.DeepEqual(actionsByRoute, want) {
		t.Fatalf("actions = %#v, want %#v", actionsByRoute, want)
	}
}

func TestScanMiddleware(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "middleware.go", `package routes

import "net/http"

func Middleware(next http.Handler) http.Handler {
	return next
}
`)
	writeFile(t, root, "users/middleware.go", `package users

import "net/http"

func Middleware(next http.Handler) http.Handler {
	return next
}
`)
	writeFile(t, root, "users/by_id/middleware.go", `package by_id

import "net/http"

func Middleware(next http.Handler) http.Handler {
	return next
}
`)

	tree := scanOK(t, root)

	want := []Middleware{
		{RoutePrefix: "/", GoFile: "middleware.go"},
		{RoutePrefix: "/users", GoFile: "users/middleware.go"},
		{RoutePrefix: "/users/{id}", Params: []string{"id"}, GoFile: "users/by_id/middleware.go"},
	}
	if !reflect.DeepEqual(tree.Middlewares, want) {
		t.Fatalf("middleware = %#v, want %#v", tree.Middlewares, want)
	}
}

func TestScanReportsMiddlewareProblems(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "users/middleware.go", `package users

import "net/http"

func Middleware(next http.Handler) {}
`)

	_, err := Scan(root)
	var scanErr *ScanError
	if !errors.As(err, &scanErr) {
		t.Fatalf("Scan() error = %T, want *ScanError", err)
	}

	want := "Middleware: middleware must use exact form func Middleware(next http.Handler) http.Handler with unaliased net/http import"
	if !hasProblem(scanErr.Problems, "users/middleware.go", want) {
		t.Fatalf("problems = %#v, want %q", scanErr.Problems, want)
	}
}

func TestScanMissingRootErrors(t *testing.T) {
	_, err := Scan(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("Scan() error = nil, want error")
	}
}

func TestScanCollectsInvalidNames(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root,
		"Users/route.go",
		"_id/route.go",
		".hidden/route.go",
		"by_/route.go",
		"blog-posts/route.go",
		"_helper.go",
		".hidden.go",
	)

	_, err := Scan(root)
	if err == nil {
		t.Fatal("Scan() error = nil, want ScanError")
	}

	var scanErr *ScanError
	if !errors.As(err, &scanErr) {
		t.Fatalf("Scan() error type = %T, want *ScanError", err)
	}

	wantPaths := []string{
		"_helper.go",
		".hidden.go",
		".hidden",
		"Users",
		"_id",
		"blog-posts",
		"by_",
	}
	for _, wantPath := range wantPaths {
		if !hasProblemPath(scanErr.Problems, wantPath) {
			t.Fatalf("problems = %#v, want path %q", scanErr.Problems, wantPath)
		}
	}
}

func TestScanOutputOrderIsDeterministic(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "zeta/route.go", routeGoPageSource("zeta"))
	writeFile(t, root, "alpha/route.go", routeGoPageSource("alpha"))
	writeFiles(t, root, "users/layout.go", "admin/layout.go")

	tree := scanOK(t, root)

	routePaths := make([]string, 0, len(tree.Routes))
	for _, route := range tree.Routes {
		routePaths = append(routePaths, route.Route)
	}
	if !slices.IsSorted(routePaths) {
		t.Fatalf("routes = %#v, want sorted", routePaths)
	}

	layoutPrefixes := make([]string, 0, len(tree.Layouts))
	for _, layout := range tree.Layouts {
		layoutPrefixes = append(layoutPrefixes, layout.RoutePrefix)
	}
	if !slices.IsSorted(layoutPrefixes) {
		t.Fatalf("layout prefixes = %#v, want sorted", layoutPrefixes)
	}

}

func writeFiles(t *testing.T, root string, paths ...string) {
	t.Helper()

	for _, relPath := range paths {
		writeFile(t, root, relPath, "")
	}
}

func writeFile(t *testing.T, root, relPath, content string) {
	t.Helper()

	fullPath := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(fullPath), err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", fullPath, err)
	}
}

func scanOK(t *testing.T, root string) *Tree {
	t.Helper()

	tree, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan() error = %v, want nil", err)
	}
	return tree
}

func routeGoKitMountSource(packageName string, mount string) string {
	return `package ` + packageName + `

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

var Route = goldr.KitRouteMount[shared.Kit]{
	New: newKit,
	Mount: "` + mount + `",
}

func newKit(r *http.Request) shared.Kit {
	return shared.New()
}
`
}

func routeGoMountedKitPageSource(packageName string, page string) string {
	return `package ` + packageName + `

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"example.com/app/shared"
)

var Route = goldr.KitRouteDef[shared.Kit]{
	Page: ` + page + `,
}
`
}

func routeGoMountedKitActionsSource(packageName string, page string, actions ...string) string {
	var builder strings.Builder
	builder.WriteString(`package `)
	builder.WriteString(packageName)
	builder.WriteString(`

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"example.com/app/shared"
)

var Route = goldr.KitRouteDef[shared.Kit]{
	Page: `)
	builder.WriteString(page)
	builder.WriteString(`,
	Actions: goldr.KitActions[shared.Kit]{
`)
	for _, action := range actions {
		builder.WriteString("\t\t")
		builder.WriteString(action)
		builder.WriteString(",\n")
	}
	builder.WriteString(`	},
}
`)
	return builder.String()
}

func routeGoPageSource(packageName string) string {
	return `package ` + packageName + `

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

var Route = goldr.RouteDef{
	Page: page,
}

func page(r *http.Request) goldr.RouteResponse {
	return goldr.Text{Body: "ok"}
}
`
}

func routeGoActionsSource(packageName string, actions ...string) string {
	var builder strings.Builder
	builder.WriteString(`package `)
	builder.WriteString(packageName)
	builder.WriteString(`

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

var Route = goldr.RouteDef{
	Actions: goldr.Actions{
`)
	for _, action := range actions {
		builder.WriteString("\t\t")
		builder.WriteString(action)
		builder.WriteString(",\n")
	}
	builder.WriteString(`	},
}
`)
	return builder.String()
}

func hasProblemPath(problems []Problem, path string) bool {
	for _, problem := range problems {
		if strings.EqualFold(problem.Path, path) {
			return true
		}
	}
	return false
}

func hasProblem(problems []Problem, path, message string) bool {
	for _, problem := range problems {
		if problem.Path == path && problem.Message == message {
			return true
		}
	}
	return false
}
