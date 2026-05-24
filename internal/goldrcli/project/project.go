// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package project

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mobiletoly/goldr/internal/goldrcli/appfs"
	"github.com/mobiletoly/goldr/internal/routing"
	"github.com/mobiletoly/goldr/internal/wiring"
)

type GeneratedFile struct {
	Path    string
	Content []byte
}

type Paths struct {
	Root            string
	RoutesDir       string
	RouteImportPath string
}

func HasTemplFiles(root string) (bool, error) {
	hasTempl := false
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".templ" {
			hasTempl = true
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return hasTempl, nil
}

func GenerateFiles(ctx context.Context, root string) ([]GeneratedFile, error) {
	paths, err := PathsForRoot(ctx, root)
	if err != nil {
		return nil, err
	}

	return GenerateFilesForPaths(paths)
}

func GenerateFilesForPaths(paths Paths) ([]GeneratedFile, error) {
	tree, err := routing.Scan(paths.RoutesDir)
	if err != nil {
		return nil, err
	}
	manifest := routing.BuildManifest(*tree)

	return GenerateManifestFiles(paths, manifest)
}

func PathsForRoot(ctx context.Context, root string) (Paths, error) {
	appRoot, err := appfs.ResolveExistingDir(root)
	if err != nil {
		return Paths{}, fmt.Errorf("resolve --app-root %q: %w", root, err)
	}

	paths, err := PathsForResolvedRoot(ctx, appRoot)
	if err != nil {
		return Paths{}, err
	}
	if err := appfs.RequireDir(paths.RoutesDir); err != nil {
		return Paths{}, err
	}
	return paths, nil
}

func PathsForResolvedRoot(ctx context.Context, appRoot string) (Paths, error) {
	routesDir := appfs.RoutesDir(appRoot)
	routeImportPath, err := RouteRootImportPath(ctx, appRoot, routesDir)
	if err != nil {
		return Paths{}, err
	}

	return Paths{
		Root:            appRoot,
		RoutesDir:       routesDir,
		RouteImportPath: routeImportPath,
	}, nil
}

func GenerateManifestFiles(paths Paths, manifest routing.Manifest) ([]GeneratedFile, error) {
	routesFile, err := GenerateRouteManifestFile(paths, manifest)
	if err != nil {
		return nil, err
	}

	urlsFile, err := GenerateURLHelperFile(paths, manifest)
	if err != nil {
		return nil, err
	}

	inspectorFile, err := GenerateInspectorSupportFile(paths)
	if err != nil {
		return nil, err
	}

	fragmentWrapperFiles, err := GenerateFragmentWrapperFiles(paths, manifest)
	if err != nil {
		return nil, err
	}

	files := []GeneratedFile{
		routesFile,
		urlsFile,
		inspectorFile,
	}
	files = append(files, fragmentWrapperFiles...)
	return files, nil
}

func GenerateRouteManifestFile(paths Paths, manifest routing.Manifest) (GeneratedFile, error) {
	source, err := wiring.GenerateManifest(manifest, wiring.GenerateOptions{
		PackageName:         "routes",
		RouteRootImportPath: paths.RouteImportPath,
		InspectorImportPath: InspectorImportPath(paths),
	})
	if err != nil {
		return GeneratedFile{}, err
	}
	return GeneratedFile{Path: filepath.Join(paths.RoutesDir, wiring.GeneratedFileName), Content: source}, nil
}

func GenerateInspectorSupportFile(paths Paths) (GeneratedFile, error) {
	source, err := wiring.GenerateInspectorSupport("goldrinspect")
	if err != nil {
		return GeneratedFile{}, err
	}
	return GeneratedFile{Path: filepath.Join(paths.Root, "app", "internal", "goldrinspect", wiring.InspectorGeneratedFileName), Content: source}, nil
}

func GenerateFragmentWrapperFiles(paths Paths, manifest routing.Manifest) ([]GeneratedFile, error) {
	files, err := wiring.GenerateFragmentWrappers(manifest, wiring.GenerateOptions{
		RouteRootImportPath: paths.RouteImportPath,
		InspectorImportPath: InspectorImportPath(paths),
	})
	if err != nil {
		return nil, err
	}
	generated := make([]GeneratedFile, 0, len(files))
	for _, file := range files {
		generated = append(generated, GeneratedFile{
			Path:    filepath.Join(paths.RoutesDir, filepath.FromSlash(file.Dir), wiring.GeneratedFileName),
			Content: file.Content,
		})
	}
	return generated, nil
}

func GenerateURLHelperFile(paths Paths, manifest routing.Manifest) (GeneratedFile, error) {
	source, err := wiring.GenerateURLHelpers(manifest, wiring.GenerateURLOptions{
		PackageName: "urls",
	})
	if err != nil {
		return GeneratedFile{}, err
	}
	return GeneratedFile{Path: filepath.Join(paths.Root, "app", "urls", wiring.URLGeneratedFileName), Content: source}, nil
}

func InspectorImportPath(paths Paths) string {
	return path.Join(path.Dir(paths.RouteImportPath), "internal/goldrinspect")
}

func RouteRootImportPath(ctx context.Context, appRoot string, routesDir string) (string, error) {
	goModPath, err := GoModPath(ctx, appRoot)
	if err != nil {
		return "", err
	}
	moduleRoot := filepath.Dir(goModPath)

	modulePath, err := ModulePathFromGoMod(goModPath)
	if err != nil {
		return "", err
	}

	relRoutesDir, err := filepath.Rel(moduleRoot, routesDir)
	if err != nil {
		return "", err
	}
	if relRoutesDir == ".." || strings.HasPrefix(relRoutesDir, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("route root %s is outside module root %s", routesDir, moduleRoot)
	}

	return path.Join(modulePath, filepath.ToSlash(relRoutesDir)), nil
}

func GoModPath(ctx context.Context, appRoot string) (string, error) {
	command := exec.CommandContext(ctx, "go", "env", "GOMOD")
	command.Dir = appRoot

	output, err := command.Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", errors.New("go executable not found")
		}
		return "", fmt.Errorf("go env GOMOD: %w", err)
	}

	goMod := strings.TrimSpace(string(output))
	if goMod == "" || goMod == os.DevNull {
		return "", fmt.Errorf("could not find go.mod for --app-root %s", appRoot)
	}

	resolved, err := filepath.EvalSymlinks(goMod)
	if err != nil {
		return "", fmt.Errorf("resolve go.mod %s: %w", goMod, err)
	}
	return resolved, nil
}

