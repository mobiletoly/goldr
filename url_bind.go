// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import "net/http"

// BindFromRequest binds a generated dynamic route node from the current
// request path values.
func BindFromRequest[T any](r *http.Request, node interface {
	GoldrRouteParams() []string
	Bind(string) T
}) (T, bool) {
	var zero T
	if r == nil {
		return zero, false
	}
	params := node.GoldrRouteParams()
	if len(params) == 0 {
		return zero, false
	}
	value := r.PathValue(params[len(params)-1])
	if value == "" {
		return zero, false
	}
	return node.Bind(value), true
}
