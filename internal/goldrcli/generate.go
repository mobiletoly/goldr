// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldrcli

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
	cliassets "github.com/mobiletoly/goldr/internal/goldrcli/assets"
	"github.com/mobiletoly/goldr/internal/routing"
	"github.com/mobiletoly/goldr/internal/wiring"
	"github.com/urfave/cli/v3"
)

const (
	generateRootFlag  = "root"
	generateCheckFlag = "check"
)

type generateOptions struct {
	root  string
	check bool
}

type generatedFile struct {
	path    string
	content []byte
}

type appPaths struct {
	root            string
	routesDir       string
	routeImportPath string
}

func generateCommand() *cli.Command {
	return &cli.Command{
		Name:        "generate",
		Usage:       "generate goldr route and URL files",
		UsageText:   "goldr generate [--root <dir>] [--check]",
		Description: generateDescription,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        generateRootFlag,
				Value:       ".",
				Usage:       "app root directory",
				HideDefault: false,
			},
			&cli.BoolFlag{
				Name:        generateCheckFlag,
				Usage:       "check generated files without writing",
				HideDefault: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runGenerate(ctx, generateOptions{
				root:  cmd.String(generateRootFlag),
				check: cmd.Bool(generateCheckFlag),
			})
		},
	}
}

const generateDescription = `Scans app/routes and writes goldr-owned generated files:
  app/routes/goldr_gen.go
  app/routes/**/goldr_gen.go when route packages need generated helpers
  app/internal/goldrinspect/goldr_gen.go
  app/urls/goldr_gen.go
  assets/goldr_assets_gen.go when assets/build exists

Before scanning routes, this command runs:
  go tool templ generate -path .

When assets/build exists, this command also fingerprints assets/build into assets/dist.

Use --check in CI to verify templ and goldr-generated files without writing.`

func runGenerate(ctx context.Context, options generateOptions) error {
	paths, err := appPathsForRoot(ctx, options.root)
	if err != nil {
		return fmt.Errorf("goldr generate: %w", err)
	}

	if options.check {
		if err := runTemplGenerateCheck(ctx, paths.root); err != nil {
			return fmt.Errorf("goldr generate: %w", err)
		}
	} else {
		if err := runTemplGenerateFiles(ctx, paths.root); err != nil {
			return fmt.Errorf("goldr generate: %w", err)
		}
	}

	files, err := generateFilesForPaths(paths)
	if err != nil {
		return fmt.Errorf("goldr generate: %w", err)
	}

	if options.check {
		if err := checkGeneratedFiles(files); err != nil {
			return fmt.Errorf("goldr generate: %w", err)
		}
		if err := checkStaleManagedGeneratedFiles(paths, files); err != nil {
			return fmt.Errorf("goldr generate: %w", err)
		}
		if err := runGenerateAssets(options.root, true); err != nil {
			return fmt.Errorf("goldr generate: %w", err)
		}
		return nil
	}

	for _, file := range files {
		if err := writeGeneratedFile(file); err != nil {
			return fmt.Errorf("goldr generate: %w", err)
		}
	}
	if err := removeStaleManagedGeneratedFiles(paths, files); err != nil {
		return fmt.Errorf("goldr generate: %w", err)
	}
	if err := runGenerateAssets(options.root, false); err != nil {
		return fmt.Errorf("goldr generate: %w", err)
	}
	return nil
}

func runGenerateAssets(root string, check bool) error {
	hasAssets, err := cliassets.HasBuildInputs(root)
	if err != nil {
		return err
	}
	if !hasAssets {
		return nil
	}
	if check {
		if err := cliassets.Check(root); err != nil {
			return fmt.Errorf("goldr-managed assets are not current; run go tool goldr generate\n%w", err)
		}
		return nil
	}
	return cliassets.Dist(root)
}

