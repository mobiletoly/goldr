// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package wiring

import (
	"bytes"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
)

func writeHandler(buffer *bytes.Buffer, routes []runtimeRoute, rootLayouts []routing.ManifestLayout) {
	paths := runtimePaths(routes)
	root := buildDispatchTree(paths)
	helpers := newHandlerHelperPlan(routes, rootLayouts)
	buffer.WriteString(`
func Handler() http.Handler {
	return HandlerWithOptions(HandlerOptions{})
}

func HandlerWithOptions(options HandlerOptions) http.Handler {
`)
	if helpers.hasEndpointHandlers() {
		buffer.WriteString(`	handlers := goldrNewHandlers(options)

`)
	}
	buffer.WriteString(`	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if options.TemplateInspection != goldr.TemplateInspectionOff {
			r = r.WithContext(goldrinspect.WithMode(r.Context(), options.TemplateInspection))
		}
		routePath := r.URL.EscapedPath()
`)
	if hasRoutesWithoutLayouts(routes) {
		buffer.WriteString(`		r = goldr.WithRoutePageRenderer(r, goldrDirectRoutePageRenderer)
`)
	}
	dispatchArgs := "options, w, r"
	if helpers.hasEndpointHandlers() {
		dispatchArgs = "options, handlers, w, r"
	}
	if hasSegmentRoutes(routes) {
		fmt.Fprintf(buffer, `		if routePath == "/" {
			goldrDispatchRoot(%s, nil)
			return
		}
		segments := goldrPathSegments(routePath)
		if len(segments) == 0 {
			goldrRouteNotFound(options, w, r)
			return
		}
		goldrDispatchRoot(%s, segments)
	})
}
`, dispatchArgs, dispatchArgs)
	} else {
		fmt.Fprintf(buffer, `		if routePath == "/" {
			goldrDispatchRoot(%s, nil)
			return
		}
		goldrRouteNotFound(options, w, r)
	})
}
`, dispatchArgs)
	}
	writeDispatchNodes(buffer, root, helpers)
	writeHandlerHelpers(buffer, helpers)

	buffer.WriteString(`
func goldrDirectRoutePageRenderer(r *http.Request, page goldr.Page) (templ.Component, error) {
	component := page.Component
	if component == nil {
		return nil, goldr.ErrNilComponent
	}
	return component, nil
}

func goldrRouteNotFound(options HandlerOptions, w http.ResponseWriter, r *http.Request) {
	handlers := options.ErrorHandlers
	if handlers.RouteNotFound != nil {
		goldrWriteRouteFallbackResponse(w, r, handlers.RouteNotFound(r), goldrRootErrorRoutePageRenderer)
		return
	}
	http.NotFound(w, r)
}

func goldrRouteMethodNotAllowed(options HandlerOptions, w http.ResponseWriter, r *http.Request) {
	handlers := options.ErrorHandlers
	if handlers.RouteMethodNotAllowed != nil {
		goldrWriteRouteFallbackResponse(w, r, handlers.RouteMethodNotAllowed(r), goldrRootErrorRoutePageRenderer)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func goldrRouteError(options HandlerOptions, w http.ResponseWriter, r *http.Request, err error, render goldr.RoutePageRenderer) {
	handlers := options.ErrorHandlers
	if handlers.RouteError != nil {
		goldrWriteRouteErrorResponse(w, r, handlers.RouteError(r, err), render)
		return
	}
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
`)
	writeEndpointResponseHelpers(buffer, routes)
	buffer.WriteString(`

func goldrWriteRouteFallbackResponse(w http.ResponseWriter, r *http.Request, response goldr.RouteResponse, render goldr.RoutePageRenderer) {
	r = goldr.WithRoutePageRenderer(r, render)
	if err := goldr.WriteRouteResponse(w, r, response); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func goldrWriteRouteErrorResponse(w http.ResponseWriter, r *http.Request, response goldr.RouteResponse, render goldr.RoutePageRenderer) {
	r = goldr.WithRoutePageRenderer(r, render)
	if err := goldr.WriteRouteResponse(w, r, response); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
`)

	writeRootErrorRoutePageRenderer(buffer, rootLayouts, helpers)

	if hasSegmentRoutes(routes) {
		buffer.WriteString(`
func goldrPathSegments(routePath string) []string {
	if routePath == "/" || !strings.HasPrefix(routePath, "/") || strings.HasSuffix(routePath, "/") {
		return nil
	}
	return strings.Split(strings.TrimPrefix(routePath, "/"), "/")
}
`)
	}

	if hasDynamicRoutes(routes) {
		buffer.WriteString(`
func goldrPathParam(segment string) (string, bool) {
	value, err := url.PathUnescape(segment)
	if err != nil {
		return "", false
	}
	return value, true
}
`)
	}

	if hasRequestNavRoutes(routes) {
		buffer.WriteString(`
func goldrRequestTrailKey(r *http.Request, allowed []string) string {
	key := r.URL.Query().Get("_goldr_nav_trail_key")
	if key == "" {
		return ""
	}
	if slices.Contains(allowed, key) {
		return key
	}
	return ""
}

func goldrNavHref(basePath string, segments ...string) string {
	basePath = goldrNormalizeBasePath(basePath)
	if len(segments) == 0 {
		if basePath == "" {
			return "/"
		}
		return basePath + "/"
	}

	size := len(basePath)
	for _, segment := range segments {
		if segment == "" {
			return ""
		}
		size += 1 + len(segment)
	}

	var builder strings.Builder
	builder.Grow(size)
	builder.WriteString(basePath)
	for _, segment := range segments {
		builder.WriteByte('/')
		builder.WriteString(segment)
	}
	return builder.String()
}

func goldrNormalizeBasePath(basePath string) string {
	if basePath == "" || basePath == "/" {
		return ""
	}
	if basePath[0] != '/' {
		basePath = "/" + basePath
	}
	for len(basePath) > 1 && basePath[len(basePath)-1] == '/' {
		basePath = basePath[:len(basePath)-1]
	}
	if basePath == "/" {
		return ""
	}
	return basePath
}
`)
	}

	if hasFragmentRoutes(routes) {
		buffer.WriteString(`
func goldrWrapFragmentRouteResponse(response goldr.FragmentRouteResponse, marker goldrinspect.Marker) goldr.FragmentRouteResponse {
	switch response := response.(type) {
	case goldr.Fragment:
		response.Component = goldrinspect.Wrap(response.Component, marker)
		return response
	case *goldr.Fragment:
		if response == nil {
			return response
		}
		next := *response
		next.Component = goldrinspect.Wrap(next.Component, marker)
		return next
	default:
		return response
	}
}

`)
	}

}

