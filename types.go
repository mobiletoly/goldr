// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import "github.com/a-h/templ"

// Page is the render result returned by goldr page functions.
type Page struct {
	Component templ.Component
	Metadata  PageMetadata
}

// PageMetadata is page-owned metadata passed explicitly to layouts.
type PageMetadata struct {
	Title       string
	Description string
}

// LayoutContext is the explicit layout-facing context for page rendering.
type LayoutContext struct {
	Child    templ.Component
	Metadata PageMetadata
}
