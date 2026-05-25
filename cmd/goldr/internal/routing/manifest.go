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
	Unit        RenderUnit
	Function    string
	Segment     string
	Index       bool
}

type ManifestAction struct {
	Method           string
	Route            string
	Params           []string
	GoFile           string
	SourceGoFile     string
	MiddlewareGoFile string
	Function         string
	Suffix           string
	Segment          string
	Writer           bool
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
	Page             *RouteHandlerDeclaration
	Fragments        []RouteFragmentDeclaration
	Actions          []RouteActionDeclaration
	Kit              *RouteKitDeclaration
	Mount            *RouteMountDeclaration
	Source           string
	Adapter          string
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
			Page:             cloneRouteHandlerDeclaration(route.Page),
			Fragments:        slices.Clone(route.Fragments),
			Actions:          slices.Clone(route.Actions),
			Kit:              cloneRouteKitDeclaration(route.Kit),
			Mount:            cloneRouteMountDeclaration(route.Mount),
			Source:           route.Source,
			Adapter:          route.Adapter,
		})
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
	return &next
}
