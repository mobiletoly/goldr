// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const (
	routeDeclarationKindLocal    = "local"
	routeDeclarationKindKit      = "kit"
	routeDeclarationKindKitMount = "mounted-kit"
)

var (
	routeDeclarationNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
	mountRouteStaticPattern     = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
)

type RouteDeclaration struct {
	Route            string
	Params           []string
	GoFile           string
	MiddlewareGoFile string
	Imports          []RouteImportDeclaration
	Kind             string
	Name             string
	Title            string
	Meta             []RouteMetaLabel
	Page             *RouteHandlerDeclaration
	Fragments        []RouteFragmentDeclaration
	Actions          []RouteActionDeclaration
	Kit              *RouteKitDeclaration
	Mount            *RouteMountDeclaration
	Source           string
	Adapter          string
}

type RouteImportDeclaration struct {
	Name     string
	Path     string
	Explicit bool
}

type RouteHandlerDeclaration struct {
	Handler string
}

type RouteFragmentDeclaration struct {
	Name       string
	Segment    string
	SymbolName string
	Index      bool
	Handler    string
}

type RouteActionDeclaration struct {
	Method     string
	Name       string
	Segment    string
	SymbolName string
	Index      bool
	Writer     bool
	Handler    string
}

type RouteKitDeclaration struct {
	KitType string
	New     string
}

type RouteMountDeclaration struct {
	Path            string
	Routes          []string
	RoutesSet       bool
	Owner           string
	OwnerRoute      string
	OwnerParamCount int
}

type RouteMetaLabel struct {
	Key   string
	Value string
}

type routeDeclarationProblem struct {
	Message string
}

type routeDeclarationScanError struct {
	Path     string
	Problems []routeDeclarationProblem
}

func (err *routeDeclarationScanError) Error() string {
	if len(err.Problems) == 0 {
		return "route declaration scan failed"
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "route declaration scan found %d problem(s)", len(err.Problems))
	for _, problem := range err.Problems {
		fmt.Fprintf(&builder, "; %s", problem.Message)
	}
	return builder.String()
}

func scanRouteDeclaration(path string) (RouteDeclaration, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		return RouteDeclaration{}, fmt.Errorf("parse route declaration file %q: %w", path, err)
	}

	parser := routeDeclarationParser{}
	decl := parser.parse(file)
	if len(parser.problems) > 0 {
		return decl, &routeDeclarationScanError{Path: path, Problems: parser.problems}
	}
	return decl, nil
}

type routeDeclarationParser struct {
	problems []routeDeclarationProblem
}

func (parser *routeDeclarationParser) parse(file *ast.File) RouteDeclaration {
	imports := parser.parseImports(file)
	var routeValues []routeValueSpec
	for _, decl := range file.Decls {
		parser.inspectReservedDeclaration(decl)
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for index, name := range valueSpec.Names {
				if name.Name != "Route" {
					continue
				}
				routeValues = append(routeValues, routeValueSpec{
					spec:  valueSpec,
					index: index,
				})
			}
		}
	}

	switch len(routeValues) {
	case 0:
		parser.addProblem("missing Route declaration")
		return RouteDeclaration{}
	case 1:
	default:
		parser.addProblem("more than one Route declaration")
		return RouteDeclaration{}
	}

	decl := parser.parseRouteValue(routeValues[0])
	decl.Imports = imports
	return decl
}

func (parser *routeDeclarationParser) parseImports(file *ast.File) []RouteImportDeclaration {
	imports := make([]RouteImportDeclaration, 0, len(file.Imports))
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			parser.addProblem("route.go import path must be a valid string literal")
			continue
		}
		item := RouteImportDeclaration{Path: importPath}
		if spec.Name != nil {
			switch spec.Name.Name {
			case "_":
				parser.addProblem("route.go must not use blank imports")
				continue
			case ".":
				parser.addProblem("route.go must not use dot imports")
				continue
			default:
				item.Name = spec.Name.Name
				item.Explicit = true
			}
		} else {
			item.Name = path.Base(importPath)
		}
		imports = append(imports, item)
	}
	slices.SortFunc(imports, func(a, b RouteImportDeclaration) int {
		if a.Name != b.Name {
			return strings.Compare(a.Name, b.Name)
		}
		return strings.Compare(a.Path, b.Path)
	})
	return imports
}

