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

	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
)

type runtimePage struct {
	page        routing.ManifestPage
	segments    []string
	layouts     []routing.ManifestLayout
	middlewares []routing.ManifestMiddleware
}

type runtimeFragment struct {
	fragment    routing.ManifestFragment
	route       string
	segments    []string
	layouts     []routing.ManifestLayout
	middlewares []routing.ManifestMiddleware
}

type runtimeAction struct {
	action      routing.ManifestAction
	segments    []string
	layouts     []routing.ManifestLayout
	middlewares []routing.ManifestMiddleware
}

type runtimeRoute struct {
	route     string
	navRoute  string
	params    []string
	nav       routing.RouteNavDeclaration
	navTrail  []runtimeNavStep
	trailKeys []string
	segments  []string
	page      *runtimePage
	fragment  *runtimeFragment
	action    *runtimeAction
}

type runtimeNavStep struct {
	route   string
	params  []string
	nav     routing.RouteNavDeclaration
	current bool
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
	pages, fragments, actions := executableRouteSurface(manifest)
	routes := make([]runtimeRoute, 0, len(pages)+len(fragments)+len(actions))
	for _, page := range pages {
		middlewareGoFile := page.Unit.GoFile
		if page.Unit.MiddlewareGoFile != "" {
			middlewareGoFile = page.Unit.MiddlewareGoFile
		}
		runtimePage := runtimePage{
			page:        page,
			segments:    routeSegments(page.Route),
			layouts:     layoutStack(page.Route, manifest.Layouts),
			middlewares: middlewareStack(middlewareGoFile, manifest.Middlewares),
		}
		routes = append(routes, runtimeRoute{
			route:    page.Route,
			navRoute: page.Route,
			params:   page.Params,
			nav:      cloneRuntimeRouteNav(page.Nav),
			segments: runtimePage.segments,
			page:     &runtimePage,
		})
	}
	for _, fragment := range fragments {
		route := fragmentRoute(fragment)
		middlewareGoFile := fragment.Unit.GoFile
		if fragment.Unit.MiddlewareGoFile != "" {
			middlewareGoFile = fragment.Unit.MiddlewareGoFile
		}
		runtimeFragment := runtimeFragment{
			fragment:    fragment,
			route:       route,
			segments:    routeSegments(route),
			layouts:     layoutStack(route, manifest.Layouts),
			middlewares: middlewareStack(middlewareGoFile, manifest.Middlewares),
		}
		routes = append(routes, runtimeRoute{
			route:    route,
			navRoute: fragment.RoutePrefix,
			params:   fragment.Params,
			nav:      cloneRuntimeRouteNav(fragment.Nav),
			segments: runtimeFragment.segments,
			fragment: &runtimeFragment,
		})
	}
	for _, action := range actions {
		middlewareGoFile := action.GoFile
		if action.MiddlewareGoFile != "" {
			middlewareGoFile = action.MiddlewareGoFile
		}
		runtimeAction := runtimeAction{
			action:      action,
			segments:    routeSegments(action.Route),
			layouts:     layoutStack(action.Route, manifest.Layouts),
			middlewares: middlewareStack(middlewareGoFile, manifest.Middlewares),
		}
		routes = append(routes, runtimeRoute{
			route:    action.Route,
			navRoute: action.NavRoute,
			params:   action.Params,
			nav:      cloneRuntimeRouteNav(action.Nav),
			segments: runtimeAction.segments,
			action:   &runtimeAction,
		})
	}

	if err := attachRuntimeRouteNav(routes, manifest.Routes); err != nil {
		return nil, err
	}
	if err := attachRuntimeRouteTrailKeys(routes, manifest.Routes); err != nil {
		return nil, err
	}
	if err := validateRuntimeRoutes(routes); err != nil {
		return nil, err
	}
	sortRuntimeRoutes(routes)
	return routes, nil
}

