// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldrcli

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
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
	checkCodeTemplGenerated = "GOLDR007"
)

type checkOptions struct {
	root string
}

func checkCommand() *cli.Command {
	return &cli.Command{
		Name:        "check",
		Usage:       "check goldr route tree and generated files",
		UsageText:   "goldr check [--root <dir>]",
		Description: checkDescription,
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

const checkDescription = `Read-only validation for app/routes, templ output, and generated files.

Checks route naming, page/layout/fragment file pairs, action conventions, generated route dispatch readiness, generated URL helper readiness, templ-generated file freshness, and goldr-generated file freshness.

Run after go tool templ generate and go tool goldr generate. This command runs templ check mode but does not run tests, start the app, or write files.`

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

	generatedFiles, err := checkGeneratedManifestReadiness(paths, manifest)
	if err != nil {
		return fmt.Errorf("goldr check: %w", err)
	}

	if err := checkTemplGeneratedFiles(ctx, paths.root); err != nil {
		return fmt.Errorf("goldr check: %w", err)
	}

	if err := checkGeneratedFiles(generatedFiles); err != nil {
		return fmt.Errorf("goldr check: %w", checkMultilineCodeError(checkCodeGeneratedFiles, err))
	}
	return nil
}

func checkGeneratedManifestReadiness(paths appPaths, manifest routing.Manifest) ([]generatedFile, error) {
	routesFile, err := generateRouteManifestFile(paths, manifest)
	if err != nil {
		return nil, checkCodeError(checkCodeRouteGenerate, err)
	}
	urlsFile, err := generateURLHelperFile(paths, manifest)
	if err != nil {
		return nil, checkCodeError(checkCodeURLGenerate, err)
	}
	return []generatedFile{routesFile, urlsFile}, nil
}

func checkTemplGeneratedFiles(ctx context.Context, root string) error {
	if err := checkTemplTool(ctx, root); err != nil {
		return checkCodeError(checkCodeTemplGenerated, err)
	}

	command := exec.CommandContext(ctx, "go", "tool", "templ", "generate", "-check", "-path", ".")
	command.Dir = root
	output, err := command.CombinedOutput()
	if err == nil {
		return nil
	}

	var message strings.Builder
	message.WriteString("templ generated files are not up to date; run go tool templ generate")
	if trimmed := strings.TrimSpace(string(output)); trimmed != "" {
		message.WriteString("\n")
		message.WriteString(trimmed)
	}
	return checkMultilineCodeError(checkCodeTemplGenerated, errors.New(message.String()))
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