func runTemplGenerateFiles(ctx context.Context, root string) error {
	if err := checkTemplTool(ctx, root); err != nil {
		return err
	}

	command := exec.CommandContext(ctx, "go", "tool", "templ", "generate", "-path", ".")
	command.Dir = root
	output, err := command.CombinedOutput()
	if err == nil {
		return nil
	}

	var message strings.Builder
	message.WriteString("templ generation failed")
	if trimmed := strings.TrimSpace(string(output)); trimmed != "" {
		message.WriteString("\n")
		message.WriteString(trimmed)
	}
	return errors.New(message.String())
}

func runTemplGenerateCheck(ctx context.Context, root string) error {
	if err := checkTemplTool(ctx, root); err != nil {
		return err
	}

	command := exec.CommandContext(ctx, "go", "tool", "templ", "generate", "-check", "-path", ".")
	command.Dir = root
	output, err := command.CombinedOutput()
	if err == nil {
		return nil
	}

	var message strings.Builder
	message.WriteString("templ generated files are not up to date; run go tool goldr generate")
	if trimmed := strings.TrimSpace(string(output)); trimmed != "" {
		message.WriteString("\n")
		message.WriteString(trimmed)
	}
	return errors.New(message.String())
}

func generateFiles(ctx context.Context, root string) ([]generatedFile, error) {
	paths, err := appPathsForRoot(ctx, root)
	if err != nil {
		return nil, err
	}

	return generateFilesForPaths(paths)
}

func generateFilesForPaths(paths appPaths) ([]generatedFile, error) {
	tree, err := routing.Scan(paths.routesDir)
	if err != nil {
		return nil, err
	}
	manifest := routing.BuildManifest(*tree)

	return generateManifestFiles(paths, manifest)
}

func appPathsForRoot(ctx context.Context, root string) (appPaths, error) {
	appRoot, err := appfs.ResolveExistingDir(root)
	if err != nil {
		return appPaths{}, fmt.Errorf("resolve --root %q: %w", root, err)
	}

	paths, err := appPathsForResolvedRoot(ctx, appRoot)
	if err != nil {
		return appPaths{}, err
	}
	if err := appfs.RequireDir(paths.routesDir); err != nil {
		return appPaths{}, err
	}
	return paths, nil
}

func appPathsForResolvedRoot(ctx context.Context, appRoot string) (appPaths, error) {
	routesDir := appfs.RoutesDir(appRoot)
	routeImportPath, err := routeRootImportPath(ctx, appRoot, routesDir)
	if err != nil {
		return appPaths{}, err
	}

	return appPaths{
		root:            appRoot,
		routesDir:       routesDir,
		routeImportPath: routeImportPath,
	}, nil
}

func generateManifestFiles(paths appPaths, manifest routing.Manifest) ([]generatedFile, error) {
	routesFile, err := generateRouteManifestFile(paths, manifest)
	if err != nil {
		return nil, err
	}

	urlsFile, err := generateURLHelperFile(paths, manifest)
	if err != nil {
		return nil, err
	}

	inspectorFile, err := generateInspectorSupportFile(paths)
	if err != nil {
		return nil, err
	}

	fragmentWrapperFiles, err := generateFragmentWrapperFiles(paths, manifest)
	if err != nil {
		return nil, err
	}

	files := []generatedFile{
		routesFile,
		urlsFile,
		inspectorFile,
	}
	files = append(files, fragmentWrapperFiles...)
	return files, nil
}

func generateRouteManifestFile(paths appPaths, manifest routing.Manifest) (generatedFile, error) {
	source, err := wiring.GenerateManifest(manifest, wiring.GenerateOptions{
		PackageName:         "routes",
		RouteRootImportPath: paths.routeImportPath,
		InspectorImportPath: inspectorImportPath(paths),
	})
	if err != nil {
		return generatedFile{}, err
	}
	return generatedFile{path: filepath.Join(paths.routesDir, wiring.GeneratedFileName), content: source}, nil
}

func generateInspectorSupportFile(paths appPaths) (generatedFile, error) {
	source, err := wiring.GenerateInspectorSupport("goldrinspect")
	if err != nil {
		return generatedFile{}, err
	}
	return generatedFile{path: filepath.Join(paths.root, "app", "internal", "goldrinspect", wiring.InspectorGeneratedFileName), content: source}, nil
}

