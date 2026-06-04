// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"slices"
)

type Manifest struct {
	Root        string
	MountRoot   string
	Pages       []ManifestPage
	Layouts     []ManifestLayout
	Fragments   []ManifestFragment
	Actions     []ManifestAction
	Middlewares []ManifestMiddleware
	Routes      []ManifestRouteDeclaration
	MountRoutes []ManifestMountRouteSelection
	MountSource []ManifestMountSourceRoute
}

type RenderUnit struct {
	GoFile           string
	SourceGoFile     string
	TemplFile        string
	HasTempl         bool
	MiddlewareGoFile string
}

type ManifestPage struct {
	Route    string
	Params   []string
	Nav      RouteNavDeclaration
	Unit     RenderUnit
	Function string
}

type ManifestLayout struct {
	RoutePrefix string
	Params      []string
	Unit        RenderUnit
}

type ManifestFragment struct {
	Name        string
	RoutePrefix string
	Params      []string
	Nav         RouteNavDeclaration
	Unit        RenderUnit
	Function    string
	Segment     string
	Index       bool
}

type ManifestAction struct {
	Method              string
	Route               string
	NavRoute            string
	Params              []string
	Nav                 RouteNavDeclaration
	GoFile              string
	SourceGoFile        string
	MiddlewareGoFile    string
	Function            string
	Suffix              string
	Segment             string
	Writer              bool
	AdapterReturnsError bool
}

type ManifestMiddleware struct {
	RoutePrefix string
	Params      []string
	GoFile      string
}

type ManifestRouteDeclaration struct {
	Route            string
	Params           []string
	GoFile           string
	MiddlewareGoFile string
	Imports          []RouteImportDeclaration
	Kind             string
	Name             string
	Title            string
	Meta             []RouteMetaLabel
	Nav              RouteNavDeclaration
	Page             *RouteHandlerDeclaration
	Fragments        []RouteFragmentDeclaration
	Actions          []RouteActionDeclaration
	Kit              *RouteKitDeclaration
	Mount            *RouteMountDeclaration
	Destinations     []RouteDestinationDeclaration
	Source           string
	Adapter          string
}

type ManifestMountRouteSelection struct {
	MountPath    string
	Owner        string
	Source       string
	Route        string
	Params       []string
	Nav          RouteNavDeclaration
	Destinations []RouteDestinationDeclaration
	Included     bool
}

type ManifestMountSourceRoute struct {
	MountPath string
	Source    string
	Route     string
	Params    []string
	Page      *RouteHandlerDeclaration
	Fragments []RouteFragmentDeclaration
	Actions   []RouteActionDeclaration
}

func BuildManifest(tree Tree) Manifest {
	manifest := Manifest{
		Root:        tree.Root,
		MountRoot:   tree.MountRoot,
		Pages:       make([]ManifestPage, 0, len(tree.Pages)),
		Layouts:     make([]ManifestLayout, 0, len(tree.Layouts)),
		Fragments:   make([]ManifestFragment, 0, len(tree.Fragments)),
		Actions:     make([]ManifestAction, 0, len(tree.Actions)),
		Middlewares: make([]ManifestMiddleware, 0, len(tree.Middlewares)),
		Routes:      make([]ManifestRouteDeclaration, 0, len(tree.Routes)),
	}

	for _, page := range tree.Pages {
		manifest.Pages = append(manifest.Pages, ManifestPage{
			Route:  page.Route,
			Params: slices.Clone(page.Params),
			Unit: RenderUnit{
				GoFile:    page.GoFile,
				TemplFile: page.TemplFile,
				HasTempl:  page.HasTempl,
			},
		})
	}

	for _, layout := range tree.Layouts {
		manifest.Layouts = append(manifest.Layouts, ManifestLayout{
			RoutePrefix: layout.RoutePrefix,
			Params:      slices.Clone(layout.Params),
			Unit: RenderUnit{
				GoFile:    layout.GoFile,
				TemplFile: layout.TemplFile,
				HasTempl:  layout.HasTempl,
			},
		})
	}

	for _, fragment := range tree.Fragments {
		manifest.Fragments = append(manifest.Fragments, ManifestFragment{
			Name:        fragment.Name,
			RoutePrefix: fragment.RoutePrefix,
			Params:      slices.Clone(fragment.Params),
			Unit: RenderUnit{
				GoFile:    fragment.GoFile,
				TemplFile: fragment.TemplFile,
				HasTempl:  fragment.HasTempl,
			},
			Index: fragment.Index,
		})
	}

	for _, action := range tree.Actions {
		manifest.Actions = append(manifest.Actions, ManifestAction{
			Method:   action.Method,
			Route:    action.Route,
			Params:   slices.Clone(action.Params),
			GoFile:   action.GoFile,
			Function: action.Function,
			Suffix:   action.Suffix,
			Segment:  action.Segment,
			Writer:   action.Writer,
		})
	}

	for _, middleware := range tree.Middlewares {
		manifest.Middlewares = append(manifest.Middlewares, ManifestMiddleware{
			RoutePrefix: middleware.RoutePrefix,
			Params:      slices.Clone(middleware.Params),
			GoFile:      middleware.GoFile,
		})
	}

	for _, route := range tree.Routes {
		manifest.Routes = append(manifest.Routes, ManifestRouteDeclaration{
			Route:            route.Route,
			Params:           slices.Clone(route.Params),
			GoFile:           route.GoFile,
			MiddlewareGoFile: route.MiddlewareGoFile,
			Imports:          slices.Clone(route.Imports),
			Kind:             route.Kind,
			Name:             route.Name,
			Title:            route.Title,
			Meta:             slices.Clone(route.Meta),
			Nav:              cloneRouteNavDeclaration(route.Nav),
			Page:             cloneRouteHandlerDeclaration(route.Page),
			Fragments:        slices.Clone(route.Fragments),
			Actions:          slices.Clone(route.Actions),
			Kit:              cloneRouteKitDeclaration(route.Kit),
			Mount:            cloneRouteMountDeclaration(route.Mount),
			Destinations:     cloneRouteDestinations(route.Destinations),
			Source:           route.Source,
			Adapter:          route.Adapter,
		})
	}

	if len(tree.MountRoutes) > 0 {
		manifest.MountRoutes = make([]ManifestMountRouteSelection, 0, len(tree.MountRoutes))
		for _, route := range tree.MountRoutes {
			manifest.MountRoutes = append(manifest.MountRoutes, ManifestMountRouteSelection{
				MountPath:    route.MountPath,
				Owner:        route.Owner,
				Source:       route.Source,
				Route:        route.Route,
				Params:       slices.Clone(route.Params),
				Nav:          cloneRouteNavDeclaration(route.Nav),
				Destinations: cloneRouteDestinations(route.Destinations),
				Included:     route.Included,
			})
		}
	}

	if len(tree.MountSource) > 0 {
		manifest.MountSource = make([]ManifestMountSourceRoute, 0, len(tree.MountSource))
		for _, route := range tree.MountSource {
			manifest.MountSource = append(manifest.MountSource, ManifestMountSourceRoute{
				MountPath: route.MountPath,
				Source:    route.Source,
				Route:     route.Route,
				Params:    slices.Clone(route.Params),
				Page:      cloneRouteHandlerDeclaration(route.Page),
				Fragments: slices.Clone(route.Fragments),
				Actions:   slices.Clone(route.Actions),
			})
		}
	}

	sortManifest(&manifest)

	return manifest
}

