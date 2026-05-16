// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import "github.com/a-h/templ"

// Page is the render result returned by goldr page functions.
type Page struct {
	// Component is the templ component rendered for the page body.
	Component templ.Component
	// Metadata is page-owned metadata passed to matching layouts.
	Metadata PageMetadata
}

// PageMetadata is page-owned metadata passed explicitly to layouts.
type PageMetadata struct {
	// Title is the page title.
	Title string
	// Description is the page description.
	Description string
}

// LayoutContext is the explicit layout-facing context for page rendering.
type LayoutContext struct {
	// Child is the already-composed child layout or page component.
	Child templ.Component
	// Metadata is the page metadata visible to the layout chain.
	Metadata PageMetadata
}
