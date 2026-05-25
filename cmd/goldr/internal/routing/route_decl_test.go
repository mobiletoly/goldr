// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func routeImports(namePath ...string) []RouteImportDeclaration {
	imports := make([]RouteImportDeclaration, 0, len(namePath)/2)
	for index := 0; index < len(namePath); index += 2 {
		imports = append(imports, RouteImportDeclaration{Name: namePath[index], Path: namePath[index+1]})
	}
	return imports
}

func TestScanRouteDeclarationMinimalLocalPage(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "users/route.go", `package users

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

var Route = goldr.RouteDef{
	Page: goldr.FuncPage(page),
}

func page(r *http.Request) goldr.RouteResponse {
	return goldr.Text{Body: "users"}
}
`)

	tree := scanOK(t, root)

	want := []RouteDeclaration{
		{
			Route:   "/users",
			GoFile:  "users/route.go",
			Imports: routeImports("goldr", "github.com/mobiletoly/goldr", "http", "net/http"),
			Kind:    routeDeclarationKindLocal,
			Page:    &RouteHandlerDeclaration{Handler: "page"},
		},
	}
	if !reflect.DeepEqual(tree.Routes, want) {
		t.Fatalf("routes = %#v, want %#v", tree.Routes, want)
	}
}

func TestScanRouteDeclarationLocalFullSurface(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "users/route.go", `package users

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

var Route = goldr.RouteDef{
	Name:  "users.index",
	Title: "Users",
	Page:  goldr.FuncPage(page),
	Fragments: goldr.FuncFragments{
		goldr.FuncFragment("preview", preview),
		goldr.FuncFragment("save_profile", saveProfile),
	},
	Actions: goldr.FuncActions{
		goldr.FuncPostIndex(postIndex),
		goldr.FuncPut("replace", putReplace),
		goldr.FuncPatch("save-profile", patchSaveProfile),
		goldr.FuncDelete("archive", deleteArchive),
	},
	Meta: goldr.RouteMeta{
		Labels: map[string]string{
			"app.permission": "admin.users.view",
			"app.nav": "admin.users",
		},
	},
}
`)

	tree := scanOK(t, root)

	want := []RouteDeclaration{
		{
			Route:   "/users",
			GoFile:  "users/route.go",
			Imports: routeImports("goldr", "github.com/mobiletoly/goldr", "http", "net/http"),
			Kind:    routeDeclarationKindLocal,
			Name:    "users.index",
			Title:   "Users",
			Meta: []RouteMetaLabel{
				{Key: "app.nav", Value: "admin.users"},
				{Key: "app.permission", Value: "admin.users.view"},
			},
			Page: &RouteHandlerDeclaration{Handler: "page"},
			Fragments: []RouteFragmentDeclaration{
				{Name: "preview", Segment: "preview", SymbolName: "Preview", Handler: "preview"},
				{Name: "save_profile", Segment: "save-profile", SymbolName: "SaveProfile", Handler: "saveProfile"},
			},
			Actions: []RouteActionDeclaration{
				{Method: "POST", Index: true, SymbolName: "Index", Handler: "postIndex"},
				{Method: "PUT", Name: "replace", Segment: "replace", SymbolName: "Replace", Handler: "putReplace"},
				{Method: "PATCH", Name: "save-profile", Segment: "save-profile", SymbolName: "SaveProfile", Handler: "patchSaveProfile"},
				{Method: "DELETE", Name: "archive", Segment: "archive", SymbolName: "Archive", Handler: "deleteArchive"},
			},
		},
	}
	if !reflect.DeepEqual(tree.Routes, want) {
		t.Fatalf("routes = %#v, want %#v", tree.Routes, want)
	}
}

