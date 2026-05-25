// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import "net/http"

// RouteDef declares a route surface backed by local function handlers.
type RouteDef struct {
	Name      string
	Title     string
	Page      FuncPageDef
	Fragments FuncFragments
	Actions   FuncActions
	Meta      RouteMeta
}

// KitRouteDef declares a route surface backed by a shared kit implementation.
// Live routes under app/routes must declare New. Mounted route surfaces under
// app/mounts omit New because their KitRouteMount owner supplies it.
type KitRouteDef[K any] struct {
	Name      string
	Title     string
	New       func(*http.Request) K
	Page      KitPageDef[K]
	Fragments KitFragments[K]
	Actions   KitActions[K]
	Meta      RouteMeta
}

// KitRouteMount declares a live route owner that mounts a shared kit route
// surface from app/mounts. Mount is a clean relative slash path under
// app/mounts using lowercase Go-safe route directory names.
type KitRouteMount[K any] struct {
	New   func(*http.Request) K
	Mount string
}

// RouteMeta carries app-owned opaque route metadata.
type RouteMeta struct {
	Labels map[string]string
}

// FuncPageDef is a local page handler declaration.
type FuncPageDef struct {
	fn func(*http.Request) RouteResponse
}

// FuncFragmentDef is a local fragment handler declaration.
type FuncFragmentDef struct {
	segment string
	index   bool
	fn      func(*http.Request) RouteResponse
}

// FuncActionDef is a local action handler declaration.
type FuncActionDef struct {
	method  string
	segment string
	index   bool
	fn      func(*http.Request) RouteResponse
	handler func(http.ResponseWriter, *http.Request)
}

// FuncFragments is a list of local fragment handler declarations.
type FuncFragments []FuncFragmentDef

// FuncActions is a list of local action handler declarations.
type FuncActions []FuncActionDef

// FuncPage declares a local page handler.
func FuncPage(fn func(*http.Request) RouteResponse) FuncPageDef {
	return FuncPageDef{fn: fn}
}

// FuncFragment declares a local fragment handler.
func FuncFragment(segment string, fn func(*http.Request) RouteResponse) FuncFragmentDef {
	return FuncFragmentDef{segment: segment, fn: fn}
}

// FuncFragmentIndex declares a local fragment handler at the route index path.
func FuncFragmentIndex(fn func(*http.Request) RouteResponse) FuncFragmentDef {
	return FuncFragmentDef{index: true, fn: fn}
}

// FuncPost declares a local POST action handler at segment.
func FuncPost(segment string, fn func(*http.Request) RouteResponse) FuncActionDef {
	return funcAction("POST", segment, false, fn)
}

// FuncPostIndex declares a local POST action handler at the route index path.
func FuncPostIndex(fn func(*http.Request) RouteResponse) FuncActionDef {
	return funcAction("POST", "", true, fn)
}

// FuncPut declares a local PUT action handler at segment.
func FuncPut(segment string, fn func(*http.Request) RouteResponse) FuncActionDef {
	return funcAction("PUT", segment, false, fn)
}

// FuncPutIndex declares a local PUT action handler at the route index path.
func FuncPutIndex(fn func(*http.Request) RouteResponse) FuncActionDef {
	return funcAction("PUT", "", true, fn)
}

// FuncPatch declares a local PATCH action handler at segment.
func FuncPatch(segment string, fn func(*http.Request) RouteResponse) FuncActionDef {
	return funcAction("PATCH", segment, false, fn)
}

// FuncPatchIndex declares a local PATCH action handler at the route index path.
func FuncPatchIndex(fn func(*http.Request) RouteResponse) FuncActionDef {
	return funcAction("PATCH", "", true, fn)
}

// FuncDelete declares a local DELETE action handler at segment.
func FuncDelete(segment string, fn func(*http.Request) RouteResponse) FuncActionDef {
	return funcAction("DELETE", segment, false, fn)
}

// FuncDeleteIndex declares a local DELETE action handler at the route index path.
func FuncDeleteIndex(fn func(*http.Request) RouteResponse) FuncActionDef {
	return funcAction("DELETE", "", true, fn)
}

