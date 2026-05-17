// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

// Package goldr provides the small runtime surface for goldr applications.
//
// Goldr is a server-first, HTML-first, HTMX-native web framework for Go. An
// application keeps ownership of its net/http server, mux, middleware, static
// asset serving, auth, sessions, persistence, validation, and deployment.
// Goldr adds filesystem routing under app/routes, generated route wiring, and
// generated URL helpers.
//
// A minimal goldr app has a route tree like this:
//
//	app/routes/
//	    layout.go
//	    layout.templ
//	    page.go
//	    page.templ
//
// Page functions live beside their templ views and return Page:
//
//	func Page(r *http.Request) goldr.Page {
//	    return goldr.Page{
//	        Component: PageView(),
//	        Metadata: goldr.PageMetadata{
//	            Title: "Home",
//	        },
//	    }
//	}
//
// Layout functions accept LayoutContext and return a templ.Component:
//
//	func Layout(r *http.Request, ctx goldr.LayoutContext) templ.Component {
//	    return LayoutView(ctx.Metadata, ctx.Child)
//	}
//
// After route or template edits, run templ generation first, then goldr
// generation and checks:
//
//	go tool templ generate
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
// Fragments are explicit HTMX partials named frag_<name>.go and
// frag_<name>.templ. Actions are ordinary HTTP handlers named by method and
// action, such as PostCreate, colocated with the route they mutate. HTMX
// attributes should stay visible in templ files.
//
// Action handlers that redisplay templ HTML can call Render to buffer the
// component response, handle render errors, set response headers, then write
// the buffered response. Use HTMLResponse.WriteStatus when rendered HTML needs
// a non-200 status.
//
// For server-sent events, applications keep ownership of their stream routes,
// mux registration, subscriber state, and replay policy. The sse package
// provides only event-stream wire helpers for headers, comments, event fields,
// templ-rendered HTML data, and flushing.
//
// For CSRF protection, applications keep ownership of middleware mounting,
// secrets, auth, sessions, templates, and error responses. The csrf package
// provides only signed-cookie token issue and validation helpers for unsafe
// form and HTMX requests.
//
// Static assets are application-owned and should not live under app/routes. For
// production cache safety, write final browser-ready files to assets/build,
// then run:
//
//	go tool goldr assets dist
//
// That command copies fingerprinted files to assets/dist and writes
// assets/goldr_assets_gen.go, so templates can reference assets.Path("app.css")
// and the application can serve assets.FS() under /assets/ from its own mux.
//
// For reproducible app-local tooling, add the CLI tools to go.mod:
//
//	go get -tool github.com/mobiletoly/goldr/cmd/goldr@latest
//	go get -tool github.com/a-h/templ/cmd/templ@v0.3.1020
//
// For a complete current walkthrough, see the repository README and docs/user
// documentation.
package goldr
