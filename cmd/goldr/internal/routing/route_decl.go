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

var routeDeclarationNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

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

		if decl.Kind == routeDeclarationKindKitMount && key.Name != "New" && key.Name != "Mount" {
			parser.addProblem("KitRouteMount supports only New and Mount route surface fields")
			continue
		}

		switch key.Name {
		case "Name":
			decl.Name = parser.stringLiteral("Name", field.Value)
		case "Title":
			decl.Title = parser.stringLiteral("Title", field.Value)
		case "Page":
			decl.Page = parser.parsePage(decl.Kind, field.Value)
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
			decl.Mount = &RouteMountDeclaration{Path: parser.stringLiteral("Mount", field.Value)}
		default:
			parser.addProblem("unsupported Route field: " + key.Name)
		}
	}
}

func (parser *routeDeclarationParser) parsePage(kind string, expr ast.Expr) *RouteHandlerDeclaration {
	want := "FuncPage"
	if isKitBackedKind(kind) {
		want = "KitPage"
	}
	args, ok := goldrCall(expr, want)
	if !ok || len(args) != 1 {
		parser.addProblem("Page must be goldr." + want + "(handler)")
		return nil
	}
	return &RouteHandlerDeclaration{Handler: exprString(args[0])}
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

	segmentHelper := "FuncFragment"
	indexHelper := "FuncFragmentIndex"
	if isKitBackedKind(kind) {
		segmentHelper = "KitFragment"
		indexHelper = "KitFragmentIndex"
	}

	switch selector.Sel.Name {
	case indexHelper:
		if len(call.Args) != 1 {
			parser.addProblem("index fragment helpers must use one handler argument")
			return RouteFragmentDeclaration{}, false
		}
		return RouteFragmentDeclaration{
			Name:       "index",
			SymbolName: "Index",
			Index:      true,
			Handler:    exprString(call.Args[0]),
		}, true
	case segmentHelper:
		if len(call.Args) != 2 {
			parser.addProblem("fragment helpers with route segments must use segment and handler arguments")
			return RouteFragmentDeclaration{}, false
		}
		segment := parser.stringLiteral("fragment segment", call.Args[0])
		if !validRouteDeclarationName(segment) {
			parser.addProblem("fragment segments must use lowercase ASCII letters, digits, underscores, or hyphens")
			return RouteFragmentDeclaration{}, false
		}
		return RouteFragmentDeclaration{
			Name:       segment,
			Segment:    browserPathSegment(segment),
			SymbolName: exportedDeclarationName(segment),
			Handler:    exprString(call.Args[1]),
		}, true
	default:
		parser.addProblem(fragmentHelperError(kind))
		return RouteFragmentDeclaration{}, false
	}
}

func fragmentHelperError(kind string) string {
	if kind == routeDeclarationKindKit {
		return "Fragments entries must use goldr.KitFragment(segment, handler) or goldr.KitFragmentIndex(handler)"
	}
	return "Fragments entries must use goldr.FuncFragment(segment, handler) or goldr.FuncFragmentIndex(handler)"
}

func fragmentCollectionType(kind string, expr ast.Expr) bool {
	if kind == routeDeclarationKindLocal {
		return selectorName(expr, "goldr", "FuncFragments")
	}
	x, args, ok := indexType(expr)
	return ok && selectorName(x, "goldr", "KitFragments") && len(args) == 1
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
		return selectorName(expr, "goldr", "FuncActions")
	}
	x, args, ok := indexType(expr)
	return ok && selectorName(x, "goldr", "KitActions") && len(args) == 1
}

func (parser *routeDeclarationParser) parseAction(kind string, expr ast.Expr) (RouteActionDeclaration, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		parser.addProblem("Actions entries must use goldr action helper calls")
		return RouteActionDeclaration{}, false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !identName(selector.X, "goldr") {
		parser.addProblem("Actions entries must use goldr action helper calls")
		return RouteActionDeclaration{}, false
	}

	method, index, writer, ok := routeActionHelper(kind, selector.Sel.Name)
	if !ok {
		parser.addProblem("unsupported action helper: goldr." + selector.Sel.Name)
		return RouteActionDeclaration{}, false
	}

	if index {
		if len(call.Args) != 1 {
			parser.addProblem("index action helpers must use one handler argument")
			return RouteActionDeclaration{}, false
		}
		return RouteActionDeclaration{
			Method:     method,
			Index:      true,
			SymbolName: "Index",
			Writer:     writer,
			Handler:    exprString(call.Args[0]),
		}, true
	}

	if len(call.Args) != 2 {
		parser.addProblem("action helpers with route segments must use segment and handler arguments")
		return RouteActionDeclaration{}, false
	}
	segment := parser.stringLiteral("action segment", call.Args[0])
	if !validRouteDeclarationName(segment) {
		parser.addProblem("action segments must use lowercase ASCII letters, digits, underscores, or hyphens")
		return RouteActionDeclaration{}, false
	}
	return RouteActionDeclaration{
		Method:     method,
		Name:       segment,
		Segment:    browserPathSegment(segment),
		SymbolName: exportedDeclarationName(segment),
		Writer:     writer,
		Handler:    exprString(call.Args[1]),
	}, true
}

func routeActionHelper(kind string, name string) (method string, index bool, writer bool, ok bool) {
	prefix := "Func"
	if isKitBackedKind(kind) {
		prefix = "Kit"
	}
	if !strings.HasPrefix(name, prefix) {
		return "", false, false, false
	}
	suffix := strings.TrimPrefix(name, prefix)
	for _, item := range []struct {
		helper string
		method string
	}{
		{"Post", "POST"},
		{"Put", "PUT"},
		{"Patch", "PATCH"},
		{"Delete", "DELETE"},
	} {
		switch suffix {
		case item.helper:
			return item.method, false, false, true
		case item.helper + "Index":
			return item.method, true, false, true
		case item.helper + "Handler":
			return item.method, false, true, true
		case item.helper + "HandlerIndex":
			return item.method, true, true, true
		}
	}
	return "", false, false, false
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

func goldrCall(expr ast.Expr, name string) ([]ast.Expr, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	if !selectorName(call.Fun, "goldr", name) {
		return nil, false
	}
	return call.Args, true
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

func validRouteDeclarationName(value string) bool {
	return routeDeclarationNamePattern.MatchString(value)
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
