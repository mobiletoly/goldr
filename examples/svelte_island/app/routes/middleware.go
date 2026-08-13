package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr/examples/svelte_island/app/security"
)

func Middleware(next http.Handler) http.Handler {
	return security.CSRF.TokenMiddleware(next)
}