// FuncPostHandler declares a local writer-based POST action handler at segment.
func FuncPostHandler(segment string, fn func(http.ResponseWriter, *http.Request)) FuncActionDef {
	return funcActionHandler("POST", segment, false, fn)
}

// FuncPostHandlerIndex declares a local writer-based POST action handler at
// the route index path.
func FuncPostHandlerIndex(fn func(http.ResponseWriter, *http.Request)) FuncActionDef {
	return funcActionHandler("POST", "", true, fn)
}

// FuncPutHandler declares a local writer-based PUT action handler at segment.
func FuncPutHandler(segment string, fn func(http.ResponseWriter, *http.Request)) FuncActionDef {
	return funcActionHandler("PUT", segment, false, fn)
}

// FuncPutHandlerIndex declares a local writer-based PUT action handler at the
// route index path.
func FuncPutHandlerIndex(fn func(http.ResponseWriter, *http.Request)) FuncActionDef {
	return funcActionHandler("PUT", "", true, fn)
}

// FuncPatchHandler declares a local writer-based PATCH action handler at segment.
func FuncPatchHandler(segment string, fn func(http.ResponseWriter, *http.Request)) FuncActionDef {
	return funcActionHandler("PATCH", segment, false, fn)
}

// FuncPatchHandlerIndex declares a local writer-based PATCH action handler at
// the route index path.
func FuncPatchHandlerIndex(fn func(http.ResponseWriter, *http.Request)) FuncActionDef {
	return funcActionHandler("PATCH", "", true, fn)
}

// FuncDeleteHandler declares a local writer-based DELETE action handler at segment.
func FuncDeleteHandler(segment string, fn func(http.ResponseWriter, *http.Request)) FuncActionDef {
	return funcActionHandler("DELETE", segment, false, fn)
}

// FuncDeleteHandlerIndex declares a local writer-based DELETE action handler at
// the route index path.
func FuncDeleteHandlerIndex(fn func(http.ResponseWriter, *http.Request)) FuncActionDef {
	return funcActionHandler("DELETE", "", true, fn)
}

func funcAction(method string, segment string, index bool, fn func(*http.Request) RouteResponse) FuncActionDef {
	return FuncActionDef{
		method:  method,
		segment: segment,
		index:   index,
		fn:      fn,
	}
}

func funcActionHandler(method string, segment string, index bool, fn func(http.ResponseWriter, *http.Request)) FuncActionDef {
	return FuncActionDef{
		method:  method,
		segment: segment,
		index:   index,
		handler: fn,
	}
}

// KitPageDef is a kit page handler declaration.
type KitPageDef[K any] struct {
	fn func(K, *http.Request) RouteResponse
}

// KitFragmentDef is a kit fragment handler declaration.
type KitFragmentDef[K any] struct {
	segment string
	index   bool
	fn      func(K, *http.Request) RouteResponse
}

// KitActionDef is a kit action handler declaration.
type KitActionDef[K any] struct {
	method  string
	segment string
	index   bool
	fn      func(K, *http.Request) RouteResponse
	handler func(K, http.ResponseWriter, *http.Request)
}

// KitFragments is a list of kit fragment handler declarations.
type KitFragments[K any] []KitFragmentDef[K]

// KitActions is a list of kit action handler declarations.
type KitActions[K any] []KitActionDef[K]

// KitPage declares a kit page handler.
func KitPage[K any](fn func(K, *http.Request) RouteResponse) KitPageDef[K] {
	return KitPageDef[K]{fn: fn}
}

// KitFragment declares a kit fragment handler.
func KitFragment[K any](segment string, fn func(K, *http.Request) RouteResponse) KitFragmentDef[K] {
	return KitFragmentDef[K]{segment: segment, fn: fn}
}

// KitFragmentIndex declares a kit fragment handler at the route index path.
func KitFragmentIndex[K any](fn func(K, *http.Request) RouteResponse) KitFragmentDef[K] {
	return KitFragmentDef[K]{index: true, fn: fn}
}

// KitPost declares a kit POST action handler at segment.
func KitPost[K any](segment string, fn func(K, *http.Request) RouteResponse) KitActionDef[K] {
	return kitAction("POST", segment, false, fn)
}

