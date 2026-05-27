// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package wiring

import (
	"bytes"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
)

func writeHandler(buffer *bytes.Buffer, routes []runtimeRoute, rootLayouts []routing.ManifestLayout) {
	paths := runtimePaths(routes)
	root := buildDispatchTree(paths)
	buffer.WriteString(`
func Handler() http.Handler {
	return HandlerWithOptions(HandlerOptions{})
}

func HandlerWithOptions(options HandlerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if options.TemplateInspection != goldr.TemplateInspectionOff {
			r = r.WithContext(goldrinspect.WithMode(r.Context(), options.TemplateInspection))
		}
		routePath := r.URL.EscapedPath()
`)
	if hasRoutesWithoutLayouts(routes) {
		buffer.WriteString(`		r = goldr.WithRoutePageRenderer(r, goldrDirectRoutePageRenderer)
`)
	}
	if hasSegmentRoutes(routes) {
		buffer.WriteString(`		if routePath == "/" {
			goldrDispatchRoot(options, w, r, nil)
			return
		}
		segments := goldrPathSegments(routePath)
		if len(segments) == 0 {
			goldrNotFound(options, w, r)
			return
		}
		goldrDispatchRoot(options, w, r, segments)
	})
}
`)
	} else {
		buffer.WriteString(`		if routePath == "/" {
			goldrDispatchRoot(options, w, r, nil)
			return
		}
		goldrNotFound(options, w, r)
	})
}
`)
	}
	writeDispatchNodes(buffer, root)

	buffer.WriteString(`
func goldrDirectRoutePageRenderer(r *http.Request, page goldr.Page) (templ.Component, error) {
	component := page.Component
	if component == nil {
		return nil, goldr.ErrNilComponent
	}
	return component, nil
}

func goldrNotFound(options HandlerOptions, w http.ResponseWriter, r *http.Request) {
	handlers := options.ErrorHandlers
	if handlers.NotFound != nil {
		goldrWriteErrorRouteResponse(w, r, handlers.NotFound(r), goldrRootErrorRoutePageRenderer)
		return
	}
	http.NotFound(w, r)
}

func goldrMethodNotAllowed(options HandlerOptions, w http.ResponseWriter, r *http.Request) {
	handlers := options.ErrorHandlers
	if handlers.MethodNotAllowed != nil {
		goldrWriteErrorRouteResponse(w, r, handlers.MethodNotAllowed(r), goldrRootErrorRoutePageRenderer)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func goldrInternalServerError(options HandlerOptions, w http.ResponseWriter, r *http.Request, err error) {
	handlers := options.ErrorHandlers
	if handlers.InternalServerError != nil {
		goldrWriteInternalServerErrorRouteResponse(w, r, handlers.InternalServerError(r, err))
		return
	}
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func goldrWriteErrorRouteResponse(w http.ResponseWriter, r *http.Request, response goldr.RouteResponse, render goldr.RoutePageRenderer) {
	if err := goldr.WritePageRouteResponse(w, r, response, render); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func goldrWriteInternalServerErrorRouteResponse(w http.ResponseWriter, r *http.Request, response goldr.RouteResponse) {
	if err := goldr.WriteRouteResponse(w, r, response); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
`)

	writeRootErrorRoutePageRenderer(buffer, rootLayouts)

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

	if hasFragmentRoutes(routes) {
		buffer.WriteString(`
func goldrWrapFragmentRouteResponse(response goldr.RouteResponse, wrap func(templ.Component) templ.Component) goldr.RouteResponse {
	switch response := response.(type) {
	case goldr.Fragment:
		response.Component = wrap(response.Component)
		return response
	case *goldr.Fragment:
		if response == nil {
			return response
		}
		next := *response
		next.Component = wrap(next.Component)
		return next
	default:
		return response
	}
}

`)
	}

}

func writeDispatchNodes(buffer *bytes.Buffer, root *dispatchNode) {
	writeDispatchNode(buffer, root)
	for _, child := range sortedStaticChildren(root) {
		writeDispatchNodes(buffer, child.node)
	}
	if root.dynamicChild != nil {
		writeDispatchNodes(buffer, root.dynamicChild)
	}
}

