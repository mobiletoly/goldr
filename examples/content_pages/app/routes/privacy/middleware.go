// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package privacy

import (
	"context"
	"net/http"
)

type requestContextKey struct{}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Privacy-Middleware", "privacy")
		ctx := context.WithValue(r.Context(), requestContextKey{}, "privacy middleware")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requestContext(r *http.Request) string {
	value, _ := r.Context().Value(requestContextKey{}).(string)
	return value
}