func writeEndpointResponseHelpers(buffer *bytes.Buffer, routes []runtimeRoute) {
	needsPage, needsFragment, needsAction := endpointResponseHelperNeeds(routes)
	if needsPage {
		buffer.WriteString(`

func goldrWritePageEndpointResponse(options HandlerOptions, w http.ResponseWriter, r *http.Request, response goldr.PageRouteResponse, marker goldrinspect.Marker, layouts []goldrLayoutStep, routeErrorRender goldr.RoutePageRenderer) {
	render := func(r *http.Request, page goldr.Page) (templ.Component, error) {
		return goldrRenderPageWithMarker(r, page, marker, layouts)
	}
	if err := goldr.WritePageRouteResponse(w, r, response, render); err != nil {
		goldrRouteError(options, w, r, err, routeErrorRender)
	}
}
`)
	}
	if needsFragment {
		buffer.WriteString(`

func goldrWriteFragmentEndpointResponse(options HandlerOptions, w http.ResponseWriter, r *http.Request, response goldr.FragmentRouteResponse, render goldr.RoutePageRenderer) {
	if err := goldr.WriteFragmentRouteResponse(w, r, response); err != nil {
		goldrRouteError(options, w, r, err, render)
	}
}
`)
	}
	if needsAction {
		buffer.WriteString(`

func goldrWriteEndpointResponse(options HandlerOptions, w http.ResponseWriter, r *http.Request, response goldr.RouteResponse, render goldr.RoutePageRenderer) {
	if err := goldr.WriteRouteResponse(w, r, response); err != nil {
		goldrRouteError(options, w, r, err, render)
	}
}
`)
	}
}

func endpointResponseHelperNeeds(routes []runtimeRoute) (bool, bool, bool) {
	var needsPage bool
	var needsFragment bool
	var needsAction bool
	for _, route := range routes {
		switch {
		case route.page != nil:
			needsPage = true
		case route.fragment != nil:
			needsFragment = true
		case route.action != nil && !route.action.action.Writer:
			needsAction = true
		}
		if needsPage && needsFragment && needsAction {
			return needsPage, needsFragment, needsAction
		}
	}
	return needsPage, needsFragment, needsAction
}

type handlerHelperPlan struct {
	layoutStacks           []layoutStackHelper
	layoutStackNames       map[string]string
	layoutRendererNames    map[string]string
	middlewareStacks       []middlewareStackHelper
	middlewareStackNames   map[string]string
	endpointHandlers       []endpointHandlerHelper
	endpointHandlerNames   map[string]string
	needsPageRenderHelpers bool
}