type routeValueSpec struct {
	spec  *ast.ValueSpec
	index int
}

func (parser *routeDeclarationParser) inspectReservedDeclaration(decl ast.Decl) {
	switch decl := decl.(type) {
	case *ast.FuncDecl:
		parser.inspectReservedName(decl.Name.Name)
	case *ast.GenDecl:
		for _, spec := range decl.Specs {
			switch spec := spec.(type) {
			case *ast.TypeSpec:
				parser.inspectReservedName(spec.Name.Name)
			case *ast.ValueSpec:
				for _, name := range spec.Names {
					parser.inspectReservedName(name.Name)
				}
			}
		}
	}
}

func (parser *routeDeclarationParser) inspectReservedName(name string) {
	if name == "Route" {
		return
	}
	if strings.HasPrefix(name, "GoldrRoute") {
		parser.addProblem("reserved GoldrRoute* symbol declared: " + name)
	}
}

func (parser *routeDeclarationParser) parseRouteValue(routeValue routeValueSpec) RouteDeclaration {
	var value ast.Expr
	if routeValue.index < len(routeValue.spec.Values) {
		value = routeValue.spec.Values[routeValue.index]
	} else if len(routeValue.spec.Values) == 1 && len(routeValue.spec.Names) == 1 {
		value = routeValue.spec.Values[0]
	}
	if value == nil {
		parser.addProblem("Route must use a static goldr.RouteDef, goldr.KitRouteDef, or goldr.KitRouteMount composite literal")
		return RouteDeclaration{}
	}

	literal, ok := value.(*ast.CompositeLit)
	if !ok {
		parser.addProblem("Route must use a static goldr.RouteDef, goldr.KitRouteDef, or goldr.KitRouteMount composite literal")
		return RouteDeclaration{}
	}

	kind, kit, ok := routeDeclarationType(literal.Type)
	if !ok {
		parser.addProblem("Route must use goldr.RouteDef, goldr.KitRouteDef[K], or goldr.KitRouteMount[K]")
		return RouteDeclaration{}
	}

	decl := RouteDeclaration{
		Kind: kind,
		Kit:  kit,
	}
	parser.parseRouteFields(&decl, literal)
	parser.validateRouteSurface(decl)
	return decl
}

func routeDeclarationType(expr ast.Expr) (string, *RouteKitDeclaration, bool) {
	if selectorName(expr, "goldr", "RouteDef") {
		return routeDeclarationKindLocal, nil, true
	}

	x, args, ok := indexType(expr)
	if !ok || len(args) != 1 {
		return "", nil, false
	}
	switch {
	case selectorName(x, "goldr", "KitRouteDef"):
		return routeDeclarationKindKit, &RouteKitDeclaration{
			KitType: exprString(args[0]),
		}, true
	case selectorName(x, "goldr", "KitRouteMount"):
		return routeDeclarationKindKitMount, &RouteKitDeclaration{
			KitType: exprString(args[0]),
		}, true
	}
	return "", nil, false
}

func indexType(expr ast.Expr) (ast.Expr, []ast.Expr, bool) {
	switch expr := expr.(type) {
	case *ast.IndexExpr:
		return expr.X, []ast.Expr{expr.Index}, true
	case *ast.IndexListExpr:
		return expr.X, expr.Indices, true
	default:
		return nil, nil, false
	}
}

