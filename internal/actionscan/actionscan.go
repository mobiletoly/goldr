// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package actionscan

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"unicode"
)

const FileName = "actions.go"

type Action struct {
	Method   string
	Function string
	Suffix   string
	Segment  string
}

type Problem struct {
	Function string
	Message  string
}

type ScanError struct {
	Path     string
	Problems []Problem
}

func (err *ScanError) Error() string {
	if len(err.Problems) == 0 {
		return "action scan failed"
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "action scan found %d problem(s)", len(err.Problems))
	for _, problem := range err.Problems {
		fmt.Fprintf(&builder, "; %s: %s", problem.Function, problem.Message)
	}
	return builder.String()
}

func Scan(path string) ([]Action, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse actions file %q: %w", path, err)
	}

	var actions []Action
	var problems []Problem
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}

		action, problem, ok := inspectFunc(fn)
		if !ok {
			continue
		}
		if problem.Message != "" {
			problems = append(problems, problem)
			continue
		}
		actions = append(actions, action)
	}
	if len(problems) > 0 {
		return actions, &ScanError{Path: path, Problems: problems}
	}
	return actions, nil
}

func inspectFunc(fn *ast.FuncDecl) (Action, Problem, bool) {
	method, suffix, supported := supportedMethod(fn.Name.Name)
	if !supported {
		if suffix, ok := strings.CutPrefix(fn.Name.Name, "Get"); ok {
			if validSuffix(suffix) {
				return Action{}, Problem{
					Function: fn.Name.Name,
					Message:  "GET action handlers are not supported; pages and fragments own GET and HEAD",
				}, true
			}
			return Action{}, Problem{
				Function: fn.Name.Name,
				Message:  "action function names must use Get<Name> with an exported ASCII suffix",
			}, true
		}
		return Action{}, Problem{}, false
	}

	if !validSuffix(suffix) {
		return Action{}, Problem{
			Function: fn.Name.Name,
			Message:  "action function names must use Post<Name>, Put<Name>, Patch<Name>, or Delete<Name> with an exported ASCII suffix",
		}, true
	}
	if !validSignature(fn.Type) {
		return Action{}, Problem{
			Function: fn.Name.Name,
			Message:  "action handlers must use func Name(w http.ResponseWriter, r *http.Request)",
		}, true
	}

	return Action{
		Method:   method,
		Function: fn.Name.Name,
		Suffix:   suffix,
		Segment:  suffixSegment(suffix),
	}, Problem{}, true
}

func supportedMethod(name string) (method, suffix string, ok bool) {
	prefixes := []struct {
		prefix string
		method string
	}{
		{"Delete", "DELETE"},
		{"Patch", "PATCH"},
		{"Post", "POST"},
		{"Put", "PUT"},
	}
	for _, item := range prefixes {
		if suffix, ok := strings.CutPrefix(name, item.prefix); ok {
			return item.method, suffix, true
		}
	}
	return "", "", false
}

func validSuffix(value string) bool {
	if value == "" {
		return false
	}
	for index, r := range value {
		if r > unicode.MaxASCII {
			return false
		}
		if index == 0 && (r < 'A' || r > 'Z') {
			return false
		}
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func validSignature(fnType *ast.FuncType) bool {
	if fnType.Results != nil && len(fnType.Results.List) > 0 {
		return false
	}

	params := expandedParamTypes(fnType.Params)
	if len(params) != 2 {
		return false
	}
	return isSelector(params[0], "http", "ResponseWriter") && isStarSelector(params[1], "http", "Request")
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

func suffixSegment(value string) string {
	if value == "Index" {
		return ""
	}

	var builder strings.Builder
	runes := []rune(value)
	for index, r := range runes {
		if index > 0 && startsWord(runes, index) {
			builder.WriteByte('-')
		}
		builder.WriteRune(unicode.ToLower(r))
	}
	return builder.String()
}

func startsWord(runes []rune, index int) bool {
	current := runes[index]
	previous := runes[index-1]
	if current < 'A' || current > 'Z' {
		return false
	}
	if previous >= 'a' && previous <= 'z' {
		return true
	}
	if previous >= '0' && previous <= '9' {
		return true
	}
	if index+1 < len(runes) {
		next := runes[index+1]
		return next >= 'a' && next <= 'z'
	}
	return false
}
