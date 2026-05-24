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

const (
	// SSEEventHelperPath is the embedded named SSE event helper file name.
	SSEEventHelperPath = "goldr-sse-event.js"
	// TemplateInspectorHelperPath is the embedded template inspector helper
	// file name.
	TemplateInspectorHelperPath = "goldr-template-inspector.js"
)

//go:embed goldr-sse-event.js goldr-template-inspector.js
var embedded embed.FS

var helpers = map[string]helperFile{
	SSEEventHelperPath:          mustReadHelper(SSEEventHelperPath),
	TemplateInspectorHelperPath: mustReadHelper(TemplateInspectorHelperPath),
}

type helperFile struct {
	content []byte
	etag    string
}

// FS returns Goldr's embedded browser helper files.
func FS() fs.FS {
	return embedded
}

// Handler returns an HTTP handler for Goldr's browser helper files.
func Handler() http.Handler {
	return http.HandlerFunc(serveHelper)
}

func serveHelper(w http.ResponseWriter, r *http.Request) {
	helperName := strings.TrimPrefix(r.URL.Path, "/")
	helper, ok := helpers[helperName]
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("ETag", helper.etag)

	if etagMatches(r.Header.Get("If-None-Match"), helper.etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	http.ServeContent(w, r, helperName, time.Time{}, bytes.NewReader(helper.content))
}

func mustReadHelper(name string) helperFile {
	content, err := embedded.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return helperFile{
		content: content,
		etag:    contentETag(content),
	}
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
