// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import "net/http"

// RouteDef declares a route surface backed by local function handlers.
type RouteDef struct {
	Name      string
	Title     string
	Page      PageHandler
	Fragments Fragments
	Actions   Actions
	Meta      RouteMeta
}

// KitRouteDef declares a route surface backed by a shared kit implementation.
// Live routes under app/routes must declare New. Mounted route surfaces under
// app/mounts omit New because their KitRouteMount owner supplies it.
type KitRouteDef[K any] struct {
	Name      string
	Title     string
	New       func(*http.Request) K
	Page      KitPageHandler[K]
	Fragments KitFragments[K]
	Actions   KitActions[K]
	Meta      RouteMeta
}

// KitRouteMount declares a live route owner that mounts a shared kit route
// surface from app/mounts. Mount is a clean relative slash path under
// app/mounts using lowercase Go-safe route directory names.
type KitRouteMount[K any] struct {
	New    func(*http.Request) K
	Mount  string
	Routes MountRoutes
}

// MountRoutes declares the mount-relative route declarations exposed by one
// KitRouteMount owner. Omit Routes to expose the full mounted subtree.
type MountRoutes []string

// RouteMeta carries app-owned opaque route metadata.
type RouteMeta struct {
	Labels map[string]string
}

// PageHandler is a local page handler declaration.
type PageHandler func(*http.Request) PageRouteResponse

// FragmentRouteDef is a local fragment route declaration.
type FragmentRouteDef struct {
	path string
	fn   func(*http.Request) FragmentRouteResponse
}

// ActionDef is a local action route declaration.
type ActionDef struct {
	method  string
	path    string
	fn      func(*http.Request) RouteResponse
	handler func(http.ResponseWriter, *http.Request)
}

// Fragments is a list of local fragment route declarations.
type Fragments []FragmentRouteDef

// Actions is a list of local action route declarations.
type Actions []ActionDef

// FragmentRoute declares a local fragment route at path.
func FragmentRoute(path string, fn func(*http.Request) FragmentRouteResponse) FragmentRouteDef {
	return FragmentRouteDef{path: path, fn: fn}
}

// Action declares a local action route that returns a RouteResponse.
func Action(method string, path string, fn func(*http.Request) RouteResponse) ActionDef {
	return ActionDef{method: method, path: path, fn: fn}
}

// HTTPAction declares a local action route that writes directly to the HTTP response.
func HTTPAction(method string, path string, fn func(http.ResponseWriter, *http.Request)) ActionDef {
	return ActionDef{method: method, path: path, handler: fn}
}

// KitPageHandler is a kit page handler declaration.
type KitPageHandler[K any] func(K, *http.Request) PageRouteResponse

// KitFragmentRouteDef is a kit fragment route declaration.
type KitFragmentRouteDef[K any] struct {
	path string
	fn   func(K, *http.Request) FragmentRouteResponse
}

// KitActionDef is a kit action route declaration.
type KitActionDef[K any] struct {
	method  string
	path    string
	fn      func(K, *http.Request) RouteResponse
	handler func(K, http.ResponseWriter, *http.Request)
}

// KitFragments is a list of kit fragment route declarations.
type KitFragments[K any] []KitFragmentRouteDef[K]

// KitActions is a list of kit action route declarations.
type KitActions[K any] []KitActionDef[K]

// KitFragmentRoute declares a kit fragment route at path.
func KitFragmentRoute[K any](path string, fn func(K, *http.Request) FragmentRouteResponse) KitFragmentRouteDef[K] {
	return KitFragmentRouteDef[K]{path: path, fn: fn}
}

// KitAction declares a kit action route that returns a RouteResponse.
func KitAction[K any](method string, path string, fn func(K, *http.Request) RouteResponse) KitActionDef[K] {
	return KitActionDef[K]{method: method, path: path, fn: fn}
}

// KitHTTPAction declares a kit action route that writes directly to the HTTP response.
func KitHTTPAction[K any](method string, path string, fn func(K, http.ResponseWriter, *http.Request)) KitActionDef[K] {
	return KitActionDef[K]{method: method, path: path, handler: fn}
}