type layoutStackHelper struct {
	name         string
	rendererName string
	layouts      []routing.ManifestLayout
}

type middlewareStackHelper struct {
	name        string
	middlewares []routing.ManifestMiddleware
}

type endpointHandlerHelper struct {
	name  string
	route runtimeRoute
}

func newHandlerHelperPlan(routes []runtimeRoute, rootLayouts []routing.ManifestLayout) handlerHelperPlan {
	plan := handlerHelperPlan{
		layoutStackNames:     make(map[string]string),
		layoutRendererNames:  make(map[string]string),
		middlewareStackNames: make(map[string]string),
		endpointHandlerNames: make(map[string]string),
	}
	plan.addLayoutStack(rootLayouts)
	for _, route := range routes {
		if route.page != nil {
			plan.addLayoutStack(route.page.layouts)
			plan.needsPageRenderHelpers = true
		}
		if route.fragment != nil {
			plan.addLayoutStack(route.fragment.layouts)
		}
		if route.action != nil {
			plan.addLayoutStack(route.action.layouts)
		}
		plan.addMiddlewareStack(routeMiddlewares(route))
		plan.addEndpointHandler(route)
	}
	return plan
}

func (plan *handlerHelperPlan) addLayoutStack(layouts []routing.ManifestLayout) {
	if len(layouts) == 0 {
		return
	}
	key := layoutStackKey(layouts)
	if _, ok := plan.layoutStackNames[key]; ok {
		return
	}
	index := len(plan.layoutStacks)
	name := fmt.Sprintf("goldrLayoutStack%d", index)
	rendererName := fmt.Sprintf("goldrRoutePageRenderer%d", index)
	plan.layoutStackNames[key] = name
	plan.layoutRendererNames[key] = rendererName
	plan.layoutStacks = append(plan.layoutStacks, layoutStackHelper{
		name:         name,
		rendererName: rendererName,
		layouts:      layouts,
	})
}

func (plan *handlerHelperPlan) addMiddlewareStack(middlewares []routing.ManifestMiddleware) {
	if len(middlewares) == 0 {
		return
	}
	key := middlewareStackKey(middlewares)
	if _, ok := plan.middlewareStackNames[key]; ok {
		return
	}
	name := fmt.Sprintf("goldrMiddlewareStack%d", len(plan.middlewareStacks))
	plan.middlewareStackNames[key] = name
	plan.middlewareStacks = append(plan.middlewareStacks, middlewareStackHelper{
		name:        name,
		middlewares: middlewares,
	})
}

func (plan *handlerHelperPlan) addEndpointHandler(route runtimeRoute) {
	if len(routeMiddlewares(route)) == 0 {
		return
	}
	key := endpointHandlerKey(route)
	if _, ok := plan.endpointHandlerNames[key]; ok {
		return
	}
	name := fmt.Sprintf("endpoint%d", len(plan.endpointHandlers))
	plan.endpointHandlerNames[key] = name
	plan.endpointHandlers = append(plan.endpointHandlers, endpointHandlerHelper{
		name:  name,
		route: route,
	})
}

func (plan handlerHelperPlan) layoutStackName(layouts []routing.ManifestLayout) string {
	if len(layouts) == 0 {
		return "nil"
	}
	return plan.layoutStackNames[layoutStackKey(layouts)]
}

func (plan handlerHelperPlan) layoutRendererName(layouts []routing.ManifestLayout) string {
	if len(layouts) == 0 {
		return ""
	}
	return plan.layoutRendererNames[layoutStackKey(layouts)]
}

func (plan handlerHelperPlan) routePageRendererName(layouts []routing.ManifestLayout) string {
	if len(layouts) == 0 {
		return "goldrDirectRoutePageRenderer"
	}
	return plan.layoutRendererName(layouts)
}

func (plan handlerHelperPlan) middlewareStackName(middlewares []routing.ManifestMiddleware) string {
	if len(middlewares) == 0 {
		return ""
	}
	return plan.middlewareStackNames[middlewareStackKey(middlewares)]
}

func (plan handlerHelperPlan) endpointHandlerName(route runtimeRoute) string {
	return plan.endpointHandlerNames[endpointHandlerKey(route)]
}

func (plan handlerHelperPlan) hasEndpointHandlers() bool {
	return len(plan.endpointHandlers) > 0
}

func layoutStackKey(layouts []routing.ManifestLayout) string {
	var builder strings.Builder
	for _, layout := range layouts {
		builder.WriteString(layout.RoutePrefix)
		builder.WriteByte('\x00')
		builder.WriteString(layout.Unit.GoFile)
		builder.WriteByte('\x00')
		builder.WriteString(layout.Unit.SourceGoFile)
		builder.WriteByte('\x00')
		builder.WriteString(layout.Unit.TemplFile)
		builder.WriteByte('\x00')
	}
	return builder.String()
}

