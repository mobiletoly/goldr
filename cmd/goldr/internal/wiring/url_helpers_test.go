package wiring

import (
	"errors"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
)

func TestGenerateURLHelpersWritesRouteNodes(t *testing.T) {
	source := generateURLHelpersOK(t, urlHelperManifest())

	for _, want := range []string{
		"package urls",
		"var Root = newRootRoute(\"\")",
		"var Orgs = newOrgsRoute(\"\")",
		"var Settings = newSettingsRoute(\"\")",
		"var Users = newUsersRoute(\"\")",
		"type MountedRoutes struct",
		"func WithBasePath(basePath string) MountedRoutes",
		"func BySlug(slug string) bySlugRoute",
		"func (r MountedRoutes) BySlug(slug string) bySlugRoute",
		"type goldrURLPath string",
		"func (p goldrURLPath) Path() string",
		"type usersRoute struct {\n\tgoldrURLPath",
		"type usersCreateRoute = goldrURLPath",
		"type usersTableRoute = goldrURLPath",
		"type usersByIDRoute struct {\n\tgoldrURLPath",
		"func (r usersRoute) ByID(id string) usersByIDRoute",
		`Table:        usersTableRoute(path + "/table")`,
		"BuildInfo settingsBuildInfoRoute",
		"SavePreview usersSavePreviewRoute",
		"url.PathEscape(id)",
		"normalizeBasePath(basePath)",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("generated URL helper source missing %q:\n%s", want, source)
		}
	}
	if got := strings.Count(source, "func (p goldrURLPath) Path() string"); got != 1 {
		t.Fatalf("goldrURLPath Path method count = %d, want 1\n%s", got, source)
	}
	if strings.Contains(source, "func (r usersByIDProfileRoute) Path() string") {
		t.Fatalf("generated URL helper source emits per-route Path method:\n%s", source)
	}
	if strings.Contains(source, "func (r orgsRoute) Path() string") {
		t.Fatalf("namespace-only helper exposes Path method:\n%s", source)
	}
	if strings.Contains(source, "type usersCreateRoute struct") {
		t.Fatalf("leaf path-only helper emits struct instead of alias:\n%s", source)
	}
	if strings.Contains(source, "func newUsersCreateRoute(") {
		t.Fatalf("leaf path-only helper emits constructor instead of inline alias construction:\n%s", source)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), URLGeneratedFileName, source, parser.SkipObjectResolution); err != nil {
		t.Fatalf("ParseFile() error = %v\n%s", err, source)
	}
}

func TestGenerateURLHelpersIsDeterministic(t *testing.T) {
	manifest := urlHelperManifest()
	first := generateURLHelpersOK(t, manifest)
	second := generateURLHelpersOK(t, manifest)

	if first != second {
		t.Fatalf("generated URL helper source differs between runs")
	}
}

func TestGenerateURLHelpersUsesRoutePathForIndexFragments(t *testing.T) {
	source := generateURLHelpersOK(t, routing.Manifest{
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/users/status-options",
				GoFile: "users/status_options/route.go",
				Kind:   "local",
				Fragments: []routing.RouteFragmentDeclaration{
					{Name: "index", SymbolName: "Index", Index: true, Handler: "options"},
				},
			},
		},
	})

	for _, want := range []string{
		"var Users = newUsersRoute(\"\")",
		"StatusOptions usersStatusOptionsRoute",
		`path := basePath + "/users"`,
		"type usersStatusOptionsRoute = goldrURLPath",
		`StatusOptions: usersStatusOptionsRoute(path + "/status-options")`,
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("generated URL helper source missing %q:\n%s", want, source)
		}
	}
	if strings.Contains(source, `"/" + "users"`) {
		t.Fatalf("generated URL helper source uses segment-by-segment static concatenation:\n%s", source)
	}
	if strings.Contains(source, "Index") {
		t.Fatalf("generated URL helper source uses fragment kind/name:\n%s", source)
	}
}

