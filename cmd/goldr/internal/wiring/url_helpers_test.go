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
		"var BySlug = newBySlugRouteNode(\"\")",
		"type MountedRoutes struct",
		"func WithBasePath(basePath string) MountedRoutes",
		"BySlug   bySlugRouteNode",
		"type goldrURLPath struct",
		"func (p goldrURLPath) Path() string",
		"func (p goldrURLPath) GoldrRoutePattern() string",
		"func (p goldrURLPath) GoldrRouteParams() []string",
		"type usersRoute struct {\n\tgoldrURLPath",
		"type usersCreateRoute = goldrURLPath",
		"type usersTableRoute = goldrURLPath",
		"type usersByIDRoute struct {\n\tgoldrURLPath",
		"type usersByIDRouteNode struct",
		"func (r usersByIDRouteNode) Bind(id string) usersByIDRoute",
		"func (r usersByIDRouteNode) GoldrRoutePattern() string",
		`Table:        usersTableRoute(goldrURLPath{path: path + "/table", pattern: "/users/table"})`,
		`ByID:         newUsersByIDRouteNode(path)`,
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
		`StatusOptions: usersStatusOptionsRoute(goldrURLPath{path: path + "/status-options", pattern: "/users/status-options"})`,
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

func TestGenerateURLHelpersWritesRouteScopedTrailKeys(t *testing.T) {
	source := generateURLHelpersOK(t, routing.Manifest{
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/dashboard",
				GoFile: "dashboard/route.go",
				Kind:   "local",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
			},
			{
				Route:  "/users/{id}/profile",
				Params: []string{"id"},
				GoFile: "users/by_id/profile/route.go",
				Kind:   "local",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
			},
			{
				Route:  "/users/{id}/settings",
				Params: []string{"id"},
				GoFile: "users/by_id/settings/route.go",
				Kind:   "local",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
			},
			{
				Route:  "/workflow",
				GoFile: "workflow/route.go",
				Kind:   "local",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
				Destinations: []routing.RouteDestinationDeclaration{
					{Name: "dashboard", SymbolName: "Dashboard", Target: []string{"Dashboard"}, TrailKey: "workflow-a"},
					{Name: "profile-workflow", SymbolName: "ProfileWorkflow", Target: []string{"Users", "ByID", "Profile"}, TrailKey: "workflow-a"},
				},
			},
			{
				Route:  "/workspaces",
				GoFile: "workspaces/route.go",
				Kind:   "local",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
				Destinations: []routing.RouteDestinationDeclaration{
					{Name: "profile", SymbolName: "Profile", Target: []string{"Users", "ByID", "Profile"}, TrailKey: "project-search"},
				},
			},
		},
	})

	for _, want := range []string{
		"type dashboardRouteTrailKeys struct {\n\tWorkflowA string\n}",
		"type usersByIDProfileRouteTrailKeys struct {\n\tProjectSearch string\n\tWorkflowA     string\n}",
		"type dashboardRoute struct {\n\tgoldrURLPath\n\tTrailKeys dashboardRouteTrailKeys\n}",
		"type usersByIDProfileRouteRef struct {\n\tTrailKeys usersByIDProfileRouteTrailKeys\n}",
		`TrailKeys:    dashboardRouteTrailKeys{WorkflowA: "workflow-a"}`,
		`TrailKeys: usersByIDProfileRouteTrailKeys{ProjectSearch: "project-search", WorkflowA: "workflow-a"}`,
		"Profile  usersByIDProfileRouteRef",
		"Settings usersByIDSettingsRouteRef",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("generated URL helper source missing %q:\n%s", want, source)
		}
	}
	if strings.Contains(source, "type usersByIDSettingsRouteTrailKeys") {
		t.Fatalf("generated URL helper source emits TrailKeys for route without accepted keys:\n%s", source)
	}
	if strings.Contains(source, "usersByIDProfileRoute struct {\n\tgoldrURLPath\n\tid        string\n\tTrailKeys") {
		t.Fatalf("generated URL helper source emits duplicate TrailKeys on bound route type:\n%s", source)
	}
	if strings.Contains(source, "const WorkflowA") {
		t.Fatalf("generated URL helper source emits global nav trail constants:\n%s", source)
	}
}

