// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

// Package goldr provides the small runtime surface for goldr applications.
//
// Goldr is a server-first, HTML-first, HTMX-native web framework for Go. An
// application keeps ownership of its net/http server, mux, middleware, static
// asset serving, auth, sessions, persistence, validation, and deployment.
// Goldr adds filesystem routing under app/routes, mounted non-live Kit route
// subtrees under app/mounts, route-derived navigation trails, generated route
// wiring, and generated URL helpers.
//
// A minimal goldr app has a route tree like this:
//
//	app/routes/
//	    layout.go
//	    layout.templ
//	    route.go
//	    page.templ
//
// Route declarations live in route.go. They declare the page, fragments, and
// actions exposed by that filesystem route directory:
//
//	var Route = goldr.RouteDef{
//	    Page: Page,
//	}
//
// Page functions can live in route.go or ordinary helper files beside their
// templ views. They return PageRouteResponse:
//
//	func Page(r *http.Request) goldr.PageRouteResponse {
//	    return goldr.NewPage(
//	        PageView(),
//	        goldr.PageMetadata{Title: "Home"},
//	    )
//	}
//
// Layout functions accept LayoutContext and return a templ.Component:
//
//	func Layout(r *http.Request, ctx goldr.LayoutContext) templ.Component {
//	    return LayoutView(ctx.Metadata, ctx.Child)
//	}
//
// After route or template edits, run goldr generation and checks. The generate
// command runs templ generation first when .templ files are present:
//
//	go tool goldr generate
//	go tool goldr check
//
// Goldr writes generated route dispatch to app/routes/goldr_gen.go and
// generated URL helpers to app/urls/goldr_gen.go. Applications register the
// generated routes.Handler() with their own mux:
//
//	mux := http.NewServeMux()
//	mux.Handle("/", routes.Handler())
//
// Nested directories define nested routes. A directory named by_<name>
// captures a path value that handlers read with r.PathValue("<name>").
// Fragments are explicit HTMX partials declared in route.go with route-local
// paths such as goldr.FragmentRoute("/table", FragTable), which maps to /table
// under that route directory. goldr.FragmentRoute("/", FragIndex) declares a
// fragment at the route directory path itself. Fragment functions return
// FragmentRouteResponse values and use NewFragment for normal fragment HTML.
// Fragment responses default to
// Cache-Control: no-store unless the application sets Cache-Control itself.
// Actions are mutation endpoints declared in route.go with helpers such as
// goldr.Action(http.MethodPost, "/create", PostCreate) or goldr.Action(http.MethodPost, "/", PostIndex).
// Ordinary action handlers return RouteResponse values.
// HTMX attributes should stay visible in templ files.
//
// Action handlers that redisplay templ HTML can return Fragment responses with
// explicit headers and status. When an action needs to return a full page
// through the matched layout stack, return a Page response. Page, fragment,
// redirect, text, and no-content route responses can carry explicit headers
// with WithHeader and AddHeader.
//
// Routes may declare canonical navigation metadata with RouteDef.Nav. Generated
// dispatch attaches the matched route's canonical navigation plan to the
// request; handlers call Nav(r), resolve dynamic labels with app data, and pass
// the resulting Navigation to templates. Destination helpers can still select
// an explicit alternate TrailKey for shared target workflows.
//
// For server-sent events, applications keep ownership of their stream routes,
// mux registration, subscriber state, and replay policy. The sse package
// provides only event-stream wire helpers for headers, comments, event fields,
// templ-rendered HTML data, and flushing. The browser package provides an
// explicitly mounted helper for swapping selected named SSE events in htmx
// templates. Its file name is browser.SSEEventHelperPath. Goldr does not
// inject that script automatically.
//
// For CSRF protection, applications keep ownership of middleware mounting,
// secrets, auth, sessions, templates, and error responses. The csrf package
// provides only signed-cookie token issue and validation helpers for unsafe
// form and HTMX requests.
//
// Static assets are application-owned and should not live under app/routes. For
// production cache safety, write final browser-ready files to assets/build. The
// normal generate command fingerprints those files when assets/build exists:
//
//	go tool goldr generate
//
// Goldr copies fingerprinted files to assets/dist and writes
// assets/goldr_assets_gen.go, so templates can reference assets.Path("app.css")
// and the application can serve assets.FS() under /assets/ from its own mux.
// The asset-only command remains available as go tool goldr assets dist.
//
// For reproducible app-local tooling, add the CLI tools to go.mod:
//
//	go get -tool github.com/mobiletoly/goldr/cmd/goldr@latest
//	go get -tool github.com/a-h/templ/cmd/templ@v0.3.1020
//
// For a complete current walkthrough, see the repository README and docs/user
// documentation.
package goldr
