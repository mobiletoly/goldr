// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

// Package csrf provides explicit signed-cookie CSRF helpers for Goldr
// applications.
//
// The package owns token generation, cookie signing, and validation helpers.
// Applications still own middleware mounting, secrets, auth, sessions,
// templates, request body limits, and error responses.
package csrf

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
)

const (
	// FieldName is the default form field name for CSRF tokens.
	FieldName = "csrf_token"
	// HeaderName is the request header name for non-form CSRF tokens.
	HeaderName = "X-CSRF-Token"
	// MetaName is the token meta tag name for app-owned JavaScript requests.
	MetaName = "csrf-token"
	// DefaultCookieName is the default cookie name for signed CSRF tokens.
	DefaultCookieName = "goldr_csrf"
)

const (
	defaultCookiePath = "/"
	defaultMaxAge     = 12 * time.Hour
	minSecretBytes    = 32
	tokenVersion      = "v1"
	nonceBytes        = 32
)

var canonicalHeaderName = http.CanonicalHeaderKey(HeaderName)

var (
	// ErrWeakSecret reports a CSRF secret shorter than 32 bytes.
	ErrWeakSecret = errors.New("csrf: secret must be at least 32 bytes")
	// ErrNilRequest reports a nil request passed to CSRF validation.
	ErrNilRequest = errors.New("csrf: nil request")
	// ErrMissingCookie reports a missing CSRF cookie on an unsafe request.
	ErrMissingCookie = errors.New("csrf: missing cookie token")
	// ErrMissingToken reports a missing submitted CSRF token on an unsafe request.
	ErrMissingToken = errors.New("csrf: missing submitted token")
	// ErrMalformedToken reports a token that cannot be parsed.
	ErrMalformedToken = errors.New("csrf: malformed token")
	// ErrBadToken reports a mismatched token or invalid token signature.
	ErrBadToken = errors.New("csrf: bad token")
	// ErrExpiredToken reports a valid signed token older than the configured max age.
	ErrExpiredToken = errors.New("csrf: expired token")
)

// Config configures a CSRF guard.
type Config struct {
	// Secret signs tokens. It must contain at least 32 bytes.
	Secret []byte
	// CookieName is the token cookie name. It defaults to DefaultCookieName.
	CookieName string
	// CookiePath is the token cookie path. It defaults to "/".
	CookiePath string
	// MaxAge is the token lifetime. It defaults to 12 hours.
	MaxAge time.Duration
	// Secure sets the cookie Secure attribute.
	Secure bool
	// SameSite sets the cookie SameSite attribute. It defaults to Lax.
	SameSite http.SameSite
}

// Guard issues and validates CSRF tokens.
type Guard struct {
	secret     []byte
	cookieName string
	cookiePath string
	maxAge     time.Duration
	secure     bool
	sameSite   http.SameSite
	now        func() time.Time
}

type tokenContextKey struct{}

// New creates a CSRF guard.
func New(config Config) (*Guard, error) {
	if len(config.Secret) < minSecretBytes {
		return nil, ErrWeakSecret
	}
	cookieName := config.CookieName
	if cookieName == "" {
		cookieName = DefaultCookieName
	}
	cookiePath := config.CookiePath
	if cookiePath == "" {
		cookiePath = defaultCookiePath
	}
	maxAge := config.MaxAge
	if maxAge == 0 {
		maxAge = defaultMaxAge
	}
	sameSite := config.SameSite
	if sameSite == 0 {
		sameSite = http.SameSiteLaxMode
	}
	return &Guard{
		secret:     append([]byte(nil), config.Secret...),
		cookieName: cookieName,
		cookiePath: cookiePath,
		maxAge:     maxAge,
		secure:     config.Secure,
		sameSite:   sameSite,
		now:        time.Now,
	}, nil
}

// TokenMiddleware makes a CSRF token available for templates and sets the token
// cookie when the request does not already carry a valid token cookie.
func (g *Guard) TokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, issueCookie, err := g.requestToken(r)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if issueCookie {
			http.SetCookie(w, g.cookie(token))
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), tokenContextKey{}, token)))
	})
}

// Token returns the request token stored by TokenMiddleware.
func Token(r *http.Request) string {
	if r == nil {
		return ""
	}
	token, _ := r.Context().Value(tokenContextKey{}).(string)
	return token
}

