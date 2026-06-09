// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package initcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/goldrcli/appfs"
	"github.com/mobiletoly/goldr/cmd/goldr/internal/goldrcli/project"
	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
	"github.com/urfave/cli/v3"
)

const initAppRootFlag = "app-root"

type initOptions struct {
	root string
}

func Command() *cli.Command {
	return &cli.Command{
		Name:        "init",
		Usage:       "initialize goldr app structure",
		UsageText:   "goldr init [--app-root <dir>]",
		Description: initDescription,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        initAppRootFlag,
				Value:       ".",
				Usage:       "Goldr app root directory",
				Config:      cli.StringConfig{TrimSpace: true},
				HideDefault: false,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runInit(ctx, initOptions{
				root: cmd.String(initAppRootFlag),
			})
		},
	}
}

const initDescription = `Creates the minimal app/routes and app/urls skeleton for an existing Go module.

Writes:
  app/routes/route.go
  app/routes/page.templ
  app/routes/layout.go
  app/routes/layout.templ
  app/routes/goldr_gen.go
  app/internal/goldrinspect/goldr_gen.go
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
		if err := project.WriteGeneratedFile(file); err != nil {
			return fmt.Errorf("goldr init: %w", err)
		}
	}
	return nil
}

func initAppPathsForRoot(ctx context.Context, root string) (project.Paths, error) {
	appRoot, err := appfs.ResolveExistingDir(root)
	if err != nil {
		return project.Paths{}, fmt.Errorf("resolve --app-root %q: %w", root, err)
	}

	if err := requireMissingPath(filepath.Join(appRoot, "app")); err != nil {
		return project.Paths{}, err
	}

	return project.PathsForResolvedRoot(ctx, appRoot)
}

func requireMissingPath(name string) error {
	if _, err := os.Lstat(name); err == nil {
		return fmt.Errorf("%s already exists", name)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func initFiles(paths project.Paths) ([]project.GeneratedFile, error) {
	files := []project.GeneratedFile{
		{
			Path:    filepath.Join(paths.RoutesDir, "route.go"),
			Content: []byte(initRouteGoSource),
		},
		{
			Path:    filepath.Join(paths.RoutesDir, "page.templ"),
			Content: []byte(initPageTemplSource),
		},
		{
			Path:    filepath.Join(paths.RoutesDir, "layout.go"),
			Content: []byte(initLayoutGoSource),
		},
		{
			Path:    filepath.Join(paths.RoutesDir, "layout.templ"),
			Content: []byte(initLayoutTemplSource),
		},
	}

	generated, err := project.GenerateManifestFiles(paths, initManifest())
	if err != nil {
		return nil, err
	}
	return append(files, generated...), nil
}

func initManifest() routing.Manifest {
	return routing.Manifest{
		Routes: []routing.ManifestRouteDeclaration{
			{
				Route:  "/",
				GoFile: "route.go",
				Kind:   "local",
				Page: &routing.RouteHandlerDeclaration{
					Handler:   "page",
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

const initRouteGoSource = `package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

var Route = goldr.RouteDef{
	Page: page,
}

func page(_ *http.Request) goldr.PageRouteResponse {
	return goldr.NewPage(
		PageView(),
		goldr.PageMetadata{
			Title: "Hello goldr",
		},
	)
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
			<script src="https://cdn.jsdelivr.net/npm/htmx.org@4.0.0-beta4" integrity="sha384-aWZK1NtOs/aWb/+YZdTM8q2JkWEshlMc9mgZ189numT9bwFhyAyYEoO4nO/2dTXt" crossorigin="anonymous" defer></script>
		</head>
		<body>
			<main>
				@child
			</main>
		</body>
	</html>
}
`