func TestGenerateMountURLHelpersWritesMountRelativeRouteNodes(t *testing.T) {
	source := generateMountURLHelpersOK(t, mountedURLHelperManifest(), "reports")

	for _, want := range []string{
		"package reports",
		"type GoldrMountURLs struct",
		"func NewGoldrMountURLs(route interface{ Path() string }) GoldrMountURLs",
		"func newGoldrMountURLs(mountPath string) GoldrMountURLs",
		"func (r GoldrMountURLs) Path() string",
		"type goldrURLPath string",
		"func (p goldrURLPath) Path() string",
		"ByID(id string) goldrMountByIDURL",
		"Table    goldrMountTableURL",
		"type goldrMountTableURL = goldrURLPath",
		"Export   goldrMountExportURL",
		"url.PathEscape(id)",
		"normalizeGoldrMountPath(mountPath)",
		"goldrMountRootPath(r.basePath)",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("generated mounted URL helper source missing %q:\n%s", want, source)
		}
	}
	for _, unwanted := range []string{
		"var Root",
		"Root     goldrMountRootURL",
		"goldrMountRootURL",
		"func WithBasePath",
		"type MountedRoutes",
		`"/" + "admin"`,
		`"/" + "user"`,
		"func (r goldrMountTableURL) Path() string",
		"func newGoldrMountTableURL(",
	} {
		if strings.Contains(source, unwanted) {
			t.Fatalf("generated mounted URL helper source contains %q:\n%s", unwanted, source)
		}
	}
	if _, err := parser.ParseFile(token.NewFileSet(), URLGeneratedFileName, source, parser.SkipObjectResolution); err != nil {
		t.Fatalf("ParseFile() error = %v\n%s", err, source)
	}
}

func TestGenerateMountURLHelpersIncludesMountedSourceRoutesSelectedByOneOwner(t *testing.T) {
	manifest := mountedURLHelperManifest()
	manifest.MountSource = append(manifest.MountSource, routing.ManifestMountSourceRoute{
		MountPath: "reports",
		Route:     "/audit",
		Source:    "../mounts/reports/audit/route.go",
		Page:      &routing.RouteHandlerDeclaration{Handler: "Audit"},
	})

	source := generateMountURLHelpersOK(t, manifest, "reports")

	for _, want := range []string{
		"Audit    goldrMountAuditURL",
		"ByID(id string) goldrMountByIDURL",
		"Table    goldrMountTableURL",
		"type goldrMountAuditURL = goldrURLPath",
		`Audit:    goldrMountAuditURL(normalizedMountPath + "/audit")`,
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("generated mounted URL helper source missing %q:\n%s", want, source)
		}
	}
	for _, unwanted := range []string{
		`"/" + "admin"`,
		`"/" + "user"`,
	} {
		if strings.Contains(source, unwanted) {
			t.Fatalf("generated mounted URL helper source contains owner path %q:\n%s", unwanted, source)
		}
	}
}

func TestGenerateMountURLHelpersCompileAndEscapeParams(t *testing.T) {
	manifest := mountedURLHelperManifest()
	tempDir := tempGoldrModule(t)
	writeTempFile(t, tempDir, "mounts/reports/goldr_gen.go", generateMountURLHelpersOK(t, manifest, "reports"))
	writeTempFile(t, tempDir, "mounts/reports/url_test.go", `package reports

import "testing"

type testRoutePath string

func (r testRoutePath) Path() string {
	return string(r)
}

func TestMountURLHelpers(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"root", NewGoldrMountURLs(testRoutePath("/admin/reports")).Path(), "/admin/reports"},
		{"root empty", NewGoldrMountURLs(testRoutePath("")).Path(), "/"},
		{"root slash", NewGoldrMountURLs(testRoutePath("/")).Path(), "/"},
		{"root missing leading slash", NewGoldrMountURLs(testRoutePath("admin/reports")).Path(), "/admin/reports"},
		{"root trailing slash", NewGoldrMountURLs(testRoutePath("/admin/reports/")).Path(), "/admin/reports"},
		{"fragment", NewGoldrMountURLs(testRoutePath("/admin/reports")).Table.Path(), "/admin/reports/table"},
		{"action", NewGoldrMountURLs(testRoutePath("/admin/reports")).Export.Path(), "/admin/reports/export"},
		{"dynamic", NewGoldrMountURLs(testRoutePath("/admin/reports")).ByID("a/b").Path(), "/admin/reports/a%2Fb"},
		{"dynamic fragment", NewGoldrMountURLs(testRoutePath("/admin/reports")).ByID("a b").Panel.Path(), "/admin/reports/a%20b/panel"},
	}
	for _, test := range tests {
		if test.got != test.want {
			t.Fatalf("%s = %q, want %q", test.name, test.got, test.want)
		}
	}
}
`)

	runGoTest(t, filepath.Join(tempDir, "mounts", "reports"))
}