func TestScanRouteDeclarationKitRoute(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "analytics/cohort_explorer/route.go", `package cohort_explorer

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"github.com/example/app/pages/cohortexplorer"
)

var Route = goldr.KitRouteDef[cohortexplorer.Kit]{
	Name:    "admin.analytics.cohort_explorer",
	Title:   "Cohort Explorer",
	New:     newKit,
	Page:    goldr.KitPage(cohortexplorer.Kit.Page),
	Fragments: goldr.KitFragments[cohortexplorer.Kit]{
		goldr.KitFragment("filters", cohortexplorer.Kit.FragFilters),
		goldr.KitFragment("results", cohortexplorer.Kit.FragResults),
	},
	Actions: goldr.KitActions[cohortexplorer.Kit]{
		goldr.KitPostIndex(cohortexplorer.Kit.PostIndex),
		goldr.KitPost("export", cohortexplorer.Kit.PostExport),
	},
}

func newKit(r *http.Request) cohortexplorer.Kit {
	return cohortexplorer.New(portal(r))
}

func portal(r *http.Request) cohortexplorer.Portal {
	return cohortexplorer.Portal{}
}
`)

	tree := scanOK(t, root)

	want := []RouteDeclaration{
		{
			Route:  "/analytics/cohort-explorer",
			GoFile: "analytics/cohort_explorer/route.go",
			Imports: []RouteImportDeclaration{
				{Name: "cohortexplorer", Path: "github.com/example/app/pages/cohortexplorer"},
				{Name: "goldr", Path: "github.com/mobiletoly/goldr"},
				{Name: "http", Path: "net/http"},
			},
			Kind:  routeDeclarationKindKit,
			Name:  "admin.analytics.cohort_explorer",
			Title: "Cohort Explorer",
			Page:  &RouteHandlerDeclaration{Handler: "cohortexplorer.Kit.Page"},
			Fragments: []RouteFragmentDeclaration{
				{Name: "filters", Segment: "filters", SymbolName: "Filters", Handler: "cohortexplorer.Kit.FragFilters"},
				{Name: "results", Segment: "results", SymbolName: "Results", Handler: "cohortexplorer.Kit.FragResults"},
			},
			Actions: []RouteActionDeclaration{
				{Method: "POST", Index: true, SymbolName: "Index", Handler: "cohortexplorer.Kit.PostIndex"},
				{Method: "POST", Name: "export", Segment: "export", SymbolName: "Export", Handler: "cohortexplorer.Kit.PostExport"},
			},
			Kit: &RouteKitDeclaration{
				KitType: "cohortexplorer.Kit",
				New:     "newKit",
			},
		},
	}
	if !reflect.DeepEqual(tree.Routes, want) {
		t.Fatalf("routes = %#v, want %#v", tree.Routes, want)
	}
}

func TestScanRouteDeclarationAllActionHelpers(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "local/route.go", `package local

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Actions: goldr.FuncActions{
		goldr.FuncPostIndex(postIndex),
		goldr.FuncPost("create", postCreate),
		goldr.FuncPutIndex(putIndex),
		goldr.FuncPut("replace", putReplace),
		goldr.FuncPatchIndex(patchIndex),
		goldr.FuncPatch("update", patchUpdate),
		goldr.FuncDeleteIndex(deleteIndex),
		goldr.FuncDelete("archive", deleteArchive),
	},
}
`)
	writeFile(t, root, "kit/route.go", `package kit

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

type Kit struct{}

func New(*http.Request) Kit { return Kit{} }

var Route = goldr.KitRouteDef[Kit]{
	New: New,
	Actions: goldr.KitActions[Kit]{
		goldr.KitPostIndex(Kit.PostIndex),
		goldr.KitPost("create", Kit.PostCreate),
		goldr.KitPutIndex(Kit.PutIndex),
		goldr.KitPut("replace", Kit.PutReplace),
		goldr.KitPatchIndex(Kit.PatchIndex),
		goldr.KitPatch("update", Kit.PatchUpdate),
		goldr.KitDeleteIndex(Kit.DeleteIndex),
		goldr.KitDelete("archive", Kit.DeleteArchive),
	},
}
`)

	tree := scanOK(t, root)
	actionsByRoute := make(map[string][]RouteActionDeclaration)
	for _, route := range tree.Routes {
		actionsByRoute[route.Route] = route.Actions
	}

	wantLocal := []RouteActionDeclaration{
		{Method: "POST", Index: true, SymbolName: "Index", Handler: "postIndex"},
		{Method: "POST", Name: "create", Segment: "create", SymbolName: "Create", Handler: "postCreate"},
		{Method: "PUT", Index: true, SymbolName: "Index", Handler: "putIndex"},
		{Method: "PUT", Name: "replace", Segment: "replace", SymbolName: "Replace", Handler: "putReplace"},
		{Method: "PATCH", Index: true, SymbolName: "Index", Handler: "patchIndex"},
		{Method: "PATCH", Name: "update", Segment: "update", SymbolName: "Update", Handler: "patchUpdate"},
		{Method: "DELETE", Index: true, SymbolName: "Index", Handler: "deleteIndex"},
		{Method: "DELETE", Name: "archive", Segment: "archive", SymbolName: "Archive", Handler: "deleteArchive"},
	}
	if !reflect.DeepEqual(actionsByRoute["/local"], wantLocal) {
		t.Fatalf("local actions = %#v, want %#v", actionsByRoute["/local"], wantLocal)
	}

	wantKit := []RouteActionDeclaration{
		{Method: "POST", Index: true, SymbolName: "Index", Handler: "Kit.PostIndex"},
		{Method: "POST", Name: "create", Segment: "create", SymbolName: "Create", Handler: "Kit.PostCreate"},
		{Method: "PUT", Index: true, SymbolName: "Index", Handler: "Kit.PutIndex"},
		{Method: "PUT", Name: "replace", Segment: "replace", SymbolName: "Replace", Handler: "Kit.PutReplace"},
		{Method: "PATCH", Index: true, SymbolName: "Index", Handler: "Kit.PatchIndex"},
		{Method: "PATCH", Name: "update", Segment: "update", SymbolName: "Update", Handler: "Kit.PatchUpdate"},
		{Method: "DELETE", Index: true, SymbolName: "Index", Handler: "Kit.DeleteIndex"},
		{Method: "DELETE", Name: "archive", Segment: "archive", SymbolName: "Archive", Handler: "Kit.DeleteArchive"},
	}
	if !reflect.DeepEqual(actionsByRoute["/kit"], wantKit) {
		t.Fatalf("kit actions = %#v, want %#v", actionsByRoute["/kit"], wantKit)
	}
}

