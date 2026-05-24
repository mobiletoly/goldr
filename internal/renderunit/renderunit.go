// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package renderunit

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"path/filepath"
	"strings"

	"github.com/a-h/templ"

	"github.com/mobiletoly/goldr/internal/routing"
)

const (
	KindPage     = "page"
	KindLayout   = "layout"
	KindFragment = "fragment"
)

var ErrNilComponent = errors.New("render templ component: nil component")

type Problem struct {
	Kind       string
	Identifier string
	GoFile     string
	Message    string
}

type ValidationError struct {
	Problems []Problem
}

func (err *ValidationError) Error() string {
	if len(err.Problems) == 0 {
		return "render-unit validation failed"
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "render-unit validation found %d problem(s)", len(err.Problems))
	for _, problem := range err.Problems {
		fmt.Fprintf(&builder, "; %s %s (%s): %s", problem.Kind, problem.Identifier, problem.GoFile, problem.Message)
	}
	return builder.String()
}

func ValidateManifest(manifest routing.Manifest) error {
	var problems []Problem

	for _, page := range manifest.Pages {
		validateSignature(&problems, manifest.Root, KindPage, page.Route, page.Unit.GoFile, signatureRule{
			function:       "Page",
			message:        "page handlers must use func Page(*http.Request) goldr.RouteResponse",
			missingMessage: "page handlers must use func Page(*http.Request) goldr.RouteResponse",
			valid:          validPageSignature,
		})
	}
	for _, layout := range manifest.Layouts {
		validateUnit(&problems, manifest.Root, KindLayout, layout.RoutePrefix, layout.Unit, signatureRule{
			function: "Layout",
			message:  "layouts must use func Layout(*http.Request, goldr.LayoutContext) templ.Component",
			valid:    validLayoutSignature,
		})
	}
	for _, fragment := range manifest.Fragments {
		validateUnit(&problems, manifest.Root, KindFragment, fragment.RoutePrefix+":"+fragment.Name, fragment.Unit, signatureRule{
			function: fragmentFuncName(fragment.Name),
			message:  "fragments must use func FragName(*http.Request) goldr.RouteResponse",
			valid:    validFragmentSignature,
		})
	}

	if len(problems) > 0 {
		return &ValidationError{Problems: problems}
	}
	return nil
}

func Render(ctx context.Context, writer io.Writer, component templ.Component) error {
	if component == nil {
		return ErrNilComponent
	}
	return component.Render(ctx, writer)
}

type signatureRule struct {
	function       string
	message        string
	missingMessage string
	valid          func(*ast.FuncType) bool
}

func validateUnit(problems *[]Problem, root, kind, identifier string, unit routing.RenderUnit, rule signatureRule) {
	if unit.HasTempl && unit.TemplFile != "" {
		validateSignature(problems, root, kind, identifier, unit.GoFile, rule)
		return
	}

	*problems = append(*problems, Problem{
		Kind:       kind,
		Identifier: identifier,
		GoFile:     unit.GoFile,
		Message:    "missing matching .templ file",
	})
}

func validateSignature(problems *[]Problem, root, kind, identifier, goFile string, rule signatureRule) {
	if root == "" {
		return
	}

	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, filepath.FromSlash(goFile)), nil, parser.SkipObjectResolution)
	if err != nil {
		*problems = append(*problems, Problem{
			Kind:       kind,
			Identifier: identifier,
			GoFile:     goFile,
			Message:    "parse Go file: " + err.Error(),
		})
		return
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name.Name != rule.function {
			continue
		}
		if !rule.valid(fn.Type) {
			*problems = append(*problems, Problem{
				Kind:       kind,
				Identifier: identifier,
				GoFile:     goFile,
				Message:    rule.message,
			})
		}
		return
	}

	message := rule.missingMessage
	if message == "" {
		message = "missing " + rule.function + " function"
	}

	*problems = append(*problems, Problem{
		Kind:       kind,
		Identifier: identifier,
		GoFile:     goFile,
		Message:    message,
	})
}

func validPageSignature(fnType *ast.FuncType) bool {
	params := expandedParamTypes(fnType.Params)
	results := expandedParamTypes(fnType.Results)
	return len(params) == 1 &&
		len(results) == 1 &&
		isStarSelector(params[0], "http", "Request") &&
		isSelector(results[0], "goldr", "RouteResponse")
}

func validLayoutSignature(fnType *ast.FuncType) bool {
	params := expandedParamTypes(fnType.Params)
	results := expandedParamTypes(fnType.Results)
	return len(params) == 2 &&
		len(results) == 1 &&
		isStarSelector(params[0], "http", "Request") &&
		isSelector(params[1], "goldr", "LayoutContext") &&
		isSelector(results[0], "templ", "Component")
}

func validFragmentSignature(fnType *ast.FuncType) bool {
	params := expandedParamTypes(fnType.Params)
	results := expandedParamTypes(fnType.Results)
	return len(params) == 1 &&
		len(results) == 1 &&
		isStarSelector(params[0], "http", "Request") &&
		isSelector(results[0], "goldr", "RouteResponse")
}

func expandedParamTypes(fields *ast.FieldList) []ast.Expr {
	if fields == nil {
		return nil
	}

	var result []ast.Expr
	for _, field := range fields.List {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for range count {
			result = append(result, field.Type)
		}
	}
	return result
}

func isStarSelector(expr ast.Expr, pkg, name string) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	return isSelector(star.X, pkg, name)
}

func isSelector(expr ast.Expr, pkg, name string) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	return ok && ident.Name == pkg && selector.Sel.Name == name
}

func fragmentFuncName(name string) string {
	parts := strings.Split(name, "_")
	var builder strings.Builder
	builder.WriteString("Frag")
	for _, part := range parts {
		if part == "" {
			builder.WriteByte('_')
			continue
		}
		builder.WriteString(strings.ToUpper(part[:1]))
		builder.WriteString(part[1:])
	}
	return builder.String()
}
