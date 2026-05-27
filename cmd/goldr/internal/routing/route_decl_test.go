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
	Page: page,
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
	Page:  page,
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/preview", preview),
		goldr.FragmentRoute("/save_profile", saveProfile),
	},
	Actions: goldr.Actions{
		goldr.Action(http.MethodPost, "/", postIndex),
		goldr.Action(http.MethodPut, "/replace", putReplace),
		goldr.Action(http.MethodPatch, "/save-profile", patchSaveProfile),
		goldr.Action(http.MethodDelete, "/archive", deleteArchive),
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
	Page:    cohortexplorer.Kit.Page,
	Fragments: goldr.KitFragments[cohortexplorer.Kit]{
		goldr.KitFragmentRoute("/filters", cohortexplorer.Kit.FragFilters),
		goldr.KitFragmentRoute("/results", cohortexplorer.Kit.FragResults),
	},
	Actions: goldr.KitActions[cohortexplorer.Kit]{
		goldr.KitAction(http.MethodPost, "/", cohortexplorer.Kit.PostIndex),
		goldr.KitAction(http.MethodPost, "/export", cohortexplorer.Kit.PostExport),
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

func TestScanRouteDeclarationPointerKitMethodExpressions(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "reports/route.go", `package reports

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

type Kit struct{}

func newKit(*http.Request) *Kit { return &Kit{} }

var Route = goldr.KitRouteDef[*Kit]{
	New:  newKit,
	Page: (*Kit).Page,
	Fragments: goldr.KitFragments[*Kit]{
		goldr.KitFragmentRoute("/panel", (*Kit).Panel),
	},
	Actions: goldr.KitActions[*Kit]{
		goldr.KitAction(http.MethodPost, "/save", (*Kit).PostSave),
		goldr.KitHTTPAction(http.MethodDelete, "/", (*Kit).DeleteIndex),
	},
}
`)

	tree := scanOK(t, root)

	want := []RouteDeclaration{
		{
			Route:   "/reports",
			GoFile:  "reports/route.go",
			Imports: routeImports("goldr", "github.com/mobiletoly/goldr", "http", "net/http"),
			Kind:    routeDeclarationKindKit,
			Page:    &RouteHandlerDeclaration{Handler: "(*Kit).Page"},
			Fragments: []RouteFragmentDeclaration{
				{Name: "panel", Segment: "panel", SymbolName: "Panel", Handler: "(*Kit).Panel"},
			},
			Actions: []RouteActionDeclaration{
				{Method: "POST", Name: "save", Segment: "save", SymbolName: "Save", Handler: "(*Kit).PostSave"},
				{Method: "DELETE", Index: true, SymbolName: "Index", Writer: true, Handler: "(*Kit).DeleteIndex"},
			},
			Kit: &RouteKitDeclaration{
				KitType: "*Kit",
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
	Actions: goldr.Actions{
		goldr.Action(http.MethodPost, "/", postIndex),
		goldr.Action(http.MethodPost, "/create", postCreate),
		goldr.Action(http.MethodPut, "/", putIndex),
		goldr.Action(http.MethodPut, "/replace", putReplace),
		goldr.Action(http.MethodPatch, "/", patchIndex),
		goldr.Action(http.MethodPatch, "/update", patchUpdate),
		goldr.Action(http.MethodDelete, "/", deleteIndex),
		goldr.Action(http.MethodDelete, "/archive", deleteArchive),
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
		goldr.KitAction(http.MethodPost, "/", Kit.PostIndex),
		goldr.KitAction(http.MethodPost, "/create", Kit.PostCreate),
		goldr.KitAction(http.MethodPut, "/", Kit.PutIndex),
		goldr.KitAction(http.MethodPut, "/replace", Kit.PutReplace),
		goldr.KitAction(http.MethodPatch, "/", Kit.PatchIndex),
		goldr.KitAction(http.MethodPatch, "/update", Kit.PatchUpdate),
		goldr.KitAction(http.MethodDelete, "/", Kit.DeleteIndex),
		goldr.KitAction(http.MethodDelete, "/archive", Kit.DeleteArchive),
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
	Actions: goldr.Actions{
		goldr.HTTPAction(http.MethodPost, "/", postIndex),
		goldr.HTTPAction(http.MethodPost, "/create", postCreate),
		goldr.HTTPAction(http.MethodPut, "/", putIndex),
		goldr.HTTPAction(http.MethodPut, "/replace", putReplace),
		goldr.HTTPAction(http.MethodPatch, "/", patchIndex),
		goldr.HTTPAction(http.MethodPatch, "/update", patchUpdate),
		goldr.HTTPAction(http.MethodDelete, "/", deleteIndex),
		goldr.HTTPAction(http.MethodDelete, "/archive", deleteArchive),
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
		goldr.KitHTTPAction(http.MethodPost, "/", Kit.PostIndex),
		goldr.KitHTTPAction(http.MethodPost, "/create", Kit.PostCreate),
		goldr.KitHTTPAction(http.MethodPut, "/", Kit.PutIndex),
		goldr.KitHTTPAction(http.MethodPut, "/replace", Kit.PutReplace),
		goldr.KitHTTPAction(http.MethodPatch, "/", Kit.PatchIndex),
		goldr.KitHTTPAction(http.MethodPatch, "/update", Kit.PatchUpdate),
		goldr.KitHTTPAction(http.MethodDelete, "/", Kit.DeleteIndex),
		goldr.KitHTTPAction(http.MethodDelete, "/archive", Kit.DeleteArchive),
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
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/", options),
	},
	Actions: goldr.Actions{
		goldr.Action(http.MethodPost, "/", postIndex),
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
		goldr.KitFragmentRoute("/", Kit.Options),
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
	Page: page,
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
	Page: Kit.Page,
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
	Page: Kit.Page,
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
	Page: page,
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/", options),
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
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/", "index", options),
	},
}
`,
			message: "fragment route helpers must use path and handler arguments",
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
		goldr.KitFragmentRoute("/", "index", Kit.Options),
	},
}
`,
			message: "fragment route helpers must use path and handler arguments",
		},
		{
			name: "fragment path without leading slash",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("table", table),
	},
}
`,
			message: `fragment path must start with "/"`,
		},
		{
			name: "fragment path empty",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Page: page,
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("", table),
	},
}
`,
			message: "fragment path must not be empty",
		},
		{
			name: "fragment path trailing slash",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/table/", table),
	},
}
`,
			message: "fragment path must not have a trailing slash",
		},
		{
			name: "fragment path nested",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/table/row", row),
	},
}
`,
			message: "fragment path must be route-local; nested paths belong in nested route directories",
		},
		{
			name: "fragment path uppercase",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/Table", table),
	},
}
`,
			message: "fragment path segments must use lowercase ASCII letters, digits, underscores, or hyphens and start with a lowercase ASCII letter",
		},
		{
			name: "fragment path digit first",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/1table", table),
	},
}
`,
			message: "fragment path segments must use lowercase ASCII letters, digits, underscores, or hyphens and start with a lowercase ASCII letter",
		},
		{
			name: "index fragment collides with named index fragment",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Fragments: goldr.Fragments{
		goldr.FragmentRoute("/", options),
		goldr.FragmentRoute("/index", namedOptions),
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
			name: "action path without leading slash",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Actions: goldr.Actions{
		goldr.Action(http.MethodPost, "create", postCreate),
	},
}
`,
			message: `action path must start with "/"`,
		},
		{
			name: "action path empty",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Page: page,
	Actions: goldr.Actions{
		goldr.Action(http.MethodPost, "", postCreate),
	},
}
`,
			message: "action path must not be empty",
		},
		{
			name: "action path nested",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Actions: goldr.Actions{
		goldr.Action(http.MethodPost, "/create/confirm", postCreate),
	},
}
`,
			message: "action path must be route-local; nested paths belong in nested route directories",
		},
		{
			name: "unsupported action method selector",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Actions: goldr.Actions{
		goldr.Action(http.MethodGet, "/search", getSearch),
	},
}
`,
			message: "action methods must use http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, or matching string literals",
		},
		{
			name: "unsupported action method literal",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Actions: goldr.Actions{
		goldr.Action("GET", "/search", getSearch),
	},
}
`,
			message: "action methods must be POST, PUT, PATCH, or DELETE",
		},
		{
			name: "computed action method",
			source: `package users