func TestScanRouteDeclarationAllActionHandlerHelpers(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "local/route.go", `package local

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Actions: goldr.FuncActions{
		goldr.FuncPostHandlerIndex(postIndex),
		goldr.FuncPostHandler("create", postCreate),
		goldr.FuncPutHandlerIndex(putIndex),
		goldr.FuncPutHandler("replace", putReplace),
		goldr.FuncPatchHandlerIndex(patchIndex),
		goldr.FuncPatchHandler("update", patchUpdate),
		goldr.FuncDeleteHandlerIndex(deleteIndex),
		goldr.FuncDeleteHandler("archive", deleteArchive),
	},
}
`)
	writeFile(t, root, "kit/route.go", `package kit

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

type Kit struct{}

func New(*http.Request) Kit { return Kit{} }

var Route = goldr.KitRouteDef[Kit]{
	New: New,
	Actions: goldr.KitActions[Kit]{
		goldr.KitPostHandlerIndex(Kit.PostIndex),
		goldr.KitPostHandler("create", Kit.PostCreate),
		goldr.KitPutHandlerIndex(Kit.PutIndex),
		goldr.KitPutHandler("replace", Kit.PutReplace),
		goldr.KitPatchHandlerIndex(Kit.PatchIndex),
		goldr.KitPatchHandler("update", Kit.PatchUpdate),
		goldr.KitDeleteHandlerIndex(Kit.DeleteIndex),
		goldr.KitDeleteHandler("archive", Kit.DeleteArchive),
	},
}
`)

	tree := scanOK(t, root)
	actionsByRoute := make(map[string][]RouteActionDeclaration)
	for _, route := range tree.Routes {
		actionsByRoute[route.Route] = route.Actions
	}

	wantLocal := []RouteActionDeclaration{
		{Method: "POST", Index: true, SymbolName: "Index", Writer: true, Handler: "postIndex"},
		{Method: "POST", Name: "create", Segment: "create", SymbolName: "Create", Writer: true, Handler: "postCreate"},
		{Method: "PUT", Index: true, SymbolName: "Index", Writer: true, Handler: "putIndex"},
		{Method: "PUT", Name: "replace", Segment: "replace", SymbolName: "Replace", Writer: true, Handler: "putReplace"},
		{Method: "PATCH", Index: true, SymbolName: "Index", Writer: true, Handler: "patchIndex"},
		{Method: "PATCH", Name: "update", Segment: "update", SymbolName: "Update", Writer: true, Handler: "patchUpdate"},
		{Method: "DELETE", Index: true, SymbolName: "Index", Writer: true, Handler: "deleteIndex"},
		{Method: "DELETE", Name: "archive", Segment: "archive", SymbolName: "Archive", Writer: true, Handler: "deleteArchive"},
	}
	if !reflect.DeepEqual(actionsByRoute["/local"], wantLocal) {
		t.Fatalf("local actions = %#v, want %#v", actionsByRoute["/local"], wantLocal)
	}

	wantKit := []RouteActionDeclaration{
		{Method: "POST", Index: true, SymbolName: "Index", Writer: true, Handler: "Kit.PostIndex"},
		{Method: "POST", Name: "create", Segment: "create", SymbolName: "Create", Writer: true, Handler: "Kit.PostCreate"},
		{Method: "PUT", Index: true, SymbolName: "Index", Writer: true, Handler: "Kit.PutIndex"},
		{Method: "PUT", Name: "replace", Segment: "replace", SymbolName: "Replace", Writer: true, Handler: "Kit.PutReplace"},
		{Method: "PATCH", Index: true, SymbolName: "Index", Writer: true, Handler: "Kit.PatchIndex"},
		{Method: "PATCH", Name: "update", Segment: "update", SymbolName: "Update", Writer: true, Handler: "Kit.PatchUpdate"},
		{Method: "DELETE", Index: true, SymbolName: "Index", Writer: true, Handler: "Kit.DeleteIndex"},
		{Method: "DELETE", Name: "archive", Segment: "archive", SymbolName: "Archive", Writer: true, Handler: "Kit.DeleteArchive"},
	}
	if !reflect.DeepEqual(actionsByRoute["/kit"], wantKit) {
		t.Fatalf("kit actions = %#v, want %#v", actionsByRoute["/kit"], wantKit)
	}
}

