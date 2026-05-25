// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/middlewarescan"
)

const (
	goFileExtension            = ".go"
	goTestFileSuffix           = "_test" + goFileExtension
	templGeneratedGoFileSuffix = "_templ" + goFileExtension
	templFileExtension         = ".templ"

	layoutRenderUnit = "layout"
	fragmentPrefix   = "frag_"

	pageGoFile    = "page" + goFileExtension
	layoutGoFile  = layoutRenderUnit + goFileExtension
	routeGoFile   = "route" + goFileExtension
	actionsGoFile = "actions" + goFileExtension

	dynamicRoutePrefix  = "by_"
	dynamicRoutePattern = dynamicRoutePrefix + "<param>"
	goInternalDir       = "internal"
	goIgnoredTestdata   = "testdata"
	goVendorDir         = "vendor"
)

var routeIdentPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// Tree is the scanner output for one route root.
type Tree struct {
	Root        string
	MountRoot   string
	Pages       []Page
	Layouts     []Layout
	Fragments   []Fragment
	Actions     []Action
	Middlewares []Middleware
	Routes      []RouteDeclaration
}

type Page struct {
	Route     string
	Params    []string
	GoFile    string
	TemplFile string
	HasTempl  bool
}

type Layout struct {
	RoutePrefix string
	Params      []string
	GoFile      string
	TemplFile   string
	HasTempl    bool
}

type Fragment struct {
	Name        string
	RoutePrefix string
	Params      []string
	GoFile      string
	TemplFile   string
	HasTempl    bool
	Index       bool
}

type Action struct {
	Method   string
	Route    string
	Params   []string
	GoFile   string
	Function string
	Suffix   string
	Segment  string
	Writer   bool
}

type Middleware struct {
	RoutePrefix string
	Params      []string
	GoFile      string
}

type Problem struct {
	Path    string
	Message string
}

type ScanError struct {
	Problems []Problem
}

type scanMode int

const (
	scanModeLive scanMode = iota
	scanModeMounted
)

func (err *ScanError) Error() string {
	if len(err.Problems) == 0 {
		return "routing scan failed"
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "routing scan found %d problem(s)", len(err.Problems))
	for _, problem := range err.Problems {
		fmt.Fprintf(&builder, "; %s: %s", problem.Path, problem.Message)
	}
	return builder.String()
}

func Scan(root string) (*Tree, error) {
	return scan(root, scanModeLive)
}

func scan(root string, mode scanMode) (*Tree, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("scan route root %q: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scan route root %q: not a directory", root)
	}

	scanner := scanner{
		root: filepath.Clean(root),
		mode: mode,
		tree: &Tree{
			Root: filepath.Clean(root),
		},
	}

	scanner.scanDir("", nil, nil)
	scanner.sort()

	if len(scanner.problems) > 0 {
		return scanner.tree, &ScanError{Problems: scanner.problems}
	}

	return scanner.tree, nil
}

func ScanWithMounts(root string, mountRoot string) (*Tree, error) {
	tree, err := Scan(root)
	if err != nil {
		return tree, err
	}
	tree.MountRoot = filepath.Clean(mountRoot)
	expander := mountExpander{
		root:      filepath.Clean(root),
		mountRoot: filepath.Clean(mountRoot),
		tree:      tree,
	}
	expander.expand()
	if len(expander.problems) > 0 {
		return tree, &ScanError{Problems: expander.problems}
	}
	return tree, nil
}

type scanner struct {
	root     string
	mode     scanMode
	tree     *Tree
	problems []Problem
}

type routeSegment struct {
	pathSegment string
	paramName   string
}

func (scanner *scanner) scanDir(relDir string, routeSegments []string, params []string) {
	entries, err := os.ReadDir(filepath.Join(scanner.root, filepath.FromSlash(relDir)))
	if err != nil {
		scanner.addProblem(relDir, err.Error())
		return
	}

	files := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			files[entry.Name()] = true
		}
	}

	route := routePath(routeSegments)
	dirParams := slices.Clone(params)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		scanner.scanFile(relDir, entry.Name(), route, dirParams, files)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		childRel := joinPath(relDir, name)
		segment, ok := scanner.routeSegment(childRel, name)
		if !ok {
			continue
		}

		childParams := slices.Clone(params)
		if segment.paramName != "" {
			childParams = append(childParams, segment.paramName)
		}
		scanner.scanDir(childRel, append(slices.Clone(routeSegments), segment.pathSegment), childParams)
	}
}

