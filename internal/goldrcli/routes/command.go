// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	"context"
	"fmt"

	"github.com/mobiletoly/goldr/internal/goldrcli/appfs"
	"github.com/mobiletoly/goldr/internal/goldrcli/scandiag"
	"github.com/mobiletoly/goldr/internal/routing"
	"github.com/urfave/cli/v3"
)

const (
	rootFlag     = "root"
	listJSONFlag = "json"
)

type routeTreePaths struct {
	routesDir string
}

func Command() *cli.Command {
	return &cli.Command{
		Name:        "routes",
		Usage:       "inspect goldr routes",
		UsageText:   "goldr routes <command> [options]",
		Description: routesDescription,
		Commands: []*cli.Command{
			listCommand(),
			layoutsCommand(),
			explainCommand(),
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return cli.ShowSubcommandHelp(cmd)
		},
	}
}

const routesDescription = `Read-only inspection for the app/routes filesystem route tree.

Use before editing routes:
  go tool goldr routes list
  go tool goldr routes layouts
  go tool goldr routes explain /users/7

These commands do not write generated files. Run "go tool goldr generate" after route changes.`

func rootStringFlag() *cli.StringFlag {
	return &cli.StringFlag{
		Name:        rootFlag,
		Value:       ".",
		Usage:       "app root directory",
		HideDefault: false,
	}
}

func scanRouteManifest(root string) (routeTreePaths, routing.Manifest, error) {
	paths, err := routeTreePathsForRoot(root)
	if err != nil {
		return routeTreePaths{}, routing.Manifest{}, err
	}

	tree, err := routing.Scan(paths.routesDir)
	if err != nil {
		return routeTreePaths{}, routing.Manifest{}, scandiag.Error(paths.routesDir, err)
	}
	return paths, routing.BuildManifest(*tree), nil
}

func routeTreePathsForRoot(root string) (routeTreePaths, error) {
	appRoot, err := appfs.ResolveExistingDir(root)
	if err != nil {
		return routeTreePaths{}, fmt.Errorf("resolve --root %q: %w", root, err)
	}

	routesDir := appfs.RoutesDir(appRoot)
	if err := appfs.RequireDir(routesDir); err != nil {
		return routeTreePaths{}, err
	}
	return routeTreePaths{
		routesDir: routesDir,
	}, nil
}
