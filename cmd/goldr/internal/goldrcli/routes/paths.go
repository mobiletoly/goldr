// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	"os"
	"path/filepath"
)

func routeSourceDisplayPath(routesDir string, source string) string {
	return cwdRelativePath(filepath.Join(routesDir, filepath.FromSlash(source)))
}

func cwdRelativePath(name string) string {
	cwd, cwdErr := os.Getwd()
	rel, relErr := filepath.Rel(cwd, name)
	if cwdErr == nil && relErr == nil {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(name)
}
