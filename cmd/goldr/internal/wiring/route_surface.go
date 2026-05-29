// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package wiring

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
)

const (
	RouteSurfaceKindLayout   = "layout"
	RouteSurfaceKindRoute    = "route"
	RouteSurfaceKindPage     = "page"
	RouteSurfaceKindFragment = "fragment"
	RouteSurfaceKindAction   = "action"

	RouteSurfaceSelectionIncluded = "included"
	RouteSurfaceSelectionExcluded = "excluded"
)

type RouteSurfaceRow struct {
	Kind        string
	Methods     []string
	Path        string
	Params      []string
	Source      string
	Helper      string
	Selection   string
	Declaration *RouteDeclarationInfo
}

type RouteDeclarationInfo struct {
	Source              string
	Kind                string
	Name                string
	Title               string
	Labels              []RouteDeclarationLabel
	Nav                 RouteDeclarationNav
	TrailKeys           []string
	Destinations        []RouteDeclarationDestination
	InboundDestinations []RouteDeclarationInboundDestination
	Mount               *RouteDeclarationMount
	Kit                 *RouteDeclarationKit
	Page                *RouteDeclarationPage
	Fragment            *RouteDeclarationFragment
	Action              *RouteDeclarationAction
}

type RouteDeclarationLabel struct {
	Key   string
	Value string
}

type RouteDeclarationDestination struct {
	Name     string
	Helper   string
	Target   string
	TrailKey string
}

type RouteDeclarationInboundDestination struct {
	Source   string
	Name     string
	Helper   string
	TrailKey string
}

type RouteDeclarationNav struct {
	Label string
	Key   string
}

type RouteDeclarationKit struct {
	KitType string
	New     string
}

type RouteDeclarationMount struct {
	Path  string
	Owner string
}

type RouteDeclarationPage struct {
	Handler string
	Adapter string
}

type RouteDeclarationFragment struct {
	Name    string
	Segment string
	Index   bool
	Handler string
	Adapter string
}

type RouteDeclarationAction struct {
	Method  string
	Name    string
	Segment string
	Index   bool
	Writer  bool
	Handler string
	Adapter string
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

	inboundDestinations, err := inboundDestinationTrailEdgesByRoute(manifest.Routes)
	if err != nil {
		return nil, err
	}

	return routeSurfaceRows(manifest, routes, inboundDestinations), nil
}

func RouteSurfaceWithMountSelections(manifest routing.Manifest) ([]RouteSurfaceRow, error) {
	routes, err := runtimeRoutes(manifest)
	if err != nil {
		return nil, err
	}
	paths := runtimePaths(routes)
	if _, err := buildURLHelperTree(paths); err != nil {
		return nil, err
	}

	inboundDestinations, err := inboundDestinationTrailEdgesByRoute(manifest.Routes)
	if err != nil {
		return nil, err
	}

	rows := routeSurfaceRows(manifest, routes, inboundDestinations)
	for index := range rows {
		if rows[index].Declaration != nil && rows[index].Declaration.Mount != nil {
			rows[index].Selection = RouteSurfaceSelectionIncluded
		}
	}
	for _, selection := range manifest.MountRoutes {
		if selection.Included {
			continue
		}
		rows = append(rows, routeSurfaceMountSelectionRow(selection))
	}
	slices.SortFunc(rows, compareRouteSurfaceRows)
	return rows, nil
}

func routeSurfaceMountSelectionRow(selection routing.ManifestMountRouteSelection) RouteSurfaceRow {
	return RouteSurfaceRow{
		Kind:      RouteSurfaceKindRoute,
		Path:      selection.Route,
		Params:    slices.Clone(selection.Params),
		Source:    selection.Source,
		Selection: RouteSurfaceSelectionExcluded,
		Declaration: &RouteDeclarationInfo{
			Source:       selection.Source,
			Kind:         "mounted-kit",
			Nav:          routeDeclarationNav(selection.Nav),
			Destinations: routeDeclarationDestinationsForRoute(selection.Route, selection.Destinations),
			Mount: &RouteDeclarationMount{
				Path:  selection.MountPath,
				Owner: selection.Owner,
			},
		},
	}
}

func routeSurfaceRows(manifest routing.Manifest, routes []runtimeRoute, inboundDestinations map[string][]destinationTrailEdge) []RouteSurfaceRow {
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
		rows = append(rows, routeSurfaceRuntimeRow(route, manifest.Routes, inboundDestinations))
	}
	slices.SortFunc(rows, compareRouteSurfaceRows)
	return rows
}