func middlewareStackKey(middlewares []routing.ManifestMiddleware) string {
	var builder strings.Builder
	for _, middleware := range middlewares {
		builder.WriteString(middleware.RoutePrefix)
		builder.WriteByte('\x00')
		builder.WriteString(middleware.GoFile)
		builder.WriteByte('\x00')
	}
	return builder.String()
}

func endpointHandlerKey(route runtimeRoute) string {
	var builder strings.Builder
	builder.WriteString(route.route)
	builder.WriteByte('\x00')
	switch {
	case route.page != nil:
		builder.WriteString("page")
		builder.WriteByte('\x00')
		builder.WriteString(route.page.page.Unit.GoFile)
		builder.WriteByte('\x00')
		builder.WriteString(route.page.page.Function)
	case route.fragment != nil:
		builder.WriteString("fragment")
		builder.WriteByte('\x00')
		builder.WriteString(route.fragment.fragment.Unit.GoFile)
		builder.WriteByte('\x00')
		builder.WriteString(route.fragment.fragment.Name)
		builder.WriteByte('\x00')
		builder.WriteString(route.fragment.fragment.Function)
	case route.action != nil:
		builder.WriteString("action")
		builder.WriteByte('\x00')
		builder.WriteString(route.action.action.Method)
		builder.WriteByte('\x00')
		builder.WriteString(route.action.action.GoFile)
		builder.WriteByte('\x00')
		builder.WriteString(route.action.action.Function)
		builder.WriteByte('\x00')
		builder.WriteString(strconv.FormatBool(route.action.action.Writer))
		builder.WriteByte('\x00')
		builder.WriteString(strconv.FormatBool(route.action.action.AdapterReturnsError))
	}
	return builder.String()
}

func writeHandlerHelpers(buffer *bytes.Buffer, helpers handlerHelperPlan) {
	if helpers.needsPageRenderHelpers || len(helpers.layoutStacks) > 0 {
		buffer.WriteString(`
type goldrLayoutFunc func(*http.Request, goldr.LayoutContext) templ.Component

type goldrLayoutStep struct {
	render goldrLayoutFunc
	marker goldrinspect.Marker
}

func goldrRenderPageWithMarker(r *http.Request, page goldr.Page, marker goldrinspect.Marker, layouts []goldrLayoutStep) (templ.Component, error) {
	component, metadata, err := goldrPageComponent(page)
	if err != nil {
		return nil, err
	}
	component = goldrinspect.Wrap(component, marker)
	return goldrRenderPageLayouts(r, component, metadata, layouts)
}

func goldrRenderPage(r *http.Request, page goldr.Page, layouts []goldrLayoutStep) (templ.Component, error) {
	component, metadata, err := goldrPageComponent(page)
	if err != nil {
		return nil, err
	}
	return goldrRenderPageLayouts(r, component, metadata, layouts)
}

func goldrPageComponent(page goldr.Page) (templ.Component, goldr.PageMetadata, error) {
	component := page.Component
	if component == nil {
		return nil, goldr.PageMetadata{}, goldr.ErrNilComponent
	}
	return component, page.Metadata, nil
}

func goldrRenderPageLayouts(r *http.Request, component templ.Component, metadata goldr.PageMetadata, layouts []goldrLayoutStep) (templ.Component, error) {
	layoutContext := goldr.LayoutContext{Metadata: metadata}
	for index := len(layouts) - 1; index >= 0; index-- {
		layoutContext.Child = component
		component = layouts[index].render(r, layoutContext)
		if component == nil {
			return nil, goldr.ErrNilComponent
		}
		component = goldrinspect.Wrap(component, layouts[index].marker)
	}
	return component, nil
}
`)
	}

	for _, stack := range helpers.layoutStacks {
		writeLayoutStackHelper(buffer, stack)
	}
	for _, stack := range helpers.layoutStacks {
		fmt.Fprintf(buffer, "\nfunc %s(r *http.Request, page goldr.Page) (templ.Component, error) {\n", stack.rendererName)
		fmt.Fprintf(buffer, "\treturn goldrRenderPage(r, page, %s)\n", stack.name)
		buffer.WriteString("}\n")
	}
	for _, stack := range helpers.middlewareStacks {
		writeMiddlewareStackHelper(buffer, stack)
	}
	writeEndpointHandlerHelpers(buffer, helpers)
}

