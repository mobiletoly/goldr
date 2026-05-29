// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

type navTrailKeyContextKey struct{}

// NavTrail is app-owned navigation presentation state.
type NavTrail []NavTrailStep

// NavTrailStep is one rendered navigation trail item.
type NavTrailStep struct {
	Label   string
	Href    string
	Current bool
}

// NavStep returns a linked navigation trail step.
func NavStep(label string, href string) NavTrailStep {
	return NavTrailStep{Label: label, Href: href}
}

// CurrentNavStep returns the current navigation trail step.
func CurrentNavStep(label string) NavTrailStep {
	return NavTrailStep{Label: label, Current: true}
}

// BackHref returns the nearest previous linked step in a navigation trail.
func BackHref(trail NavTrail) (string, bool) {
	for index := len(trail) - 1; index >= 0; index-- {
		step := trail[index]
		if step.Current {
			continue
		}
		if strings.TrimSpace(step.Href) != "" {
			return step.Href, true
		}
	}
	return "", false
}

// QueryValues copies selected app-owned query values from a request.
//
// Goldr owns _goldr_trail, so that key is never copied.
func QueryValues(r *http.Request, keys ...string) url.Values {
	values := url.Values{}
	if r == nil || r.URL == nil {
		return values
	}
	query := r.URL.Query()
	for _, key := range keys {
		if key == "" || key == "_goldr_trail" {
			continue
		}
		incomingValues, ok := query[key]
		if !ok {
			continue
		}
		values[key] = append([]string(nil), incomingValues...)
	}
	return values
}

// NavTrails declares route-local allowed navigation trail keys.
type NavTrails struct {
	Allowed []string
}

// WithNavTrailKey returns a request carrying an already validated navigation
// trail key. Generated Goldr dispatch calls this before route handlers.
func WithNavTrailKey(r *http.Request, key string) *http.Request {
	if r == nil || key == "" {
		return r
	}
	return r.WithContext(context.WithValue(r.Context(), navTrailKeyContextKey{}, key))
}

// NavTrailKey returns the validated navigation trail key for the matched route.
func NavTrailKey(r *http.Request) string {
	if r == nil {
		return ""
	}
	key, _ := r.Context().Value(navTrailKeyContextKey{}).(string)
	return key
}

// NavTrailSelected reports whether the request selected a validated navigation
// trail key.
func NavTrailSelected(r *http.Request, key string) bool {
	return key != "" && NavTrailKey(r) == key
}
