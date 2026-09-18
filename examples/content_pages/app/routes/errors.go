// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func RouteNotFound(r *http.Request) goldr.RouteResponse {
	return goldr.NewPage(
		NotFoundView(r.URL.EscapedPath()),
		goldr.PageMetadata{Title: "Page not found - Goldr Content Pages"},
	).WithStatus(http.StatusNotFound)
}

func InternalError() goldr.RouteResponse {
	return goldr.NewPage(
		InternalErrorView(),
		goldr.PageMetadata{Title: "Content error - Goldr Content Pages"},
	).WithStatus(http.StatusInternalServerError)
}