func (parser *routeDeclarationParser) parseRouteFields(decl *RouteDeclaration, literal *ast.CompositeLit) {
	for _, item := range literal.Elts {
		field, ok := item.(*ast.KeyValueExpr)
		if !ok {
			parser.addProblem("Route fields must use keyed composite literal entries")
			continue
		}
		key, ok := field.Key.(*ast.Ident)
		if !ok {
			parser.addProblem("Route fields must use identifier keys")
			continue
		}

		if decl.Kind == routeDeclarationKindKitMount && key.Name != "New" && key.Name != "Mount" && key.Name != "Routes" {
			parser.addProblem("KitRouteMount supports only New, Mount, and Routes route surface fields")
			continue
		}

		switch key.Name {
		case "Name":
			decl.Name = parser.stringLiteral("Name", field.Value)
		case "Title":
			decl.Title = parser.stringLiteral("Title", field.Value)
		case "Page":
			decl.Page = parser.parsePage(field.Value)
		case "Fragments":
			decl.Fragments = parser.parseFragments(decl.Kind, field.Value)
		case "Actions":
			decl.Actions = parser.parseActions(decl.Kind, field.Value)
		case "Meta":
			decl.Meta = parser.parseMeta(field.Value)
		case "New":
			if decl.Kind != routeDeclarationKindKit && decl.Kind != routeDeclarationKindKitMount {
				parser.addProblem("New is only supported on goldr.KitRouteDef and goldr.KitRouteMount")
				continue
			}
			if decl.Kind == routeDeclarationKindKitMount {
				decl.Kit.New = parser.ident("KitRouteMount.New", field.Value)
			} else {
				decl.Kit.New = parser.identOrSelector("New", field.Value)
			}
		case "Mount":
			if decl.Kind != routeDeclarationKindKitMount {
				parser.addProblem("Mount is only supported on goldr.KitRouteMount")
				continue
			}
			if decl.Mount == nil {
				decl.Mount = &RouteMountDeclaration{}
			}
			decl.Mount.Path = parser.stringLiteral("Mount", field.Value)
		case "Routes":
			if decl.Kind != routeDeclarationKindKitMount {
				parser.addProblem("Routes is only supported on goldr.KitRouteMount")
				continue
			}
			routes := parser.parseMountRoutes(field.Value)
			if decl.Mount == nil {
				decl.Mount = &RouteMountDeclaration{}
			}
			decl.Mount.RoutesSet = true
			decl.Mount.Routes = routes
		default:
			parser.addProblem("unsupported Route field: " + key.Name)
		}
	}
}

func (parser *routeDeclarationParser) parseMountRoutes(expr ast.Expr) []string {
	literal, ok := expr.(*ast.CompositeLit)
	if !ok || !selectorName(literal.Type, "goldr", "MountRoutes") {
		parser.addProblem("KitRouteMount.Routes must use a literal goldr.MountRoutes value")
		return nil
	}
	if len(literal.Elts) == 0 {
		parser.addProblem("KitRouteMount.Routes must not be empty")
		return nil
	}
	routes := make([]string, 0, len(literal.Elts))
	seen := make(map[string]bool, len(literal.Elts))
	for _, item := range literal.Elts {
		route := parser.stringLiteral("KitRouteMount.Routes entry", item)
		if route == "" {
			continue
		}
		if !validMountRouteSelector(route) {
			parser.addProblem("KitRouteMount.Routes entries must be mount-relative browser route patterns like \"/\", \"/table\", or \"/{id}\"")
			continue
		}
		if seen[route] {
			parser.addProblem("KitRouteMount.Routes contains duplicate route pattern: " + route)
			continue
		}
		seen[route] = true
		routes = append(routes, route)
	}
	return routes
}

func (parser *routeDeclarationParser) parsePage(expr ast.Expr) *RouteHandlerDeclaration {
	handler := parser.handlerExpression("Page", expr)
	if handler == "" {
		return nil
	}
	return &RouteHandlerDeclaration{Handler: handler}
}