func ModulePathFromGoMod(goModPath string) (string, error) {
	file, err := os.Open(goModPath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "module") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "module" {
			continue
		}
		modulePath := fields[1]
		if unquoted, err := strconv.Unquote(modulePath); err == nil {
			modulePath = unquoted
		}
		if modulePath == "" {
			return "", fmt.Errorf("empty module path in %s", goModPath)
		}
		return modulePath, nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("module directive not found in %s", goModPath)
}

func WriteGeneratedFile(file GeneratedFile) error {
	existing, err := os.ReadFile(file.Path)
	if err == nil && bytes.Equal(existing, file.Content) {
		return nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file.Path), 0755); err != nil {
		return err
	}
	return os.WriteFile(file.Path, file.Content, 0644)
}

func CheckGeneratedFiles(files []GeneratedFile) error {
	var stale []string
	for _, file := range files {
		existing, err := os.ReadFile(file.Path)
		switch {
		case errors.Is(err, os.ErrNotExist):
			stale = append(stale, fmt.Sprintf("%s is missing", file.Path))
		case err != nil:
			return err
		case !bytes.Equal(existing, file.Content):
			stale = append(stale, fmt.Sprintf("%s is stale", file.Path))
		}
	}
	if len(stale) > 0 {
		return errors.New(strings.Join(stale, "\n"))
	}
	return nil
}

func CheckStaleManagedGeneratedFiles(paths Paths, files []GeneratedFile) error {
	stale, err := StaleManagedGeneratedFiles(paths, files)
	if err != nil {
		return err
	}
	if len(stale) > 0 {
		for index, file := range stale {
			stale[index] = fmt.Sprintf("%s is stale", file)
		}
		return errors.New(strings.Join(stale, "\n"))
	}
	return nil
}

func RemoveStaleManagedGeneratedFiles(paths Paths, files []GeneratedFile) error {
	stale, err := StaleManagedGeneratedFiles(paths, files)
	if err != nil {
		return err
	}
	for _, file := range stale {
		if err := os.Remove(file); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func StaleManagedGeneratedFiles(paths Paths, files []GeneratedFile) ([]string, error) {
	expected := make(map[string]bool, len(files))
	for _, file := range files {
		expected[filepath.Clean(file.Path)] = true
	}

	var stale []string
	err := filepath.WalkDir(paths.RoutesDir, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != wiring.GeneratedFileName {
			return nil
		}
		cleanName := filepath.Clean(name)
		if !expected[cleanName] {
			generated, err := IsGoldrGeneratedFile(cleanName)
			if err != nil {
				return err
			}
			if generated {
				stale = append(stale, cleanName)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return stale, nil
}

func IsGoldrGeneratedFile(name string) (bool, error) {
	content, err := os.ReadFile(name)
	if err != nil {
		return false, err
	}
	return bytes.HasPrefix(content, []byte("// Code generated by goldr; DO NOT EDIT.")), nil
}