func TestGenerateURLHelpersCompileRouteScopedTrailKeys(t *testing.T) {
	manifest := routing.Manifest{
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/users/{id}/profile",
				Params: []string{"id"},
				GoFile: "users/by_id/profile/route.go",
				Kind:   "local",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
			},
			{
				Route:  "/workspaces",
				GoFile: "workspaces/route.go",
				Kind:   "local",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
				Destinations: []routing.RouteDestinationDeclaration{{
					Name:       "profile",
					SymbolName: "Profile",
					Target:     []string{"Users", "ByID", "Profile"},
					TrailKey:   "project-search",
				}},
			},
		},
	}
	tempDir := tempGoldrModule(t)
	writeTempFile(t, tempDir, "urls/goldr_gen.go", generateURLHelpersOK(t, manifest))
	writeTempFile(t, tempDir, "urls/trail_keys_test.go", `package urls

import "testing"

func TestRouteScopedTrailKeys(t *testing.T) {
	if got, want := Users.ByID.Profile.TrailKeys.ProjectSearch, "project-search"; got != want {
		t.Fatalf("Users.ByID.Profile.TrailKeys.ProjectSearch = %q, want %q", got, want)
	}
	if got, want := Users.ByID.Profile.GoldrRoutePattern(), "/users/{id}/profile"; got != want {
		t.Fatalf("Users.ByID.Profile.GoldrRoutePattern() = %q, want %q", got, want)
	}
}
`)

	runGoTest(t, filepath.Join(tempDir, "urls"))
}

func TestGenerateURLHelpersWritesMountedOwnerTrailKeys(t *testing.T) {
	source := generateURLHelpersOK(t, routing.Manifest{
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/admin/reports/audit",
				GoFile: "admin/reports/route.go",
				Kind:   "mounted-kit",
				Source: "../mounts/reports/audit/route.go",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
				Mount: &routing.RouteMountDeclaration{
					Path:       "reports",
					Owner:      "admin/reports/route.go",
					OwnerRoute: "/admin/reports",
				},
			},
			{
				Route:  "/user/reports",
				GoFile: "user/reports/route.go",
				Kind:   "mounted-kit",
				Source: "../mounts/reports/route.go",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
				Mount: &routing.RouteMountDeclaration{
					Path:       "reports",
					Owner:      "user/reports/route.go",
					OwnerRoute: "/user/reports",
				},
			},
			{
				Route:  "/workflow",
				GoFile: "workflow/route.go",
				Kind:   "local",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
				Destinations: []routing.RouteDestinationDeclaration{{
					Name:       "audit",
					SymbolName: "Audit",
					Target:     []string{"Admin", "Reports", "Audit"},
					TrailKey:   "workflow-a",
				}},
			},
		},
	})

	if !strings.Contains(source, "type adminReportsAuditRouteTrailKeys struct {\n\tWorkflowA string\n}") {
		t.Fatalf("generated URL helper source missing mounted owner trail key constants:\n%s", source)
	}
	if strings.Contains(source, "User.Reports.TrailKeys") {
		t.Fatalf("generated URL helper source emits trail keys for mounted owner without metadata:\n%s", source)
	}
}

