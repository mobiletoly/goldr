package deps

import (
	"context"
	"net/http"

	"github.com/mobiletoly/goldr/csrf"
)

type Dependencies struct {
	CSRF *csrf.Guard
}

type contextKey struct{}

func Middleware(value *Dependencies, next http.Handler) http.Handler {
	mustValid(value)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, WithRequest(r, value))
	})
}

func WithRequest(r *http.Request, value *Dependencies) *http.Request {
	mustValid(value)
	return r.WithContext(context.WithValue(r.Context(), contextKey{}, value))
}

func From(r *http.Request) *Dependencies {
	value, ok := r.Context().Value(contextKey{}).(*Dependencies)
	if !ok || value == nil {
		panic("full_feature deps: missing Dependencies; wrap generated routes with deps.Middleware")
	}
	return value
}

func mustValid(value *Dependencies) {
	if value == nil {
		panic("full_feature deps: nil Dependencies")
	}
	if value.CSRF == nil {
		panic("full_feature deps: nil CSRF guard")
	}
}
