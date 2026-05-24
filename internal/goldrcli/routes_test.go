package goldrcli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRoutesPrintsRouteTable(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "app/routes/layout.go", "package routes\n")
	writeFile(t, root, "app/routes/page.go", "package routes\n")
	writeFile(t, root, "app/routes/users/page.go", "package users\n")
	writeFile(t, root, "app/routes/users/frag_table.go", "package users\n")
	writeFile(t, root, "app/routes/users/by_id/page.go", "package by_id\n")
	writeFile(t, root, "app/routes/users/actions.go", `package users

import "net/http"

func PostCreate(w http.ResponseWriter, r *http.Request) {}
`)

	code, stdout, stderr := runGoldr(t, "routes", "list", "--app-root", root)

	if code != 0 {
		t.Fatalf("Run(routes list) exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	want := [][]string{
		{"KIND", "METHOD", "PATH", "PARAMS", "SOURCE", "HELPER"},
		{"layout", "-", "/", "-", "layout.go", "-"},
		{"page", "GET,HEAD", "/", "-", "page.go", "urls.Root.Path()"},
		{"action", "POST", "/users/create", "-", "users/actions.go:PostCreate", "urls.Users.Create.Path()"},
		{"fragment", "GET,HEAD", "/users/frag-table", "-", "users/frag_table.go", "urls.Users.FragTable.Path()"},
		{"page", "GET,HEAD", "/users", "-", "users/page.go", "urls.Users.Path()"},
		{"page", "GET,HEAD", "/users/{id}", "id", "users/by_id/page.go", "urls.Users.ByID(id).Path()"},
	}
	requireRouteTableRows(t, stdout, want)
	requireMissingFile(t, filepath.Join(root, "app", "routes", "goldr_gen.go"))
	requireMissingFile(t, filepath.Join(root, "app", "urls", "goldr_gen.go"))
}