func TestGenerateURLHelpersWritesDestinations(t *testing.T) {
	manifest := routing.Manifest{
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/analytics",
				GoFile: "analytics/route.go",
				Kind:   "local",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
				Destinations: []routing.RouteDestinationDeclaration{
					{
						Name:       "project-report",
						SymbolName: "ProjectReport",
						Target:     []string{"Workspace", "Projects", "ByID", "Report"},
						TrailKey:   "workflow-a",
					},
					{
						Name:       "projects",
						SymbolName: "Projects",
						Target:     []string{"Workspace", "Projects"},
					},
				},
			},
			{
				Route:  "/workspace/projects",
				GoFile: "workspace/projects/route.go",
				Kind:   "local",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
			},
			{
				Route:  "/workspace/projects/{id}/report",
				Params: []string{"id"},
				GoFile: "workspace/projects/by_id/report/route.go",
				Kind:   "local",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
			},
		},
	}
	source := generateURLHelpersOK(t, manifest)

	for _, want := range []string{
		"type analyticsRouteDestinations struct",
		"ProjectReport analyticsRouteProjectReportDestination",
		"Projects      analyticsRouteProjectsDestination",
		"type analyticsRouteProjectReportDestination struct",
		"func (d analyticsRouteProjectReportDestination) Bind(id string) analyticsRouteProjectReportDestinationBound1",
		"func (d analyticsRouteProjectReportDestinationBound1) Href() string",
		`return goldrURLWithTrail(path, "workflow-a")`,
		"func (d analyticsRouteProjectReportDestinationBound1) NavigationHref(nav goldr.Navigation) string",
		`return goldr.NavigationHref(path, "workflow-a", nav)`,
		"func (d analyticsRouteProjectsDestination) Href() string",
		"func goldrURLWithTrail(path string, trail string) string",
		"func goldrDestinationBasePath(sourcePath string, sourcePattern string) string",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("generated URL helper source missing %q:\n%s", want, source)
		}
	}
	if strings.Contains(source, "HrefWithQuery") {
		t.Fatalf("generated destination exposes removed HrefWithQuery helper:\n%s", source)
	}
	if strings.Contains(source, "HrefWithRequestQuery") {
		t.Fatalf("generated destination exposes removed HrefWithRequestQuery helper:\n%s", source)
	}
	if strings.Contains(source, "func (p goldrURLPath) Href() string") {
		t.Fatalf("generated route helpers expose Href on Path carrier:\n%s", source)
	}
	if strings.Contains(source, "func (d analyticsRouteProjectsDestination) NavigationHref") {
		t.Fatalf("generated destination without trail key exposes NavigationHref:\n%s", source)
	}
}

func TestGenerateURLHelpersCompileDestinations(t *testing.T) {
	manifest := routing.Manifest{
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/analytics",
				GoFile: "analytics/route.go",
				Kind:   "local",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
				Destinations: []routing.RouteDestinationDeclaration{
					{
						Name:       "project-report",
						SymbolName: "ProjectReport",
						Target:     []string{"Workspace", "Projects", "ByID", "Report"},
						TrailKey:   "workflow-a",
					},
					{
						Name:       "projects",
						SymbolName: "Projects",
						Target:     []string{"Workspace", "Projects"},
					},
				},
			},
			{
				Route:  "/workspace/projects",
				GoFile: "workspace/projects/route.go",
				Kind:   "local",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
			},
			{
				Route:  "/workspace/projects/{id}/report",
				Params: []string{"id"},
				GoFile: "workspace/projects/by_id/report/route.go",
				Kind:   "local",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
			},
		},
	}
	tempDir := tempGoldrModule(t)
	writeTempFile(t, tempDir, "urls/goldr_gen.go", generateURLHelpersOK(t, manifest))
	writeTempFile(t, tempDir, "urls/destinations_test.go", `package urls

import (
	"testing"

	"github.com/mobiletoly/goldr"
)

func TestDestinations(t *testing.T) {
	if got, want := Analytics.Destinations.ProjectReport.Bind("a/b").Href(), "/workspace/projects/a%2Fb/report?_goldr_nav_trail_key=workflow-a"; got != want {
		t.Fatalf("contextual href = %q, want %q", got, want)
	}
	if got, want := Analytics.Destinations.ProjectReport.Bind("a/b").NavigationHref(goldr.Navigation{}), "/workspace/projects/a%2Fb/report?_goldr_nav_trail_key=workflow-a"; got != want {
		t.Fatalf("zero-navigation href = %q, want %q", got, want)
	}
	if got, want := Analytics.Destinations.Projects.Href(), "/workspace/projects"; got != want {
		t.Fatalf("clean destination href = %q, want %q", got, want)
	}
	if got, want := Analytics.Path(), "/analytics"; got != want {
		t.Fatalf("route Path() = %q, want %q", got, want)
	}
	if got, want := WithBasePath("/webapp").Analytics.Destinations.ProjectReport.Bind("42").Href(), "/webapp/workspace/projects/42/report?_goldr_nav_trail_key=workflow-a"; got != want {
		t.Fatalf("base-path contextual href = %q, want %q", got, want)
	}
	if got, want := goldrURLWithTrail("/workspace/projects?tab=active", "workflow-a"), "/workspace/projects?tab=active&_goldr_nav_trail_key=workflow-a"; got != want {
		t.Fatalf("query-preserving trail href = %q, want %q", got, want)
	}
}
`)

	runGoTest(t, filepath.Join(tempDir, "urls"))
}