func sortManifest(manifest *Manifest) {
	slices.SortFunc(manifest.Pages, func(a, b ManifestPage) int {
		return compareRouteOrder(a.Route, a.Unit.GoFile, b.Route, b.Unit.GoFile)
	})
	slices.SortFunc(manifest.Layouts, func(a, b ManifestLayout) int {
		return compareRouteOrder(a.RoutePrefix, a.Unit.GoFile, b.RoutePrefix, b.Unit.GoFile)
	})
	slices.SortFunc(manifest.Fragments, func(a, b ManifestFragment) int {
		return compareFragmentOrder(a.RoutePrefix, a.Name, a.Unit.GoFile, b.RoutePrefix, b.Name, b.Unit.GoFile)
	})
	slices.SortFunc(manifest.Actions, func(a, b ManifestAction) int {
		return compareActionOrder(a.Route, a.Method, a.Function, b.Route, b.Method, b.Function)
	})
	slices.SortFunc(manifest.Middlewares, func(a, b ManifestMiddleware) int {
		return compareRouteOrder(a.RoutePrefix, a.GoFile, b.RoutePrefix, b.GoFile)
	})
	slices.SortFunc(manifest.Routes, func(a, b ManifestRouteDeclaration) int {
		return compareRouteOrder(a.Route, a.GoFile, b.Route, b.GoFile)
	})
	slices.SortFunc(manifest.MountRoutes, func(a, b ManifestMountRouteSelection) int {
		return compareRouteOrder(a.Route, a.Source, b.Route, b.Source)
	})
	slices.SortFunc(manifest.MountSource, func(a, b ManifestMountSourceRoute) int {
		return compareRouteOrder(a.Route, a.Source, b.Route, b.Source)
	})
}

func cloneRouteHandlerDeclaration(value *RouteHandlerDeclaration) *RouteHandlerDeclaration {
	if value == nil {
		return nil
	}
	next := *value
	return &next
}

func cloneRouteKitDeclaration(value *RouteKitDeclaration) *RouteKitDeclaration {
	if value == nil {
		return nil
	}
	next := *value
	return &next
}

func cloneRouteMountDeclaration(value *RouteMountDeclaration) *RouteMountDeclaration {
	if value == nil {
		return nil
	}
	next := *value
	next.Routes = cloneMountRoutes(value.Routes)
	return &next
}

func cloneMountRoutes(values []MountRouteDeclaration) []MountRouteDeclaration {
	if len(values) == 0 {
		return nil
	}
	next := make([]MountRouteDeclaration, len(values))
	for index, value := range values {
		next[index] = value
		next[index].Nav = cloneRouteNavDeclaration(value.Nav)
		next[index].Destinations = cloneRouteDestinations(value.Destinations)
	}
	return next
}

func cloneRouteDestinations(values []RouteDestinationDeclaration) []RouteDestinationDeclaration {
	if len(values) == 0 {
		return nil
	}
	next := make([]RouteDestinationDeclaration, len(values))
	for index, value := range values {
		next[index] = value
		next[index].Target = slices.Clone(value.Target)
	}
	return next
}
