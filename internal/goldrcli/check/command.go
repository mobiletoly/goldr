// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package check

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mobiletoly/goldr/internal/goldrcli/appfs"
	cliassets "github.com/mobiletoly/goldr/internal/goldrcli/assets"
	"github.com/mobiletoly/goldr/internal/goldrcli/project"
	"github.com/mobiletoly/goldr/internal/goldrcli/scandiag"
	"github.com/mobiletoly/goldr/internal/goldrcli/templtool"
	"github.com/mobiletoly/goldr/internal/renderunit"
	"github.com/mobiletoly/goldr/internal/routing"
	"github.com/urfave/cli/v3"
)

const (
	checkAppRootFlag = "app-root"

	checkCodeAppRoot        = "GOLDR001"
	checkCodeRouteScan      = "GOLDR002"
	checkCodeRenderUnit     = "GOLDR003"
	checkCodeRouteGenerate  = "GOLDR004"
	checkCodeURLGenerate    = "GOLDR005"
	checkCodeGeneratedFiles = "GOLDR006"
	checkCodeTemplGenerated = "GOLDR007"
	checkCodeAssets         = "GOLDR008"
)

type checkOptions struct {
	root string
}

func Command() *cli.Command {
	return &cli.Command{
		Name:        "check",
		Usage:       "check goldr route tree and generated files",
		UsageText:   "goldr check [--app-root <dir>]",
		Description: checkDescription,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        checkAppRootFlag,
				Value:       ".",
				Usage:       "Goldr app root directory",
				Config:      cli.StringConfig{TrimSpace: true},
				HideDefault: false,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runCheck(ctx, checkOptions{
				root: cmd.String(checkAppRootFlag),
			})
		},
	}
}

const checkDescription = `Read-only validation for app/routes, templ output when present, generated files, and Goldr-managed assets.

Checks route naming, page handler signatures, layout/fragment file pairs, action conventions, generated route dispatch readiness, generated URL helper readiness, templ-generated file freshness when .templ files exist, goldr-generated file freshness, and Goldr-managed asset freshness when asset outputs exist.

Run after go tool goldr generate. This command runs templ check mode when .templ files exist but does not run tests, start the app, or write files.`

func runCheck(ctx context.Context, options checkOptions) error {
	paths, err := project.PathsForRoot(ctx, options.root)
	if err != nil {
		return fmt.Errorf("goldr check: %w", checkCodeError(checkCodeAppRoot, err))
	}

	tree, err := routing.Scan(paths.RoutesDir)
	if err != nil {
		return fmt.Errorf("goldr check: %w", scandiag.CodeError(paths.RoutesDir, err, checkCodeRouteScan))
	}
	manifest := routing.BuildManifest(*tree)

	if err := renderunit.ValidateManifest(manifest); err != nil {
		return fmt.Errorf("goldr check: %w", checkRenderUnitError(paths.RoutesDir, err))
	}

	generatedFiles, err := checkGeneratedManifestReadiness(paths, manifest)
	if err != nil {
		return fmt.Errorf("goldr check: %w", err)
	}

	hasTempl, err := project.HasTemplFiles(paths.Root)
	if err != nil {
		return fmt.Errorf("goldr check: %w", checkCodeError(checkCodeTemplGenerated, err))
	}
	if hasTempl {
		if err := checkTemplGeneratedFiles(ctx, paths.Root); err != nil {
			return fmt.Errorf("goldr check: %w", err)
		}
	}

	if err := project.CheckGeneratedFiles(generatedFiles); err != nil {
		return fmt.Errorf("goldr check: %w", checkMultilineCodeError(checkCodeGeneratedFiles, err))
	}
	if err := checkManagedAssets(paths.Root); err != nil {
		return fmt.Errorf("goldr check: %w", err)
	}
	return nil
}

func checkGeneratedManifestReadiness(paths project.Paths, manifest routing.Manifest) ([]project.GeneratedFile, error) {
	routesFile, err := project.GenerateRouteManifestFile(paths, manifest)
	if err != nil {
		return nil, checkCodeError(checkCodeRouteGenerate, err)
	}
	urlsFile, err := project.GenerateURLHelperFile(paths, manifest)
	if err != nil {
		return nil, checkCodeError(checkCodeURLGenerate, err)
	}
	inspectorFile, err := project.GenerateInspectorSupportFile(paths)
	if err != nil {
		return nil, checkCodeError(checkCodeRouteGenerate, err)
	}
	fragmentWrapperFiles, err := project.GenerateFragmentWrapperFiles(paths, manifest)
	if err != nil {
		return nil, checkCodeError(checkCodeRouteGenerate, err)
	}
	files := []project.GeneratedFile{routesFile, urlsFile, inspectorFile}
	files = append(files, fragmentWrapperFiles...)
	if err := project.CheckStaleManagedGeneratedFiles(paths, files); err != nil {
		return nil, checkCodeError(checkCodeGeneratedFiles, err)
	}
	return files, nil
}

func checkTemplGeneratedFiles(ctx context.Context, root string) error {
	if err := templtool.Require(ctx, root); err != nil {
		return checkCodeError(checkCodeTemplGenerated, err)
	}

	if err := templtool.GenerateCheck(ctx, root); err != nil {
		return checkMultilineCodeError(checkCodeTemplGenerated, err)
	}
	return nil
}

func checkManagedAssets(root string) error {
	hasAssets, err := cliassets.HasManagedOutputs(root)
	if err != nil {
		return checkCodeError(checkCodeAssets, err)
	}
	if !hasAssets {
		return nil
	}
	if err := cliassets.Check(root); err != nil {
		message := fmt.Errorf("goldr-managed assets are not current; run go tool goldr generate\n%w", err)
		return checkMultilineCodeError(checkCodeAssets, message)
	}
	return nil
}

func checkRenderUnitError(routesDir string, err error) error {
	var validationErr *renderunit.ValidationError
	if !errors.As(err, &validationErr) {
		return checkCodeError(checkCodeRenderUnit, err)
	}

	messages := make([]string, 0, len(validationErr.Problems))
	for _, problem := range validationErr.Problems {
		messages = append(messages, fmt.Sprintf(
			"%s: %s %s %s: %s",
			appfs.RouteDiagnosticPath(routesDir, problem.GoFile),
			checkCodeRenderUnit,
			problem.Kind,
			problem.Identifier,
			problem.Message,
		))
	}
	return errors.New(strings.Join(messages, "\n"))
}

func checkCodeError(code string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s %w", code, err)
}

func checkMultilineCodeError(code string, err error) error {
	if err == nil {
		return nil
	}
	lines := strings.Split(err.Error(), "\n")
	for index, line := range lines {
		lines[index] = fmt.Sprintf("%s %s", code, line)
	}
	return errors.New(strings.Join(lines, "\n"))
}