func TestGenerateURLHelpersCompileRouteRefDestinationsWithBasePath(t *testing.T) {
	manifest := routing.Manifest{
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/main/hq/teams/{team_id}/analytics",
				Params: []string{"team_id"},
				GoFile: "main/hq/teams/by_team_id/analytics/route.go",
				Kind:   "local",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
				Destinations: []routing.RouteDestinationDeclaration{
					{
						Name:       "project-report",
						SymbolName: "ProjectReport",
						Target:     []string{"Main", "Reports", "ByProjectID"},
					},
				},
			},
			{
				Route:  "/main/reports/{project_id}",
				Params: []string{"project_id"},
				GoFile: "main/reports/by_project_id/route.go",
				Kind:   "local",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
			},
		},
	}
	tempDir := tempGoldrModule(t)
	writeTempFile(t, tempDir, "urls/goldr_gen.go", generateURLHelpersOK(t, manifest))
	writeTempFile(t, tempDir, "urls/route_ref_destinations_test.go", `package urls

import "testing"

func TestRouteRefDestinationsKeepBasePath(t *testing.T) {
	if got, want := WithBasePath("/webapp").Main.Hq.Teams.ByTeamID.Analytics.Destinations.ProjectReport.Bind("c/1").Href(), "/webapp/main/reports/c%2F1"; got != want {
		t.Fatalf("route-ref destination href = %q, want %q", got, want)
	}
}
`)

	runGoTest(t, filepath.Join(tempDir, "urls"))
}

func TestGenerateURLHelpersCompileDestinationsTargetingNestedRoot(t *testing.T) {
	manifest := routing.Manifest{
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/analytics",
				GoFile: "analytics/route.go",
				Kind:   "local",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
				Destinations: []routing.RouteDestinationDeclaration{
					{
						Name:       "main-root",
						SymbolName: "MainRoot",
						Target:     []string{"Main", "Root"},
					},
				},
			},
			{
				Route:  "/main/root",
				GoFile: "main/root/route.go",
				Kind:   "local",
				Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
			},
		},
	}
	tempDir := tempGoldrModule(t)
	writeTempFile(t, tempDir, "urls/goldr_gen.go", generateURLHelpersOK(t, manifest))
	writeTempFile(t, tempDir, "urls/nested_root_destination_test.go", `package urls

import "testing"

func TestNestedRootDestination(t *testing.T) {
	if got, want := Analytics.Destinations.MainRoot.Href(), "/main/root"; got != want {
		t.Fatalf("nested Root destination href = %q, want %q", got, want)
	}
}
`)

	runGoTest(t, filepath.Join(tempDir, "urls"))
}

