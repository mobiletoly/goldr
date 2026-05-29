// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

type requestNavContextKey struct{}

const (
	navTrailKeyQuery     = "_goldr_nav_trail_key"
	navReturnToQuery     = "_goldr_return_to"
	navReturnToMaxLength = 2048
)

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

// Navigation is resolved request navigation state for templates.
type Navigation struct {
	Trail   NavTrail
	Back    NavigationBack
	Current NavigationCurrent

	returnTo string
}

// NavigationBack is the app-level Back link for a resolved navigation value.
type NavigationBack struct {
	Href  string
	Label string
	OK    bool
}

// NavigationCurrent is the current step for a resolved navigation value.
type NavigationCurrent struct {
	Href  string
	Label string
	OK    bool
}

type requestNavState struct {
	trailKey         string
	currentReturnTo  string
	incomingReturnTo string
	steps            []requestNavStep
	resolutions      map[string]requestNavResolution
}

type requestNavStep struct {
	label   string
	key     string
	href    string
	current bool
}

type requestNavResolution struct {
	label string
	href  string
}

// RequestNav exposes generated route navigation state for one request.
type RequestNav struct {
	state *requestNavState
}

// WithRequestNav returns a request carrying generated route navigation state.
// Generated Goldr dispatch calls this before route handlers.
func WithRequestNav(r *http.Request, trailKey string, nav []RouteNav, hrefs []string, currentIndex int) *http.Request {
	if r == nil {
		return r
	}
	steps := make([]requestNavStep, 0, len(nav))
	for index, item := range nav {
		href := ""
		if index < len(hrefs) {
			href = hrefs[index]
		}
		steps = append(steps, requestNavStep{
			label:   item.Label,
			key:     item.Key,
			href:    href,
			current: index == currentIndex,
		})
	}
	state := &requestNavState{
		trailKey:         trailKey,
		currentReturnTo:  currentNavigationReturnTo(r),
		incomingReturnTo: incomingNavigationReturnTo(r, trailKey),
		steps:            steps,
	}
	return r.WithContext(context.WithValue(r.Context(), requestNavContextKey{}, state))
}

// Nav returns generated route navigation state for a request.
func Nav(r *http.Request) RequestNav {
	if r == nil {
		return RequestNav{}
	}
	state, _ := r.Context().Value(requestNavContextKey{}).(*requestNavState)
	return RequestNav{state: state}
}

// Trail returns the resolved navigation trail for the matched route.
func (nav RequestNav) Trail() NavTrail {
	if nav.state == nil {
		return nil
	}
	trail := make(NavTrail, 0, len(nav.state.steps))
	for _, step := range nav.state.steps {
		label := step.label
		href := step.href
		if step.key != "" {
			resolution, ok := nav.state.resolutions[step.key]
			if !ok || strings.TrimSpace(resolution.label) == "" {
				continue
			}
			label = resolution.label
			if resolution.href != "" {
				href = resolution.href
			}
		}
		if strings.TrimSpace(label) == "" {
			continue
		}
		if step.current {
			trail = append(trail, CurrentNavStep(label))
			continue
		}
		if strings.TrimSpace(href) == "" {
			continue
		}
		trail = append(trail, NavStep(label, href))
	}
	if len(trail) == 0 {
		return nil
	}
	return trail
}

// TrailKey returns the validated alternate trail key for the matched route.
func (nav RequestNav) TrailKey() string {
	if nav.state == nil {
		return ""
	}
	return nav.state.trailKey
}

// Navigation returns resolved request navigation state for templates.
func (nav RequestNav) Navigation() Navigation {
	return nav.NavigationWithTrail(nav.Trail())
}

// NavigationWithTrail returns request navigation state for a custom trail.
func (nav RequestNav) NavigationWithTrail(trail NavTrail) Navigation {
	back := backNavigationStep(trail)
	if nav.state == nil {
		return Navigation{
			Trail:   trail,
			Back:    back,
			Current: currentNavigationStep(trail),
		}
	}

	if back.OK && nav.state.trailKey != "" && nav.state.incomingReturnTo != "" {
		back.Href = nav.state.incomingReturnTo
	}

	return Navigation{
		Trail:    trail,
		Back:     back,
		Current:  currentNavigationStep(trail),
		returnTo: nav.state.currentReturnTo,
	}
}

