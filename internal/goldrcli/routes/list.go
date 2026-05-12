// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/mobiletoly/goldr/internal/wiring"
	"github.com/urfave/cli/v3"
)

type listOptions struct {
	root string
	json bool
}

func listCommand() *cli.Command {
	return &cli.Command{
		Name:      "list",
		Usage:     "list route surface",
		UsageText: "goldr routes list [--root <dir>] [--json]",
		Flags: []cli.Flag{
			rootStringFlag(),
			&cli.BoolFlag{
				Name:        listJSONFlag,
				Usage:       "print JSON output",
				HideDefault: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runList(ctx, listOptions{
				root: cmd.String(rootFlag),
				json: cmd.Bool(listJSONFlag),
			}, cmd.Root().Writer)
		},
	}
}

func runList(_ context.Context, options listOptions, writer io.Writer) error {
	_, manifest, err := scanRouteManifest(options.root)
	if err != nil {
		return fmt.Errorf("goldr routes list: %w", err)
	}

	rows, err := wiring.RouteSurface(manifest)
	if err != nil {
		return fmt.Errorf("goldr routes list: %w", err)
	}
	if options.json {
		if err := renderRouteSurfaceJSON(writer, rows); err != nil {
			return fmt.Errorf("goldr routes list: %w", err)
		}
		return nil
	}
	if err := renderRouteSurfaceTable(writer, rows); err != nil {
		return fmt.Errorf("goldr routes list: %w", err)
	}
	return nil
}

func renderRouteSurfaceTable(writer io.Writer, rows []wiring.RouteSurfaceRow) error {
	table := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "KIND\tMETHOD\tPATH\tPARAMS\tSOURCE\tHELPER"); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := fmt.Fprintf(
			table,
			"%s\t%s\t%s\t%s\t%s\t%s\n",
			row.Kind,
			wiring.FormatRouteSurfaceList(row.Methods),
			row.Path,
			wiring.FormatRouteSurfaceList(row.Params),
			row.Source,
			routeSurfaceHelperText(row.Helper),
		); err != nil {
			return err
		}
	}
	return table.Flush()
}

type routeSurfaceJSONRow struct {
	Kind    string   `json:"kind"`
	Methods []string `json:"methods"`
	Path    string   `json:"path"`
	Params  []string `json:"params"`
	Source  string   `json:"source"`
	Helper  string   `json:"helper"`
}

func renderRouteSurfaceJSON(writer io.Writer, rows []wiring.RouteSurfaceRow) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(routeSurfaceJSONRows(rows))
}

func routeSurfaceJSONRows(rows []wiring.RouteSurfaceRow) []routeSurfaceJSONRow {
	result := make([]routeSurfaceJSONRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, routeSurfaceJSONRow{
			Kind:    row.Kind,
			Methods: routeSurfaceJSONStrings(row.Methods),
			Path:    row.Path,
			Params:  routeSurfaceJSONStrings(row.Params),
			Source:  row.Source,
			Helper:  row.Helper,
		})
	}
	return result
}

func routeSurfaceJSONStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	copied := make([]string, len(values))
	copy(copied, values)
	return copied
}

func routeSurfaceHelperText(helper string) string {
	if helper == "" {
		return "-"
	}
	return helper
}