import "github.com/mobiletoly/goldr"

const methodPost = "POST"

var Route = goldr.RouteDef{
	Actions: goldr.Actions{
		goldr.Action(methodPost, "/create", postCreate),
	},
}
`,
			message: "action methods must use http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, or matching string literals",
		},
		{
			name: "computed metadata",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Page: page,
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
	Page: page,
}

func GoldrRoutePage() {}
`,
			message: "reserved GoldrRoute* symbol declared: GoldrRoutePage",
		},
		{
			name: "inline page function",
			source: `package users

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

var Route = goldr.RouteDef{
	Page: func(r *http.Request) goldr.RouteResponse {
		return goldr.Text{Body: "ok"}
	},
}
`,
			message: "Page must be an identifier, selector, or pointer method expression",
		},
		{
			name: "blank import",
			source: `package users

import _ "example.com/sideeffect"
import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Page: page,
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
	Page: page,
}
`,
			message: "route.go must not use dot imports",
		},
		{
			name: "duplicate action symbol",
			source: `package users

import "github.com/mobiletoly/goldr"

var Route = goldr.RouteDef{
	Actions: goldr.Actions{
		goldr.Action(http.MethodPost, "/save-profile", postSaveProfile),
		goldr.Action(http.MethodPost, "/save_profile", postSaveProfileAgain),
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
	Actions: goldr.Actions{
		goldr.FuncGet("search", getSearch),
	},
}
`,
			message: "Actions entries must use goldr.Action(method, path, handler) or goldr.HTTPAction(method, path, handler)",
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