func (parser *routeDeclarationParser) parseFragments(kind string, expr ast.Expr) []RouteFragmentDeclaration {
	literal, ok := expr.(*ast.CompositeLit)
	if !ok || !fragmentCollectionType(kind, literal.Type) {
		parser.addProblem("Fragments must use a literal goldr fragment collection")
		return nil
	}

	var fragments []RouteFragmentDeclaration
	seen := make(map[string]string)
	for _, item := range literal.Elts {
		fragment, ok := parser.parseFragment(kind, item)
		if !ok {
			continue
		}
		label := fragment.Name
		if fragment.Index {
			label = "Index"
		}
		if previous, ok := seen[fragment.SymbolName]; ok {
			parser.addProblem(fmt.Sprintf("fragment segments %q and %q normalize to the same generated symbol %s", previous, label, fragment.SymbolName))
			continue
		}
		seen[fragment.SymbolName] = label
		fragments = append(fragments, fragment)
	}
	return fragments
}

func (parser *routeDeclarationParser) parseFragment(kind string, expr ast.Expr) (RouteFragmentDeclaration, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		parser.addProblem(fragmentHelperError(kind))
		return RouteFragmentDeclaration{}, false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !identName(selector.X, "goldr") {
		parser.addProblem(fragmentHelperError(kind))
		return RouteFragmentDeclaration{}, false
	}

	helper := "FragmentRoute"
	if isKitBackedKind(kind) {
		helper = "KitFragmentRoute"
	}

	if selector.Sel.Name != helper {
		parser.addProblem(fragmentHelperError(kind))
		return RouteFragmentDeclaration{}, false
	}
	if len(call.Args) != 2 {
		parser.addProblem("fragment route helpers must use path and handler arguments")
		return RouteFragmentDeclaration{}, false
	}
	path, ok := parser.routeLocalPath("fragment path", call.Args[0])
	if !ok {
		return RouteFragmentDeclaration{}, false
	}
	handler := parser.handlerExpression("fragment handler", call.Args[1])
	if handler == "" {
		return RouteFragmentDeclaration{}, false
	}
	if path.index {
		return RouteFragmentDeclaration{
			Name:       "index",
			SymbolName: "Index",
			Index:      true,
			Handler:    handler,
		}, true
	}
	return RouteFragmentDeclaration{
		Name:       path.segment,
		Segment:    browserPathSegment(path.segment),
		SymbolName: exportedDeclarationName(path.segment),
		Handler:    handler,
	}, true
}

func fragmentHelperError(kind string) string {
	return routeHelperError(
		kind,
		"Fragments entries must use goldr.FragmentRoute(path, handler)",
		"Fragments entries must use goldr.KitFragmentRoute(path, handler)",
	)
}

func fragmentCollectionType(kind string, expr ast.Expr) bool {
	if kind == routeDeclarationKindLocal {
		return selectorName(expr, "goldr", "Fragments")
	}
	x, args, ok := indexType(expr)
	return ok && selectorName(x, "goldr", "KitFragments") && len(args) == 1
}

type routeLocalPath struct {
	segment string
	index   bool
}

func (parser *routeDeclarationParser) routeLocalPath(label string, expr ast.Expr) (routeLocalPath, bool) {
	literal, ok := expr.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		parser.addProblem(label + " must be a string literal")
		return routeLocalPath{}, false
	}
	value, err := strconv.Unquote(literal.Value)
	if err != nil {
		parser.addProblem(label + " must be a valid string literal")
		return routeLocalPath{}, false
	}
	if value == "" {
		parser.addProblem(label + " must not be empty")
		return routeLocalPath{}, false
	}
	if value == "/" {
		return routeLocalPath{index: true}, true
	}
	if !strings.HasPrefix(value, "/") {
		parser.addProblem(label + ` must start with "/"`)
		return routeLocalPath{}, false
	}
	if strings.HasSuffix(value, "/") {
		parser.addProblem(label + " must not have a trailing slash")
		return routeLocalPath{}, false
	}
	segment := strings.TrimPrefix(value, "/")
	if strings.Contains(segment, "/") {
		parser.addProblem(label + " must be route-local; nested paths belong in nested route directories")
		return routeLocalPath{}, false
	}
	if !validRouteDeclarationName(segment) {
		parser.addProblem(label + " segments must use lowercase ASCII letters, digits, underscores, or hyphens and start with a lowercase ASCII letter")
		return routeLocalPath{}, false
	}
	return routeLocalPath{segment: segment}, true
}