func generateFragmentWrapperFiles(paths appPaths, manifest routing.Manifest) ([]generatedFile, error) {
	files, err := wiring.GenerateFragmentWrappers(manifest, wiring.GenerateOptions{
		RouteRootImportPath: paths.routeImportPath,
		InspectorImportPath: inspectorImportPath(paths),
	})
	if err != nil {
		return nil, err
	}
	generated := make([]generatedFile, 0, len(files))
	for _, file := range files {
		generated = append(generated, generatedFile{
			path:    filepath.Join(paths.routesDir, filepath.FromSlash(file.Dir), wiring.GeneratedFileName),
			content: file.Content,
		})
	}
	return generated, nil
}

func generateURLHelperFile(paths appPaths, manifest routing.Manifest) (generatedFile, error) {
	source, err := wiring.GenerateURLHelpers(manifest, wiring.GenerateURLOptions{
		PackageName: "urls",
	})
	if err != nil {
		return generatedFile{}, err
	}
	return generatedFile{path: filepath.Join(paths.root, "app", "urls", wiring.URLGeneratedFileName), content: source}, nil
}

func inspectorImportPath(paths appPaths) string {
	return path.Join(path.Dir(paths.routeImportPath), "internal/goldrinspect")
}

func routeRootImportPath(ctx context.Context, appRoot string, routesDir string) (string, error) {
	goModPath, err := goModPath(ctx, appRoot)
	if err != nil {
		return "", err
	}
	moduleRoot := filepath.Dir(goModPath)

	modulePath, err := modulePathFromGoMod(goModPath)
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

func goModPath(ctx context.Context, appRoot string) (string, error) {
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
		return "", fmt.Errorf("could not find go.mod for --root %s", appRoot)
	}

	resolved, err := filepath.EvalSymlinks(goMod)
	if err != nil {
		return "", fmt.Errorf("resolve go.mod %s: %w", goMod, err)
	}
	return resolved, nil
}

func modulePathFromGoMod(goModPath string) (string, error) {
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

func writeGeneratedFile(file generatedFile) error {
	existing, err := os.ReadFile(file.path)
	if err == nil && bytes.Equal(existing, file.content) {
		return nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file.path), 0755); err != nil {
		return err
	}
	return os.WriteFile(file.path, file.content, 0644)
}

func checkGeneratedFiles(files []generatedFile) error {
	var stale []string
	for _, file := range files {
		existing, err := os.ReadFile(file.path)
		switch {
		case errors.Is(err, os.ErrNotExist):
			stale = append(stale, fmt.Sprintf("%s is missing", file.path))
		case err != nil:
			return err
		case !bytes.Equal(existing, file.content):
			stale = append(stale, fmt.Sprintf("%s is stale", file.path))
		}
	}
	if len(stale) > 0 {
		return errors.New(strings.Join(stale, "\n"))
	}
	return nil
}

func checkStaleManagedGeneratedFiles(paths appPaths, files []generatedFile) error {
	stale, err := staleManagedGeneratedFiles(paths, files)
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

func removeStaleManagedGeneratedFiles(paths appPaths, files []generatedFile) error {
	stale, err := staleManagedGeneratedFiles(paths, files)
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

func staleManagedGeneratedFiles(paths appPaths, files []generatedFile) ([]string, error) {
	expected := make(map[string]bool, len(files))
	for _, file := range files {
		expected[filepath.Clean(file.path)] = true
	}

	var stale []string
	err := filepath.WalkDir(paths.routesDir, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != wiring.GeneratedFileName {
			return nil
		}
		cleanName := filepath.Clean(name)
		if !expected[cleanName] {
			generated, err := isGoldrGeneratedFile(cleanName)
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

func isGoldrGeneratedFile(name string) (bool, error) {
	content, err := os.ReadFile(name)
	if err != nil {
		return false, err
	}
	return bytes.HasPrefix(content, []byte("// Code generated by goldr; DO NOT EDIT.")), nil
}
