// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldrcli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mobiletoly/goldr/internal/goldrcli/appfs"
	"github.com/mobiletoly/goldr/internal/routing"
	"github.com/urfave/cli/v3"
)

const initRootFlag = "root"

type initOptions struct {
	root string
}

func initCommand() *cli.Command {
	return &cli.Command{
		Name:        "init",
		Usage:       "initialize goldr app structure",
		UsageText:   "goldr init [--root <dir>]",
		Description: initDescription,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        initRootFlag,
				Value:       ".",
				Usage:       "app root directory",
				HideDefault: false,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runInit(ctx, initOptions{
				root: cmd.String(initRootFlag),
			})
		},
	}
}

const initDescription = `Creates the minimal app/routes and app/urls skeleton for an existing Go module.

Writes:
  app/routes/page.go
  app/routes/page.templ
  app/routes/layout.go
  app/routes/layout.templ
  app/routes/goldr_gen.go
  app/urls/goldr_gen.go

Does not create go.mod, write main.go, run templ generation, or start a server.`

func runInit(ctx context.Context, options initOptions) error {
	paths, err := initAppPathsForRoot(ctx, options.root)
	if err != nil {
		return fmt.Errorf("goldr init: %w", err)
	}

	files, err := initFiles(paths)
	if err != nil {
		return fmt.Errorf("goldr init: %w", err)
	}
	for _, file := range files {
		if err := writeGeneratedFile(file); err != nil {
			return fmt.Errorf("goldr init: %w", err)
		}
	}
	return nil
}

func initAppPathsForRoot(ctx context.Context, root string) (appPaths, error) {
	appRoot, err := appfs.ResolveExistingDir(root)
	if err != nil {
		return appPaths{}, fmt.Errorf("resolve --root %q: %w", root, err)
	}

	if err := requireMissingPath(filepath.Join(appRoot, "app")); err != nil {
		return appPaths{}, err
	}

	return appPathsForResolvedRoot(ctx, appRoot)
}

func requireMissingPath(name string) error {
	if _, err := os.Lstat(name); err == nil {
		return fmt.Errorf("%s already exists", name)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func initFiles(paths appPaths) ([]generatedFile, error) {
	files := []generatedFile{
		{
			path:    filepath.Join(paths.routesDir, "page.go"),
			content: []byte(initPageGoSource),
		},
		{
			path:    filepath.Join(paths.routesDir, "page.templ"),
			content: []byte(initPageTemplSource),
		},
		{
			path:    filepath.Join(paths.routesDir, "layout.go"),
			content: []byte(initLayoutGoSource),
		},
		{
			path:    filepath.Join(paths.routesDir, "layout.templ"),
			content: []byte(initLayoutTemplSource),
		},
	}

	generated, err := generateManifestFiles(paths, initManifest())
	if err != nil {
		return nil, err
	}
	return append(files, generated...), nil
}

func initManifest() routing.Manifest {
	return routing.Manifest{
		Pages: []routing.ManifestPage{
			{
				Route: "/",
				Unit: routing.RenderUnit{
					GoFile:    "page.go",
					TemplFile: "page.templ",
					HasTempl:  true,
				},
			},
		},
		Layouts: []routing.ManifestLayout{
			{
				RoutePrefix: "/",
				Unit: routing.RenderUnit{
					GoFile:    "layout.go",
					TemplFile: "layout.templ",
					HasTempl:  true,
				},
			},
		},
	}
}

const initPageGoSource = `package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(_ *http.Request) goldr.Page {
	return goldr.Page{
		Component: PageView(),
		Metadata: goldr.PageMetadata{
			Title: "Hello goldr",
		},
	}
}
`

const initPageTemplSource = `package routes

templ PageView() {
	<section>
		<h1>Hello goldr</h1>
		<p>Edit app/routes/page.templ to start building.</p>
	</section>
}
`

const initLayoutGoSource = `package routes

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

const defaultTitle = "Hello goldr"

func Layout(_ *http.Request, ctx goldr.LayoutContext) templ.Component {
	return LayoutView(ctx.Metadata, ctx.Child)
}

func pageTitle(metadata goldr.PageMetadata) string {
	if metadata.Title != "" {
		return metadata.Title
	}
	return defaultTitle
}
`

const initLayoutTemplSource = `package routes

import "github.com/mobiletoly/goldr"

templ LayoutView(metadata goldr.PageMetadata, child templ.Component) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1"/>
			<title>{ pageTitle(metadata) }</title>
			<script src="https://unpkg.com/htmx.org@2.0.4" defer></script>
		</head>
		<body>
			<main>
				@child
			</main>
		</body>
	</html>
}
`