func (parser *routeDeclarationParser) parseActions(kind string, expr ast.Expr) []RouteActionDeclaration {
	literal, ok := expr.(*ast.CompositeLit)
	if !ok || !actionCollectionType(kind, literal.Type) {
		parser.addProblem("Actions must use a literal goldr action collection")
		return nil
	}

	var actions []RouteActionDeclaration
	seen := make(map[string]string)
	for _, item := range literal.Elts {
		action, ok := parser.parseAction(kind, item)
		if !ok {
			continue
		}
		key := action.Method + " " + action.SymbolName
		label := action.Name
		if action.Index {
			label = "Index"
		}
		if previous, ok := seen[key]; ok {
			parser.addProblem(fmt.Sprintf("action segments %q and %q normalize to the same generated symbol %s", previous, label, action.SymbolName))
			continue
		}
		seen[key] = label
		actions = append(actions, action)
	}
	return actions
}

func actionCollectionType(kind string, expr ast.Expr) bool {
	if kind == routeDeclarationKindLocal {
		return selectorName(expr, "goldr", "Actions")
	}
	x, args, ok := indexType(expr)
	return ok && selectorName(x, "goldr", "KitActions") && len(args) == 1
}

func (parser *routeDeclarationParser) parseAction(kind string, expr ast.Expr) (RouteActionDeclaration, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		parser.addProblem(actionHelperError(kind))
		return RouteActionDeclaration{}, false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !identName(selector.X, "goldr") {
		parser.addProblem(actionHelperError(kind))
		return RouteActionDeclaration{}, false
	}

	responseHelper := "Action"
	httpHelper := "HTTPAction"
	if isKitBackedKind(kind) {
		responseHelper = "KitAction"
		httpHelper = "KitHTTPAction"
	}

	var writer bool
	switch selector.Sel.Name {
	case responseHelper:
	case httpHelper:
		writer = true
	default:
		parser.addProblem(actionHelperError(kind))
		return RouteActionDeclaration{}, false
	}

	if len(call.Args) != 3 {
		parser.addProblem("action route helpers must use method, path, and handler arguments")
		return RouteActionDeclaration{}, false
	}
	method, ok := parser.actionMethod(call.Args[0])
	if !ok {
		return RouteActionDeclaration{}, false
	}
	path, ok := parser.routeLocalPath("action path", call.Args[1])
	if !ok {
		return RouteActionDeclaration{}, false
	}
	handler := parser.handlerExpression("action handler", call.Args[2])
	if handler == "" {
		return RouteActionDeclaration{}, false
	}
	if path.index {
		return RouteActionDeclaration{
			Method:     method,
			Index:      true,
			SymbolName: "Index",
			Writer:     writer,
			Handler:    handler,
		}, true
	}
	return RouteActionDeclaration{
		Method:     method,
		Name:       path.segment,
		Segment:    browserPathSegment(path.segment),
		SymbolName: exportedDeclarationName(path.segment),
		Writer:     writer,
		Handler:    handler,
	}, true
}

func actionHelperError(kind string) string {
	return routeHelperError(
		kind,
		"Actions entries must use goldr.Action(method, path, handler) or goldr.HTTPAction(method, path, handler)",
		"Actions entries must use goldr.KitAction(method, path, handler) or goldr.KitHTTPAction(method, path, handler)",
	)
}

