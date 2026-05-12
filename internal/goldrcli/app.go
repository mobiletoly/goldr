// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldrcli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/mobiletoly/goldr/internal/goldrcli/routes"
	"github.com/urfave/cli/v3"
)

// New returns the root CLI command. It is internal so goldr can keep the public API small while the CLI grows.
func New(version string) *cli.Command {
	cmd := &cli.Command{
		Name:            "goldr",
		Usage:           "server-first Go framework for HTMX applications",
		UsageText:       "goldr <command>",
		Version:         version,
		HideVersion:     true,
		HideHelpCommand: true,
		ExitErrHandler:  func(context.Context, *cli.Command, error) {},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "version",
				Usage:       "show version",
				Local:       true,
				OnlyOnce:    true,
				HideDefault: true,
				Action: func(_ context.Context, cmd *cli.Command, _ bool) error {
					fmt.Fprintf(cmd.Root().Writer, "goldr %s\n", version)
					return cli.Exit("", 0)
				},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() > 0 {
				fmt.Fprintf(cmd.Root().ErrWriter, "goldr: unknown command %q\n\n", cmd.Args().First())
				if err := showRootHelpTo(cmd, cmd.Root().ErrWriter); err != nil {
					return err
				}
				return cli.Exit("", 2)
			}

			return cli.ShowRootCommandHelp(cmd)
		},
		Commands: []*cli.Command{
			initCommand(),
			checkCommand(),
			generateCommand(),
			routes.Command(),
			{
				Name:  "help",
				Usage: "show help",
				Action: func(_ context.Context, cmd *cli.Command) error {
					return cli.ShowRootCommandHelp(cmd.Root())
				},
			},
			{
				Name:  "version",
				Usage: "show version",
				Action: func(_ context.Context, cmd *cli.Command) error {
					fmt.Fprintf(cmd.Root().Writer, "goldr %s\n", version)
					return nil
				},
			},
		},
	}

	return cmd
}

// Run executes the root command and converts urfave exit errors into process exit codes.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer, version string) int {
	cmd := New(version)
	cmd.Writer = stdout
	cmd.ErrWriter = stderr

	err := cmd.Run(ctx, args)
	if err == nil {
		return 0
	}

	if exitCoder, ok := errors.AsType[cli.ExitCoder](err); ok {
		return exitCoder.ExitCode()
	}

	fmt.Fprintln(stderr, err)
	return 1
}

func showRootHelpTo(cmd *cli.Command, writer io.Writer) error {
	root := cmd.Root()
	original := root.Writer
	root.Writer = writer
	defer func() {
		root.Writer = original
	}()

	return cli.ShowRootCommandHelp(root)
}