func writeLayoutStackHelper(buffer *bytes.Buffer, stack layoutStackHelper) {
	fmt.Fprintf(buffer, "\nvar %s = []goldrLayoutStep{\n", stack.name)
	for _, layout := range stack.layouts {
		writeLayoutCallComment(buffer, "\t", layout)
		buffer.WriteString("\t{\n")
		fmt.Fprintf(buffer, "\t\trender: %s,\n", routeFunc(layout.Unit.GoFile, "Layout"))
		fmt.Fprintf(buffer, "\t\tmarker: %s,\n", templateMarker("layout", layout.RoutePrefix, layout.Unit))
		buffer.WriteString("\t},\n")
	}
	buffer.WriteString("}\n")
}

func writeMiddlewareStackHelper(buffer *bytes.Buffer, stack middlewareStackHelper) {
	fmt.Fprintf(buffer, "\nfunc %s(next http.Handler) http.Handler {\n", stack.name)
	for index := len(stack.middlewares) - 1; index >= 0; index-- {
		middlewareCall := routeFunc(stack.middlewares[index].GoFile, "Middleware")
		writeMiddlewareCallComment(buffer, "\t", stack.middlewares[index])
		fmt.Fprintf(buffer, "\tnext = %s(next)\n", middlewareCall)
	}
	buffer.WriteString("\treturn next\n")
	buffer.WriteString("}\n")
}

func writeEndpointHandlerHelpers(buffer *bytes.Buffer, helpers handlerHelperPlan) {
	if !helpers.hasEndpointHandlers() {
		return
	}
	buffer.WriteString("\ntype goldrHandlers struct {\n")
	for _, handler := range helpers.endpointHandlers {
		fmt.Fprintf(buffer, "\t%s http.Handler\n", handler.name)
	}
	buffer.WriteString("}\n")
	buffer.WriteString("\nfunc goldrNewHandlers(options HandlerOptions) *goldrHandlers {\n")
	buffer.WriteString("\treturn &goldrHandlers{\n")
	for _, handler := range helpers.endpointHandlers {
		middlewareStack := helpers.middlewareStackName(routeMiddlewares(handler.route))
		fmt.Fprintf(buffer, "\t\t%s: %s(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {\n", handler.name, middlewareStack)
		writeRouteEndpointBody(buffer, handler.route, helpers, "\t\t\t")
		buffer.WriteString("\t\t})),\n")
	}
	buffer.WriteString("\t}\n")
	buffer.WriteString("}\n")
}

func writeRouteEndpointBody(buffer *bytes.Buffer, route runtimeRoute, helpers handlerHelperPlan, indent string) {
	if route.action != nil {
		writeActionRoute(buffer, route, helpers, indent)
		return
	}
	writeRenderRoute(buffer, route, helpers, indent)
}

func writeDispatchNodes(buffer *bytes.Buffer, root *dispatchNode, helpers handlerHelperPlan) {
	writeDispatchNode(buffer, root, helpers)
	for _, child := range sortedStaticChildren(root) {
		writeDispatchNodes(buffer, child.node, helpers)
	}
	if root.dynamicChild != nil {
		writeDispatchNodes(buffer, root.dynamicChild, helpers)
	}
}

func writeDispatchNode(buffer *bytes.Buffer, node *dispatchNode, helpers handlerHelperPlan) {
	if helpers.hasEndpointHandlers() {
		fmt.Fprintf(buffer, "\nfunc %s(options HandlerOptions, handlers *goldrHandlers, w http.ResponseWriter, r *http.Request, segments []string) {\n", node.name)
	} else {
		fmt.Fprintf(buffer, "\nfunc %s(options HandlerOptions, w http.ResponseWriter, r *http.Request, segments []string) {\n", node.name)
	}
	if node.path == nil {
		fmt.Fprintf(buffer, "\tif len(segments) <= %d {\n", node.depth)
		buffer.WriteString("\t\tgoldrRouteNotFound(options, w, r)\n")
		buffer.WriteString("\t\treturn\n")
		buffer.WriteString("\t}\n")
	}
	if node.path != nil {
		fmt.Fprintf(buffer, "\tif len(segments) == %d {\n", node.depth)
		writePathDispatch(buffer, *node.path, helpers, "\t\t")
		buffer.WriteString("\t}\n")
	}

	children := sortedStaticChildren(node)
	if len(children) > 0 {
		fmt.Fprintf(buffer, "\tswitch segments[%d] {\n", node.depth)
		for _, child := range children {
			fmt.Fprintf(buffer, "\tcase %s:\n", strconv.Quote(child.segment))
			if helpers.hasEndpointHandlers() {
				fmt.Fprintf(buffer, "\t\t%s(options, handlers, w, r, segments)\n", child.node.name)
			} else {
				fmt.Fprintf(buffer, "\t\t%s(options, w, r, segments)\n", child.node.name)
			}
			buffer.WriteString("\t\treturn\n")
		}
		buffer.WriteString("\t}\n")
	}
	if node.dynamicChild != nil {
		fmt.Fprintf(buffer, "\tif segments[%d] != \"\" {\n", node.depth)
		if helpers.hasEndpointHandlers() {
			fmt.Fprintf(buffer, "\t\t%s(options, handlers, w, r, segments)\n", node.dynamicChild.name)
		} else {
			fmt.Fprintf(buffer, "\t\t%s(options, w, r, segments)\n", node.dynamicChild.name)
		}
		buffer.WriteString("\t\treturn\n")
		buffer.WriteString("\t}\n")
	}
	buffer.WriteString("\tgoldrRouteNotFound(options, w, r)\n")
	buffer.WriteString("}\n")
}

