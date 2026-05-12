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

func TestScanPagesAndParams(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root,
		"page.go",
		"admin_v1/page.go",
		"settings/build_info/page.go",
		"settings/by_build_id/page.go",
		"users/page.go",
		"users/by_id/page.go",
		"orgs/by_org_id/users/by_user_id/page.go",
	)

	tree := scanOK(t, root)

	got := make(map[string][]string)
	for _, page := range tree.Pages {
		got[page.Route] = page.Params
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
		t.Fatalf("pages = %#v, want %#v", got, want)
	}
}

func TestScanLayoutsAndFragments(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root,
		"layout.go",
		"layout.templ",
		"settings/build_info/layout.go",
		"users/layout.go",
		"users/build_info/frag_build_info.go",
		"users/frag_table.go",
		"users/frag_row.go",
		"users/frag_row.templ",
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

	wantFragments := []Fragment{
		{Name: "row", RoutePrefix: "/users", GoFile: "users/frag_row.go", TemplFile: "users/frag_row.templ", HasTempl: true},
		{Name: "table", RoutePrefix: "/users", GoFile: "users/frag_table.go"},
		{Name: "build_info", RoutePrefix: "/users/build-info", GoFile: "users/build_info/frag_build_info.go"},
	}
	if !reflect.DeepEqual(tree.Fragments, wantFragments) {
		t.Fatalf("fragments = %#v, want %#v", tree.Fragments, wantFragments)
	}
}

func TestScanRecordsMissingTemplPairsWithoutError(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root,
		"page.go",
		"layout.go",
		"frag_row.go",
	)

	tree := scanOK(t, root)

	if tree.Pages[0].HasTempl {
		t.Fatalf("page HasTempl = true, want false")
	}
	if tree.Layouts[0].HasTempl {
		t.Fatalf("layout HasTempl = true, want false")
	}
	if tree.Fragments[0].HasTempl {
		t.Fatalf("fragment HasTempl = true, want false")
	}
}

func TestScanIgnoresNonConventionGoFiles(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root,
		"users/helpers.go",
		"users/frag_row_test.go",
		"users/frag_table_templ.go",
		"users/page.go",
		"users/page_templ.go",
	)
	writeFile(t, root, "users/actions.go", "package users\n\nfunc Helper() {}\n")

	tree := scanOK(t, root)

	if len(tree.Pages) != 1 || tree.Pages[0].Route != "/users" {
		t.Fatalf("pages = %#v, want one /users page", tree.Pages)
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

func TestScanActions(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "actions.go", `package routes

import "net/http"

func PostIndex(w http.ResponseWriter, r *http.Request) {}
func PutSearch(w http.ResponseWriter, r *http.Request) {}
`)
	writeFile(t, root, "users/actions.go", `package users

import "net/http"

func PostCreate(w http.ResponseWriter, r *http.Request) {}
func PatchSavePreview(w http.ResponseWriter, r *http.Request) {}
func helper() {}
`)
	writeFile(t, root, "users/by_id/actions.go", `package by_id

import "net/http"

func DeleteIndex(w http.ResponseWriter, r *http.Request) {}
func PatchProfile(w http.ResponseWriter, r *http.Request) {}
`)

	tree := scanOK(t, root)

	want := []Action{
		{Method: "POST", Route: "/", GoFile: "actions.go", Function: "PostIndex", Suffix: "Index"},
		{Method: "PUT", Route: "/search", GoFile: "actions.go", Function: "PutSearch", Suffix: "Search", Segment: "search"},
		{Method: "POST", Route: "/users/create", GoFile: "users/actions.go", Function: "PostCreate", Suffix: "Create", Segment: "create"},
		{Method: "PATCH", Route: "/users/save-preview", GoFile: "users/actions.go", Function: "PatchSavePreview", Suffix: "SavePreview", Segment: "save-preview"},
		{Method: "DELETE", Route: "/users/{id}", Params: []string{"id"}, GoFile: "users/by_id/actions.go", Function: "DeleteIndex", Suffix: "Index"},
		{Method: "PATCH", Route: "/users/{id}/profile", Params: []string{"id"}, GoFile: "users/by_id/actions.go", Function: "PatchProfile", Suffix: "Profile", Segment: "profile"},
	}
	if !reflect.DeepEqual(tree.Actions, want) {
		t.Fatalf("actions = %#v, want %#v", tree.Actions, want)
	}
}

func TestScanReportsActionProblems(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "users/actions.go", `package users

import "net/http"

func GetCreate(w http.ResponseWriter, r *http.Request) {}
func PostCreate(w http.ResponseWriter) {}
`)

	_, err := Scan(root)
	var scanErr *ScanError
	if !errors.As(err, &scanErr) {
		t.Fatalf("Scan() error = %T, want *ScanError", err)
	}

	wantMessages := []string{
		"GetCreate: GET action handlers are not supported; pages and fragments own GET and HEAD",
		"PostCreate: action handlers must use func Name(w http.ResponseWriter, r *http.Request)",
	}
	for _, want := range wantMessages {
		if !hasProblem(scanErr.Problems, "users/actions.go", want) {
			t.Fatalf("problems = %#v, want %q", scanErr.Problems, want)
		}
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
		"Users/page.go",
		"_id/page.go",
		".hidden/page.go",
		"by_/page.go",
		"blog-posts/page.go",
		"testdata/page.go",
		"_helper.go",
		".hidden.go",
		"frag_.go",
		"frag_Row.go",
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
		"frag_.go",
		"frag_Row.go",
		".hidden",
		"Users",
		"_id",
		"blog-posts",
		"by_",
		"testdata",
	}
	for _, wantPath := range wantPaths {
		if !hasProblemPath(scanErr.Problems, wantPath) {
			t.Fatalf("problems = %#v, want path %q", scanErr.Problems, wantPath)
		}
	}
}

func TestScanOutputOrderIsDeterministic(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root,
		"zeta/page.go",
		"alpha/page.go",
		"users/frag_table.go",
		"users/frag_row.go",
		"users/layout.go",
		"admin/layout.go",
	)

	tree := scanOK(t, root)

	pageRoutes := make([]string, 0, len(tree.Pages))
	for _, page := range tree.Pages {
		pageRoutes = append(pageRoutes, page.Route)
	}
	if !slices.IsSorted(pageRoutes) {
		t.Fatalf("page routes = %#v, want sorted", pageRoutes)
	}

	layoutPrefixes := make([]string, 0, len(tree.Layouts))
	for _, layout := range tree.Layouts {
		layoutPrefixes = append(layoutPrefixes, layout.RoutePrefix)
	}
	if !slices.IsSorted(layoutPrefixes) {
		t.Fatalf("layout prefixes = %#v, want sorted", layoutPrefixes)
	}

	fragmentKeys := make([]string, 0, len(tree.Fragments))
	for _, fragment := range tree.Fragments {
		fragmentKeys = append(fragmentKeys, fragment.RoutePrefix+"/"+fragment.Name)
	}
	if !slices.IsSorted(fragmentKeys) {
		t.Fatalf("fragment keys = %#v, want sorted", fragmentKeys)
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
