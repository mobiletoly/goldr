// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	"context"
	"net/http"
)

type requestContextKey struct{}

func WithOuterContext(r *http.Request, value string) *http.Request {
	ctx := context.WithValue(r.Context(), requestContextKey{}, value)
	return r.WithContext(ctx)
}

func requestContext(r *http.Request) string {
	value, _ := r.Context().Value(requestContextKey{}).(string)
	return value
}