func TestRunRoutesPrintsJSON(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "app/routes/layout.go", "package routes\n")
	writeFile(t, root, "app/routes/page.go", "package routes\n")
	writeFile(t, root, "app/routes/users/page.go", "package users\n")
	writeFile(t, root, "app/routes/users/frag_table.go", "package users\n")
	writeFile(t, root, "app/routes/users/by_id/page.go", "package by_id\n")
	writeFile(t, root, "app/routes/users/actions.go", `package users

import "net/http"

func PostCreate(w http.ResponseWriter, r *http.Request) {}
`)

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

	var rows []routeSurfaceJSONRow
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil {
		t.Fatalf("Unmarshal(routes list --json) error = %v; stdout = %q", err, stdout)
	}
	want := []routeSurfaceJSONRow{
		{Kind: "layout", Methods: []string{}, Path: "/", Params: []string{}, Source: "layout.go"},
		{Kind: "page", Methods: []string{"GET", "HEAD"}, Path: "/", Params: []string{}, Source: "page.go", Helper: "urls.Root.Path()"},
		{Kind: "action", Methods: []string{"POST"}, Path: "/users/create", Params: []string{}, Source: "users/actions.go:PostCreate", Helper: "urls.Users.Create.Path()"},
		{Kind: "fragment", Methods: []string{"GET", "HEAD"}, Path: "/users/frag-table", Params: []string{}, Source: "users/frag_table.go", Helper: "urls.Users.FragTable.Path()"},
		{Kind: "page", Methods: []string{"GET", "HEAD"}, Path: "/users", Params: []string{}, Source: "users/page.go", Helper: "urls.Users.Path()"},
		{Kind: "page", Methods: []string{"GET", "HEAD"}, Path: "/users/{id}", Params: []string{"id"}, Source: "users/by_id/page.go", Helper: "urls.Users.ByID(id).Path()"},
	}
	if len(rows) != len(want) {
		t.Fatalf("JSON rows = %#v, want %#v", rows, want)
	}
	for index := range want {
		if strings.Join(rows[index].Methods, "\x00") != strings.Join(want[index].Methods, "\x00") ||
			strings.Join(rows[index].Params, "\x00") != strings.Join(want[index].Params, "\x00") ||
			rows[index].Kind != want[index].Kind ||
			rows[index].Path != want[index].Path ||
			rows[index].Source != want[index].Source ||
			rows[index].Helper != want[index].Helper {
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

func TestRunRoutesFullFeatureOutputIsDeterministic(t *testing.T) {
	root := fullFeatureRoot(t)
	stdout := runGoldrDeterministic(t, "routes list", "routes", "list", "--app-root", root)

	rows := routeTableRows(t, stdout)
	for _, want := range [][]string{
		{"layout", "-", "/", "-", "layout.go", "-"},
		{"page", "GET,HEAD", "/", "-", "page.go", "urls.Root.Path()"},
		{"page", "GET,HEAD", "/settings", "-", "settings/page.go", "urls.Settings.Path()"},
		{"layout", "-", "/users", "-", "users/layout.go", "-"},
		{"page", "GET,HEAD", "/users/{id}", "id", "users/by_id/page.go", "urls.Users.ByID(id).Path()"},
		{"action", "POST", "/users/create", "-", "users/actions.go:PostCreate", "urls.Users.Create.Path()"},
		{"action", "POST", "/users/save-preview", "-", "users/actions.go:PostSavePreview", "urls.Users.SavePreview.Path()"},
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
		Source:  "users/by_id/page.go",
		Helper:  "urls.Users.ByID(id).Path()",
	})
	requireRouteJSONContainsRow(t, rows, routeSurfaceJSONRow{
		Kind:    "action",
		Methods: []string{"POST"},
		Path:    "/users/save-preview",
		Params:  []string{},
		Source:  "users/actions.go:PostSavePreview",
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
		"  source   " + source("users/by_id/page.go"),
		"  params   id = \"a/b\"",
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
		"  source   " + source("users/actions.go") + " (PostCreate)",
		"LAYOUT STACK",
		"  /      " + source("layout.go"),
		"  /users " + source("users/layout.go"),
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
	writeFile(t, root, "app/routes/users/actions.go", `package users

import "net/http"

func PostPath(w http.ResponseWriter, r *http.Request) {}
`)

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
	Kind    string   `json:"kind"`
	Methods []string `json:"methods"`
	Path    string   `json:"path"`
	Params  []string `json:"params"`
	Source  string   `json:"source"`
	Helper  string   `json:"helper"`
}

func requireRouteJSONContainsRow(t *testing.T, rows []routeSurfaceJSONRow, want routeSurfaceJSONRow) {
	t.Helper()

	for _, row := range rows {
		if row.Kind == want.Kind &&
			strings.Join(row.Methods, "\x00") == strings.Join(want.Methods, "\x00") &&
			row.Path == want.Path &&
			strings.Join(row.Params, "\x00") == strings.Join(want.Params, "\x00") &&
			row.Source == want.Source &&
			row.Helper == want.Helper {
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
	lines := []string{
		"Layout map",
		"",
		rootPath,
		"└─ /  layout: " + source("layout.go"),
		"   ├─ page: GET,HEAD /  " + source("page.go"),
		"   ├─ admin/",
		"   │  └─ page: GET,HEAD /admin  " + source("admin/page.go"),
		"   ├─ protected_resource_demo/",
		"   │  ├─ page: GET,HEAD /protected-resource-demo  " + source("protected_resource_demo/page.go"),
		"   │  ├─ action (layout-aware): POST /protected-resource-demo/reveal-secret  " + source("protected_resource_demo/actions.go") + " (PostRevealSecret)",
		"   │  └─ action (layout-aware): POST /protected-resource-demo/sign-out  " + source("protected_resource_demo/actions.go") + " (PostSignOut)",
		"   ├─ settings/",
		"   │  └─ page: GET,HEAD /settings  " + source("settings/page.go"),
		"   ├─ sign_in/",
		"   │  ├─ page: GET,HEAD /sign-in  " + source("sign_in/page.go"),
		"   │  └─ action (layout-aware): POST /sign-in  " + source("sign_in/actions.go") + " (PostIndex)",
		"   └─ users/  layout: " + source("users/layout.go"),
		"      ├─ page: GET,HEAD /users  " + source("users/page.go"),
		"      ├─ by_id/",
		"      │  └─ page: GET,HEAD /users/{id}  params: id  " + source("users/by_id/page.go"),
		"      ├─ fragment (not wrapped): GET,HEAD /users/frag-table  " + source("users/frag_table.go"),
		"      ├─ action (layout-aware): POST /users/create  " + source("users/actions.go") + " (PostCreate)",
		"      └─ action (layout-aware): POST /users/save-preview  " + source("users/actions.go") + " (PostSavePreview)",
		"",
		"Rule:",
		"  pages inherit every layout above them",
		"  actions can use the same layout stack with goldr.WriteRouteResponse",
		"  fragments are not layout-wrapped",
	}
	return strings.Join(lines, "\n") + "\n"
}

func fullFeatureRouteSourcePath(source string) string {
	return filepath.ToSlash(filepath.Join("..", "..", "examples", "full_feature", "app", "routes", filepath.FromSlash(source)))
}

func fullFeatureRoutesDisplayRoot() string {
	return filepath.ToSlash(filepath.Join("..", "..", "examples", "full_feature", "app", "routes"))
}