func writeDispatchNode(buffer *bytes.Buffer, node *dispatchNode) {
	fmt.Fprintf(buffer, "\nfunc %s(options HandlerOptions, w http.ResponseWriter, r *http.Request, segments []string) {\n", node.name)
	if node.depth > 0 {
		fmt.Fprintf(buffer, "\tif len(segments) < %d {\n", node.depth)
		buffer.WriteString("\t\tgoldrNotFound(options, w, r)\n")
		buffer.WriteString("\t\treturn\n")
		buffer.WriteString("\t}\n")
	}
	fmt.Fprintf(buffer, "\tif len(segments) == %d {\n", node.depth)
	if node.path != nil {
		writePathDispatch(buffer, *node.path, "\t\t")
	} else {
		buffer.WriteString("\t\tgoldrNotFound(options, w, r)\n")
		buffer.WriteString("\t\treturn\n")
	}
	buffer.WriteString("\t}\n")

	children := sortedStaticChildren(node)
	if len(children) > 0 {
		fmt.Fprintf(buffer, "\tswitch segments[%d] {\n", node.depth)
		for _, child := range children {
			fmt.Fprintf(buffer, "\tcase %s:\n", strconv.Quote(child.segment))
			fmt.Fprintf(buffer, "\t\t%s(options, w, r, segments)\n", child.node.name)
			buffer.WriteString("\t\treturn\n")
		}
		buffer.WriteString("\t}\n")
	}
	if node.dynamicChild != nil {
		fmt.Fprintf(buffer, "\tif segments[%d] != \"\" {\n", node.depth)
		fmt.Fprintf(buffer, "\t\t%s(options, w, r, segments)\n", node.dynamicChild.name)
		buffer.WriteString("\t\treturn\n")
		buffer.WriteString("\t}\n")
	}
	buffer.WriteString("\tgoldrNotFound(options, w, r)\n")
	buffer.WriteString("}\n")
}

func writePathDispatch(buffer *bytes.Buffer, routePath runtimePath, indent string) {
	if len(routePath.params) > 0 {
		for index, name := range routePath.params {
			fmt.Fprintf(buffer, "%sgoldrParam%d, ok := goldrPathParam(segments[%d])\n", indent, index, paramSegmentIndex(routePath.segments, name, index))
			fmt.Fprintf(buffer, "%sif !ok {\n", indent)
			fmt.Fprintf(buffer, "%s\tgoldrNotFound(options, w, r)\n", indent)
			fmt.Fprintf(buffer, "%s\treturn\n", indent)
			fmt.Fprintf(buffer, "%s}\n", indent)
		}
		for index, name := range routePath.params {
			fmt.Fprintf(buffer, "%sr.SetPathValue(%s, goldrParam%d)\n", indent, strconv.Quote(name), index)
		}
	}

	for _, route := range routePath.routes {
		writeMethodDispatch(buffer, route, indent)
	}
	fmt.Fprintf(buffer, "%sw.Header().Set(\"Allow\", %s)\n", indent, strconv.Quote(allowHeader(routePath.routes)))
	fmt.Fprintf(buffer, "%sgoldrMethodNotAllowed(options, w, r)\n", indent)
	fmt.Fprintf(buffer, "%sreturn\n", indent)
}

func writeMethodDispatch(buffer *bytes.Buffer, route runtimeRoute, indent string) {
	if route.page != nil {
		fmt.Fprintf(buffer, "%sif r.Method == http.MethodGet || r.Method == http.MethodHead {\n", indent)
		writeEndpointDispatch(buffer, route, indent+"\t", writeRenderRoute)
		fmt.Fprintf(buffer, "%s}\n", indent)
		return
	}
	if route.fragment != nil {
		fmt.Fprintf(buffer, "%sif r.Method == http.MethodGet || r.Method == http.MethodHead {\n", indent)
		writeEndpointDispatch(buffer, route, indent+"\t", writeRenderRoute)
		fmt.Fprintf(buffer, "%s}\n", indent)
		return
	}

	methodConst := httpMethodConst(route.action.action.Method)
	fmt.Fprintf(buffer, "%sif r.Method == %s {\n", indent, methodConst)
	writeEndpointDispatch(buffer, route, indent+"\t", writeActionRoute)
	fmt.Fprintf(buffer, "%s}\n", indent)
}

func writeEndpointDispatch(buffer *bytes.Buffer, route runtimeRoute, indent string, writeEndpoint func(*bytes.Buffer, runtimeRoute, string)) {
	middlewares := routeMiddlewares(route)
	if len(middlewares) == 0 {
		writeEndpoint(buffer, route, indent)
		return
	}

	fmt.Fprintf(buffer, "%sgoldrEndpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {\n", indent)
	writeEndpoint(buffer, route, indent+"\t")
	fmt.Fprintf(buffer, "%s})\n", indent)
	fmt.Fprintf(buffer, "%sgoldrHandler := http.Handler(goldrEndpoint)\n", indent)
	for index := len(middlewares) - 1; index >= 0; index-- {
		middlewareCall := routeFunc(middlewares[index].GoFile, "Middleware")
		writeMiddlewareCallComment(buffer, indent, middlewares[index])
		fmt.Fprintf(buffer, "%sgoldrHandler = %s(goldrHandler)\n", indent, middlewareCall)
	}
	fmt.Fprintf(buffer, "%sgoldrHandler.ServeHTTP(w, r)\n", indent)
	fmt.Fprintf(buffer, "%sreturn\n", indent)
}

