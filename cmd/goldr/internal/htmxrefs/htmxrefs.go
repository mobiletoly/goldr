// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package htmxrefs

import (
	"net/url"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/templscan"
	"github.com/mobiletoly/goldr/cmd/goldr/internal/wiring"
)

const (
	StatusResolved  = "resolved"
	StatusUnmatched = "unmatched"
	StatusDynamic   = "dynamic"
	StatusExternal  = "external"
	StatusInvalid   = "invalid"
)

type Root struct {
	Dir          string
	SourcePrefix string
}

type Reference struct {
	Status    string
	Method    string
	Attribute string
	Source    string
	Line      int
	Column    int
	Value     string
	Route     string
	Match     *RouteMatch
}

type RouteMatch struct {
	Path   string
	Kind   string
	Source string
	Helper string
}

func Scan(roots []Root, rows []wiring.RouteSurfaceRow) ([]Reference, error) {
	resolver := newResolver(rows)
	var references []Reference
	for _, root := range roots {
		files, err := templscan.ScanDir(root.Dir)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			source := sourcePath(root, file.Path)
			for _, attribute := range file.Attributes {
				method, ok := requestMethod(attribute.Name)
				if !ok {
					continue
				}
				reference := Reference{
					Method:    method,
					Attribute: attribute.Name,
					Source:    source,
					Line:      attribute.Line,
					Column:    attribute.Column,
					Value:     attribute.Value,
				}
				resolver.resolve(&reference, attribute)
				references = append(references, reference)
			}
		}
	}
	sortReferences(references)
	return references, nil
}

func sourcePath(root Root, path string) string {
	rel, err := filepath.Rel(root.Dir, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)
	prefix := strings.Trim(root.SourcePrefix, "/")
	if prefix == "" {
		return rel
	}
	return prefix + "/" + rel
}

func requestMethod(attribute string) (string, bool) {
	switch strings.ToLower(attribute) {
	case "hx-get", "data-hx-get":
		return "GET", true
	case "hx-post", "data-hx-post":
		return "POST", true
	case "hx-put", "data-hx-put":
		return "PUT", true
	case "hx-patch", "data-hx-patch":
		return "PATCH", true
	case "hx-delete", "data-hx-delete":
		return "DELETE", true
	default:
		return "", false
	}
}

type resolver struct {
	byMethodPath        map[string]wiring.RouteSurfaceRow
	byMethodHelper      map[string]wiring.RouteSurfaceRow
	byHelper            map[string]wiring.RouteSurfaceRow
	byMethodHelperShape map[string]wiring.RouteSurfaceRow
	byHelperShape       map[string]wiring.RouteSurfaceRow
}

func newResolver(rows []wiring.RouteSurfaceRow) resolver {
	resolver := resolver{
		byMethodPath:        map[string]wiring.RouteSurfaceRow{},
		byMethodHelper:      map[string]wiring.RouteSurfaceRow{},
		byHelper:            map[string]wiring.RouteSurfaceRow{},
		byMethodHelperShape: map[string]wiring.RouteSurfaceRow{},
		byHelperShape:       map[string]wiring.RouteSurfaceRow{},
	}
	for _, row := range rows {
		if row.Kind != wiring.RouteSurfaceKindPage && row.Kind != wiring.RouteSurfaceKindFragment && row.Kind != wiring.RouteSurfaceKindAction {
			continue
		}
		if row.Path != "" {
			for _, method := range row.Methods {
				putFirst(resolver.byMethodPath, methodKey(method, row.Path), row)
			}
		}
		if row.Helper == "" || row.Helper == "-" {
			continue
		}
		helperShape := normalizeHelperShape(row.Helper)
		putFirst(resolver.byHelper, row.Helper, row)
		putFirst(resolver.byHelperShape, helperShape, row)
		for _, method := range row.Methods {
			putFirst(resolver.byMethodHelper, methodKey(method, row.Helper), row)
			putFirst(resolver.byMethodHelperShape, methodKey(method, helperShape), row)
		}
	}
	return resolver
}

func putFirst(index map[string]wiring.RouteSurfaceRow, key string, row wiring.RouteSurfaceRow) {
	if _, ok := index[key]; ok {
		return
	}
	index[key] = row
}