func routeSurfaceRuntimeRow(route runtimeRoute, declarations []routing.ManifestRouteDeclaration, inboundDestinations map[string][]destinationTrailEdge) RouteSurfaceRow {
	declarationInfo := routeDeclarationInfoForRuntimeRoute(route, declarations, inboundDestinations)
	row := RouteSurfaceRow{
		Methods:     routeMethods(route),
		Path:        route.route,
		Params:      slices.Clone(route.params),
		Helper:      routeSurfaceHelper(route.route),
		Declaration: declarationInfo,
	}

	switch {
	case route.page != nil:
		row.Kind = RouteSurfaceKindPage
		row.Source = renderUnitSourceGoFile(route.page.page.Unit)
	case route.fragment != nil:
		row.Kind = RouteSurfaceKindFragment
		row.Source = renderUnitSourceGoFile(route.fragment.fragment.Unit)
	case route.action != nil:
		row.Kind = RouteSurfaceKindAction
		row.Source = fmt.Sprintf("%s:%s", manifestActionSourceGoFile(route.action.action), route.action.action.Function)
	}
	if declarationInfo != nil && declarationInfo.Source != "" {
		row.Source = declarationInfo.Source
		if route.action != nil && declarationInfo.Action != nil {
			row.Source = fmt.Sprintf("%s:%s", declarationInfo.Source, route.action.action.Function)
		}
	}
	return row
}

func routeDeclarationInfoForRuntimeRoute(route runtimeRoute, declarations []routing.ManifestRouteDeclaration, inboundDestinations map[string][]destinationTrailEdge) *RouteDeclarationInfo {
	for _, declaration := range declarations {
		if route.page != nil {
			if info := routeDeclarationInfoForPage(route, declaration, inboundDestinations[route.navRoute]); info != nil {
				return info
			}
			continue
		}
		if route.fragment != nil {
			if info := routeDeclarationInfoForFragment(route, declaration, inboundDestinations[route.navRoute]); info != nil {
				return info
			}
			continue
		}
		if route.action != nil {
			if info := routeDeclarationInfoForAction(route, declaration, inboundDestinations[route.navRoute]); info != nil {
				return info
			}
		}
	}
	return nil
}

func routeDeclarationInfoForPage(route runtimeRoute, declaration routing.ManifestRouteDeclaration, inboundDestinations []destinationTrailEdge) *RouteDeclarationInfo {
	if route.page.page.Unit.GoFile != declaration.GoFile || route.page.page.Route != declaration.Route || declaration.Page == nil {
		return nil
	}
	info := baseRouteDeclarationInfo(declaration, route.trailKeys, inboundDestinations)
	info.Page = &RouteDeclarationPage{
		Handler: declaration.Page.Handler,
		Adapter: routePageAdapterName(declaration),
	}
	return info
}

func routeDeclarationInfoForFragment(route runtimeRoute, declaration routing.ManifestRouteDeclaration, inboundDestinations []destinationTrailEdge) *RouteDeclarationInfo {
	fragment := route.fragment.fragment
	if fragment.Unit.GoFile != declaration.GoFile || fragment.RoutePrefix != declaration.Route {
		return nil
	}
	for _, declarationFragment := range declaration.Fragments {
		if fragment.Name != declarationFragment.Name || fragment.Segment != declarationFragment.Segment || fragment.Index != declarationFragment.Index {
			continue
		}
		info := baseRouteDeclarationInfo(declaration, route.trailKeys, inboundDestinations)
		info.Fragment = &RouteDeclarationFragment{
			Name:    declarationFragment.Name,
			Segment: declarationFragment.Segment,
			Index:   declarationFragment.Index,
			Handler: declarationFragment.Handler,
			Adapter: routeFragmentAdapterName(declaration, declarationFragment),
		}
		return info
	}
	return nil
}

func routeDeclarationInfoForAction(route runtimeRoute, declaration routing.ManifestRouteDeclaration, inboundDestinations []destinationTrailEdge) *RouteDeclarationInfo {
	action := route.action.action
	if action.GoFile != declaration.GoFile {
		return nil
	}
	for _, declarationAction := range declaration.Actions {
		if action.Method != declarationAction.Method || action.Route != routeDeclarationActionPath(declaration.Route, declarationAction) {
			continue
		}
		info := baseRouteDeclarationInfo(declaration, route.trailKeys, inboundDestinations)
		info.Action = &RouteDeclarationAction{
			Method:  declarationAction.Method,
			Name:    declarationAction.Name,
			Segment: declarationAction.Segment,
			Index:   declarationAction.Index,
			Writer:  declarationAction.Writer,
			Handler: declarationAction.Handler,
			Adapter: routeActionAdapterName(declaration, declarationAction),
		}
		return info
	}
	return nil
}

