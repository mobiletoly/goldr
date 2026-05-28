// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package csrf

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)

func TestNewRejectsWeakSecret(t *testing.T) {
	if _, err := New(Config{Secret: []byte("short")}); !errors.Is(err, ErrWeakSecret) {
		t.Fatalf("New() error = %v, want ErrWeakSecret", err)
	}
}

func TestNewAppliesDefaults(t *testing.T) {
	guard := newTestGuard(t)

	if guard.cookieName != DefaultCookieName {
		t.Fatalf("cookieName = %q, want %q", guard.cookieName, DefaultCookieName)
	}
	if guard.cookiePath != "/" {
		t.Fatalf("cookiePath = %q, want /", guard.cookiePath)
	}
	if guard.maxAge != 12*time.Hour {
		t.Fatalf("maxAge = %s, want 12h", guard.maxAge)
	}
	if guard.sameSite != http.SameSiteLaxMode {
		t.Fatalf("sameSite = %v, want Lax", guard.sameSite)
	}
}

func TestHeaderNameUsesCanonicalPublicSpelling(t *testing.T) {
	if HeaderName != "X-CSRF-Token" {
		t.Fatalf("HeaderName = %q, want X-CSRF-Token", HeaderName)
	}
}

func TestTokenMiddlewareIssuesTokenCookieAndStoresToken(t *testing.T) {
	guard := newTestGuard(t)
	var requestToken string
	handler := guard.TokenMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		requestToken = Token(r)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil))

	if requestToken == "" {
		t.Fatal("Token() = empty, want issued token")
	}
	cookie := findCookie(t, recorder.Result().Cookies(), DefaultCookieName)
	if cookie.Value != requestToken {
		t.Fatalf("cookie token = %q, want request token %q", cookie.Value, requestToken)
	}
	if !cookie.HttpOnly {
		t.Fatal("cookie HttpOnly = false, want true")
	}
	if cookie.Path != "/" {
		t.Fatalf("cookie Path = %q, want /", cookie.Path)
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie SameSite = %v, want Lax", cookie.SameSite)
	}
}

func TestTokenMiddlewareReusesValidCookie(t *testing.T) {
	guard := newTestGuard(t)
	token := newToken(t, guard, fixedNow)
	var requestToken string
	handler := guard.TokenMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		requestToken = Token(r)
	}))
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: DefaultCookieName, Value: token})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if requestToken != token {
		t.Fatalf("Token() = %q, want existing token %q", requestToken, token)
	}
	if len(recorder.Result().Cookies()) != 0 {
		t.Fatalf("cookies = %#v, want no replacement cookie", recorder.Result().Cookies())
	}
}

func TestTokenReturnsEmptyForNilOrMissingRequestToken(t *testing.T) {
	if token := Token(nil); token != "" {
		t.Fatalf("Token(nil) = %q, want empty", token)
	}
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	if token := Token(request); token != "" {
		t.Fatalf("Token(request) = %q, want empty", token)
	}
}

func TestValidateSkipsSafeMethods(t *testing.T) {
	guard := newTestGuard(t)
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace} {
		t.Run(method, func(t *testing.T) {
			request := httptest.NewRequestWithContext(context.Background(), method, "/", nil)
			if err := guard.Validate(request, ""); err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
			}
		})
	}
}