func TestScanRouteDeclarationIndexFragments(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "local/route.go", `package local

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Fragments: goldr.FuncFragments{
		goldr.FuncFragmentIndex(options),
	},
	Actions: goldr.FuncActions{
		goldr.FuncPostIndex(postIndex),
	},
}
`)
	writeFile(t, root, "kit/route.go", `package kit

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

type Kit struct{}

func New(*http.Request) Kit { return Kit{} }

var Route = goldr.KitRouteDef[Kit]{
	New: New,
	Fragments: goldr.KitFragments[Kit]{
		goldr.KitFragmentIndex(Kit.Options),
	},
}
`)

	tree := scanOK(t, root)
	fragmentsByRoute := make(map[string][]RouteFragmentDeclaration)
	for _, route := range tree.Routes {
		fragmentsByRoute[route.Route] = route.Fragments
	}

	wantLocal := []RouteFragmentDeclaration{{Name: "index", SymbolName: "Index", Index: true, Handler: "options"}}
	if !reflect.DeepEqual(fragmentsByRoute["/local"], wantLocal) {
		t.Fatalf("local fragments = %#v, want %#v", fragmentsByRoute["/local"], wantLocal)
	}
	wantKit := []RouteFragmentDeclaration{{Name: "index", SymbolName: "Index", Index: true, Handler: "Kit.Options"}}
	if !reflect.DeepEqual(fragmentsByRoute["/kit"], wantKit) {
		t.Fatalf("kit fragments = %#v, want %#v", fragmentsByRoute["/kit"], wantKit)
	}
}

func TestScanRouteDeclarationRejectsSameDirectoryOldRouteSurface(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "users/route.go", `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Page: goldr.FuncPage(page),
}
`)
	writeFile(t, root, "users/page.go", "package users\n")
	writeFile(t, root, "users/frag_table.go", "package users\n")
	writeFile(t, root, "users/actions.go", `package users

import "net/http"

func PostCreate(w http.ResponseWriter, r *http.Request) {}
`)

	_, err := Scan(root)
	if err == nil {
		t.Fatal("Scan() error = nil, want same-directory route surface rejection")
	}
	for _, want := range []string{"users/page.go", "users/frag_table.go", "users/actions.go", "route surface belongs in route.go"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("Scan() error missing %q:\n%v", want, err)
		}
	}
}