func TestGenerateURLHelpersRejectsInvalidDestinations(t *testing.T) {
	tests := []struct {
		name     string
		manifest routing.Manifest
		want     string
	}{
		{
			name: "unknown target",
			manifest: routing.Manifest{
				Routes: []routing.ManifestRouteDeclaration{
					{
						Route:  "/analytics",
						GoFile: "analytics/route.go",
						Kind:   "local",
						Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
						Destinations: []routing.RouteDestinationDeclaration{{
							Name:       "missing",
							SymbolName: "Missing",
							Target:     []string{"Missing"},
						}},
					},
				},
			},
			want: `destination "missing" targets unknown route helper Missing`,
		},
		{
			name: "owner-excluded mounted target",
			manifest: routing.Manifest{
				Routes: []routing.ManifestRouteDeclaration{
					{
						Route:  "/admin/reports",
						GoFile: "admin/reports/route.go",
						Kind:   "mounted-kit",
						Page:   &routing.RouteHandlerDeclaration{Handler: "Kit.Page"},
						Destinations: []routing.RouteDestinationDeclaration{{
							Name:       "audit",
							SymbolName: "Audit",
							Target:     []string{"Admin", "Reports", "Audit"},
							TrailKey:   "admin-reports",
						}},
					},
				},
				MountRoutes: []routing.ManifestMountRouteSelection{
					{
						MountPath: "reports",
						Owner:     "admin/reports/route.go",
						Source:    "../mounts/reports/audit/route.go",
						Route:     "/admin/reports/audit",
						Included:  false,
					},
				},
			},
			want: `destination "audit" targets unknown route helper Admin.Reports.Audit`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := GenerateURLHelpers(test.manifest, GenerateURLOptions{PackageName: "urls"})
			if !errors.Is(err, ErrAmbiguousURLHelper) {
				t.Fatalf("GenerateURLHelpers() error = %v, want ErrAmbiguousURLHelper", err)
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("GenerateURLHelpers() error = %v, want %q", err, test.want)
			}
		})
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
		"type goldrURLPath struct",
		"func (p goldrURLPath) Path() string",
		"ByID     goldrMountByIDURLNode",
		"func (r goldrMountByIDURLNode) Bind(id string) goldrMountByIDURL",
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
		"ByID     goldrMountByIDURLNode",
		"func (r goldrMountByIDURLNode) Bind(id string) goldrMountByIDURL",
		"Table    goldrMountTableURL",
		"type goldrMountAuditURL = goldrURLPath",
		`Audit:    goldrMountAuditURL(goldrURLPath{path: normalizedMountPath + "/audit", pattern: "/audit"})`,
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

import (
	"net/http"
	"testing"

	"github.com/mobiletoly/goldr"
)

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
		{"dynamic", NewGoldrMountURLs(testRoutePath("/admin/reports")).ByID.Bind("a/b").Path(), "/admin/reports/a%2Fb"},
		{"dynamic fragment", NewGoldrMountURLs(testRoutePath("/admin/reports")).ByID.Bind("a b").Panel.Path(), "/admin/reports/a%20b/panel"},
	}
	for _, test := range tests {
		if test.got != test.want {
			t.Fatalf("%s = %q, want %q", test.name, test.got, test.want)
		}
	}
}

