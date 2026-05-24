package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr/examples/full_feature/app/deps"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deps.From(r).CSRF.TokenMiddleware(next).ServeHTTP(w, r)
	})
}
