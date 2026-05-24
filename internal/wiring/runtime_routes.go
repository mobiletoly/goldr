// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package wiring

import (
	"fmt"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/mobiletoly/goldr/internal/routing"
)

type runtimePage struct {
	page     routing.ManifestPage
	segments []string
	layouts  []routing.ManifestLayout
}

type runtimeFragment struct {
	fragment routing.ManifestFragment
	route    string
	segments []string
}

type runtimeAction struct {
	action   routing.ManifestAction
	segments []string
	layouts  []routing.ManifestLayout
}

type runtimeRoute struct {
	route    string
	params   []string
	segments []string
	page     *runtimePage
	fragment *runtimeFragment
	action   *runtimeAction
}

type runtimePath struct {
	route    string
	params   []string
	segments []string
	routes   []runtimeRoute
}

type routeImport struct {
	Dir        string
	Alias      string
	ImportPath string
}

func runtimeRoutes(manifest routing.Manifest) ([]runtimeRoute, error) {
	routes := make([]runtimeRoute, 0, len(manifest.Pages)+len(manifest.Fragments)+len(manifest.Actions))
	for _, page := range manifest.Pages {
		runtimePage := runtimePage{
			page:     page,
			segments: routeSegments(page.Route),
			layouts:  layoutStack(page.Route, manifest.Layouts),
		}
		routes = append(routes, runtimeRoute{
			route:    page.Route,
			params:   page.Params,
			segments: runtimePage.segments,
			page:     &runtimePage,
		})
	}
	for _, fragment := range manifest.Fragments {
		route := fragmentRoute(fragment)
		runtimeFragment := runtimeFragment{
			fragment: fragment,
			route:    route,
			segments: routeSegments(route),
		}
		routes = append(routes, runtimeRoute{
			route:    route,
			params:   fragment.Params,
			segments: runtimeFragment.segments,
			fragment: &runtimeFragment,
		})
	}
	for _, action := range manifest.Actions {
		runtimeAction := runtimeAction{
			action:   action,
			segments: routeSegments(action.Route),
			layouts:  layoutStack(action.Route, manifest.Layouts),
		}
		routes = append(routes, runtimeRoute{
			route:    action.Route,
			params:   action.Params,
			segments: runtimeAction.segments,
			action:   &runtimeAction,
		})
	}

	if err := validateRuntimeRoutes(routes); err != nil {
		return nil, err
	}
	sortRuntimeRoutes(routes)
	return routes, nil
}

func fragmentRoute(fragment routing.ManifestFragment) string {
	segment := "frag-" + browserPathSegment(fragment.Name)
	if fragment.RoutePrefix == "/" {
		return "/" + segment
	}
	return fragment.RoutePrefix + "/" + segment
}

func browserPathSegment(sourceName string) string {
	return strings.ReplaceAll(sourceName, "_", "-")
}

func validateRuntimeRoutes(routes []runtimeRoute) error {
	seenMethods := make(map[string]runtimeRoute, len(routes))
	seenShapes := make(map[string]runtimeRoute, len(routes))
	for _, route := range routes {
		for _, method := range routeMethods(route) {
			key := method + " " + route.route
			if previous, ok := seenMethods[key]; ok {
				return fmt.Errorf("%w %q between %s and %s", ErrAmbiguousRuntimeRoute, key, routeGoFile(previous), routeGoFile(route))
			}
			seenMethods[key] = route
		}

		shape := routeShape(route.route)
		if previous, ok := seenShapes[shape]; ok && previous.route != route.route {
			return fmt.Errorf("%w shape %q between %s and %s", ErrAmbiguousRuntimeRoute, shape, routeGoFile(previous), routeGoFile(route))
		}
		seenShapes[shape] = route
	}
	return nil
}

func runtimePaths(routes []runtimeRoute) []runtimePath {
	groups := make(map[string][]runtimeRoute)
	for _, route := range routes {
		groups[route.route] = append(groups[route.route], route)
	}

	paths := make([]runtimePath, 0, len(groups))
	for route, group := range groups {
		sortRuntimeRoutes(group)
		paths = append(paths, runtimePath{
			route:    route,
			params:   group[0].params,
			segments: group[0].segments,
			routes:   group,
		})
	}
	sortRuntimePaths(paths)
	return paths
}

