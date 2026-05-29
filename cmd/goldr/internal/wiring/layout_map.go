// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package wiring

import (
	"path"
	"slices"
	"strings"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
)

type RouteLayoutMap struct {
	Root *RouteLayoutMapNode
}

type RouteLayoutMapNode struct {
	Name      string
	Dir       string
	Layout    *RouteLayoutMapLayout
	Pages     []RouteLayoutMapPage
	Fragments []RouteLayoutMapFragment
	Actions   []RouteLayoutMapAction
	Children  []*RouteLayoutMapNode
}

type RouteLayoutMapLayout struct {
	RoutePrefix string
	Params      []string
	Source      string
}

type RouteLayoutMapPage struct {
	Methods []string
	Route   string
	Params  []string
	Source  string
	Owner   string
	Layouts []RouteLayoutMapLayout
}

type RouteLayoutMapFragment struct {
	Methods []string
	Route   string
	Params  []string
	Source  string
	Owner   string
}

type RouteLayoutMapAction struct {
	Methods  []string
	Route    string
	Params   []string
	Source   string
	Owner    string
	Function string
	Layouts  []RouteLayoutMapLayout
}

func BuildRouteLayoutMap(manifest routing.Manifest) (RouteLayoutMap, error) {
	routes, err := runtimeRoutes(manifest)
	if err != nil {
		return RouteLayoutMap{}, err
	}
	inboundDestinations, err := inboundDestinationTrailEdgesByRoute(manifest.Routes)
	if err != nil {
		return RouteLayoutMap{}, err
	}

	builder := newRouteLayoutMapBuilder()
	for _, layout := range manifest.Layouts {
		node := builder.nodeForSource(layout.Unit.GoFile)
		node.Layout = new(routeLayoutMapLayout(layout))
	}

	for _, route := range routes {
		declarationInfo := routeDeclarationInfoForRuntimeRoute(route, manifest.Routes, inboundDestinations)
		owner := ""
		if declarationInfo != nil && declarationInfo.Mount != nil {
			owner = declarationInfo.Mount.Owner
		}
		switch {
		case route.page != nil:
			page := route.page.page
			source := renderUnitSourceGoFile(page.Unit)
			node := builder.nodeForSource(routeLayoutMapNodeSource(source, owner))
			node.Pages = append(node.Pages, RouteLayoutMapPage{
				Methods: routeMethods(route),
				Route:   route.route,
				Params:  slices.Clone(route.params),
				Source:  source,
				Owner:   owner,
				Layouts: routeLayoutMapLayouts(route.page.layouts),
			})
		case route.fragment != nil:
			fragment := route.fragment.fragment
			source := renderUnitSourceGoFile(fragment.Unit)
			node := builder.nodeForSource(routeLayoutMapNodeSource(source, owner))
			node.Fragments = append(node.Fragments, RouteLayoutMapFragment{
				Methods: routeMethods(route),
				Route:   route.route,
				Params:  slices.Clone(route.params),
				Source:  source,
				Owner:   owner,
			})
		case route.action != nil:
			action := route.action.action
			source := manifestActionSourceGoFile(action)
			node := builder.nodeForSource(routeLayoutMapNodeSource(source, owner))
			node.Actions = append(node.Actions, RouteLayoutMapAction{
				Methods:  routeMethods(route),
				Route:    route.route,
				Params:   slices.Clone(route.params),
				Source:   source,
				Owner:    owner,
				Function: action.Function,
				Layouts:  routeLayoutMapLayouts(route.action.layouts),
			})
		}
	}

	sortRouteLayoutMapNode(builder.root)
	return RouteLayoutMap{Root: builder.root}, nil
}

type routeLayoutMapBuilder struct {
	root  *RouteLayoutMapNode
	nodes map[string]*RouteLayoutMapNode
}

func newRouteLayoutMapBuilder() routeLayoutMapBuilder {
	root := &RouteLayoutMapNode{Name: "/"}
	return routeLayoutMapBuilder{
		root:  root,
		nodes: map[string]*RouteLayoutMapNode{"": root},
	}
}

func (builder routeLayoutMapBuilder) nodeForDir(dir string) *RouteLayoutMapNode {
	if dir == "" {
		return builder.root
	}
	if node, ok := builder.nodes[dir]; ok {
		return node
	}

	parentDir, name := path.Split(dir)
	parent := builder.nodeForDir(strings.TrimSuffix(parentDir, "/"))
	node := &RouteLayoutMapNode{
		Name: name,
		Dir:  dir,
	}
	builder.nodes[dir] = node
	parent.Children = append(parent.Children, node)
	return node
}

func (builder routeLayoutMapBuilder) nodeForSource(source string) *RouteLayoutMapNode {
	dir, _ := path.Split(source)
	return builder.nodeForDir(strings.TrimSuffix(dir, "/"))
}

func routeLayoutMapLayout(layout routing.ManifestLayout) RouteLayoutMapLayout {
	return RouteLayoutMapLayout{
		RoutePrefix: layout.RoutePrefix,
		Params:      slices.Clone(layout.Params),
		Source:      layout.Unit.GoFile,
	}
}

func routeLayoutMapLayouts(layouts []routing.ManifestLayout) []RouteLayoutMapLayout {
	result := make([]RouteLayoutMapLayout, 0, len(layouts))
	for _, layout := range layouts {
		result = append(result, routeLayoutMapLayout(layout))
	}
	return result
}

func routeLayoutMapNodeSource(source string, owner string) string {
	if owner != "" {
		return owner
	}
	return source
}

func sortRouteLayoutMapNode(node *RouteLayoutMapNode) {
	slices.SortFunc(node.Pages, func(a, b RouteLayoutMapPage) int {
		if result := compareRouteSurfacePath(a.Route, b.Route); result != 0 {
			return result
		}
		return strings.Compare(a.Source, b.Source)
	})
	slices.SortFunc(node.Fragments, func(a, b RouteLayoutMapFragment) int {
		if result := compareRouteSurfacePath(a.Route, b.Route); result != 0 {
			return result
		}
		return strings.Compare(a.Source, b.Source)
	})
	slices.SortFunc(node.Actions, func(a, b RouteLayoutMapAction) int {
		if result := compareRouteSurfacePath(a.Route, b.Route); result != 0 {
			return result
		}
		if result := routeSurfaceMethodRank(a.Methods) - routeSurfaceMethodRank(b.Methods); result != 0 {
			return result
		}
		return strings.Compare(a.Function, b.Function)
	})
	slices.SortFunc(node.Children, func(a, b *RouteLayoutMapNode) int {
		return strings.Compare(a.Name, b.Name)
	})
	for _, child := range node.Children {
		sortRouteLayoutMapNode(child)
	}
}
