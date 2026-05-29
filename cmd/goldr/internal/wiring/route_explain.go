// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package wiring

import (
	"net/url"
	"slices"
	"strings"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
)

const (
	RouteExplainStatusMatched          = "matched"
	RouteExplainStatusMethodNotAllowed = "method_not_allowed"
	RouteExplainStatusNotFound         = "not_found"
)

type RouteExplanation struct {
	Status         string
	Method         string
	Path           string
	AllowedMethods []string
	Match          RouteExplanationMatch
}

type RouteExplanationMatch struct {
	Kind        string
	Methods     []string
	Route       string
	Params      []RouteExplanationParam
	Source      string
	Function    string
	Layouts     []RouteExplanationLayout
	Declaration *RouteDeclarationInfo
}

type RouteExplanationParam struct {
	Name  string
	Value string
}

type RouteExplanationLayout struct {
	RoutePrefix string
	Source      string
}

func ExplainRoute(manifest routing.Manifest, method string, escapedPath string) (RouteExplanation, error) {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = "GET"
	}
	explanation := RouteExplanation{
		Status: RouteExplainStatusNotFound,
		Method: method,
		Path:   escapedPath,
	}

	routes, err := runtimeRoutes(manifest)
	if err != nil {
		return explanation, err
	}
	pathSegments, ok := routeExplainPathSegments(escapedPath)
	if !ok {
		return explanation, nil
	}

	path := routeExplainDispatchPath(buildDispatchTree(runtimePaths(routes)), pathSegments)
	if path == nil {
		return explanation, nil
	}

	params, ok := routeExplainPathParams(*path, pathSegments)
	if !ok {
		return explanation, nil
	}

	explanation.AllowedMethods = routeExplainAllowedMethods(path.routes)
	for _, route := range path.routes {
		if !routeExplainSupportsMethod(route, method) {
			continue
		}
		explanation.Status = RouteExplainStatusMatched
		explanation.Match = routeExplainMatch(route, params, manifest.Routes)
		return explanation, nil
	}

	explanation.Status = RouteExplainStatusMethodNotAllowed
	return explanation, nil
}

func routeExplainDispatchPath(node *dispatchNode, segments []string) *runtimePath {
	if len(segments) < node.depth {
		return nil
	}
	if len(segments) == node.depth {
		return node.path
	}

	segment := segments[node.depth]
	if child := node.staticChildren[segment]; child != nil {
		return routeExplainDispatchPath(child, segments)
	}
	if node.dynamicChild != nil && segment != "" {
		return routeExplainDispatchPath(node.dynamicChild, segments)
	}
	return nil
}

func routeExplainPathSegments(escapedPath string) ([]string, bool) {
	if escapedPath == "/" {
		return nil, true
	}
	if !strings.HasPrefix(escapedPath, "/") || strings.HasSuffix(escapedPath, "/") {
		return nil, false
	}

	segments := strings.Split(strings.TrimPrefix(escapedPath, "/"), "/")
	if slices.Contains(segments, "") {
		return nil, false
	}
	return segments, true
}

func routeExplainPathParams(path runtimePath, pathSegments []string) ([]RouteExplanationParam, bool) {
	if len(path.segments) != len(pathSegments) {
		return nil, false
	}

	values := make(map[string]string, len(path.params))
	for index, routeSegment := range path.segments {
		pathSegment := pathSegments[index]
		if paramName, ok := paramSegmentName(routeSegment); ok {
			value, err := url.PathUnescape(pathSegment)
			if err != nil {
				return nil, false
			}
			values[paramName] = value
			continue
		}

		if routeSegment != pathSegment {
			return nil, false
		}
	}

	params := make([]RouteExplanationParam, 0, len(path.params))
	for _, name := range path.params {
		params = append(params, RouteExplanationParam{
			Name:  name,
			Value: values[name],
		})
	}
	return params, true
}

func routeExplainAllowedMethods(routes []runtimeRoute) []string {
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
	return methods
}

func routeExplainSupportsMethod(route runtimeRoute, method string) bool {
	return slices.Contains(routeMethods(route), method)
}

func routeExplainMatch(route runtimeRoute, params []RouteExplanationParam, declarations []routing.ManifestRouteDeclaration) RouteExplanationMatch {
	inboundDestinations, _ := inboundDestinationTrailEdgesByRoute(declarations)
	declarationInfo := routeDeclarationInfoForRuntimeRoute(route, declarations, inboundDestinations)
	match := RouteExplanationMatch{
		Methods:     routeMethods(route),
		Route:       route.route,
		Params:      slices.Clone(params),
		Declaration: declarationInfo,
	}

	switch {
	case route.page != nil:
		match.Kind = RouteSurfaceKindPage
		match.Source = renderUnitSourceGoFile(route.page.page.Unit)
		match.Layouts = routeExplainLayouts(route.page.layouts)
	case route.fragment != nil:
		match.Kind = RouteSurfaceKindFragment
		match.Source = renderUnitSourceGoFile(route.fragment.fragment.Unit)
	case route.action != nil:
		match.Kind = RouteSurfaceKindAction
		match.Source = manifestActionSourceGoFile(route.action.action)
		match.Function = route.action.action.Function
		match.Layouts = routeExplainLayouts(route.action.layouts)
	}
	if declarationInfo != nil && declarationInfo.Source != "" {
		match.Source = declarationInfo.Source
	}
	return match
}

func routeExplainLayouts(layouts []routing.ManifestLayout) []RouteExplanationLayout {
	result := make([]RouteExplanationLayout, 0, len(layouts))
	for _, layout := range layouts {
		result = append(result, RouteExplanationLayout{
			RoutePrefix: layout.RoutePrefix,
			Source:      layout.Unit.GoFile,
		})
	}
	return result
}
