// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package appfs

import (
	"fmt"
	"os"
	"path/filepath"
)

func RoutesDir(appRoot string) string {
	return filepath.Join(appRoot, "app", "routes")
}

func RouteDiagnosticPath(routesDir string, relPath string) string {
	if relPath == "" || relPath == "." {
		return routesDir
	}
	return filepath.Join(routesDir, filepath.FromSlash(relPath))
}

func ResolveExistingDir(name string) (string, error) {
	absolute, err := filepath.Abs(name)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	if err := RequireDir(resolved); err != nil {
		return "", err
	}
	return resolved, nil
}

func RequireDir(name string) error {
	info, err := os.Stat(name)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", name)
	}
	return nil
}