func writeActionRoute(buffer *bytes.Buffer, route runtimeRoute, indent string) {
	actionCall := routeFunc(route.action.action.GoFile, route.action.action.Function)
	writeActionCallComment(buffer, indent, route.action.action)
	writeRoutePageRendererAssignment(buffer, route.action.layouts, indent, "goldrRoutePageRenderer")
	if route.action.action.Writer {
		fmt.Fprintf(buffer, "%s%s(w, r)\n", indent, actionCall)
		fmt.Fprintf(buffer, "%sreturn\n", indent)
		return
	}
	fmt.Fprintf(buffer, "%srouteResponse := %s(r)\n", indent, actionCall)
	fmt.Fprintf(buffer, "%serr := goldr.WriteRouteResponse(w, r, routeResponse)\n", indent)
	fmt.Fprintf(buffer, "%sif err != nil {\n", indent)
	fmt.Fprintf(buffer, "%s\tgoldrInternalServerError(options, w, r, err)\n", indent)
	fmt.Fprintf(buffer, "%s\treturn\n", indent)
	fmt.Fprintf(buffer, "%s}\n", indent)
	fmt.Fprintf(buffer, "%sreturn\n", indent)
}

func writeRenderRoute(buffer *bytes.Buffer, route runtimeRoute, indent string) {
	if route.page != nil {
		pageCall := routeFunc(route.page.page.Unit.GoFile, pageFuncName(route.page.page))
		writePageCallComment(buffer, indent, route.page.page)
		writeRoutePageRendererAssignment(buffer, route.page.layouts, indent, "goldrErrorRoutePageRenderer")
		fmt.Fprintf(buffer, "%srouteResponse := %s(r)\n", indent, pageCall)
		fmt.Fprintf(buffer, "%serr := goldr.WritePageRouteResponse(w, r, routeResponse, func(r *http.Request, page goldr.Page) (templ.Component, error) {\n", indent)
		fmt.Fprintf(buffer, "%s\tcomponent := page.Component\n", indent)
		fmt.Fprintf(buffer, "%s\tif component == nil {\n", indent)
		fmt.Fprintf(buffer, "%s\t\treturn nil, goldr.ErrNilComponent\n", indent)
		fmt.Fprintf(buffer, "%s\t}\n", indent)
		if len(route.page.layouts) > 0 {
			fmt.Fprintf(buffer, "%s\tmetadata := page.Metadata\n", indent)
		}
		fmt.Fprintf(buffer, "%s\tcomponent = goldrinspect.Wrap(component, %s)\n", indent, templateMarker("page", route.page.page.Route, route.page.page.Unit))
		if len(route.page.layouts) > 0 {
			fmt.Fprintf(buffer, "%s\tlayoutContext := goldr.LayoutContext{Metadata: metadata}\n", indent)
		}
		for index := len(route.page.layouts) - 1; index >= 0; index-- {
			layoutCall := routeFunc(route.page.layouts[index].Unit.GoFile, "Layout")
			fmt.Fprintf(buffer, "%s\tlayoutContext.Child = component\n", indent)
			writeLayoutCallComment(buffer, indent+"\t", route.page.layouts[index])
			fmt.Fprintf(buffer, "%s\tcomponent = %s(r, layoutContext)\n", indent, layoutCall)
			fmt.Fprintf(buffer, "%s\tif component == nil {\n", indent)
			fmt.Fprintf(buffer, "%s\t\treturn nil, goldr.ErrNilComponent\n", indent)
			fmt.Fprintf(buffer, "%s\t}\n", indent)
			fmt.Fprintf(buffer, "%s\tcomponent = goldrinspect.Wrap(component, %s)\n", indent, templateMarker("layout", route.page.layouts[index].RoutePrefix, route.page.layouts[index].Unit))
		}
		fmt.Fprintf(buffer, "%s\treturn component, nil\n", indent)
		fmt.Fprintf(buffer, "%s})\n", indent)
	} else {
		fragmentCall := routeFunc(route.fragment.fragment.Unit.GoFile, manifestFragmentFuncName(route.fragment.fragment))
		writeFragmentCallComment(buffer, indent, *route.fragment)
		writeRoutePageRendererAssignment(buffer, route.fragment.layouts, indent, "goldrRoutePageRenderer")
		fmt.Fprintf(buffer, "%srouteResponse := %s(r)\n", indent, fragmentCall)
		fmt.Fprintf(buffer, "%srouteResponse = goldrWrapFragmentRouteResponse(routeResponse, func(component templ.Component) templ.Component {\n", indent)
		fmt.Fprintf(buffer, "%s\treturn goldrinspect.Wrap(component, %s)\n", indent, templateMarker("fragment", route.fragment.route, route.fragment.fragment.Unit))
		fmt.Fprintf(buffer, "%s})\n", indent)
		fmt.Fprintf(buffer, "%serr := goldr.WriteFragmentRouteResponse(w, r, routeResponse)\n", indent)
	}
	fmt.Fprintf(buffer, "%sif err != nil {\n", indent)
	fmt.Fprintf(buffer, "%s\tgoldrInternalServerError(options, w, r, err)\n", indent)
	fmt.Fprintf(buffer, "%s\treturn\n", indent)
	fmt.Fprintf(buffer, "%s}\n", indent)
	fmt.Fprintf(buffer, "%sreturn\n", indent)
}