func sortRuntimeRoutes(routes []runtimeRoute) {
	slices.SortFunc(routes, func(a, b runtimeRoute) int {
		if a.route == "/" && b.route != "/" {
			return -1
		}
		if b.route == "/" && a.route != "/" {
			return 1
		}
		if staticCount(a.segments) != staticCount(b.segments) {
			return staticCount(b.segments) - staticCount(a.segments)
		}
		if len(a.segments) != len(b.segments) {
			return len(a.segments) - len(b.segments)
		}
		if a.route != b.route {
			return strings.Compare(a.route, b.route)
		}
		if result := compareMethodOrder(a, b); result != 0 {
			return result
		}
		return strings.Compare(routeGoFile(a), routeGoFile(b))
	})
}

func sortRuntimePaths(paths []runtimePath) {
	slices.SortFunc(paths, func(a, b runtimePath) int {
		if a.route == "/" && b.route != "/" {
			return -1
		}
		if b.route == "/" && a.route != "/" {
			return 1
		}
		if staticCount(a.segments) != staticCount(b.segments) {
			return staticCount(b.segments) - staticCount(a.segments)
		}
		if len(a.segments) != len(b.segments) {
			return len(a.segments) - len(b.segments)
		}
		return strings.Compare(a.route, b.route)
	})
}

func compareMethodOrder(a, b runtimeRoute) int {
	aMethods := routeMethods(a)
	bMethods := routeMethods(b)
	return methodRank(aMethods[0]) - methodRank(bMethods[0])
}

func allowHeader(routes []runtimeRoute) string {
	seen := make(map[string]bool)
	for _, route := range routes {
		for _, method := range routeMethods(route) {
			seen[method] = true
		}
	}

	var methods []string
	for _, method := range []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE"} {
		if seen[method] {
			methods = append(methods, method)
		}
	}
	return strings.Join(methods, ", ")
}

func methodRank(method string) int {
	switch method {
	case "GET":
		return 0
	case "HEAD":
		return 1
	case "POST":
		return 2
	case "PUT":
		return 3
	case "PATCH":
		return 4
	case "DELETE":
		return 5
	default:
		return 99
	}
}

func httpMethodConst(method string) string {
	switch method {
	case "POST":
		return "http.MethodPost"
	case "PUT":
		return "http.MethodPut"
	case "PATCH":
		return "http.MethodPatch"
	case "DELETE":
		return "http.MethodDelete"
	default:
		return strconv.Quote(method)
	}
}

func layoutStack(pageRoute string, layouts []routing.ManifestLayout) []routing.ManifestLayout {
	stack := make([]routing.ManifestLayout, 0, len(layouts))
	for _, layout := range layouts {
		if routePrefixMatches(layout.RoutePrefix, pageRoute) {
			stack = append(stack, layout)
		}
	}
	slices.SortFunc(stack, func(a, b routing.ManifestLayout) int {
		if len(routeSegments(a.RoutePrefix)) != len(routeSegments(b.RoutePrefix)) {
			return len(routeSegments(a.RoutePrefix)) - len(routeSegments(b.RoutePrefix))
		}
		return strings.Compare(a.RoutePrefix, b.RoutePrefix)
	})
	return stack
}

func routePrefixMatches(prefix, route string) bool {
	prefixSegments := routeSegments(prefix)
	routeSegments := routeSegments(route)
	if len(prefixSegments) > len(routeSegments) {
		return false
	}
	for index, segment := range prefixSegments {
		if segment != routeSegments[index] {
			return false
		}
	}
	return true
}

