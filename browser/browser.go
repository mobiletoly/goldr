// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

// Package browser serves Goldr-maintained browser helper assets.
//
// Applications mount these helpers explicitly from their own mux. Goldr does
// not inject scripts or generate browser helper routes.
package browser

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

const helperPath = "goldr-sse-event.js"

//go:embed goldr-sse-event.js
var embedded embed.FS

var (
	helperContent = mustReadHelper()
	helperETag    = contentETag(helperContent)
)

// FS returns Goldr's embedded browser helper files.
func FS() fs.FS {
	return embedded
}

// Handler returns an HTTP handler for Goldr's browser helper files.
func Handler() http.Handler {
	return http.HandlerFunc(serveHelper)
}

func serveHelper(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/"+helperPath && r.URL.Path != helperPath {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("ETag", helperETag)

	if etagMatches(r.Header.Get("If-None-Match"), helperETag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	http.ServeContent(w, r, helperPath, time.Time{}, bytes.NewReader(helperContent))
}

func mustReadHelper() []byte {
	content, err := embedded.ReadFile(helperPath)
	if err != nil {
		panic(err)
	}
	return content
}

func contentETag(content []byte) string {
	sum := sha256.Sum256(content)
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func etagMatches(header string, etag string) bool {
	for candidate := range strings.SplitSeq(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || candidate == etag || strings.TrimPrefix(candidate, "W/") == etag {
			return true
		}
	}
	return false
}