func (resolver resolver) resolve(reference *Reference, attribute templscan.Attribute) {
	value := strings.TrimSpace(attribute.Value)
	if value == "" {
		reference.Status = StatusInvalid
		return
	}

	if attribute.Kind == templscan.AttributeKindConstant || attribute.Kind == templscan.AttributeKindBool {
		resolver.resolveLiteral(reference, value)
		return
	}

	if literal, ok := expressionStringLiteral(value); ok {
		resolver.resolveLiteral(reference, literal)
		return
	}

	helper, ok := expressionHelper(value)
	if !ok {
		reference.Status = StatusDynamic
		return
	}
	resolver.resolveHelper(reference, helper)
}

func (resolver resolver) resolveLiteral(reference *Reference, value string) {
	if isExternalURL(value) {
		reference.Status = StatusExternal
		return
	}
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		reference.Status = StatusInvalid
		return
	}

	path := literalPath(value)
	if row, ok := resolver.byMethodPath[methodKey(reference.Method, path)]; ok {
		reference.Status = StatusResolved
		reference.Route = row.Path
		reference.Match = routeMatch(row)
		return
	}
	reference.Status = StatusUnmatched
	reference.Route = path
}

func (resolver resolver) resolveHelper(reference *Reference, helper string) {
	if row, ok := resolver.byMethodHelper[methodKey(reference.Method, helper)]; ok {
		reference.Status = StatusResolved
		reference.Route = row.Path
		reference.Match = routeMatch(row)
		return
	}

	helperShape := normalizeHelperShape(helper)
	if row, ok := resolver.byMethodHelperShape[methodKey(reference.Method, helperShape)]; ok {
		reference.Status = StatusResolved
		reference.Route = row.Path
		reference.Match = routeMatch(row)
		return
	}

	if row, ok := resolver.byHelper[helper]; ok {
		reference.Status = StatusUnmatched
		reference.Route = row.Path
		return
	}
	if row, ok := resolver.byHelperShape[helperShape]; ok {
		reference.Status = StatusUnmatched
		reference.Route = row.Path
		return
	}

	reference.Status = StatusDynamic
}

func routeMatch(row wiring.RouteSurfaceRow) *RouteMatch {
	return &RouteMatch{
		Path:   row.Path,
		Kind:   row.Kind,
		Source: row.Source,
		Helper: row.Helper,
	}
}

func methodKey(method string, value string) string {
	return method + "\x00" + value
}

func literalPath(value string) string {
	if index := strings.IndexAny(value, "?#"); index >= 0 {
		return value[:index]
	}
	return value
}

func isExternalURL(value string) bool {
	if strings.HasPrefix(value, "//") {
		return true
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	return parsed.Scheme != "" && parsed.Host != ""
}

func expressionStringLiteral(expression string) (string, bool) {
	value, err := strconv.Unquote(strings.TrimSpace(expression))
	if err != nil {
		return "", false
	}
	return value, true
}

func expressionHelper(expression string) (string, bool) {
	expression = strings.TrimSpace(expression)
	if plus := topLevelPlusIndex(expression); plus >= 0 {
		expression = strings.TrimSpace(expression[:plus])
	}
	expression = removeWhitespace(expression)
	if !strings.HasPrefix(expression, "urls.") || !strings.HasSuffix(expression, ".Path()") {
		return "", false
	}
	return expression, true
}

func topLevelPlusIndex(expression string) int {
	depth := 0
	var quote rune
	escaped := false
	for index, r := range expression {
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' && quote != '`' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
			}
			continue
		}
		switch r {
		case '\'', '"', '`':
			quote = r
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case '+':
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func removeWhitespace(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, value)
}

func normalizeHelperShape(helper string) string {
	var builder strings.Builder
	for index := 0; index < len(helper); {
		if strings.HasPrefix(helper[index:], ".Bind(") {
			builder.WriteString(".Bind(*)")
			index += len(".Bind(")
			depth := 1
			for index < len(helper) && depth > 0 {
				switch helper[index] {
				case '(':
					depth++
				case ')':
					depth--
				}
				index++
			}
			continue
		}
		builder.WriteByte(helper[index])
		index++
	}
	return builder.String()
}

func sortReferences(references []Reference) {
	slices.SortFunc(references, func(a, b Reference) int {
		if result := strings.Compare(a.Source, b.Source); result != 0 {
			return result
		}
		if a.Line != b.Line {
			return a.Line - b.Line
		}
		if a.Column != b.Column {
			return a.Column - b.Column
		}
		if result := strings.Compare(a.Attribute, b.Attribute); result != 0 {
			return result
		}
		return strings.Compare(a.Value, b.Value)
	})
}
