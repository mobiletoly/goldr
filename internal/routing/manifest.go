// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"slices"
)

type Manifest struct {
	Root        string
	Pages       []ManifestPage
	Layouts     []ManifestLayout
	Fragments   []ManifestFragment
	Actions     []ManifestAction
	Middlewares []ManifestMiddleware
}

type RenderUnit struct {
	GoFile    string
	TemplFile string
	HasTempl  bool
}

type ManifestPage struct {
	Route  string
	Params []string
	Unit   RenderUnit
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
}

type ManifestAction struct {
	Method   string
	Route    string
	Params   []string
	GoFile   string
	Function string
	Suffix   string
	Segment  string
}

type ManifestMiddleware struct {
	RoutePrefix string
	Params      []string
	GoFile      string
}

func BuildManifest(tree Tree) Manifest {
	manifest := Manifest{
		Root:        tree.Root,
		Pages:       make([]ManifestPage, 0, len(tree.Pages)),
		Layouts:     make([]ManifestLayout, 0, len(tree.Layouts)),
		Fragments:   make([]ManifestFragment, 0, len(tree.Fragments)),
		Actions:     make([]ManifestAction, 0, len(tree.Actions)),
		Middlewares: make([]ManifestMiddleware, 0, len(tree.Middlewares)),
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
		})
	}

	for _, middleware := range tree.Middlewares {
		manifest.Middlewares = append(manifest.Middlewares, ManifestMiddleware{
			RoutePrefix: middleware.RoutePrefix,
			Params:      slices.Clone(middleware.Params),
			GoFile:      middleware.GoFile,
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
}