func (scanner *scanner) scanFile(relDir, name, route string, params []string, files map[string]bool) {
	relPath := joinPath(relDir, name)

	if strings.HasSuffix(name, goFileExtension) && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
		scanner.addProblem(relPath, "Go route files must not start with . or _")
		return
	}
	if strings.HasSuffix(name, goTestFileSuffix) {
		return
	}
	if strings.HasSuffix(name, templGeneratedGoFileSuffix) {
		return
	}
	if isOldRouteSurfaceFile(name) {
		scanner.addProblem(relPath, "route surface belongs in route.go")
		return
	}

	switch name {
	case routeGoFile:
		routeDeclaration, err := scanRouteDeclaration(filepath.Join(scanner.root, filepath.FromSlash(relPath)))
		if err != nil {
			scanner.addRouteDeclarationProblems(relPath, err)
			return
		}
		routeDeclaration.Route = route
		routeDeclaration.Params = slices.Clone(params)
		routeDeclaration.GoFile = relPath
		scanner.validateRouteDeclaration(relPath, routeDeclaration)
		scanner.tree.Routes = append(scanner.tree.Routes, routeDeclaration)
	case layoutGoFile:
		templFile, hasTempl := pairFile(relDir, layoutRenderUnit, files)
		scanner.tree.Layouts = append(scanner.tree.Layouts, Layout{
			RoutePrefix: route,
			Params:      slices.Clone(params),
			GoFile:      relPath,
			TemplFile:   templFile,
			HasTempl:    hasTempl,
		})
	case middlewarescan.FileName:
		if err := middlewarescan.Scan(filepath.Join(scanner.root, filepath.FromSlash(relPath))); err != nil {
			scanner.addMiddlewareProblems(relPath, err)
			return
		}
		scanner.tree.Middlewares = append(scanner.tree.Middlewares, Middleware{
			RoutePrefix: route,
			Params:      slices.Clone(params),
			GoFile:      relPath,
		})
	}
}

func (scanner *scanner) validateRouteDeclaration(relPath string, route RouteDeclaration) {
	switch scanner.mode {
	case scanModeLive:
		if route.Kind == routeDeclarationKindKit && route.Kit != nil && route.Kit.New == "" {
			scanner.addProblem(relPath, "KitRouteDef requires New under app/routes")
		}
	case scanModeMounted:
		switch route.Kind {
		case routeDeclarationKindKit:
			if route.Kit != nil && route.Kit.New != "" {
				scanner.addProblem(relPath, "KitRouteDef.New is not supported under app/mounts; the KitRouteMount owner supplies New")
			}
		case routeDeclarationKindLocal:
			scanner.addProblem(relPath, "mounted route files must use goldr.KitRouteDef[K]")
		case routeDeclarationKindKitMount:
			scanner.addProblem(relPath, "KitRouteMount is only supported under app/routes")
		}
	}
}

func isOldRouteSurfaceFile(name string) bool {
	return name == pageGoFile ||
		name == actionsGoFile ||
		(strings.HasPrefix(name, fragmentPrefix) && strings.HasSuffix(name, goFileExtension))
}

func (scanner *scanner) routeSegment(relPath, name string) (routeSegment, bool) {
	switch {
	case strings.HasPrefix(name, "."):
		scanner.addProblem(relPath, "route directories must not start with .")
		return routeSegment{}, false
	case strings.HasPrefix(name, "_"):
		scanner.addProblem(relPath, "route directories must not start with _")
		return routeSegment{}, false
	case isGoSpecialDir(name):
		return routeSegment{}, false
	case strings.HasPrefix(name, dynamicRoutePrefix):
		param := strings.TrimPrefix(name, dynamicRoutePrefix)
		if !isRouteIdent(param) {
			scanner.addProblem(relPath, "dynamic route directories must use "+dynamicRoutePattern+" with a lowercase Go-safe parameter")
			return routeSegment{}, false
		}
		return routeSegment{
			pathSegment: "{" + param + "}",
			paramName:   param,
		}, true
	case !isRouteIdent(name):
		scanner.addProblem(relPath, "static route directories must use lowercase Go-safe names")
		return routeSegment{}, false
	default:
		return routeSegment{pathSegment: browserPathSegment(name)}, true
	}
}

func isGoSpecialDir(name string) bool {
	return name == goInternalDir || name == goIgnoredTestdata || name == goVendorDir
}

func (scanner *scanner) sort() {
	slices.SortFunc(scanner.tree.Pages, func(a, b Page) int {
		return compareRouteOrder(a.Route, a.GoFile, b.Route, b.GoFile)
	})
	slices.SortFunc(scanner.tree.Layouts, func(a, b Layout) int {
		return compareRouteOrder(a.RoutePrefix, a.GoFile, b.RoutePrefix, b.GoFile)
	})
	slices.SortFunc(scanner.tree.Fragments, func(a, b Fragment) int {
		return compareFragmentOrder(a.RoutePrefix, a.Name, a.GoFile, b.RoutePrefix, b.Name, b.GoFile)
	})
	slices.SortFunc(scanner.tree.Actions, func(a, b Action) int {
		return compareActionOrder(a.Route, a.Method, a.Function, b.Route, b.Method, b.Function)
	})
	slices.SortFunc(scanner.tree.Middlewares, func(a, b Middleware) int {
		return compareRouteOrder(a.RoutePrefix, a.GoFile, b.RoutePrefix, b.GoFile)
	})
	slices.SortFunc(scanner.tree.Routes, func(a, b RouteDeclaration) int {
		return compareRouteOrder(a.Route, a.GoFile, b.Route, b.GoFile)
	})
}