func routeHelperError(kind string, local string, kit string) string {
	if kind == routeDeclarationKindKit {
		return kit
	}
	return local
}

func (parser *routeDeclarationParser) actionMethod(expr ast.Expr) (string, bool) {
	if literal, ok := expr.(*ast.BasicLit); ok && literal.Kind == token.STRING {
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			parser.addProblem("action method must be a valid string literal")
			return "", false
		}
		if validActionMethod(value) {
			return value, true
		}
		parser.addProblem("action methods must be POST, PUT, PATCH, or DELETE")
		return "", false
	}
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || !identName(selector.X, "http") {
		parser.addProblem("action methods must use http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, or matching string literals")
		return "", false
	}
	switch selector.Sel.Name {
	case "MethodPost":
		return "POST", true
	case "MethodPut":
		return "PUT", true
	case "MethodPatch":
		return "PATCH", true
	case "MethodDelete":
		return "DELETE", true
	default:
		parser.addProblem("action methods must use http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, or matching string literals")
		return "", false
	}
}

func validActionMethod(value string) bool {
	switch value {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func (parser *routeDeclarationParser) parseMeta(expr ast.Expr) []RouteMetaLabel {
	literal, ok := expr.(*ast.CompositeLit)
	if !ok || !selectorName(literal.Type, "goldr", "RouteMeta") {
		parser.addProblem("Meta must use a literal goldr.RouteMeta value")
		return nil
	}

	var labels []RouteMetaLabel
	for _, item := range literal.Elts {
		field, ok := item.(*ast.KeyValueExpr)
		if !ok {
			parser.addProblem("Meta fields must use keyed composite literal entries")
			continue
		}
		key, ok := field.Key.(*ast.Ident)
		if !ok || key.Name != "Labels" {
			parser.addProblem("Meta supports only the Labels field")
			continue
		}
		labels = append(labels, parser.parseLabels(field.Value)...)
	}
	slices.SortFunc(labels, func(a, b RouteMetaLabel) int {
		return strings.Compare(a.Key, b.Key)
	})
	return labels
}

func (parser *routeDeclarationParser) parseLabels(expr ast.Expr) []RouteMetaLabel {
	literal, ok := expr.(*ast.CompositeLit)
	if !ok || !mapStringStringType(literal.Type) {
		parser.addProblem("Meta.Labels must use a literal map[string]string value")
		return nil
	}

	var labels []RouteMetaLabel
	for _, item := range literal.Elts {
		field, ok := item.(*ast.KeyValueExpr)
		if !ok {
			parser.addProblem("Meta.Labels entries must use key-value string literals")
			continue
		}
		key := parser.stringLiteral("metadata label key", field.Key)
		value := parser.stringLiteral("metadata label value", field.Value)
		if key == "" {
			parser.addProblem("metadata label keys must not be empty")
			continue
		}
		labels = append(labels, RouteMetaLabel{Key: key, Value: value})
	}
	return labels
}

func mapStringStringType(expr ast.Expr) bool {
	mapType, ok := expr.(*ast.MapType)
	if !ok {
		return false
	}
	return identName(mapType.Key, "string") && identName(mapType.Value, "string")
}

func (parser *routeDeclarationParser) validateRouteSurface(decl RouteDeclaration) {
	if decl.Kind == routeDeclarationKindKitMount {
		if decl.Kit == nil || decl.Kit.New == "" {
			parser.addProblem("KitRouteMount requires New")
		}
		if decl.Mount == nil {
			parser.addProblem("KitRouteMount requires Mount")
		}
		return
	}
	if decl.Page == nil && len(decl.Fragments) == 0 && len(decl.Actions) == 0 {
		parser.addProblem("Route must declare at least one of Page, Fragments, or Actions")
	}
	if decl.Page != nil && slices.ContainsFunc(decl.Fragments, func(fragment RouteFragmentDeclaration) bool {
		return fragment.Index
	}) {
		parser.addProblem("Route cannot declare both Page and an index fragment")
	}
}

func isKitBackedKind(kind string) bool {
	return kind == routeDeclarationKindKit || kind == routeDeclarationKindKitMount
}

func (parser *routeDeclarationParser) stringLiteral(label string, expr ast.Expr) string {
	literal, ok := expr.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		parser.addProblem(label + " must be a string literal")
		return ""
	}
	value, err := strconv.Unquote(literal.Value)
	if err != nil {
		parser.addProblem(label + " must be a valid string literal")
		return ""
	}
	return value
}

func (parser *routeDeclarationParser) identOrSelector(label string, expr ast.Expr) string {
	if !isIdentOrSelector(expr) {
		parser.addProblem(label + " must be an identifier or selector expression")
		return ""
	}
	return exprString(expr)
}

func (parser *routeDeclarationParser) handlerExpression(label string, expr ast.Expr) string {
	if !isHandlerExpression(expr) {
		parser.addProblem(label + " must be an identifier, selector, or pointer method expression")
		return ""
	}
	return exprString(expr)
}

func (parser *routeDeclarationParser) ident(label string, expr ast.Expr) string {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		parser.addProblem(label + " must be a local identifier")
		return ""
	}
	return ident.Name
}