func writePathDispatch(buffer *bytes.Buffer, routePath runtimePath, helpers handlerHelperPlan, indent string) {
	if len(routePath.params) > 0 {
		for index, name := range routePath.params {
			fmt.Fprintf(buffer, "%sgoldrParam%d, ok := goldrPathParam(segments[%d])\n", indent, index, paramSegmentIndex(routePath.segments, name, index))
			fmt.Fprintf(buffer, "%sif !ok {\n", indent)
			fmt.Fprintf(buffer, "%s\tgoldrRouteNotFound(options, w, r)\n", indent)
			fmt.Fprintf(buffer, "%s\treturn\n", indent)
			fmt.Fprintf(buffer, "%s}\n", indent)
		}
		for index, name := range routePath.params {
			fmt.Fprintf(buffer, "%sr.SetPathValue(%s, goldrParam%d)\n", indent, strconv.Quote(name), index)
		}
	}

	for _, route := range routePath.routes {
		writeMethodDispatch(buffer, route, helpers, indent)
	}
	fmt.Fprintf(buffer, "%sw.Header().Set(\"Allow\", %s)\n", indent, strconv.Quote(allowHeader(routePath.routes)))
	fmt.Fprintf(buffer, "%sgoldrRouteMethodNotAllowed(options, w, r)\n", indent)
	fmt.Fprintf(buffer, "%sreturn\n", indent)
}

func writeMethodDispatch(buffer *bytes.Buffer, route runtimeRoute, helpers handlerHelperPlan, indent string) {
	if route.page != nil {
		fmt.Fprintf(buffer, "%sif r.Method == http.MethodGet || r.Method == http.MethodHead {\n", indent)
		writeEndpointDispatch(buffer, route, helpers, indent+"\t", writeRenderRoute)
		fmt.Fprintf(buffer, "%s}\n", indent)
		return
	}
	if route.fragment != nil {
		fmt.Fprintf(buffer, "%sif r.Method == http.MethodGet || r.Method == http.MethodHead {\n", indent)
		writeEndpointDispatch(buffer, route, helpers, indent+"\t", writeRenderRoute)
		fmt.Fprintf(buffer, "%s}\n", indent)
		return
	}

	methodConst := httpMethodConst(route.action.action.Method)
	fmt.Fprintf(buffer, "%sif r.Method == %s {\n", indent, methodConst)
	writeEndpointDispatch(buffer, route, helpers, indent+"\t", writeActionRoute)
	fmt.Fprintf(buffer, "%s}\n", indent)
}

func writeEndpointDispatch(buffer *bytes.Buffer, route runtimeRoute, helpers handlerHelperPlan, indent string, writeEndpoint func(*bytes.Buffer, runtimeRoute, handlerHelperPlan, string)) {
	writeRequestNavAssignment(buffer, route, indent)
	middlewares := routeMiddlewares(route)
	if len(middlewares) == 0 {
		writeEndpoint(buffer, route, helpers, indent)
		return
	}

	fmt.Fprintf(buffer, "%shandlers.%s.ServeHTTP(w, r)\n", indent, helpers.endpointHandlerName(route))
	fmt.Fprintf(buffer, "%sreturn\n", indent)
}