func TestScanRouteDeclarationReportsProblems(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		message string
	}{
		{
			name: "missing Route",
			source: `package users
`,
			message: "missing Route declaration",
		},
		{
			name: "dynamic route",
			source: `package users

var Route = buildRoute()
`,
			message: "Route must use a static goldr.RouteDef, goldr.KitRouteDef, or goldr.KitRouteMount composite literal",
		},
		{
			name: "live kit route missing New",
			source: `package users

import "github.com/mobiletoly/goldr"

type Kit struct{}

var Route = goldr.KitRouteDef[Kit]{
	Page: goldr.KitPage(Kit.Page),
}
`,
			message: "KitRouteDef requires New under app/routes",
		},
		{
			name: "removed kit route surface",
			source: `package users

import "github.com/mobiletoly/goldr"

type Kit struct{}

var Route = goldr.KitRouteSurface[Kit]{
	Page: goldr.KitPage(Kit.Page),
}
`,
			message: "Route must use goldr.RouteDef, goldr.KitRouteDef[K], or goldr.KitRouteMount[K]",
		},
		{
			name: "kit route mount new selector",
			source: `package users

import (
	"net/http"

	"github.com/mobiletoly/goldr"
	"example.com/app/shared"
)

type Kit struct{}

func New(*http.Request) Kit { return Kit{} }

var Route = goldr.KitRouteMount[Kit]{
	New: shared.New,
	Mount: "reports",
}
`,
			message: "KitRouteMount.New must be a local identifier",
		},
		{
			name: "kit route mount metadata",
			source: `package users

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

type Kit struct{}

func New(*http.Request) Kit { return Kit{} }

var Route = goldr.KitRouteMount[Kit]{
	New: New,
	Mount: "reports",
	Name: "reports",
}
`,
			message: "KitRouteMount supports only New and Mount route surface fields",
		},
		{
			name: "empty route surface",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{}
`,
			message: "Route must declare at least one of Page, Fragments, or Actions",
		},
		{
			name: "computed fragments",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Fragments: sharedFragments(),
}
`,
			message: "Fragments must use a literal goldr fragment collection",
		},
		{
			name: "index fragment with page",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Page: goldr.FuncPage(page),
	Fragments: goldr.FuncFragments{
		goldr.FuncFragmentIndex(options),
	},
}
`,
			message: "Route cannot declare both Page and an index fragment",
		},
		{
			name: "local index fragment wrong arity",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Fragments: goldr.FuncFragments{
		goldr.FuncFragmentIndex("index", options),
	},
}
`,
			message: "index fragment helpers must use one handler argument",
		},
		{
			name: "kit index fragment wrong arity",
			source: `package users

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

type Kit struct{}

func New(*http.Request) Kit { return Kit{} }

var Route = goldr.KitRouteDef[Kit]{
	New: New,
	Fragments: goldr.KitFragments[Kit]{
		goldr.KitFragmentIndex("index", Kit.Options),
	},
}
`,
			message: "index fragment helpers must use one handler argument",
		},
		{
			name: "index fragment collides with named index fragment",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Fragments: goldr.FuncFragments{
		goldr.FuncFragmentIndex(options),
		goldr.FuncFragment("index", namedOptions),
	},
}
`,
			message: `fragment segments "Index" and "index" normalize to the same generated symbol Index`,
		},
		{
			name: "computed actions",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Actions: append(baseActions, extra...),
}
`,
			message: "Actions must use a literal goldr action collection",
		},
		{
			name: "computed metadata",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Page: goldr.FuncPage(page),
	Meta: buildMeta(),
}
`,
			message: "Meta must use a literal goldr.RouteMeta value",
		},
		{
			name: "reserved symbol",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Page: goldr.FuncPage(page),
}

func GoldrRoutePage() {}
`,
			message: "reserved GoldrRoute* symbol declared: GoldrRoutePage",
		},
		{
			name: "blank import",
			source: `package users

import _ "example.com/sideeffect"
import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Page: goldr.FuncPage(page),
}
`,
			message: "route.go must not use blank imports",
		},
		{
			name: "dot import",
			source: `package users

import . "example.com/dot"
import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Page: goldr.FuncPage(page),
}
`,
			message: "route.go must not use dot imports",
		},
		{
			name: "duplicate action symbol",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Actions: goldr.FuncActions{
		goldr.FuncPost("save-profile", postSaveProfile),
		goldr.FuncPost("save_profile", postSaveProfileAgain),
	},
}
`,
			message: `action segments "save-profile" and "save_profile" normalize to the same generated symbol SaveProfile`,
		},
		{
			name: "old kit type arguments",
			source: `package users

import "github.com/mobiletoly/goldr"

type Kit struct{}
type Context struct{}

var Route = goldr.KitRouteDef[Kit, Context]{}
`,
			message: "Route must use goldr.RouteDef, goldr.KitRouteDef[K], or goldr.KitRouteMount[K]",
		},
		{
			name: "unsupported action helper",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Actions: goldr.FuncActions{
		goldr.FuncGet("search", getSearch),
	},
}
`,
			message: "unsupported action helper: goldr.FuncGet",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, "users/route.go", test.source)

			_, err := Scan(root)
			var scanErr *ScanError
			if !errors.As(err, &scanErr) {
				t.Fatalf("Scan() error = %T, want *ScanError", err)
			}
			if !hasProblem(scanErr.Problems, "users/route.go", test.message) {
				t.Fatalf("problems = %#v, want %q", scanErr.Problems, test.message)
			}
		})
	}
}