func TestValidateAcceptsFormToken(t *testing.T) {
	guard := newTestGuard(t)
	token := newToken(t, guard, fixedNow)
	request := unsafeRequest(token)

	if err := guard.Validate(request, token); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestValidateAcceptsHeaderTokenBeforeFormToken(t *testing.T) {
	guard := newTestGuard(t)
	token := newToken(t, guard, fixedNow)
	request := unsafeRequest(token)
	request.Header.Set("X-Csrf-Token", token)

	if err := guard.Validate(request, "bad-form-token"); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestValidateAcceptsCaseInsensitiveHeaderName(t *testing.T) {
	guard := newTestGuard(t)
	token := newToken(t, guard, fixedNow)
	request := unsafeRequest(token)
	request.Header.Set("x-csrf-token", token) //nolint:canonicalheader // covers case-insensitive input spelling.

	if err := guard.Validate(request, "bad-form-token"); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestValidateRejectsInvalidUnsafeRequests(t *testing.T) {
	guard := newTestGuard(t)
	validToken := newToken(t, guard, fixedNow)
	otherToken := newToken(t, guard, fixedNow.Add(time.Second))
	expiredToken := newToken(t, guard, fixedNow.Add(-13*time.Hour))
	badSignature := replaceSignature(validToken)
	shortNonce := signedToken(t, guard, "v1."+strconv.FormatInt(fixedNow.Unix(), 10)+".AA")

	tests := []struct {
		name      string
		request   *http.Request
		formToken string
		want      error
	}{
		{
			name:      "nil request",
			request:   nil,
			formToken: validToken,
			want:      ErrNilRequest,
		},
		{
			name:      "missing submitted token",
			request:   unsafeRequest(validToken),
			formToken: "",
			want:      ErrMissingToken,
		},
		{
			name:      "missing cookie",
			request:   httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil),
			formToken: validToken,
			want:      ErrMissingCookie,
		},
		{
			name:      "mismatched token",
			request:   unsafeRequest(validToken),
			formToken: otherToken,
			want:      ErrBadToken,
		},
		{
			name:      "malformed token",
			request:   unsafeRequest("malformed"),
			formToken: "malformed",
			want:      ErrMalformedToken,
		},
		{
			name:      "short nonce",
			request:   unsafeRequest(shortNonce),
			formToken: shortNonce,
			want:      ErrMalformedToken,
		},
		{
			name:      "bad signature",
			request:   unsafeRequest(badSignature),
			formToken: badSignature,
			want:      ErrBadToken,
		},
		{
			name:      "expired token",
			request:   unsafeRequest(expiredToken),
			formToken: expiredToken,
			want:      ErrExpiredToken,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := guard.Validate(test.request, test.formToken)
			if !errors.Is(err, test.want) {
				t.Fatalf("Validate() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestInputRendersEscapedHiddenField(t *testing.T) {
	html := renderComponent(t, Input(`tok"en<&`))
	want := `<input type="hidden" name="csrf_token" value="tok&#34;en&lt;&amp;">`
	if html != want {
		t.Fatalf("Input() = %q, want %q", html, want)
	}
}

func TestHeadersReturnsHTMXHeaderJSON(t *testing.T) {
	headers := Headers(`tok"en`)
	var decoded map[string]string
	if err := json.Unmarshal([]byte(headers), &decoded); err != nil {
		t.Fatalf("Headers() JSON error = %v, body = %q", err, headers)
	}
	if decoded[HeaderName] != `tok"en` {
		t.Fatalf("Headers()[%q] = %q, want token", HeaderName, decoded[HeaderName])
	}
}

func TestMetaRendersEscapedTokenMeta(t *testing.T) {
	html := renderComponent(t, Meta(`tok"en<&`))
	want := `<meta name="csrf-token" content="tok&#34;en&lt;&amp;">`
	if html != want {
		t.Fatalf("Meta() = %q, want %q", html, want)
	}
}

func TestConfiguredCookieAttributes(t *testing.T) {
	guard, err := New(Config{
		Secret:     testSecret(),
		CookieName: "app_csrf",
		CookiePath: "/app",
		MaxAge:     time.Hour,
		Secure:     true,
		SameSite:   http.SameSiteStrictMode,
	})
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
	guard.now = func() time.Time { return fixedNow }
	handler := guard.TokenMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/app", nil))

	cookie := findCookie(t, recorder.Result().Cookies(), "app_csrf")
	if cookie.Path != "/app" {
		t.Fatalf("Path = %q, want /app", cookie.Path)
	}
	if cookie.MaxAge != 3600 {
		t.Fatalf("MaxAge = %d, want 3600", cookie.MaxAge)
	}
	if !cookie.Secure {
		t.Fatal("Secure = false, want true")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("SameSite = %v, want Strict", cookie.SameSite)
	}
}

func newTestGuard(t *testing.T) *Guard {
	t.Helper()
	guard, err := New(Config{Secret: testSecret()})
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
	guard.now = func() time.Time { return fixedNow }
	return guard
}

func testSecret() []byte {
	return []byte("0123456789abcdef0123456789abcdef")
}

func newToken(t *testing.T, guard *Guard, now time.Time) string {
	t.Helper()
	token, err := guard.newToken(now)
	if err != nil {
		t.Fatalf("newToken() error = %v, want nil", err)
	}
	return token
}

func signedToken(t *testing.T, guard *Guard, payload string) string {
	t.Helper()
	return payload + "." + base64.RawURLEncoding.EncodeToString(guard.sign(payload))
}

func unsafeRequest(cookieToken string) *http.Request {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
	request.AddCookie(&http.Cookie{Name: DefaultCookieName, Value: cookieToken})
	return request
}

func findCookie(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %q not found in %#v", name, cookies)
	return nil
}

func renderComponent(t *testing.T, component interface {
	Render(context.Context, io.Writer) error
}) string {
	t.Helper()
	var buffer bytes.Buffer
	if err := component.Render(context.Background(), &buffer); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	return buffer.String()
}

func replaceSignature(token string) string {
	index := strings.LastIndex(token, ".")
	if index < 0 {
		return token
	}
	return token[:index+1] + strings.Repeat("A", 43)
}
