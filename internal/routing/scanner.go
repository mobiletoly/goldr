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

	"github.com/mobiletoly/goldr/internal/actionscan"
	"github.com/mobiletoly/goldr/internal/middlewarescan"
)

const (
	goFileExtension            = ".go"
	goTestFileSuffix           = "_test" + goFileExtension
	templGeneratedGoFileSuffix = "_templ" + goFileExtension
	templFileExtension         = ".templ"

	pageRenderUnit   = "page"
	layoutRenderUnit = "layout"
	fragmentPrefix   = "frag_"

	pageGoFile   = pageRenderUnit + goFileExtension
	layoutGoFile = layoutRenderUnit + goFileExtension

	fragmentGoPattern   = fragmentPrefix + "<name>" + goFileExtension
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
	Pages       []Page
	Layouts     []Layout
	Fragments   []Fragment
	Actions     []Action
	Middlewares []Middleware
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
}

type Action struct {
	Method   string
	Route    string
	Params   []string
	GoFile   string
	Function string
	Suffix   string
	Segment  string
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
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("scan route root %q: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scan route root %q: not a directory", root)
	}

	scanner := scanner{
		root: filepath.Clean(root),
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

type scanner struct {
	root     string
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

	switch {
	case name == pageGoFile:
		templFile, hasTempl := pairFile(relDir, pageRenderUnit, files)
		scanner.tree.Pages = append(scanner.tree.Pages, Page{
			Route:     route,
			Params:    slices.Clone(params),
			GoFile:    relPath,
			TemplFile: templFile,
			HasTempl:  hasTempl,
		})
	case name == layoutGoFile:
		templFile, hasTempl := pairFile(relDir, layoutRenderUnit, files)
		scanner.tree.Layouts = append(scanner.tree.Layouts, Layout{
			RoutePrefix: route,
			Params:      slices.Clone(params),
			GoFile:      relPath,
			TemplFile:   templFile,
			HasTempl:    hasTempl,
		})
	case strings.HasPrefix(name, fragmentPrefix) && strings.HasSuffix(name, goFileExtension):
		fragmentName := strings.TrimSuffix(strings.TrimPrefix(name, fragmentPrefix), goFileExtension)
		if !isRouteIdent(fragmentName) {
			scanner.addProblem(relPath, "fragment files must use "+fragmentGoPattern+" with a lowercase Go-safe name")
			return
		}

		templFile, hasTempl := pairFile(relDir, fragmentPrefix+fragmentName, files)
		scanner.tree.Fragments = append(scanner.tree.Fragments, Fragment{
			Name:        fragmentName,
			RoutePrefix: route,
			Params:      slices.Clone(params),
			GoFile:      relPath,
			TemplFile:   templFile,
			HasTempl:    hasTempl,
		})
	case name == actionscan.FileName:
		actions, err := actionscan.Scan(filepath.Join(scanner.root, filepath.FromSlash(relPath)))
		if err != nil {
			scanner.addActionProblems(relPath, err)
			return
		}
		for _, action := range actions {
			scanner.tree.Actions = append(scanner.tree.Actions, Action{
				Method:   action.Method,
				Route:    actionRoute(route, action.Segment),
				Params:   slices.Clone(params),
				GoFile:   relPath,
				Function: action.Function,
				Suffix:   action.Suffix,
				Segment:  action.Segment,
			})
		}
	case name == middlewarescan.FileName:
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

func (scanner *scanner) addActionProblems(relPath string, err error) {
	var scanErr *actionscan.ScanError
	if !errors.As(err, &scanErr) {
		scanner.addProblem(relPath, err.Error())
		return
	}
	for _, problem := range scanErr.Problems {
		scanner.addProblem(relPath, problem.Function+": "+problem.Message)
	}
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

func routePath(segments []string) string {
	if len(segments) == 0 {
		return "/"
	}
	return "/" + strings.Join(segments, "/")
}

func browserPathSegment(sourceName string) string {
	return strings.ReplaceAll(sourceName, "_", "-")
}

func actionRoute(route, segment string) string {
	if segment == "" {
		return route
	}
	if route == "/" {
		return "/" + segment
	}
	return route + "/" + segment
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