func (scanner *scanner) addProblem(relPath, message string) {
	if relPath == "" {
		relPath = "."
	}
	scanner.problems = append(scanner.problems, Problem{
		Path:    relPath,
		Message: message,
	})
}

type mountExpander struct {
	root      string
	mountRoot string
	tree      *Tree
	problems  []Problem
}

func (expander *mountExpander) expand() {
	originalRoutes := slices.Clone(expander.tree.Routes)
	nextRoutes := make([]RouteDeclaration, 0, len(originalRoutes))
	for _, route := range originalRoutes {
		if route.Kind != routeDeclarationKindKitMount {
			nextRoutes = append(nextRoutes, route)
			continue
		}
		nextRoutes = append(nextRoutes, expander.expandMount(route)...)
	}
	expander.tree.Routes = nextRoutes
	expander.sort()
}

func (expander *mountExpander) expandMount(owner RouteDeclaration) []RouteDeclaration {
	if owner.Mount == nil {
		return nil
	}
	mountPath, ok := cleanMountPath(owner.Mount.Path)
	if !ok {
		expander.addProblem(owner.GoFile, "Mount must be a clean relative path under app/mounts using lowercase Go-safe slash components")
		return nil
	}

	mountedRoot := filepath.Join(expander.mountRoot, filepath.FromSlash(mountPath))
	mountedTree, err := scan(mountedRoot, scanModeMounted)
	if err != nil {
		expander.addMountScanProblems(owner.GoFile, mountPath, err)
		if mountedTree == nil {
			return nil
		}
	}

	for _, middleware := range mountedTree.Middlewares {
		expander.addProblem(prefixedMountPath(mountPath, middleware.GoFile), "middleware.go is not supported in app/mounts")
	}
	if len(expander.problems) > 0 {
		return nil
	}

	for _, layout := range mountedTree.Layouts {
		rebased, ok := expander.rebaseLayout(owner, mountPath, layout)
		if ok {
			expander.tree.Layouts = append(expander.tree.Layouts, rebased)
		}
	}

	var routes []RouteDeclaration
	for _, route := range mountedTree.Routes {
		rebased, ok := expander.rebaseRoute(owner, mountPath, route)
		if ok {
			routes = append(routes, rebased)
		}
	}
	return routes
}

func (expander *mountExpander) rebaseLayout(owner RouteDeclaration, mountPath string, layout Layout) (Layout, bool) {
	params, ok := joinParams(owner.Params, layout.Params)
	if !ok {
		expander.addProblem(owner.GoFile, "mounted layout reuses a dynamic param name from the mount owner")
		return Layout{}, false
	}
	return Layout{
		RoutePrefix: joinRoute(owner.Route, layout.RoutePrefix),
		Params:      params,
		GoFile:      prefixedMountPath(mountPath, layout.GoFile),
		TemplFile:   prefixedOptionalMountPath(mountPath, layout.TemplFile),
		HasTempl:    layout.HasTempl,
	}, true
}

func (expander *mountExpander) rebaseRoute(owner RouteDeclaration, mountPath string, route RouteDeclaration) (RouteDeclaration, bool) {
	params, ok := joinParams(owner.Params, route.Params)
	if !ok {
		expander.addProblem(owner.GoFile, "mounted route reuses a dynamic param name from the mount owner")
		return RouteDeclaration{}, false
	}
	source := prefixedMountPath(mountPath, route.GoFile)
	rebased := route
	rebased.Route = joinRoute(owner.Route, route.Route)
	rebased.Params = params
	rebased.GoFile = owner.GoFile
	rebased.MiddlewareGoFile = mountedRouteMiddlewareGoFile(owner.GoFile, route.GoFile)
	rebased.Source = source
	rebased.Adapter = mountedAdapterPrefix(mountPath, route.Route)
	rebased.Imports = slices.Clone(route.Imports)
	rebased.Kind = routeDeclarationKindKitMount
	rebased.Kit = cloneRouteKitDeclaration(owner.Kit)
	rebased.Mount = &RouteMountDeclaration{
		Path:            mountPath,
		Owner:           owner.GoFile,
		OwnerRoute:      owner.Route,
		OwnerParamCount: len(owner.Params),
	}
	return rebased, true
}