func TestGenerateURLHelpersPreservesMountBaseForChildOnlyMountSelection(t *testing.T) {
	manifest := routing.Manifest{
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/admin/reports/audit",
				GoFile: "admin/reports/route.go",
				Kind:   "mounted-kit",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
				Source: "../mounts/reports/audit/route.go",
				Mount: &routing.RouteMountDeclaration{
					Path:            "reports",
					Owner:           "admin/reports/route.go",
					OwnerRoute:      "/admin/reports",
					OwnerParamCount: 0,
				},
			},
		},
		MountSource: []routing.ManifestMountSourceRoute{
			{
				MountPath: "reports",
				Route:     "/",
				Source:    "../mounts/reports/route.go",
				Page:      &routing.RouteHandlerDeclaration{Handler: "Page"},
			},
			{
				MountPath: "reports",
				Route:     "/audit",
				Source:    "../mounts/reports/audit/route.go",
				Page:      &routing.RouteHandlerDeclaration{Handler: "Audit"},
			},
		},
	}
	tempDir := tempGoldrModule(t)
	writeTempFile(t, tempDir, "urls/goldr_gen.go", generateURLHelpersOK(t, manifest))
	writeTempFile(t, tempDir, "mounts/reports/goldr_gen.go", generateMountURLHelpersOK(t, manifest, "reports"))
	writeTempFile(t, tempDir, "mount_binding_test.go", `package app_test

import (
	"testing"

	reports "example.com/app/mounts/reports"
	"example.com/app/urls"
)

func TestChildOnlyMountURLBinding(t *testing.T) {
	reportURLs := reports.NewGoldrMountURLs(urls.Admin.Reports)
	if got, want := reportURLs.Path(), "/admin/reports"; got != want {
		t.Fatalf("reportURLs.Path() = %q, want %q", got, want)
	}
	if got, want := reportURLs.Audit.Path(), "/admin/reports/audit"; got != want {
		t.Fatalf("reportURLs.Audit.Path() = %q, want %q", got, want)
	}
}
`)

	runGoTest(t, tempDir)
}

func TestGenerateMountURLHelpersDropsOwnerParams(t *testing.T) {
	source := generateMountURLHelpersOK(t, routing.Manifest{
		MountSource: []routing.ManifestMountSourceRoute{
			{
				MountPath: "reports",
				Route:     "/{id}",
				Params:    []string{"id"},
				Source:    "../mounts/reports/by_id/route.go",
				Page:      &routing.RouteHandlerDeclaration{Handler: "Page"},
			},
		},
	}, "reports")

	if !strings.Contains(source, "func (r GoldrMountURLs) ByID(id string) goldrMountByIDURL") {
		t.Fatalf("generated mounted URL helper source missing mount-local dynamic param:\n%s", source)
	}
	if strings.Contains(source, "orgID") || strings.Contains(source, "org_id") {
		t.Fatalf("generated mounted URL helper source includes owner param:\n%s", source)
	}
}