func TestMountURLHelpersBindFromRequest(t *testing.T) {
	req := new(http.Request)
	req.SetPathValue("id", "a/b")

	route, ok := goldr.BindFromRequest(req, NewGoldrMountURLs(testRoutePath("/admin/reports")).ByID)
	if !ok {
		t.Fatal("BindFromRequest() ok = false, want true")
	}
	if got, want := route.Path(), "/admin/reports/a%2Fb"; got != want {
		t.Fatalf("BindFromRequest().Path() = %q, want %q", got, want)
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

	if !strings.Contains(source, "func (r goldrMountByIDURLNode) Bind(id string) goldrMountByIDURL") {
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
	_ = urls.WithBasePath("/webapp").BySlug.Bind("x/y").Path()
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
	_ = urls.Users.ByID.Bind("42").Profile.Path()
	_ = urls.WithBasePath("/webapp").Users.ByID.Bind("42").Profile.Path()
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

import (
	"net/http"
	"testing"

	"github.com/mobiletoly/goldr"
)

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
		{"root dynamic", BySlug.Bind("x/y").Path(), "/x%2Fy"},
		{"dynamic", Users.ByID.Bind("a/b").Path(), "/users/a%2Fb"},
		{"dynamic empty", Users.ByID.Bind("").Path(), "/users/"},
		{"profile", Users.ByID.Bind("a b").Profile.Path(), "/users/a%20b/profile"},
		{"nested", Orgs.ByOrgID.Bind("o/1").Users.ByUserID.Bind("u/2").Path(), "/orgs/o%2F1/users/u%2F2"},
		{"mounted root", WithBasePath("/webapp").Root.Path(), "/webapp/"},
		{"mounted static", WithBasePath("/webapp").Users.Path(), "/webapp/users"},
		{"mounted action", WithBasePath("/webapp").Users.Create.Path(), "/webapp/users/create"},
		{"mounted fragment", WithBasePath("/webapp").Users.Table.Path(), "/webapp/users/table"},
		{"mounted root dynamic", WithBasePath("/webapp").BySlug.Bind("x/y").Path(), "/webapp/x%2Fy"},
		{"mounted dynamic", WithBasePath("/webapp").Users.ByID.Bind("a/b").Path(), "/webapp/users/a%2Fb"},
		{"mounted nested", WithBasePath("/webapp").Orgs.ByOrgID.Bind("o/1").Users.ByUserID.Bind("u/2").Path(), "/webapp/orgs/o%2F1/users/u%2F2"},
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
	if got, want := Users.ByID.GoldrRoutePattern(), "/users/{id}"; got != want {
		t.Fatalf("Users.ByID.GoldrRoutePattern() = %q, want %q", got, want)
	}
	if got, want := Users.ByID.Bind("42").GoldrRoutePattern(), "/users/{id}"; got != want {
		t.Fatalf("Users.ByID.Bind(...).GoldrRoutePattern() = %q, want %q", got, want)
	}
	if got, want := Users.ByID.Bind("42").Profile.GoldrRoutePattern(), "/users/{id}/profile"; got != want {
		t.Fatalf("Users.ByID.Bind(...).Profile.GoldrRoutePattern() = %q, want %q", got, want)
	}
}

func TestURLHelpersBindFromRequest(t *testing.T) {
	req := new(http.Request)
	req.SetPathValue("slug", "x/y")
	req.SetPathValue("id", "a/b")
	req.SetPathValue("org_id", "o/1")
	req.SetPathValue("user_id", "u/2")

	slugRoute, ok := goldr.BindFromRequest(req, BySlug)
	if !ok {
		t.Fatal("BindFromRequest(BySlug) ok = false, want true")
	}
	if got, want := slugRoute.Path(), "/x%2Fy"; got != want {
		t.Fatalf("BindFromRequest(BySlug).Path() = %q, want %q", got, want)
	}

	userRoute, ok := goldr.BindFromRequest(req, Users.ByID)
	if !ok {
		t.Fatal("BindFromRequest(Users.ByID) ok = false, want true")
	}
	if got, want := userRoute.Profile.Path(), "/users/a%2Fb/profile"; got != want {
		t.Fatalf("BindFromRequest(Users.ByID).Profile.Path() = %q, want %q", got, want)
	}

	orgRoute, ok := goldr.BindFromRequest(req, Orgs.ByOrgID)
	if !ok {
		t.Fatal("BindFromRequest(Orgs.ByOrgID) ok = false, want true")
	}
	orgUserRoute, ok := goldr.BindFromRequest(req, orgRoute.Users.ByUserID)
	if !ok {
		t.Fatal("BindFromRequest(Users.ByUserID) ok = false, want true")
	}
	if got, want := orgUserRoute.Path(), "/orgs/o%2F1/users/u%2F2"; got != want {
		t.Fatalf("nested BindFromRequest().Path() = %q, want %q", got, want)
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
			name: "static child collides with TrailKeys",
			manifest: routing.Manifest{
				Pages: []routing.ManifestPage{
					{Route: "/users/trail-keys", Unit: completeUnit("users/trail_keys/page.go")},
				},
			},
		},
		{
			name: "static child collides with Destinations",
			manifest: routing.Manifest{
				Routes: []routing.ManifestRouteDeclaration{
					{
						Route:  "/users",
						GoFile: "users/route.go",
						Kind:   "local",
						Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
						Destinations: []routing.RouteDestinationDeclaration{{
							Name:       "projects",
							SymbolName: "Projects",
							Target:     []string{"Projects"},
						}},
					},
					{
						Route:  "/users/destinations",
						GoFile: "users/destinations/route.go",
						Kind:   "local",
						Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
					},
					{
						Route:  "/projects",
						GoFile: "projects/route.go",
						Kind:   "local",
						Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
					},
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
		{
			name: "trail keys normalize to same field",
			manifest: routing.Manifest{
				Routes: []routing.ManifestRouteDeclaration{
					{
						Route:  "/users",
						GoFile: "users/route.go",
						Kind:   "local",
						Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
					},
					{
						Route:  "/a",
						GoFile: "a/route.go",
						Kind:   "local",
						Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
						Destinations: []routing.RouteDestinationDeclaration{{
							Name:       "users-id",
							SymbolName: "UsersID",
							Target:     []string{"Users"},
							TrailKey:   "id",
						}},
					},
					{
						Route:  "/b",
						GoFile: "b/route.go",
						Kind:   "local",
						Page:   &routing.RouteHandlerDeclaration{Handler: "Page"},
						Destinations: []routing.RouteDestinationDeclaration{{
							Name:       "users-i-d",
							SymbolName: "UsersID",
							Target:     []string{"Users"},
							TrailKey:   "i-d",
						}},
					},
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
