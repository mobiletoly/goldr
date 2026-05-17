package testcsrf

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mobiletoly/goldr/csrf"
)

func Pair(t *testing.T, guard *csrf.Guard) (*http.Cookie, string) {
	t.Helper()

	recorder := httptest.NewRecorder()
	handler := guard.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(guard.Token(r)))
	}))
	handler.ServeHTTP(recorder, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil))

	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == csrf.DefaultCookieName {
			return cookie, recorder.Body.String()
		}
	}
	t.Fatalf("CSRF cookie %q not found", csrf.DefaultCookieName)
	return nil, ""
}