func writeActionRoute(buffer *bytes.Buffer, route runtimeRoute, helpers handlerHelperPlan, indent string) {
	actionCall := routeFunc(route.action.action.GoFile, route.action.action.Function)
	writeActionCallComment(buffer, indent, route.action.action)
	writeRoutePageRendererAssignment(buffer, route.action.layouts, helpers, indent)
	if route.action.action.Writer {
		if route.action.action.AdapterReturnsError {
			fmt.Fprintf(buffer, "%sif err := %s(w, r); err != nil {\n", indent, actionCall)
			fmt.Fprintf(buffer, "%s\tgoldrRouteError(options, w, r, err, %s)\n", indent, helpers.routePageRendererName(route.action.layouts))
			fmt.Fprintf(buffer, "%s\treturn\n", indent)
			fmt.Fprintf(buffer, "%s}\n", indent)
		} else {
			fmt.Fprintf(buffer, "%s%s(w, r)\n", indent, actionCall)
		}
		fmt.Fprintf(buffer, "%sreturn\n", indent)
		return
	}
	fmt.Fprintf(buffer, "%srouteResponse := %s(r)\n", indent, actionCall)
	fmt.Fprintf(buffer, "%sgoldrWriteEndpointResponse(options, w, r, routeResponse, %s)\n", indent, helpers.routePageRendererName(route.action.layouts))
	fmt.Fprintf(buffer, "%sreturn\n", indent)
}

func writeRenderRoute(buffer *bytes.Buffer, route runtimeRoute, helpers handlerHelperPlan, indent string) {
	if route.page != nil {
		pageCall := routeFunc(route.page.page.Unit.GoFile, pageFuncName(route.page.page))
		writePageCallComment(buffer, indent, route.page.page)
		writeRoutePageRendererAssignment(buffer, route.page.layouts, helpers, indent)
		fmt.Fprintf(buffer, "%srouteResponse := %s(r)\n", indent, pageCall)
		fmt.Fprintf(buffer, "%sgoldrWritePageEndpointResponse(options, w, r, routeResponse, %s, %s, %s)\n", indent, templateMarker("page", route.page.page.Route, route.page.page.Unit), helpers.layoutStackName(route.page.layouts), helpers.routePageRendererName(route.page.layouts))
	} else {
		fragmentCall := routeFunc(route.fragment.fragment.Unit.GoFile, manifestFragmentFuncName(route.fragment.fragment))
		writeFragmentCallComment(buffer, indent, *route.fragment)
		writeRoutePageRendererAssignment(buffer, route.fragment.layouts, helpers, indent)
		fmt.Fprintf(buffer, "%srouteResponse := %s(r)\n", indent, fragmentCall)
		fmt.Fprintf(buffer, "%srouteResponse = goldrWrapFragmentRouteResponse(routeResponse, %s)\n", indent, templateMarker("fragment", route.fragment.route, route.fragment.fragment.Unit))
		fmt.Fprintf(buffer, "%sgoldrWriteFragmentEndpointResponse(options, w, r, routeResponse, %s)\n", indent, helpers.routePageRendererName(route.fragment.layouts))
	}
	fmt.Fprintf(buffer, "%sreturn\n", indent)
}

func writeRequestNavAssignment(buffer *bytes.Buffer, route runtimeRoute, indent string) {
	if len(route.navTrail) == 0 && len(route.trailKeys) == 0 {
		return
	}
	trailKey := strconv.Quote("")
	if len(route.trailKeys) > 0 {
		trailKey = "goldrRequestTrailKey(r, []string{"
		for index, key := range route.trailKeys {
			if index > 0 {
				trailKey += ", "
			}
			trailKey += strconv.Quote(key)
		}
		trailKey += "})"
	}
	fmt.Fprintf(buffer, "%sr = goldr.WithRequestNav(r, %s, ", indent, trailKey)
	writeRouteNavLiteral(buffer, route.navTrail)
	buffer.WriteString(", ")
	writeRouteNavHrefsLiteral(buffer, route.navTrail)
	fmt.Fprintf(buffer, ", %d)\n", currentRouteNavIndex(route.navTrail))
}

func writeRouteNavLiteral(buffer *bytes.Buffer, steps []runtimeNavStep) {
	if len(steps) == 0 {
		buffer.WriteString("nil")
		return
	}
	buffer.WriteString("[]goldr.RouteNav{")
	for _, step := range steps {
		buffer.WriteString("{")
		if step.nav.Label != "" {
			fmt.Fprintf(buffer, "Label: %s, ", strconv.Quote(step.nav.Label))
		}
		if step.nav.Key != "" {
			fmt.Fprintf(buffer, "Key: %s, ", strconv.Quote(step.nav.Key))
		}
		buffer.WriteString("}, ")
	}
	buffer.WriteString("}")
}

func writeRouteNavHrefsLiteral(buffer *bytes.Buffer, steps []runtimeNavStep) {
	if len(steps) == 0 {
		buffer.WriteString("nil")
		return
	}
	buffer.WriteString("[]string{")
	for _, step := range steps {
		buffer.WriteString(routeNavHrefExpression(step))
		buffer.WriteString(", ")
	}
	buffer.WriteString("}")
}

