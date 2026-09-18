// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package content

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/mobiletoly/goldr"
)

var errFilesystemRequired = errors.New("content: filesystem is required")

// Config configures content pages.
type Config struct {
	FS fs.FS
}

// Pages resolves validated content entries from a filesystem.
//
// A configured Pages value is safe for concurrent Resolve calls when its
// filesystem supports concurrent reads.
type Pages struct {
	files fs.FS
}

// New validates config.FS and returns content pages ready for resolution.
func New(config Config) (*Pages, error) {
	if config.FS == nil {
		return nil, errFilesystemRequired
	}
	if err := Check(config.FS); err != nil {
		return nil, err
	}
	return &Pages{files: config.FS}, nil
}

// Check validates filesystem, metadata, UTF-8, and size rules for every content
// entry in files without starting a server. It does not validate or sanitize
// authored HTML.
func Check(files fs.FS) error {
	if files == nil {
		return errFilesystemRequired
	}
	return checkDirectory(files, ".", true)
}

// Resolve resolves one GET or HEAD request to a content page.
//
// The boolean is false when the request is not owned by this source. Invalid
// recognized entries and read failures return a terminal goldr.RouteError.
func (p *Pages) Resolve(r *http.Request) (goldr.PageRouteResponse, bool) {
	if p == nil || p.files == nil {
		return goldr.RouteError{Err: errors.New("content: Pages is not configured")}, true
	}
	if r == nil {
		return goldr.RouteError{Err: errors.New("content: request is nil")}, true
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return nil, false
	}

	entryPath, ok := requestEntryPath(r)
	if !ok {
		return nil, false
	}
	entry, found, err := loadRequestedEntry(p.files, entryPath)
	if err != nil {
		return goldr.RouteError{Err: err}, true
	}
	if !found {
		return nil, false
	}

	body, err := renderEntryBody(entry)
	if err != nil {
		return goldr.RouteError{Err: err}, true
	}
	return goldr.NewPage(body, entry.metadata), true
}

func requestEntryPath(r *http.Request) (string, bool) {
	if r.URL == nil {
		return "", false
	}
	path := r.URL.EscapedPath()
	if path == "" || path == "/" || path[0] != '/' || strings.HasSuffix(path, "/") {
		return "", false
	}
	path = strings.TrimPrefix(path, "/")
	for segment := range strings.SplitSeq(path, "/") {
		if !validSegment(segment) {
			return "", false
		}
	}
	if !fs.ValidPath(path) {
		return "", false
	}
	return path, true
}

func validSegment(segment string) bool {
	if segment == "" || segment[0] == '-' || segment[len(segment)-1] == '-' {
		return false
	}
	previousHyphen := false
	for _, char := range segment {
		switch {
		case char >= 'a' && char <= 'z':
			previousHyphen = false
		case char >= '0' && char <= '9':
			previousHyphen = false
		case char == '-' && !previousHyphen:
			previousHyphen = true
		default:
			return false
		}
	}
	return true
}

func contentError(source, rule string, err error) error {
	if err == nil {
		return fmt.Errorf("content %s: %s", source, rule)
	}
	return fmt.Errorf("content %s: %s: %w", source, rule, err)
}
