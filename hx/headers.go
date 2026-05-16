// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

// Package hx provides small helpers for HTMX request and response headers.
package hx

import (
	"net/http"
	"strings"
)

// HeaderBoosted and the other request header constants name HTMX request
// headers using Go's canonical HTTP header casing.
const (
	HeaderBoosted               = "Hx-Boosted"
	HeaderCurrentURL            = "Hx-Current-Url"
	HeaderHistoryRestoreRequest = "Hx-History-Restore-Request"
	HeaderPrompt                = "Hx-Prompt"
	HeaderRequest               = "Hx-Request"
	HeaderTarget                = "Hx-Target"
	HeaderTrigger               = "Hx-Trigger"
	HeaderTriggerName           = "Hx-Trigger-Name"
)

// HeaderLocation and the other response header constants name HTMX response
// headers using Go's canonical HTTP header casing.
const (
	HeaderLocation           = "Hx-Location"
	HeaderPushURL            = "Hx-Push-Url"
	HeaderRedirect           = "Hx-Redirect"
	HeaderRefresh            = "Hx-Refresh"
	HeaderReplaceURL         = "Hx-Replace-Url"
	HeaderReselect           = "Hx-Reselect"
	HeaderRetarget           = "Hx-Retarget"
	HeaderReswap             = "Hx-Reswap"
	HeaderTriggerAfterSettle = "Hx-Trigger-After-Settle"
	HeaderTriggerAfterSwap   = "Hx-Trigger-After-Swap"
)

// IsRequest reports whether the request was made by HTMX.
func IsRequest(r *http.Request) bool {
	return headerTrue(r, HeaderRequest)
}

// IsBoosted reports whether the request came from an hx-boost element.
func IsBoosted(r *http.Request) bool {
	return headerTrue(r, HeaderBoosted)
}

// IsHistoryRestoreRequest reports whether HTMX is restoring history.
func IsHistoryRestoreRequest(r *http.Request) bool {
	return headerTrue(r, HeaderHistoryRestoreRequest)
}

// CurrentURL returns the HTMX current browser URL request header.
func CurrentURL(r *http.Request) string {
	return r.Header.Get(HeaderCurrentURL)
}

// Prompt returns the HTMX prompt request header.
func Prompt(r *http.Request) string {
	return r.Header.Get(HeaderPrompt)
}

// Target returns the HTMX target element id request header.
func Target(r *http.Request) string {
	return r.Header.Get(HeaderTarget)
}

// TriggerID returns the HTMX triggering element id request header.
func TriggerID(r *http.Request) string {
	return r.Header.Get(HeaderTrigger)
}

// TriggerName returns the HTMX triggering element name request header.
func TriggerName(r *http.Request) string {
	return r.Header.Get(HeaderTriggerName)
}

func headerTrue(r *http.Request, header string) bool {
	return r.Header.Get(header) == "true"
}

// Location sets the HTMX location response header.
func Location(w http.ResponseWriter, value string) {
	w.Header().Set(HeaderLocation, value)
}

// PushURL sets the HTMX push-url response header.
func PushURL(w http.ResponseWriter, url string) {
	w.Header().Set(HeaderPushURL, url)
}

// PreventPushURL prevents HTMX from pushing a URL into browser history.
func PreventPushURL(w http.ResponseWriter) {
	w.Header().Set(HeaderPushURL, "false")
}

// Redirect sets the HTMX redirect response header.
func Redirect(w http.ResponseWriter, url string) {
	w.Header().Set(HeaderRedirect, url)
}

// Refresh sets the HTMX refresh response header.
func Refresh(w http.ResponseWriter) {
	w.Header().Set(HeaderRefresh, "true")
}

// ReplaceURL sets the HTMX replace-url response header.
func ReplaceURL(w http.ResponseWriter, url string) {
	w.Header().Set(HeaderReplaceURL, url)
}

// PreventReplaceURL prevents HTMX from replacing the current browser URL.
func PreventReplaceURL(w http.ResponseWriter) {
	w.Header().Set(HeaderReplaceURL, "false")
}

// Reselect sets the HTMX reselect response header.
func Reselect(w http.ResponseWriter, selector string) {
	w.Header().Set(HeaderReselect, selector)
}

// Retarget sets the HTMX retarget response header.
func Retarget(w http.ResponseWriter, selector string) {
	w.Header().Set(HeaderRetarget, selector)
}

// Reswap sets the HTMX reswap response header.
func Reswap(w http.ResponseWriter, swap string) {
	w.Header().Set(HeaderReswap, swap)
}

// Trigger sets the HTMX trigger response header.
func Trigger(w http.ResponseWriter, events ...string) {
	setTrigger(w, HeaderTrigger, events...)
}

// TriggerAfterSettle sets the HTMX trigger-after-settle response header.
func TriggerAfterSettle(w http.ResponseWriter, events ...string) {
	setTrigger(w, HeaderTriggerAfterSettle, events...)
}

// TriggerAfterSwap sets the HTMX trigger-after-swap response header.
func TriggerAfterSwap(w http.ResponseWriter, events ...string) {
	setTrigger(w, HeaderTriggerAfterSwap, events...)
}

func setTrigger(w http.ResponseWriter, header string, events ...string) {
	if len(events) == 0 {
		return
	}
	w.Header().Set(header, strings.Join(events, ", "))
}