// Resolve sets the label for a dynamic navigation step.
func (nav RequestNav) Resolve(key string, label string) {
	nav.ResolveHref(key, label, "")
}

// ResolveHref sets the label and href for a dynamic navigation step.
func (nav RequestNav) ResolveHref(key string, label string, href string) {
	if nav.state == nil || key == "" || strings.TrimSpace(label) == "" || !nav.hasKey(key) {
		return
	}
	if nav.state.resolutions == nil {
		nav.state.resolutions = make(map[string]requestNavResolution)
	}
	resolution := nav.state.resolutions[key]
	resolution.label = label
	resolution.href = href
	nav.state.resolutions[key] = resolution
}

func (nav RequestNav) hasKey(key string) bool {
	for _, step := range nav.state.steps {
		if step.key == key {
			return true
		}
	}
	return false
}

// NavigationHref returns a destination href with selected navigation state.
func NavigationHref(path string, trail string, nav Navigation) string {
	return navigationURL(path, trail, nav.returnTo)
}

func backNavigationStep(trail NavTrail) NavigationBack {
	for index := len(trail) - 1; index >= 0; index-- {
		step := trail[index]
		if step.Current {
			continue
		}
		if strings.TrimSpace(step.Href) != "" {
			return NavigationBack{
				Href:  step.Href,
				Label: step.Label,
				OK:    true,
			}
		}
	}
	return NavigationBack{}
}

func currentNavigationStep(trail NavTrail) NavigationCurrent {
	for _, step := range trail {
		if step.Current {
			return NavigationCurrent{
				Href:  step.Href,
				Label: step.Label,
				OK:    strings.TrimSpace(step.Label) != "",
			}
		}
	}
	return NavigationCurrent{}
}

func currentNavigationReturnTo(r *http.Request) string {
	if r == nil || r.URL == nil || r.Method != http.MethodGet {
		return ""
	}
	path := r.URL.EscapedPath()
	if path == "" {
		path = "/"
	}
	query := r.URL.Query()
	query.Del(navReturnToQuery)
	return cleanNavigationReturnTo(path, query)
}

func incomingNavigationReturnTo(r *http.Request, trailKey string) string {
	if r == nil || r.URL == nil || trailKey == "" {
		return ""
	}
	return sanitizeNavigationReturnTo(r.URL.Query().Get(navReturnToQuery))
}

func sanitizeNavigationReturnTo(raw string) string {
	if raw == "" || len(raw) > navReturnToMaxLength || strings.HasPrefix(raw, "//") {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Opaque != "" {
		return ""
	}
	path := parsed.EscapedPath()
	if path == "" || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return ""
	}
	query := parsed.Query()
	query.Del(navReturnToQuery)
	return cleanNavigationReturnTo(path, query)
}

func cleanNavigationReturnTo(path string, query url.Values) string {
	if path == "" || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return ""
	}
	encoded := query.Encode()
	value := path
	if encoded != "" {
		value += "?" + encoded
	}
	if len(value) > navReturnToMaxLength {
		return ""
	}
	return value
}

func navigationURL(path string, trail string, returnTo string) string {
	rawPath, rawQuery, hasQuery := strings.Cut(path, "?")
	query := url.Values{}
	if hasQuery {
		parsedQuery, err := url.ParseQuery(rawQuery)
		if err == nil {
			for key, parsedValues := range parsedQuery {
				if key == navTrailKeyQuery || key == navReturnToQuery {
					continue
				}
				for _, value := range parsedValues {
					query.Add(key, value)
				}
			}
		}
	}
	if trail != "" {
		query.Set(navTrailKeyQuery, trail)
		if returnTo != "" {
			query.Set(navReturnToQuery, returnTo)
		}
	}
	encodedQuery := query.Encode()
	if encodedQuery == "" {
		return rawPath
	}
	return rawPath + "?" + encodedQuery
}
