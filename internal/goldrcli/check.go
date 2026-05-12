// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldrcli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mobiletoly/goldr/internal/goldrcli/appfs"
	"github.com/mobiletoly/goldr/internal/goldrcli/scandiag"
	"github.com/mobiletoly/goldr/internal/renderunit"
	"github.com/mobiletoly/goldr/internal/routing"
	"github.com/urfave/cli/v3"
)

const (
	checkRootFlag = "root"

	checkCodeAppRoot        = "GOLDR001"
	checkCodeRouteScan      = "GOLDR002"
	checkCodeRenderUnit     = "GOLDR003"
	checkCodeRouteGenerate  = "GOLDR004"
	checkCodeURLGenerate    = "GOLDR005"
	checkCodeGeneratedFiles = "GOLDR006"
)

type checkOptions struct {
	root string
}

func checkCommand() *cli.Command {
	return &cli.Command{
		Name:      "check",
		Usage:     "check goldr route tree and generated files",
		UsageText: "goldr check [--root <dir>]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        checkRootFlag,
				Value:       ".",
				Usage:       "app root directory",
				HideDefault: false,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runCheck(ctx, checkOptions{
				root: cmd.String(checkRootFlag),
			})
		},
	}
}

func runCheck(ctx context.Context, options checkOptions) error {
	paths, err := appPathsForRoot(ctx, options.root)
	if err != nil {
		return fmt.Errorf("goldr check: %w", checkCodeError(checkCodeAppRoot, err))
	}

	tree, err := routing.Scan(paths.routesDir)
	if err != nil {
		return fmt.Errorf("goldr check: %w", scandiag.CodeError(paths.routesDir, err, checkCodeRouteScan))
	}
	manifest := routing.BuildManifest(*tree)

	if err := renderunit.ValidateManifest(manifest); err != nil {
		return fmt.Errorf("goldr check: %w", checkRenderUnitError(paths.routesDir, err))
	}

	if err := checkGeneratedManifestFiles(paths, manifest); err != nil {
		return fmt.Errorf("goldr check: %w", err)
	}
	return nil
}

func checkGeneratedManifestFiles(paths appPaths, manifest routing.Manifest) error {
	routesFile, err := generateRouteManifestFile(paths, manifest)
	if err != nil {
		return checkCodeError(checkCodeRouteGenerate, err)
	}
	urlsFile, err := generateURLHelperFile(paths, manifest)
	if err != nil {
		return checkCodeError(checkCodeURLGenerate, err)
	}
	if err := checkGeneratedFiles([]generatedFile{routesFile, urlsFile}); err != nil {
		return checkMultilineCodeError(checkCodeGeneratedFiles, err)
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