func (parser *routeDeclarationParser) addProblem(message string) {
	parser.problems = append(parser.problems, routeDeclarationProblem{Message: message})
}

func selectorName(expr ast.Expr, pkg string, name string) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if name != "" && selector.Sel.Name != name {
		return false
	}
	if pkg == "" {
		return true
	}
	ident, ok := selector.X.(*ast.Ident)
	return ok && ident.Name == pkg
}

func identName(expr ast.Expr, name string) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == name
}

func isIdentOrSelector(expr ast.Expr) bool {
	switch expr := expr.(type) {
	case *ast.Ident:
		return true
	case *ast.SelectorExpr:
		return isIdentOrSelector(expr.X)
	default:
		return false
	}
}

func isHandlerExpression(expr ast.Expr) bool {
	return isIdentOrSelector(expr) || isPointerMethodExpression(expr)
}

func isPointerMethodExpression(expr ast.Expr) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	paren, ok := selector.X.(*ast.ParenExpr)
	if !ok {
		return false
	}
	star, ok := paren.X.(*ast.StarExpr)
	if !ok {
		return false
	}
	return isIdentOrSelector(star.X)
}

func validRouteDeclarationName(value string) bool {
	return routeDeclarationNamePattern.MatchString(value)
}

func validMountRouteSelector(value string) bool {
	if value == "/" {
		return true
	}
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") || strings.Contains(value, "//") {
		return false
	}
	for segment := range strings.SplitSeq(strings.TrimPrefix(value, "/"), "/") {
		if segment == "" {
			return false
		}
		if strings.HasPrefix(segment, "{") || strings.HasSuffix(segment, "}") {
			if !strings.HasPrefix(segment, "{") || !strings.HasSuffix(segment, "}") {
				return false
			}
			param := strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")
			if !isRouteIdent(param) {
				return false
			}
			continue
		}
		if !mountRouteStaticPattern.MatchString(segment) {
			return false
		}
	}
	return true
}

func exportedDeclarationName(value string) string {
	var builder strings.Builder
	for _, part := range strings.FieldsFunc(value, func(r rune) bool {
		return r == '_' || r == '-'
	}) {
		if part == "" {
			continue
		}
		builder.WriteString(strings.ToUpper(part[:1]))
		builder.WriteString(part[1:])
	}
	return builder.String()
}

func exprString(expr ast.Expr) string {
	var buffer bytes.Buffer
	_ = printer.Fprint(&buffer, token.NewFileSet(), expr)
	return buffer.String()
}