func (expander *mountExpander) addMountScanProblems(ownerGoFile string, mountPath string, err error) {
	var scanErr *ScanError
	if !errors.As(err, &scanErr) {
		expander.addProblem(ownerGoFile, "scan mounted route subtree: "+err.Error())
		return
	}
	for _, problem := range scanErr.Problems {
		expander.addProblem(prefixedMountPath(mountPath, problem.Path), problem.Message)
	}
}

func mountedRouteMiddlewareGoFile(ownerGoFile string, mountedGoFile string) string {
	ownerDir := path.Dir(ownerGoFile)
	if ownerDir == "." {
		ownerDir = ""
	}
	mountedDir := path.Dir(mountedGoFile)
	if mountedDir == "." {
		return ownerGoFile
	}
	if ownerDir == "" {
		return path.Join(mountedDir, routeGoFile)
	}
	return path.Join(ownerDir, mountedDir, routeGoFile)
}

func (expander *mountExpander) sort() {
	slices.SortFunc(expander.tree.Layouts, func(a, b Layout) int {
		return compareRouteOrder(a.RoutePrefix, a.GoFile, b.RoutePrefix, b.GoFile)
	})
	slices.SortFunc(expander.tree.Routes, func(a, b RouteDeclaration) int {
		return compareRouteOrder(a.Route, a.GoFile, b.Route, b.GoFile)
	})
}

func (expander *mountExpander) addProblem(relPath, message string) {
	expander.problems = append(expander.problems, Problem{Path: relPath, Message: message})
}

func cleanMountPath(value string) (string, bool) {
	if value == "" || strings.HasPrefix(value, "/") || strings.Contains(value, "\\") {
		return "", false
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned != value || strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", false
	}
	for part := range strings.SplitSeq(cleaned, "/") {
		if !isRouteIdent(part) {
			return "", false
		}
	}
	return cleaned, true
}

func prefixedMountPath(mountPath string, relPath string) string {
	if relPath == "" {
		return "../mounts/" + mountPath
	}
	return "../mounts/" + joinPath(mountPath, relPath)
}

func prefixedOptionalMountPath(mountPath string, relPath string) string {
	if relPath == "" {
		return ""
	}
	return prefixedMountPath(mountPath, relPath)
}

func joinRoute(base string, child string) string {
	if child == "/" {
		return base
	}
	if base == "/" {
		return child
	}
	return base + child
}

func joinParams(parent []string, child []string) ([]string, bool) {
	seen := make(map[string]bool, len(parent)+len(child))
	params := slices.Clone(parent)
	for _, name := range parent {
		seen[name] = true
	}
	for _, name := range child {
		if seen[name] {
			return nil, false
		}
		seen[name] = true
		params = append(params, name)
	}
	return params, true
}

func mountedAdapterPrefix(mountPath string, route string) string {
	parts := strings.Split(strings.Trim(mountPath, "/"), "/")
	if route != "/" {
		parts = append(parts, routeSegmentsFromRoute(route)...)
	}
	var builder strings.Builder
	builder.WriteString("Mount")
	for _, part := range parts {
		builder.WriteString(exportedDeclarationName(adapterSegmentName(part)))
	}
	return builder.String()
}

func adapterSegmentName(segment string) string {
	if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
		return "by_" + strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")
	}
	return strings.ReplaceAll(segment, "-", "_")
}

func routeSegmentsFromRoute(route string) []string {
	if route == "/" {
		return nil
	}
	return strings.Split(strings.TrimPrefix(route, "/"), "/")
}

func (scanner *scanner) addMiddlewareProblems(relPath string, err error) {
	var scanErr *middlewarescan.ScanError
	if !errors.As(err, &scanErr) {
		scanner.addProblem(relPath, err.Error())
		return
	}
	for _, problem := range scanErr.Problems {
		scanner.addProblem(relPath, problem.Function+": "+problem.Message)
	}
}

func (scanner *scanner) addRouteDeclarationProblems(relPath string, err error) {
	var scanErr *routeDeclarationScanError
	if !errors.As(err, &scanErr) {
		scanner.addProblem(relPath, err.Error())
		return
	}
	for _, problem := range scanErr.Problems {
		scanner.addProblem(relPath, problem.Message)
	}
}

func routePath(segments []string) string {
	if len(segments) == 0 {
		return "/"
	}
	return "/" + strings.Join(segments, "/")
}

func browserPathSegment(sourceName string) string {
	return strings.ReplaceAll(sourceName, "_", "-")
}

func pairFile(relDir, base string, files map[string]bool) (string, bool) {
	name := base + templFileExtension
	if !files[name] {
		return "", false
	}
	return joinPath(relDir, name), true
}

func joinPath(elem ...string) string {
	joined := path.Join(elem...)
	if joined == "." {
		return ""
	}
	return joined
}

func isRouteIdent(value string) bool {
	return routeIdentPattern.MatchString(value)
}