func TestGenerateURLHelpersCompileAndEscapeParams(t *testing.T) {
	manifest := urlHelperManifest()
	tempDir := tempGoldrModule(t)
	writeTempFile(t, tempDir, "urls/goldr_gen.go", generateURLHelpersOK(t, manifest))
	writeGeneratedRoutes(t, tempDir, generateOK(t, manifest))
	writeTempFile(t, tempDir, "routes/page.go", `package routes

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
	"example.com/app/urls"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	_ = urls.WithBasePath("/webapp").BySlug("x/y").Path()
	return goldr.NewPage(templ.NopComponent, goldr.PageMetadata{})
}
`)
	writeTempFile(t, tempDir, "routes/users/page.go", `package users

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
	"example.com/app/urls"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	_ = urls.Users.Create.Path()
	return goldr.NewPage(templ.NopComponent, goldr.PageMetadata{})
}
`)
	writeTempFile(t, tempDir, "routes/users/by_id/page.go", `package by_id

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
	"example.com/app/urls"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	_ = urls.Users.ByID("42").Profile.Path()
	_ = urls.WithBasePath("/webapp").Users.ByID("42").Profile.Path()
	return goldr.NewPage(templ.NopComponent, goldr.PageMetadata{})
}
`)
	writeTempFile(t, tempDir, "routes/by_slug/page.go", `package by_slug

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	return goldr.NewPage(templ.NopComponent, goldr.PageMetadata{})
}
`)
	writeTempFile(t, tempDir, "routes/settings/build_info/page.go", `package build_info

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	return goldr.NewPage(templ.NopComponent, goldr.PageMetadata{})
}
`)
	writeTempFile(t, tempDir, "routes/orgs/by_org_id/users/by_user_id/page.go", `package by_user_id

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.PageRouteResponse {
	return goldr.NewPage(templ.NopComponent, goldr.PageMetadata{})
}
`)
	writeTempFile(t, tempDir, "routes/users/actions.go", `package users

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func PostCreate(r *http.Request) goldr.RouteResponse { return goldr.NoContent{} }
func PostSavePreview(r *http.Request) goldr.RouteResponse { return goldr.NoContent{} }
`)
	writeTempFile(t, tempDir, "routes/users/by_id/actions.go", `package by_id

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func PatchProfile(r *http.Request) goldr.RouteResponse { return goldr.NoContent{} }
func DeleteProfile(r *http.Request) goldr.RouteResponse { return goldr.NoContent{} }
`)
	writeTempFile(t, tempDir, "routes/users/frag_table.go", `package users

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func FragTable(r *http.Request) goldr.FragmentRouteResponse {
	return goldr.NewFragment(templ.NopComponent)
}
`)
	writeTempFile(t, tempDir, "urls/url_test.go", `package urls

import "testing"

func TestURLHelpers(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"root", Root.Path(), "/"},
		{"build info", Settings.BuildInfo.Path(), "/settings/build-info"},
		{"users", Users.Path(), "/users"},
		{"create", Users.Create.Path(), "/users/create"},
		{"save preview", Users.SavePreview.Path(), "/users/save-preview"},
		{"fragment", Users.Table.Path(), "/users/table"},
		{"root dynamic", BySlug("x/y").Path(), "/x%2Fy"},
		{"dynamic", Users.ByID("a/b").Path(), "/users/a%2Fb"},
		{"dynamic empty", Users.ByID("").Path(), "/users/"},
		{"profile", Users.ByID("a b").Profile.Path(), "/users/a%20b/profile"},
		{"nested", Orgs.ByOrgID("o/1").Users.ByUserID("u/2").Path(), "/orgs/o%2F1/users/u%2F2"},
		{"mounted root", WithBasePath("/webapp").Root.Path(), "/webapp/"},
		{"mounted static", WithBasePath("/webapp").Users.Path(), "/webapp/users"},
		{"mounted action", WithBasePath("/webapp").Users.Create.Path(), "/webapp/users/create"},
		{"mounted fragment", WithBasePath("/webapp").Users.Table.Path(), "/webapp/users/table"},
		{"mounted root dynamic", WithBasePath("/webapp").BySlug("x/y").Path(), "/webapp/x%2Fy"},
		{"mounted dynamic", WithBasePath("/webapp").Users.ByID("a/b").Path(), "/webapp/users/a%2Fb"},
		{"mounted nested", WithBasePath("/webapp").Orgs.ByOrgID("o/1").Users.ByUserID("u/2").Path(), "/webapp/orgs/o%2F1/users/u%2F2"},
		{"mounted empty base", WithBasePath("").Users.Path(), "/users"},
		{"mounted slash base", WithBasePath("/").Users.Path(), "/users"},
		{"mounted missing leading slash", WithBasePath("webapp").Users.Path(), "/webapp/users"},
		{"mounted trailing slash", WithBasePath("/webapp/").Users.Path(), "/webapp/users"},
		{"mounted repeated trailing slash", WithBasePath("/webapp///").Users.Path(), "/webapp/users"},
	}
	for _, test := range tests {
		if test.got != test.want {
			t.Fatalf("%s = %q, want %q", test.name, test.got, test.want)
		}
	}
}
`)

	runGoTest(t, tempDir)
}