// Input renders a hidden form field containing token.
func Input(token string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if _, err := io.WriteString(w, "<input"); err != nil {
			return err
		}
		if err := templ.RenderAttributes(ctx, w, templ.OrderedAttributes{
			{Key: "type", Value: "hidden"},
			{Key: "name", Value: FieldName},
			{Key: "value", Value: token},
		}); err != nil {
			return err
		}
		_, err := io.WriteString(w, ">")
		return err
	})
}

// Headers returns an hx-headers JSON value containing token.
func Headers(token string) string {
	value, err := json.Marshal(map[string]string{HeaderName: token})
	if err != nil {
		return "{}"
	}
	return string(value)
}

// Meta renders a meta tag containing token for app-owned JavaScript requests.
func Meta(token string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if _, err := io.WriteString(w, "<meta"); err != nil {
			return err
		}
		if err := templ.RenderAttributes(ctx, w, templ.OrderedAttributes{
			{Key: "name", Value: MetaName},
			{Key: "content", Value: token},
		}); err != nil {
			return err
		}
		_, err := io.WriteString(w, ">")
		return err
	})
}

// Validate validates the CSRF token for an unsafe request.
//
// For unsafe requests, HeaderName takes precedence over formToken.
func (g *Guard) Validate(r *http.Request, formToken string) error {
	if r == nil {
		return ErrNilRequest
	}
	if safeMethod(r.Method) {
		return nil
	}
	submittedToken := r.Header.Get(canonicalHeaderName)
	if submittedToken == "" {
		submittedToken = formToken
	}
	if submittedToken == "" {
		return ErrMissingToken
	}
	cookie, err := r.Cookie(g.cookieName)
	if err != nil || cookie.Value == "" {
		return ErrMissingCookie
	}
	if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(submittedToken)) != 1 {
		return ErrBadToken
	}
	return g.validateToken(cookie.Value, g.now())
}

func (g *Guard) requestToken(r *http.Request) (string, bool, error) {
	if r != nil {
		if cookie, err := r.Cookie(g.cookieName); err == nil && cookie.Value != "" {
			if err := g.validateToken(cookie.Value, g.now()); err == nil {
				return cookie.Value, false, nil
			}
		}
	}
	token, err := g.newToken(g.now())
	if err != nil {
		return "", false, err
	}
	return token, true, nil
}

func (g *Guard) cookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     g.cookieName,
		Value:    token,
		Path:     g.cookiePath,
		MaxAge:   int(g.maxAge.Seconds()),
		Expires:  g.now().Add(g.maxAge),
		HttpOnly: true,
		Secure:   g.secure,
		SameSite: g.sameSite,
	}
}

func (g *Guard) newToken(now time.Time) (string, error) {
	nonce := make([]byte, nonceBytes)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	payload := strings.Join([]string{
		tokenVersion,
		strconv.FormatInt(now.Unix(), 10),
		base64.RawURLEncoding.EncodeToString(nonce),
	}, ".")
	signature := g.sign(payload)
	return payload + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (g *Guard) validateToken(token string, now time.Time) error {
	payload, issuedAt, signature, err := parseToken(token)
	if err != nil {
		return err
	}
	expected := g.sign(payload)
	if subtle.ConstantTimeCompare(signature, expected) != 1 {
		return ErrBadToken
	}
	if now.Sub(issuedAt) > g.maxAge {
		return ErrExpiredToken
	}
	if issuedAt.After(now.Add(time.Minute)) {
		return ErrBadToken
	}
	return nil
}

func (g *Guard) sign(payload string) []byte {
	mac := hmac.New(sha256.New, g.secret)
	_, _ = mac.Write([]byte(payload))
	return mac.Sum(nil)
}

func parseToken(token string) (string, time.Time, []byte, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 4 || parts[0] != tokenVersion {
		return "", time.Time{}, nil, ErrMalformedToken
	}
	issuedUnix, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", time.Time{}, nil, ErrMalformedToken
	}
	nonce, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(nonce) != nonceBytes {
		return "", time.Time{}, nil, ErrMalformedToken
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[3])
	if err != nil || len(signature) != sha256.Size {
		return "", time.Time{}, nil, ErrMalformedToken
	}
	return strings.Join(parts[:3], "."), time.Unix(issuedUnix, 0), signature, nil
}

func safeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}