// KitPostIndex declares a kit POST action handler at the route index path.
func KitPostIndex[K any](fn func(K, *http.Request) RouteResponse) KitActionDef[K] {
	return kitAction("POST", "", true, fn)
}

// KitPut declares a kit PUT action handler at segment.
func KitPut[K any](segment string, fn func(K, *http.Request) RouteResponse) KitActionDef[K] {
	return kitAction("PUT", segment, false, fn)
}

// KitPutIndex declares a kit PUT action handler at the route index path.
func KitPutIndex[K any](fn func(K, *http.Request) RouteResponse) KitActionDef[K] {
	return kitAction("PUT", "", true, fn)
}

// KitPatch declares a kit PATCH action handler at segment.
func KitPatch[K any](segment string, fn func(K, *http.Request) RouteResponse) KitActionDef[K] {
	return kitAction("PATCH", segment, false, fn)
}

// KitPatchIndex declares a kit PATCH action handler at the route index path.
func KitPatchIndex[K any](fn func(K, *http.Request) RouteResponse) KitActionDef[K] {
	return kitAction("PATCH", "", true, fn)
}

// KitDelete declares a kit DELETE action handler at segment.
func KitDelete[K any](segment string, fn func(K, *http.Request) RouteResponse) KitActionDef[K] {
	return kitAction("DELETE", segment, false, fn)
}

// KitDeleteIndex declares a kit DELETE action handler at the route index path.
func KitDeleteIndex[K any](fn func(K, *http.Request) RouteResponse) KitActionDef[K] {
	return kitAction("DELETE", "", true, fn)
}

// KitPostHandler declares a kit writer-based POST action handler at segment.
func KitPostHandler[K any](segment string, fn func(K, http.ResponseWriter, *http.Request)) KitActionDef[K] {
	return kitActionHandler("POST", segment, false, fn)
}

// KitPostHandlerIndex declares a kit writer-based POST action handler at the
// route index path.
func KitPostHandlerIndex[K any](fn func(K, http.ResponseWriter, *http.Request)) KitActionDef[K] {
	return kitActionHandler("POST", "", true, fn)
}

// KitPutHandler declares a kit writer-based PUT action handler at segment.
func KitPutHandler[K any](segment string, fn func(K, http.ResponseWriter, *http.Request)) KitActionDef[K] {
	return kitActionHandler("PUT", segment, false, fn)
}

// KitPutHandlerIndex declares a kit writer-based PUT action handler at the
// route index path.
func KitPutHandlerIndex[K any](fn func(K, http.ResponseWriter, *http.Request)) KitActionDef[K] {
	return kitActionHandler("PUT", "", true, fn)
}

// KitPatchHandler declares a kit writer-based PATCH action handler at segment.
func KitPatchHandler[K any](segment string, fn func(K, http.ResponseWriter, *http.Request)) KitActionDef[K] {
	return kitActionHandler("PATCH", segment, false, fn)
}

// KitPatchHandlerIndex declares a kit writer-based PATCH action handler at the
// route index path.
func KitPatchHandlerIndex[K any](fn func(K, http.ResponseWriter, *http.Request)) KitActionDef[K] {
	return kitActionHandler("PATCH", "", true, fn)
}

// KitDeleteHandler declares a kit writer-based DELETE action handler at segment.
func KitDeleteHandler[K any](segment string, fn func(K, http.ResponseWriter, *http.Request)) KitActionDef[K] {
	return kitActionHandler("DELETE", segment, false, fn)
}

// KitDeleteHandlerIndex declares a kit writer-based DELETE action handler at
// the route index path.
func KitDeleteHandlerIndex[K any](fn func(K, http.ResponseWriter, *http.Request)) KitActionDef[K] {
	return kitActionHandler("DELETE", "", true, fn)
}

func kitAction[K any](method string, segment string, index bool, fn func(K, *http.Request) RouteResponse) KitActionDef[K] {
	return KitActionDef[K]{
		method:  method,
		segment: segment,
		index:   index,
		fn:      fn,
	}
}

func kitActionHandler[K any](method string, segment string, index bool, fn func(K, http.ResponseWriter, *http.Request)) KitActionDef[K] {
	return KitActionDef[K]{
		method:  method,
		segment: segment,
		index:   index,
		handler: fn,
	}
}
