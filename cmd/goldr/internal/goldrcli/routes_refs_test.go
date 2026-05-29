package goldrcli

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRunRoutesRefsPrintsHTMXReferences(t *testing.T) {
	root := tempRouteRefsApp(t)

	code, stdout, stderr := runGoldr(t, "routes", "refs", "--app-root", root)

	if code != 0 {
		t.Fatalf("Run(routes refs) exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	rows := routeTableRows(t, stdout)
	for _, want := range [][]string{
		{"STATUS", "METHOD", "ROUTE", "KIND", "ATTRIBUTE", "SOURCE", "VALUE"},
		{"resolved", "POST", "/users/create", "action", "hx-post", "users/page.templ:6:8", "urls.Users.Create.Path()"},
		{"resolved", "GET", "/users/table", "fragment", "data-hx-get", "users/page.templ:8:5", "/users/table?status=inactive"},
		{"resolved", "GET", "/users/{id}", "page", "hx-get", "users/page.templ:9:5", "urls.Users.ByID.Bind(userID).Path()"},
		{"dynamic", "GET", "-", "-", "hx-get", "users/page.templ:10:5", "href"},
		{"unmatched", "POST", "/missing", "-", "hx-post", "users/page.templ:11:5", "/missing"},
		{"external", "GET", "-", "-", "hx-get", "users/page.templ:12:5", "https://example.com/users"},
		{"invalid", "GET", "-", "-", "hx-get", "users/page.templ:13:5", "users/table"},
	} {
		requireRouteTableContainsRow(t, rows, want)
	}
	if !strings.Contains(stdout, `"urls.Users.Table.Path() + \"?status=active\""`) {
		t.Fatalf("stdout = %q, want quoted helper-plus-query value", stdout)
	}
	requireMissingFile(t, filepath.Join(root, "app", "routes", "goldr_gen.go"))
	requireMissingFile(t, filepath.Join(root, "app", "urls", "goldr_gen.go"))
}

func TestRunRoutesRefsPrintsJSON(t *testing.T) {
	root := tempRouteRefsApp(t)

	stdout := runGoldrDeterministic(t, "routes refs --json", "routes", "refs", "--app-root", root, "--json")

	var rows []routeRefsJSONTestRow
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil {
		t.Fatalf("Unmarshal(routes refs --json) error = %v; stdout = %q", err, stdout)
	}
	want := []routeRefsJSONTestRow{
		{
			Status: "resolved", Method: "POST", Route: "/users/create", Kind: "action", Attribute: "hx-post",
			Source: "users/page.templ", Line: 6, Column: 8, Value: "urls.Users.Create.Path()",
			Matched: &routeRefsJSONTestMatch{Path: "/users/create", Kind: "action", Source: "users/route.go:GoldrRoutePostCreate", Helper: "urls.Users.Create.Path()"},
		},
		{
			Status: "resolved", Method: "GET", Route: "/users/table", Kind: "fragment", Attribute: "hx-get",
			Source: "users/page.templ", Line: 7, Column: 10, Value: `urls.Users.Table.Path() + "?status=active"`,
			Matched: &routeRefsJSONTestMatch{Path: "/users/table", Kind: "fragment", Source: "users/route.go", Helper: "urls.Users.Table.Path()"},
		},
		{
			Status: "resolved", Method: "GET", Route: "/users/table", Kind: "fragment", Attribute: "data-hx-get",
			Source: "users/page.templ", Line: 8, Column: 5, Value: "/users/table?status=inactive",
			Matched: &routeRefsJSONTestMatch{Path: "/users/table", Kind: "fragment", Source: "users/route.go", Helper: "urls.Users.Table.Path()"},
		},
		{
			Status: "resolved", Method: "GET", Route: "/users/{id}", Kind: "page", Attribute: "hx-get",
			Source: "users/page.templ", Line: 9, Column: 5, Value: "urls.Users.ByID.Bind(userID).Path()",
			Matched: &routeRefsJSONTestMatch{Path: "/users/{id}", Kind: "page", Source: "users/by_id/route.go", Helper: "urls.Users.ByID.Bind(id).Path()"},
		},
		{Status: "dynamic", Method: "GET", Attribute: "hx-get", Source: "users/page.templ", Line: 10, Column: 5, Value: "href"},
		{Status: "unmatched", Method: "POST", Route: "/missing", Attribute: "hx-post", Source: "users/page.templ", Line: 11, Column: 5, Value: "/missing"},
		{Status: "external", Method: "GET", Attribute: "hx-get", Source: "users/page.templ", Line: 12, Column: 5, Value: "https://example.com/users"},
		{Status: "invalid", Method: "GET", Attribute: "hx-get", Source: "users/page.templ", Line: 13, Column: 5, Value: "users/table"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("JSON rows = %#v, want %#v", rows, want)
	}
	requireMissingFile(t, filepath.Join(root, "app", "routes", "goldr_gen.go"))
	requireMissingFile(t, filepath.Join(root, "app", "urls", "goldr_gen.go"))
}

func TestRunRoutesRefsReportsTemplParseErrors(t *testing.T) {
	root := t.TempDir()
	writeRouteListFixture(t, root)
	writeFile(t, root, "app/routes/page.templ", "package routes\n\ntempl Broken() { <div> }\n")

	code, stdout, stderr := runGoldr(t, "routes", "refs", "--app-root", root)

	if code != 1 {
		t.Fatalf("Run(routes refs parse error) exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	for _, want := range []string{"goldr routes refs:", "app/routes/page.templ"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr = %q, want %q", stderr, want)
		}
	}
}

func tempRouteRefsApp(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeFile(t, root, "app/routes/users/route.go", routeDeclarationSource("users", "page", routeDeclarationOptions{
		Page:      true,
		Fragments: []string{"table"},
		Actions:   []routeDeclarationAction{{Helper: "Action", Name: "create", Func: "postCreate"}},
	}))
	writeFile(t, root, "app/routes/users/by_id/route.go", routeDeclarationSource("by_id", "page", routeDeclarationOptions{Page: true}))
	writeFile(t, root, "app/routes/users/page.templ", `package users

import "example.com/refs/app/urls"

templ PageView(href string, userID string) {
	<section>
		<form hx-post={ urls.Users.Create.Path() }></form>
		<button hx-get={ urls.Users.Table.Path() + "?status=active" }>Active</button>
		<a data-hx-get="/users/table?status=inactive">Inactive</a>
		<a hx-get={ urls.Users.ByID.Bind(userID).Path() }>User</a>
		<a hx-get={ href }>Dynamic</a>
		<a hx-post="/missing">Missing</a>
		<a hx-get="https://example.com/users">External</a>
		<a hx-get="users/table">Invalid</a>
	</section>
}
`)
	return root
}

type routeRefsJSONTestRow struct {
	Status    string                  `json:"status"`
	Method    string                  `json:"method"`
	Route     string                  `json:"route"`
	Kind      string                  `json:"kind"`
	Attribute string                  `json:"attribute"`
	Source    string                  `json:"source"`
	Line      int                     `json:"line"`
	Column    int                     `json:"column"`
	Value     string                  `json:"value"`
	Matched   *routeRefsJSONTestMatch `json:"matched,omitempty"`
}

type routeRefsJSONTestMatch struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Source string `json:"source"`
	Helper string `json:"helper"`
}