func routeImports(routes []runtimeRoute, routeRootImportPath string) ([]routeImport, error) {
	dirs := make(map[string]bool)
	for _, route := range routes {
		if route.page != nil {
			addImportDir(dirs, route.page.page.Unit.GoFile)
			for _, layout := range route.page.layouts {
				addImportDir(dirs, layout.Unit.GoFile)
			}
		}
		if route.fragment != nil {
			addImportDir(dirs, route.fragment.fragment.Unit.GoFile)
		}
		if route.action != nil {
			addImportDir(dirs, route.action.action.GoFile)
			for _, layout := range route.action.layouts {
				addImportDir(dirs, layout.Unit.GoFile)
			}
		}
	}
	delete(dirs, "")
	if len(dirs) == 0 {
		return nil, nil
	}
	if routeRootImportPath == "" {
		return nil, fmt.Errorf("%w: required for nested runtime route imports", ErrInvalidRouteRootImportPath)
	}

	result := make([]routeImport, 0, len(dirs))
	for dir := range dirs {
		result = append(result, routeImport{
			Dir:        dir,
			Alias:      routeImportAlias(dir),
			ImportPath: routeRootImportPath + "/" + dir,
		})
	}
	slices.SortFunc(result, func(a, b routeImport) int {
		return strings.Compare(a.ImportPath, b.ImportPath)
	})
	return result, nil
}

func addImportDir(dirs map[string]bool, goFile string) {
	dir := path.Dir(goFile)
	if dir == "." {
		dir = ""
	}
	if dir != "" {
		dirs[dir] = true
	}
}

func hasDynamicRoutes(routes []runtimeRoute) bool {
	for _, route := range routes {
		if len(route.params) > 0 {
			return true
		}
	}
	return false
}

func hasRenderRoutes(routes []runtimeRoute) bool {
	for _, route := range routes {
		if route.page != nil || route.fragment != nil {
			return true
		}
	}
	return false
}

func hasFragmentRoutes(routes []runtimeRoute) bool {
	for _, route := range routes {
		if route.fragment != nil {
			return true
		}
	}
	return false
}

func hasActionRoutes(routes []runtimeRoute) bool {
	for _, route := range routes {
		if route.action != nil {
			return true
		}
	}
	return false
}

func hasActionRoutesWithoutLayouts(routes []runtimeRoute) bool {
	for _, route := range routes {
		if route.action != nil && len(route.action.layouts) == 0 {
			return true
		}
	}
	return false
}

func hasSegmentRoutes(routes []runtimeRoute) bool {
	for _, route := range routes {
		if route.route != "/" {
			return true
		}
	}
	return false
}

func defaultInspectorImportPath(routeRootImportPath string) string {
	if routeRootImportPath == "" {
		return ""
	}
	return path.Join(path.Dir(routeRootImportPath), "internal/goldrinspect")
}

func packageNameForGoFile(root string, goFile string) (string, error) {
	if root == "" {
		return "", ErrInvalidRouteRootImportPath
	}
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, filepath.FromSlash(goFile)), nil, parser.PackageClauseOnly)
	if err != nil {
		return "", fmt.Errorf("parse package name from %s: %w", goFile, err)
	}
	return file.Name.Name, nil
}

func routeGoFile(route runtimeRoute) string {
	if route.page != nil {
		return route.page.page.Unit.GoFile
	}
	if route.action != nil {
		return route.action.action.GoFile
	}
	return route.fragment.fragment.Unit.GoFile
}

func routeMethods(route runtimeRoute) []string {
	if route.page != nil || route.fragment != nil {
		return []string{"GET", "HEAD"}
	}
	return []string{route.action.action.Method}
}

func routeImportAlias(dir string) string {
	return "goldrroute_" + strings.NewReplacer("/", "_").Replace(dir)
}

func routeSegments(route string) []string {
	if route == "/" {
		return nil
	}
	return strings.Split(strings.TrimPrefix(route, "/"), "/")
}

func routeShape(route string) string {
	segments := routeSegments(route)
	for index, segment := range segments {
		if isParamSegment(segment) {
			segments[index] = "{}"
		}
	}
	if len(segments) == 0 {
		return "/"
	}
	return "/" + strings.Join(segments, "/")
}

func staticCount(segments []string) int {
	count := 0
	for _, segment := range segments {
		if !isParamSegment(segment) {
			count++
		}
	}
	return count
}

func isParamSegment(segment string) bool {
	return strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}")
}

func paramSegmentName(segment string) (string, bool) {
	if !isParamSegment(segment) {
		return "", false
	}
	return strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}"), true
}
