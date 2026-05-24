// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package middlewarescan

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

const (
	FileName     = "middleware.go"
	FunctionName = "Middleware"
)

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
		return "middleware scan failed"
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "middleware scan found %d problem(s)", len(err.Problems))
	for _, problem := range err.Problems {
		fmt.Fprintf(&builder, "; %s: %s", problem.Function, problem.Message)
	}
	return builder.String()
}

func Scan(path string) error {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		return fmt.Errorf("parse middleware file %q: %w", path, err)
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name.Name != FunctionName {
			continue
		}
		if !validSignature(fn.Type) {
			return &ScanError{
				Path: path,
				Problems: []Problem{{
					Function: FunctionName,
					Message:  "middleware must use func Middleware(next http.Handler) http.Handler",
				}},
			}
		}
		return nil
	}

	return &ScanError{
		Path: path,
		Problems: []Problem{{
			Function: FunctionName,
			Message:  "missing Middleware function",
		}},
	}
}

func validSignature(fnType *ast.FuncType) bool {
	params := expandedParamTypes(fnType.Params)
	results := expandedParamTypes(fnType.Results)
	return len(params) == 1 &&
		len(results) == 1 &&
		isSelector(params[0], "http", "Handler") &&
		isSelector(results[0], "http", "Handler")
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

func isSelector(expr ast.Expr, pkg, name string) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	return ok && ident.Name == pkg && selector.Sel.Name == name
}