func writeRootErrorRoutePageRenderer(buffer *bytes.Buffer, layouts []routing.ManifestLayout) {
	buffer.WriteString(`
func goldrRootErrorRoutePageRenderer(r *http.Request, page goldr.Page) (templ.Component, error) {
`)
	writeRoutePageRendererBody(buffer, layouts, "\t")
	buffer.WriteString(`}
`)
}

func writeRoutePageRendererAssignment(buffer *bytes.Buffer, layouts []routing.ManifestLayout, indent string, name string) {
	if len(layouts) == 0 {
		return
	}
	fmt.Fprintf(buffer, "%s%s := func(r *http.Request, page goldr.Page) (templ.Component, error) {\n", indent, name)
	writeRoutePageRendererBody(buffer, layouts, indent+"\t")
	fmt.Fprintf(buffer, "%s}\n", indent)
	fmt.Fprintf(buffer, "%sr = goldr.WithRoutePageRenderer(r, %s)\n", indent, name)
}

func writeRoutePageRendererBody(buffer *bytes.Buffer, layouts []routing.ManifestLayout, indent string) {
	fmt.Fprintf(buffer, "%s\tcomponent := page.Component\n", indent)
	fmt.Fprintf(buffer, "%s\tif component == nil {\n", indent)
	fmt.Fprintf(buffer, "%s\t\treturn nil, goldr.ErrNilComponent\n", indent)
	fmt.Fprintf(buffer, "%s\t}\n", indent)
	if len(layouts) > 0 {
		fmt.Fprintf(buffer, "%s\tmetadata := page.Metadata\n", indent)
		fmt.Fprintf(buffer, "%s\tlayoutContext := goldr.LayoutContext{Metadata: metadata}\n", indent)
	}
	for index := len(layouts) - 1; index >= 0; index-- {
		layoutCall := routeFunc(layouts[index].Unit.GoFile, "Layout")
		fmt.Fprintf(buffer, "%s\tlayoutContext.Child = component\n", indent)
		writeLayoutCallComment(buffer, indent+"\t", layouts[index])
		fmt.Fprintf(buffer, "%s\tcomponent = %s(r, layoutContext)\n", indent, layoutCall)
		fmt.Fprintf(buffer, "%s\tif component == nil {\n", indent)
		fmt.Fprintf(buffer, "%s\t\treturn nil, goldr.ErrNilComponent\n", indent)
		fmt.Fprintf(buffer, "%s\t}\n", indent)
		fmt.Fprintf(buffer, "%s\tcomponent = goldrinspect.Wrap(component, %s)\n", indent, templateMarker("layout", layouts[index].RoutePrefix, layouts[index].Unit))
	}
	fmt.Fprintf(buffer, "%s\treturn component, nil\n", indent)
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
	start := "<!--goldr:start id=" + templateCommentValue(id) +
		" kind=" + templateCommentValue(kind) +
		" route=" + templateCommentValue(route) +
		" source=" + templateCommentValue(appRouteGoFile(source)) +
		" go=" + templateCommentValue(appRouteGoFile(unit.GoFile)) +
		"-->"
	end := "<!--goldr:end id=" + templateCommentValue(id) + "-->"
	return fmt.Sprintf("goldrinspect.Marker{StartComment: %s, EndComment: %s}", strconv.Quote(start), strconv.Quote(end))
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

func templateCommentValue(value string) string {
	value = strings.ReplaceAll(value, "--", "- -")
	value = strings.ReplaceAll(value, ">", "&gt;")
	value = strings.ReplaceAll(value, " ", "%20")
	return value
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