func executableRouteSurface(manifest routing.Manifest) ([]routing.ManifestPage, []routing.ManifestFragment, []routing.ManifestAction) {
	pages := slices.Clone(manifest.Pages)
	fragments := slices.Clone(manifest.Fragments)
	actions := slices.Clone(manifest.Actions)
	for _, route := range manifest.Routes {
		unit := routing.RenderUnit{
			GoFile:           route.GoFile,
			SourceGoFile:     route.Source,
			MiddlewareGoFile: route.MiddlewareGoFile,
		}
		if route.Page != nil {
			pageUnit := unit
			pageUnit.TemplFile = route.Page.TemplFile
			pageUnit.HasTempl = route.Page.HasTempl
			pages = append(pages, routing.ManifestPage{
				Route:    route.Route,
				Params:   slices.Clone(route.Params),
				Nav:      cloneRuntimeRouteNav(route.Nav),
				Unit:     pageUnit,
				Function: routePageAdapterName(route),
			})
		}
		for _, fragment := range route.Fragments {
			fragments = append(fragments, routing.ManifestFragment{
				Name:        fragment.Name,
				RoutePrefix: route.Route,
				Params:      slices.Clone(route.Params),
				Nav:         cloneRuntimeRouteNav(route.Nav),
				Unit:        unit,
				Function:    routeFragmentAdapterName(route, fragment),
				Segment:     fragment.Segment,
				Index:       fragment.Index,
			})
		}
		for _, action := range route.Actions {
			actionPath := route.Route
			if action.Segment != "" {
				if route.Route == "/" {
					actionPath = "/" + action.Segment
				} else {
					actionPath = route.Route + "/" + action.Segment
				}
			}
			actions = append(actions, routing.ManifestAction{
				Method:              action.Method,
				Route:               actionPath,
				NavRoute:            route.Route,
				Params:              slices.Clone(route.Params),
				Nav:                 cloneRuntimeRouteNav(route.Nav),
				GoFile:              route.GoFile,
				SourceGoFile:        route.Source,
				MiddlewareGoFile:    route.MiddlewareGoFile,
				Function:            routeActionAdapterName(route, action),
				Suffix:              action.SymbolName,
				Segment:             action.Segment,
				Writer:              action.Writer,
				AdapterReturnsError: action.Writer && route.Kit != nil,
			})
		}
	}
	return pages, fragments, actions
}

func attachRuntimeRouteNav(routes []runtimeRoute, declarations []routing.ManifestRouteDeclaration) error {
	for index := range routes {
		steps, err := canonicalRuntimeNavSteps(routes[index].navRoute, declarations)
		if err != nil {
			return err
		}
		routes[index].navTrail = steps
	}
	return nil
}

func attachRuntimeRouteTrailKeys(routes []runtimeRoute, declarations []routing.ManifestRouteDeclaration) error {
	keysByRoute, err := destinationTrailKeysByRoute(declarations)
	if err != nil {
		return err
	}
	for index := range routes {
		routes[index].trailKeys = slices.Clone(keysByRoute[routes[index].navRoute])
	}
	return nil
}

func canonicalRuntimeNavSteps(route string, declarations []routing.ManifestRouteDeclaration) ([]runtimeNavStep, error) {
	if route == "" {
		return nil, nil
	}
	var steps []runtimeNavStep
	seenKeys := make(map[string]string)
	for _, declaration := range declarations {
		if !routePatternAppliesTo(route, declaration.Route) {
			continue
		}
		if declaration.Nav.Label == "" && declaration.Nav.Key == "" {
			continue
		}
		if declaration.Nav.Key != "" {
			if previous, ok := seenKeys[declaration.Nav.Key]; ok {
				return nil, fmt.Errorf("%w: duplicate Nav.Key %q in canonical trail for %s between %s and %s", ErrAmbiguousRuntimeRoute, declaration.Nav.Key, route, previous, declaration.Route)
			}
			seenKeys[declaration.Nav.Key] = declaration.Route
		}
		steps = append(steps, runtimeNavStep{
			route:   declaration.Route,
			params:  slices.Clone(declaration.Params),
			nav:     cloneRuntimeRouteNav(declaration.Nav),
			current: declaration.Route == route,
		})
	}
	return steps, nil
}

func cloneRuntimeRouteNav(value routing.RouteNavDeclaration) routing.RouteNavDeclaration {
	return value
}

func routePatternAppliesTo(route string, candidate string) bool {
	if candidate == "/" {
		return true
	}
	return route == candidate || strings.HasPrefix(route, candidate+"/")
}

func renderUnitSourceGoFile(unit routing.RenderUnit) string {
	return sourceGoFile(unit.SourceGoFile, unit.GoFile)
}

func manifestActionSourceGoFile(action routing.ManifestAction) string {
	return sourceGoFile(action.SourceGoFile, action.GoFile)
}

func sourceGoFile(source string, goFile string) string {
	if source != "" {
		return source
	}
	return goFile
}

func fragmentRoute(fragment routing.ManifestFragment) string {
	if fragment.Index {
		return fragment.RoutePrefix
	}
	segment := fragment.Segment
	if segment == "" {
		segment = browserPathSegment(fragment.Name)
	}
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
		if isMountedGoFile(a.Unit.GoFile) != isMountedGoFile(b.Unit.GoFile) {
			if isMountedGoFile(a.Unit.GoFile) {
				return 1
			}
			return -1
		}
		return strings.Compare(a.RoutePrefix, b.RoutePrefix)
	})
	return stack
}