func TestGenerateURLHelpersRejectsAmbiguousNames(t *testing.T) {
	tests := []struct {
		name     string
		manifest routing.Manifest
	}{
		{
			name: "top-level static child collides with Root",
			manifest: routing.Manifest{
				Actions: []routing.ManifestAction{
					{Method: "POST", Route: "/root", GoFile: "actions.go", Function: "PostRoot"},
				},
			},
		},
		{
			name: "top-level static child collides with WithBasePath",
			manifest: routing.Manifest{
				Actions: []routing.ManifestAction{
					{Method: "POST", Route: "/with-base-path", GoFile: "actions.go", Function: "PostWithBasePath"},
				},
			},
		},
		{
			name: "top-level static child collides with MountedRoutes",
			manifest: routing.Manifest{
				Actions: []routing.ManifestAction{
					{Method: "POST", Route: "/mounted-routes", GoFile: "actions.go", Function: "PostMountedRoutes"},
				},
			},
		},
		{
			name: "static child collides with Path",
			manifest: routing.Manifest{
				Actions: []routing.ManifestAction{
					{Method: "POST", Route: "/users/path", GoFile: "users/actions.go", Function: "PostPath"},
				},
			},
		},
		{
			name: "static child collides with dynamic method",
			manifest: routing.Manifest{
				Pages: []routing.ManifestPage{
					{Route: "/users/{id}", Params: []string{"id"}, Unit: completeUnit("users/by_id/page.go")},
				},
				Actions: []routing.ManifestAction{
					{Method: "POST", Route: "/users/by-id", GoFile: "users/actions.go", Function: "PostByID"},
				},
			},
		},
		{
			name: "dynamic child collides with static helper",
			manifest: routing.Manifest{
				Pages: []routing.ManifestPage{
					{Route: "/users/by-id", Unit: completeUnit("users/by_id_static/page.go")},
					{Route: "/users/{id}", Params: []string{"id"}, Unit: completeUnit("users/by_id/page.go")},
				},
			},
		},
		{
			name: "static children normalize to same name",
			manifest: routing.Manifest{
				Actions: []routing.ManifestAction{
					{Method: "POST", Route: "/users/save-preview", GoFile: "users/actions.go", Function: "PostSavePreview"},
					{Method: "PATCH", Route: "/users/save_preview", GoFile: "users/actions.go", Function: "PatchSavePreview"},
				},
			},
		},
		{
			name: "nested dynamic params reuse helper argument",
			manifest: routing.Manifest{
				Pages: []routing.ManifestPage{
					{Route: "/orgs/{id}/users/{id}", Params: []string{"id", "id"}, Unit: completeUnit("orgs/by_id/users/by_id/page.go")},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := GenerateURLHelpers(test.manifest, GenerateURLOptions{PackageName: "urls"})
			if !errors.Is(err, ErrAmbiguousURLHelper) {
				t.Fatalf("GenerateURLHelpers() error = %v, want ErrAmbiguousURLHelper", err)
			}
		})
	}
}

func TestGenerateURLHelpersRejectsInvalidPackageNames(t *testing.T) {
	tests := []string{"", "URLs", "route-urls", "1urls", "func"}

	for _, test := range tests {
		_, err := GenerateURLHelpers(routing.Manifest{}, GenerateURLOptions{PackageName: test})
		if !errors.Is(err, ErrInvalidPackageName) {
			t.Fatalf("GenerateURLHelpers(..., %q) error = %v, want ErrInvalidPackageName", test, err)
		}
	}
}

func TestFullFeatureExampleGeneratedURLHelpersAreCurrent(t *testing.T) {
	source, err := GenerateURLHelpers(fullFeatureManifest(), GenerateURLOptions{PackageName: "urls"})
	if err != nil {
		t.Fatalf("GenerateURLHelpers() error = %v, want nil", err)
	}
	path := filepath.Join(goldrRepoRoot(t), "examples", "full_feature", "app", "urls", URLGeneratedFileName)
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}

	if string(got) != string(source) {
		t.Fatalf("%s is stale\n--- got ---\n%s\n--- want ---\n%s", path, got, source)
	}
}
