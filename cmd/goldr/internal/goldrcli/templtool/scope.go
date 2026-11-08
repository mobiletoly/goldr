// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package templtool

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const ignoreFileName = ".goldrignore"

// Scope selects template inputs below a resolved application root.
type Scope struct {
	root   string
	ignore *regexp.Regexp
}

// LoadScope reads the application root's optional .goldrignore file.
func LoadScope(root string) (Scope, error) {
	scope := Scope{root: root}
	name := filepath.Join(root, ignoreFileName)
	content, err := os.ReadFile(name)
	if errors.Is(err, os.ErrNotExist) {
		return scope, nil
	}
	if err != nil {
		return Scope{}, fmt.Errorf("read %s: %w", name, err)
	}

	var patterns []string
	line := 0
	for text := range strings.SplitSeq(string(content), "\n") {
		line++
		entry := strings.TrimSpace(text)
		if entry == "" || strings.HasPrefix(entry, "#") {
			continue
		}
		directory := strings.HasSuffix(entry, "/")
		entry = strings.TrimSuffix(entry, "/")
		local, err := filepath.Localize(entry)
		if err != nil || entry == "." || strings.ContainsAny(entry, "\\\x00") ||
			(len(entry) >= 2 && entry[1] == ':') {
			return Scope{}, fmt.Errorf("%s:%d: invalid app-root-relative path %q", name, line, text)
		}
		patterns = append(patterns, exclusionPattern(filepath.Join(root, local)))
		if !directory {
			if prefix, ok := strings.CutSuffix(local, ".templ"); ok {
				patterns = append(patterns, exclusionPattern(filepath.Join(root, prefix+"_templ.go")))
			} else if prefix, ok := strings.CutSuffix(local, "_templ.go"); ok {
				patterns = append(patterns, exclusionPattern(filepath.Join(root, prefix+".templ")))
			}
		}
	}
	if len(patterns) > 0 {
		scope.ignore = regexp.MustCompile(strings.Join(patterns, "|"))
	}
	return scope, nil
}

func exclusionPattern(name string) string {
	segments := strings.Split(filepath.ToSlash(name), "/")
	for i, segment := range segments {
		segments[i] = regexp.QuoteMeta(segment)
	}
	return "^" + strings.Join(segments, `[\\/]`) + `([\\/]|$)`
}

// IgnorePattern returns templ's absolute-path ignore regexp, or an empty string.
func (s Scope) IgnorePattern() string {
	if s.ignore == nil {
		return ""
	}
	return s.ignore.String()
}

// HasFiles reports whether the application has any non-excluded .templ files.
func (s Scope) HasFiles() (bool, error) {
	hasTempl := false
	err := filepath.WalkDir(s.root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		ignored := s.ignore != nil && s.ignore.MatchString(name)
		if entry.IsDir() {
			if entry.Name() == ".git" || ignored {
				return fs.SkipDir
			}
			return nil
		}
		if !ignored && filepath.Ext(name) == ".templ" {
			hasTempl = true
			return fs.SkipAll
		}
		return nil
	})
	return hasTempl, err
}