func isMountedGoFile(goFile string) bool {
	return strings.HasPrefix(goFile, "../mounts/")
}

func middlewareStack(goFile string, middlewares []routing.ManifestMiddleware) []routing.ManifestMiddleware {
	endpointDir := routeSourceDir(goFile)
	stack := make([]routing.ManifestMiddleware, 0, len(middlewares))
	for _, middleware := range middlewares {
		if sourceDirContains(routeSourceDir(middleware.GoFile), endpointDir) {
			stack = append(stack, middleware)
		}
	}
	slices.SortFunc(stack, func(a, b routing.ManifestMiddleware) int {
		dirA := routeSourceDir(a.GoFile)
		dirB := routeSourceDir(b.GoFile)
		if sourceDirDepth(dirA) != sourceDirDepth(dirB) {
			return sourceDirDepth(dirA) - sourceDirDepth(dirB)
		}
		return strings.Compare(a.GoFile, b.GoFile)
	})
	return stack
}

func routeSourceDir(goFile string) string {
	dir := path.Dir(goFile)
	if dir == "." {
		return ""
	}
	return dir
}

func sourceDirContains(parent, child string) bool {
	return parent == "" || child == parent || strings.HasPrefix(child, parent+"/")
}

func sourceDirDepth(dir string) int {
	if dir == "" {
		return 0
	}
	return len(strings.Split(dir, "/"))
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

func routeImports(routes []runtimeRoute, rootLayouts []routing.ManifestLayout, routeRootImportPath string) ([]routeImport, error) {
	dirs := make(map[string]bool)
	for _, layout := range rootLayouts {
		addImportDir(dirs, layout.Unit.GoFile)
	}
	for _, route := range routes {
		if route.page != nil {
			addImportDir(dirs, route.page.page.Unit.GoFile)
			for _, layout := range route.page.layouts {
				addImportDir(dirs, layout.Unit.GoFile)
			}
		}
		if route.fragment != nil {
			addImportDir(dirs, route.fragment.fragment.Unit.GoFile)
			for _, layout := range route.fragment.layouts {
				addImportDir(dirs, layout.Unit.GoFile)
			}
		}
		if route.action != nil {
			addImportDir(dirs, route.action.action.GoFile)
			for _, layout := range route.action.layouts {
				addImportDir(dirs, layout.Unit.GoFile)
			}
		}
		for _, middleware := range routeMiddlewares(route) {
			addImportDir(dirs, middleware.GoFile)
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
			ImportPath: routeImportPath(dir, routeRootImportPath),
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

func hasFragmentRoutes(routes []runtimeRoute) bool {
	for _, route := range routes {
		if route.fragment != nil {
			return true
		}
	}
	return false
}

func hasRequestNavRoutes(routes []runtimeRoute) bool {
	for _, route := range routes {
		if len(route.navTrail) > 0 || len(route.trailKeys) > 0 {
			return true
		}
	}
	return false
}

func hasRoutesWithoutLayouts(routes []runtimeRoute) bool {
	for _, route := range routes {
		switch {
		case route.page != nil && len(route.page.layouts) == 0:
			return true
		case route.fragment != nil && len(route.fragment.layouts) == 0:
			return true
		case route.action != nil && len(route.action.layouts) == 0:
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

func routeMiddlewares(route runtimeRoute) []routing.ManifestMiddleware {
	if route.page != nil {
		return route.page.middlewares
	}
	if route.action != nil {
		return route.action.middlewares
	}
	return route.fragment.middlewares
}

func routeMethods(route runtimeRoute) []string {
	if route.page != nil || route.fragment != nil {
		return []string{"GET", "HEAD"}
	}
	return []string{route.action.action.Method}
}

func routeImportAlias(dir string) string {
	prefix := "goldrroute_"
	if strings.HasPrefix(dir, "../mounts/") {
		prefix = "goldrmount_"
		dir = strings.TrimPrefix(dir, "../mounts/")
	}
	return prefix + strings.NewReplacer("/", "_", ".", "_", "-", "_").Replace(dir)
}

func routeImportPath(dir string, routeRootImportPath string) string {
	if strings.HasPrefix(dir, "../mounts/") {
		return path.Join(path.Dir(routeRootImportPath), strings.TrimPrefix(dir, "../"))
	}
	return routeRootImportPath + "/" + dir
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