func baseRouteDeclarationInfo(declaration routing.ManifestRouteDeclaration, trailKeys []string, inboundDestinations []destinationTrailEdge) *RouteDeclarationInfo {
	info := &RouteDeclarationInfo{
		Source:              routeDeclarationSource(declaration),
		Kind:                declaration.Kind,
		Name:                declaration.Name,
		Title:               declaration.Title,
		Labels:              routeDeclarationLabels(declaration.Meta),
		Nav:                 routeDeclarationNav(declaration.Nav),
		TrailKeys:           slices.Clone(trailKeys),
		Destinations:        routeDeclarationDestinations(declaration),
		InboundDestinations: routeDeclarationInboundDestinations(inboundDestinations),
	}
	if declaration.Kit != nil {
		info.Kit = &RouteDeclarationKit{
			KitType: declaration.Kit.KitType,
			New:     declaration.Kit.New,
		}
	}
	if declaration.Mount != nil {
		info.Mount = &RouteDeclarationMount{
			Path:  declaration.Mount.Path,
			Owner: declaration.Mount.Owner,
		}
	}
	return info
}

func routeDeclarationDestinations(declaration routing.ManifestRouteDeclaration) []RouteDeclarationDestination {
	return routeDeclarationDestinationsForRoute(declaration.Route, declaration.Destinations)
}

func routeDeclarationDestinationsForRoute(route string, destinations []routing.RouteDestinationDeclaration) []RouteDeclarationDestination {
	if len(destinations) == 0 {
		return nil
	}
	result := make([]RouteDeclarationDestination, len(destinations))
	for index, destination := range destinations {
		result[index] = RouteDeclarationDestination{
			Name:     destination.Name,
			Helper:   routeSurfaceDestinationHelper(route, destination.SymbolName),
			Target:   "urls." + strings.Join(destination.Target, "."),
			TrailKey: destination.TrailKey,
		}
	}
	slices.SortFunc(result, func(a, b RouteDeclarationDestination) int {
		return strings.Compare(a.Name, b.Name)
	})
	return result
}

func routeDeclarationInboundDestinations(edges []destinationTrailEdge) []RouteDeclarationInboundDestination {
	if len(edges) == 0 {
		return nil
	}
	result := make([]RouteDeclarationInboundDestination, len(edges))
	for index, edge := range edges {
		result[index] = RouteDeclarationInboundDestination{
			Source:   edge.sourceRoute,
			Name:     edge.name,
			Helper:   routeSurfaceDestinationHelper(edge.sourceRoute, edge.symbolName),
			TrailKey: edge.trailKey,
		}
	}
	slices.SortFunc(result, func(a, b RouteDeclarationInboundDestination) int {
		if a.Source != b.Source {
			return strings.Compare(a.Source, b.Source)
		}
		if a.TrailKey != b.TrailKey {
			return strings.Compare(a.TrailKey, b.TrailKey)
		}
		return strings.Compare(a.Name, b.Name)
	})
	return result
}

func routeDeclarationNav(nav routing.RouteNavDeclaration) RouteDeclarationNav {
	return RouteDeclarationNav{
		Label: nav.Label,
		Key:   nav.Key,
	}
}

func routeSurfaceDestinationHelper(route string, destination string) string {
	helper := strings.TrimSuffix(routeSurfaceHelper(route), ".Path()")
	if helper == "-" {
		return "-"
	}
	return helper + ".Destinations." + destination
}

func routeDeclarationSource(declaration routing.ManifestRouteDeclaration) string {
	return sourceGoFile(declaration.Source, declaration.GoFile)
}

func routeDeclarationActionPath(route string, action routing.RouteActionDeclaration) string {
	if action.Segment == "" {
		return route
	}
	if route == "/" {
		return "/" + action.Segment
	}
	return route + "/" + action.Segment
}

func routeDeclarationLabels(labels []routing.RouteMetaLabel) []RouteDeclarationLabel {
	if len(labels) == 0 {
		return nil
	}
	result := make([]RouteDeclarationLabel, len(labels))
	for index, label := range labels {
		result[index] = RouteDeclarationLabel{
			Key:   label.Key,
			Value: label.Value,
		}
	}
	slices.SortFunc(result, func(a, b RouteDeclarationLabel) int {
		return strings.Compare(a.Key, b.Key)
	})
	return result
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
			fmt.Fprintf(&builder, "By%s.Bind(%s)", exportedSegmentName(paramName), unexportedIdentifier(paramName))
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
