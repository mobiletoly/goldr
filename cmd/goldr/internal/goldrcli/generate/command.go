// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package generate

import (
	"context"
	"fmt"

	cliassets "github.com/mobiletoly/goldr/cmd/goldr/internal/goldrcli/assets"
	"github.com/mobiletoly/goldr/cmd/goldr/internal/goldrcli/project"
	"github.com/mobiletoly/goldr/cmd/goldr/internal/goldrcli/templtool"
	"github.com/urfave/cli/v3"
)

const (
	generateAppRootFlag = "app-root"
	generateCheckFlag   = "check"
)

type generateOptions struct {
	root  string
	check bool
}

func Command() *cli.Command {
	return &cli.Command{
		Name:        "generate",
		Usage:       "generate goldr route and URL files",
		UsageText:   "goldr generate [--app-root <dir>] [--check]",
		Description: generateDescription,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        generateAppRootFlag,
				Value:       ".",
				Usage:       "Goldr app root directory",
				Config:      cli.StringConfig{TrimSpace: true},
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
				root:  cmd.String(generateAppRootFlag),
				check: cmd.Bool(generateCheckFlag),
			})
		},
	}
}

const generateDescription = `Scans app/routes, expands referenced app/mounts subtrees, and writes goldr-owned generated files:
  app/routes/goldr_gen.go
  app/routes/**/goldr_gen.go when route packages need generated helpers
  app/internal/goldrinspect/goldr_gen.go
  app/urls/goldr_gen.go
  app/mounts/<mount>/goldr_gen.go for referenced Kit mount subtrees
  assets/goldr_assets_gen.go when assets/build exists

When .templ files exist, this command runs:
  go tool templ generate -path .

When assets/build exists, this command also fingerprints assets/build into assets/dist.

Use --check in CI to verify templ files when present and goldr-generated files without writing.`

func runGenerate(ctx context.Context, options generateOptions) error {
	paths, err := project.PathsForRoot(ctx, options.root)
	if err != nil {
		return fmt.Errorf("goldr generate: %w", err)
	}

	hasTempl, err := project.HasTemplFiles(paths.Root)
	if err != nil {
		return fmt.Errorf("goldr generate: %w", err)
	}
	if hasTempl {
		if options.check {
			if err := templtool.GenerateCheck(ctx, paths.Root); err != nil {
				return fmt.Errorf("goldr generate: %w", err)
			}
		} else {
			if err := templtool.Generate(ctx, paths.Root); err != nil {
				return fmt.Errorf("goldr generate: %w", err)
			}
		}
	}

	files, err := project.GenerateFilesForPaths(paths)
	if err != nil {
		return fmt.Errorf("goldr generate: %w", err)
	}

	if options.check {
		if err := project.CheckGeneratedFiles(files); err != nil {
			return fmt.Errorf("goldr generate: %w", err)
		}
		if err := project.CheckStaleManagedGeneratedFiles(paths, files); err != nil {
			return fmt.Errorf("goldr generate: %w", err)
		}
		if err := runGenerateAssets(options.root, true); err != nil {
			return fmt.Errorf("goldr generate: %w", err)
		}
		return nil
	}

	for _, file := range files {
		if err := project.WriteGeneratedFile(file); err != nil {
			return fmt.Errorf("goldr generate: %w", err)
		}
	}
	if err := project.RemoveStaleManagedGeneratedFiles(paths, files); err != nil {
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