func currentRouteNavIndex(steps []runtimeNavStep) int {
	for index, step := range steps {
		if step.current {
			return index
		}
	}
	return -1
}

func routeNavHrefExpression(step runtimeNavStep) string {
	if step.route == "/" {
		return `goldrNavHref(options.BasePath)`
	}
	var parts []string
	for _, segment := range routeSegments(step.route) {
		if paramName, ok := paramSegmentName(segment); ok {
			if !slices.Contains(step.params, paramName) {
				return strconv.Quote("")
			}
			parts = append(parts, "url.PathEscape(r.PathValue("+strconv.Quote(paramName)+"))")
			continue
		}
		parts = append(parts, strconv.Quote(segment))
	}
	return "goldrNavHref(options.BasePath, " + strings.Join(parts, ", ") + ")"
}

func writeRootErrorRoutePageRenderer(buffer *bytes.Buffer, layouts []routing.ManifestLayout, helpers handlerHelperPlan) {
	buffer.WriteString(`
func goldrRootErrorRoutePageRenderer(r *http.Request, page goldr.Page) (templ.Component, error) {
`)
	if len(layouts) == 0 {
		buffer.WriteString(`	return goldrDirectRoutePageRenderer(r, page)
`)
	} else {
		fmt.Fprintf(buffer, "\treturn %s(r, page)\n", helpers.layoutRendererName(layouts))
	}
	buffer.WriteString(`}
`)
}

func writeRoutePageRendererAssignment(buffer *bytes.Buffer, layouts []routing.ManifestLayout, helpers handlerHelperPlan, indent string) {
	if len(layouts) == 0 {
		return
	}
	fmt.Fprintf(buffer, "%sr = goldr.WithRoutePageRenderer(r, %s)\n", indent, helpers.layoutRendererName(layouts))
}

func writePageCallComment(buffer *bytes.Buffer, indent string, page routing.ManifestPage) {
	writeExpectedFileComment(buffer, indent, page.Unit.GoFile)
}

func writeFragmentCallComment(buffer *bytes.Buffer, indent string, fragment runtimeFragment) {
	writeExpectedFileComment(buffer, indent, fragment.fragment.Unit.GoFile)
}

func writeLayoutCallComment(buffer *bytes.Buffer, indent string, layout routing.ManifestLayout) {
	writeExpectedFileComment(buffer, indent, layout.Unit.GoFile)
}

func writeActionCallComment(buffer *bytes.Buffer, indent string, action routing.ManifestAction) {
	writeExpectedFileComment(buffer, indent, action.GoFile)
}

func writeMiddlewareCallComment(buffer *bytes.Buffer, indent string, middleware routing.ManifestMiddleware) {
	writeExpectedFileComment(buffer, indent, middleware.GoFile)
}

func writeExpectedFileComment(buffer *bytes.Buffer, indent, goFile string) {
	fmt.Fprintf(buffer, "%s// expected in file: %s\n", indent, appRouteGoFile(goFile))
}

func appRouteGoFile(goFile string) string {
	return path.Join("app/routes", goFile)
}

func templateMarker(kind string, route string, unit routing.RenderUnit) string {
	source := unit.TemplFile
	if source == "" {
		source = renderUnitSourceGoFile(unit)
	}
	id := templateMarkerID(kind, source)
	return fmt.Sprintf(
		"goldrinspect.NewMarker(%s, %s, %s, %s, %s)",
		strconv.Quote(id),
		strconv.Quote(kind),
		strconv.Quote(route),
		strconv.Quote(appRouteGoFile(source)),
		strconv.Quote(appRouteGoFile(unit.GoFile)),
	)
}

func templateMarkerID(kind string, source string) string {
	var builder strings.Builder
	builder.WriteString("g_")
	builder.WriteString(kind)
	for _, char := range source {
		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteRune(char)
		case char >= 'A' && char <= 'Z':
			builder.WriteRune(char)
		case char >= '0' && char <= '9':
			builder.WriteRune(char)
		default:
			builder.WriteByte('_')
		}
	}
	return builder.String()
}

func paramSegmentIndex(segments []string, name string, fallback int) int {
	for index, segment := range segments {
		if segment == "{"+name+"}" {
			return index
		}
	}
	return fallback
}

func routeFunc(goFile, name string) string {
	dir := path.Dir(goFile)
	if dir == "." {
		return name
	}
	return routeImportAlias(dir) + "." + name
}

func pageFuncName(page routing.ManifestPage) string {
	if page.Function != "" {
		return page.Function
	}
	return "Page"
}

func manifestFragmentFuncName(fragment routing.ManifestFragment) string {
	if fragment.Function != "" {
		return fragment.Function
	}
	return fragmentFuncName(fragment.Name)
}
