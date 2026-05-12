// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package wiring

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mobiletoly/goldr/internal/routing"
)

const (
	RouteSurfaceKindLayout   = "layout"
	RouteSurfaceKindPage     = "page"
	RouteSurfaceKindFragment = "fragment"
	RouteSurfaceKindAction   = "action"
)

type RouteSurfaceRow struct {
	Kind    string
	Methods []string
	Path    string
	Params  []string
	Source  string
	Helper  string
}

func FormatRouteSurfaceList(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ",")
}

func RouteSurface(manifest routing.Manifest) ([]RouteSurfaceRow, error) {
	routes, err := runtimeRoutes(manifest)
	if err != nil {
		return nil, err
	}
	paths := runtimePaths(routes)
	if _, err := buildURLHelperTree(paths); err != nil {
		return nil, err
	}

	return routeSurfaceRows(manifest, routes), nil
}

func routeSurfaceRows(manifest routing.Manifest, routes []runtimeRoute) []RouteSurfaceRow {
	rows := make([]RouteSurfaceRow, 0, len(manifest.Layouts)+len(routes))
	for _, layout := range manifest.Layouts {
		rows = append(rows, RouteSurfaceRow{
			Kind:   RouteSurfaceKindLayout,
			Path:   layout.RoutePrefix,
			Params: slices.Clone(layout.Params),
			Source: layout.Unit.GoFile,
		})
	}
	for _, route := range routes {
		rows = append(rows, routeSurfaceRuntimeRow(route))
	}
	slices.SortFunc(rows, compareRouteSurfaceRows)
	return rows
}

func routeSurfaceRuntimeRow(route runtimeRoute) RouteSurfaceRow {
	row := RouteSurfaceRow{
		Methods: routeMethods(route),
		Path:    route.route,
		Params:  slices.Clone(route.params),
		Helper:  routeSurfaceHelper(route.route),
	}

	switch {
	case route.page != nil:
		row.Kind = RouteSurfaceKindPage
		row.Source = route.page.page.Unit.GoFile
	case route.fragment != nil:
		row.Kind = RouteSurfaceKindFragment
		row.Source = route.fragment.fragment.Unit.GoFile
	case route.action != nil:
		row.Kind = RouteSurfaceKindAction
		row.Source = fmt.Sprintf("%s:%s", route.action.action.GoFile, route.action.action.Function)
	}
	return row
}

func routeSurfaceHelper(route string) string {
	if route == "/" {
		return "urls.Root.Path()"
	}

	var builder strings.Builder
	builder.WriteString("urls")
	for _, segment := range routeSegments(route) {
		builder.WriteByte('.')
		if paramName, ok := paramSegmentName(segment); ok {
			fmt.Fprintf(&builder, "By%s(%s)", exportedSegmentName(paramName), unexportedIdentifier(paramName))
			continue
		}
		builder.WriteString(exportedSegmentName(segment))
	}
	builder.WriteString(".Path()")
	return builder.String()
}

func compareRouteSurfaceRows(a, b RouteSurfaceRow) int {
	if result := compareRouteSurfacePath(a.Path, b.Path); result != 0 {
		return result
	}
	if result := routeSurfaceKindRank(a.Kind) - routeSurfaceKindRank(b.Kind); result != 0 {
		return result
	}
	if result := routeSurfaceMethodRank(a.Methods) - routeSurfaceMethodRank(b.Methods); result != 0 {
		return result
	}
	return strings.Compare(a.Source, b.Source)
}

func compareRouteSurfacePath(a, b string) int {
	if a == "/" && b != "/" {
		return -1
	}
	if b == "/" && a != "/" {
		return 1
	}
	aSegments := routeSegments(a)
	bSegments := routeSegments(b)
	if staticCount(aSegments) != staticCount(bSegments) {
		return staticCount(bSegments) - staticCount(aSegments)
	}
	if len(aSegments) != len(bSegments) {
		return len(aSegments) - len(bSegments)
	}
	return strings.Compare(a, b)
}

func routeSurfaceKindRank(kind string) int {
	switch kind {
	case RouteSurfaceKindLayout:
		return 0
	case RouteSurfaceKindPage:
		return 1
	case RouteSurfaceKindFragment:
		return 2
	case RouteSurfaceKindAction:
		return 3
	default:
		return 99
	}
}

func routeSurfaceMethodRank(methods []string) int {
	if len(methods) == 0 {
		return methodRank("")
	}
	if methods[0] == "GET" {
		return methodRank("GET")
	}
	return methodRank(methods[0])
}
