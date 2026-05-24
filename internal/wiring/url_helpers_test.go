package wiring

import (
	"errors"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mobiletoly/goldr/internal/routing"
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
		"func (r rootRoute) Path() string",
		"func (r usersRoute) Path() string",
		"func (r usersRoute) ByID(id string) usersByIDRoute",
		"func (r usersByIDProfileRoute) Path() string",
		"func (r orgsByOrgIDUsersByUserIDRoute) Path() string",
		"FragTable:   newUsersFragTableRoute(basePath)",
		"BuildInfo settingsBuildInfoRoute",
		"SavePreview usersSavePreviewRoute",
		"url.PathEscape(id)",
		"normalizeBasePath(basePath)",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("generated URL helper source missing %q:\n%s", want, source)
		}
	}
	if got := strings.Count(source, "func (r usersByIDProfileRoute) Path() string"); got != 1 {
		t.Fatalf("profile Path method count = %d, want 1\n%s", got, source)
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

func Page(r *http.Request) goldr.RouteResponse {
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

func Page(r *http.Request) goldr.RouteResponse {
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

func Page(r *http.Request) goldr.RouteResponse {
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

func Page(r *http.Request) goldr.RouteResponse {
	return goldr.NewPage(templ.NopComponent, goldr.PageMetadata{})
}
`)
	writeTempFile(t, tempDir, "routes/settings/build_info/page.go", `package build_info

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.RouteResponse {
	return goldr.NewPage(templ.NopComponent, goldr.PageMetadata{})
}
`)
	writeTempFile(t, tempDir, "routes/orgs/by_org_id/users/by_user_id/page.go", `package by_user_id

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.RouteResponse {
	return goldr.NewPage(templ.NopComponent, goldr.PageMetadata{})
}
`)
	writeTempFile(t, tempDir, "routes/users/actions.go", `package users

import "net/http"

func PostCreate(w http.ResponseWriter, r *http.Request) {}
func PostSavePreview(w http.ResponseWriter, r *http.Request) {}
`)
	writeTempFile(t, tempDir, "routes/users/by_id/actions.go", `package by_id

import "net/http"

func PatchProfile(w http.ResponseWriter, r *http.Request) {}
func DeleteProfile(w http.ResponseWriter, r *http.Request) {}
`)
	writeTempFile(t, tempDir, "routes/users/frag_table.go", `package users

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func FragTable(r *http.Request) goldr.RouteResponse {
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
		{"fragment", Users.FragTable.Path(), "/users/frag-table"},
		{"root dynamic", BySlug("x/y").Path(), "/x%2Fy"},
		{"dynamic", Users.ByID("a/b").Path(), "/users/a%2Fb"},
		{"dynamic empty", Users.ByID("").Path(), "/users/"},
		{"profile", Users.ByID("a b").Profile.Path(), "/users/a%20b/profile"},
		{"nested", Orgs.ByOrgID("o/1").Users.ByUserID("u/2").Path(), "/orgs/o%2F1/users/u%2F2"},
		{"mounted root", WithBasePath("/webapp").Root.Path(), "/webapp/"},
		{"mounted static", WithBasePath("/webapp").Users.Path(), "/webapp/users"},
		{"mounted action", WithBasePath("/webapp").Users.Create.Path(), "/webapp/users/create"},
		{"mounted fragment", WithBasePath("/webapp").Users.FragTable.Path(), "/webapp/users/frag-table"},
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
	path := filepath.Join("..", "..", "examples", "full_feature", "app", "urls", URLGeneratedFileName)
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}

	if string(got) != string(source) {
		t.Fatalf("%s is stale\n--- got ---\n%s\n--- want ---\n%s", path, got, source)
	}
}
