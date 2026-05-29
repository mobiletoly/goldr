// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package htmxrefs

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/wiring"
)

func TestScanResolvesHTMXRequestReferences(t *testing.T) {
	root := t.TempDir()
	writeTempl(t, root, "app/routes/users/page.templ", `package users

templ PageView(href string, user User) {
	<form hx-post={ urls.Users.Create.Path() }></form>
	<button hx-get={ urls.Users.Table.Path() + "?status=active" }>Active</button>
	<a data-hx-get="/users/table?status=inactive">Inactive</a>
	<a hx-get={ urls.Users.ByID.Bind(user.ID).Path() }>User</a>
	<a hx-get={ urls.Users.ByID.Bind(user.ID + suffix).Path() }>Computed user</a>
	<a hx-get={ href }>Dynamic</a>
	<a hx-post="/users/missing">Missing</a>
	<a hx-delete="/users/table">Wrong method</a>
	<a hx-get="https://example.com/users">External</a>
	<a hx-get="//example.com/users">External protocol</a>
	<a hx-get="users/table">Invalid</a>
	<a hx-get={ userPath(user) }>Helper function</a>
}
`)

	refs, err := Scan([]Root{{Dir: filepath.Join(root, "app", "routes")}}, routeRows())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	got := map[string]Reference{}
	for _, ref := range refs {
		got[ref.Value] = ref
	}
	want := map[string]Reference{
		"urls.Users.Create.Path()": {
			Status:    StatusResolved,
			Method:    "POST",
			Attribute: "hx-post",
			Source:    "users/page.templ",
			Value:     "urls.Users.Create.Path()",
			Route:     "/users/create",
			Match:     &RouteMatch{Path: "/users/create", Kind: "action", Source: "users/route.go:GoldrRoutePostCreate", Helper: "urls.Users.Create.Path()"},
		},
		`urls.Users.Table.Path() + "?status=active"`: {
			Status:    StatusResolved,
			Method:    "GET",
			Attribute: "hx-get",
			Source:    "users/page.templ",
			Value:     `urls.Users.Table.Path() + "?status=active"`,
			Route:     "/users/table",
			Match:     &RouteMatch{Path: "/users/table", Kind: "fragment", Source: "users/route.go", Helper: "urls.Users.Table.Path()"},
		},
		"/users/table?status=inactive": {
			Status:    StatusResolved,
			Method:    "GET",
			Attribute: "data-hx-get",
			Source:    "users/page.templ",
			Value:     "/users/table?status=inactive",
			Route:     "/users/table",
			Match:     &RouteMatch{Path: "/users/table", Kind: "fragment", Source: "users/route.go", Helper: "urls.Users.Table.Path()"},
		},
		"urls.Users.ByID.Bind(user.ID).Path()": {
			Status:    StatusResolved,
			Method:    "GET",
			Attribute: "hx-get",
			Source:    "users/page.templ",
			Value:     "urls.Users.ByID.Bind(user.ID).Path()",
			Route:     "/users/{id}",
			Match:     &RouteMatch{Path: "/users/{id}", Kind: "page", Source: "users/by_id/route.go", Helper: "urls.Users.ByID.Bind(id).Path()"},
		},
		"urls.Users.ByID.Bind(user.ID + suffix).Path()": {
			Status:    StatusResolved,
			Method:    "GET",
			Attribute: "hx-get",
			Source:    "users/page.templ",
			Value:     "urls.Users.ByID.Bind(user.ID + suffix).Path()",
			Route:     "/users/{id}",
			Match:     &RouteMatch{Path: "/users/{id}", Kind: "page", Source: "users/by_id/route.go", Helper: "urls.Users.ByID.Bind(id).Path()"},
		},
		"href": {
			Status:    StatusDynamic,
			Method:    "GET",
			Attribute: "hx-get",
			Source:    "users/page.templ",
			Value:     "href",
		},
		"/users/missing": {
			Status:    StatusUnmatched,
			Method:    "POST",
			Attribute: "hx-post",
			Source:    "users/page.templ",
			Value:     "/users/missing",
			Route:     "/users/missing",
		},
		"/users/table": {
			Status:    StatusUnmatched,
			Method:    "DELETE",
			Attribute: "hx-delete",
			Source:    "users/page.templ",
			Value:     "/users/table",
			Route:     "/users/table",
		},
		"https://example.com/users": {
			Status:    StatusExternal,
			Method:    "GET",
			Attribute: "hx-get",
			Source:    "users/page.templ",
			Value:     "https://example.com/users",
		},
		"//example.com/users": {
			Status:    StatusExternal,
			Method:    "GET",
			Attribute: "hx-get",
			Source:    "users/page.templ",
			Value:     "//example.com/users",
		},
		"users/table": {
			Status:    StatusInvalid,
			Method:    "GET",
			Attribute: "hx-get",
			Source:    "users/page.templ",
			Value:     "users/table",
		},
		"userPath(user)": {
			Status:    StatusDynamic,
			Method:    "GET",
			Attribute: "hx-get",
			Source:    "users/page.templ",
			Value:     "userPath(user)",
		},
	}

	if len(got) != len(want) {
		t.Fatalf("refs = %#v, want %d references", refs, len(want))
	}
	for value, wantRef := range want {
		gotRef, ok := got[value]
		if !ok {
			t.Fatalf("missing reference for %q in %#v", value, refs)
		}
		gotRef.Line = 0
		gotRef.Column = 0
		if !reflect.DeepEqual(gotRef, wantRef) {
			t.Fatalf("reference %q = %#v, want %#v", value, gotRef, wantRef)
		}
	}
}

func TestScanReportsParserErrors(t *testing.T) {
	root := t.TempDir()
	path := writeTempl(t, root, "app/routes/page.templ", "package routes\n\ntempl Broken() { <div> }\n")

	_, err := Scan([]Root{{Dir: filepath.Join(root, "app", "routes")}}, nil)
	if err == nil {
		t.Fatal("Scan() error = nil, want parse error")
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("Scan() error = %v, want source path", err)
	}
}

func routeRows() []wiring.RouteSurfaceRow {
	return []wiring.RouteSurfaceRow{
		{Kind: "action", Methods: []string{"POST"}, Path: "/users/create", Source: "users/route.go:GoldrRoutePostCreate", Helper: "urls.Users.Create.Path()"},
		{Kind: "fragment", Methods: []string{"GET", "HEAD"}, Path: "/users/table", Source: "users/route.go", Helper: "urls.Users.Table.Path()"},
		{Kind: "page", Methods: []string{"GET", "HEAD"}, Path: "/users/{id}", Source: "users/by_id/route.go", Helper: "urls.Users.ByID.Bind(id).Path()"},
	}
}

func writeTempl(t *testing.T, root string, name string, source string) string {
	t.Helper()

	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
