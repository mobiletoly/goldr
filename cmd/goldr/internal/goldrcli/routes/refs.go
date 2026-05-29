// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/htmxrefs"
	"github.com/mobiletoly/goldr/cmd/goldr/internal/wiring"
	"github.com/urfave/cli/v3"
)

type refsOptions struct {
	root string
	json bool
}

func refsCommand() *cli.Command {
	return &cli.Command{
		Name:        "refs",
		Usage:       "list direct HTMX route references",
		UsageText:   "goldr routes refs [--app-root <dir>] [--json]",
		Description: refsDescription,
		Flags: []cli.Flag{
			rootStringFlag(),
			&cli.BoolFlag{
				Name:        listJSONFlag,
				Usage:       "print JSON output",
				HideDefault: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runRefs(ctx, refsOptions{
				root: cmd.String(appRootFlag),
				json: cmd.Bool(listJSONFlag),
			}, cmd.Root().Writer)
		},
	}
}

const refsDescription = `Prints direct HTMX request attributes found in .templ source and resolves obvious references to the goldr route surface.

This is source-level reference inventory. It does not render pages, execute app code, or infer inherited HTMX behavior such as hx-target or hx-swap.`

func runRefs(_ context.Context, options refsOptions, writer io.Writer) error {
	paths, manifest, err := scanRouteManifest(options.root)
	if err != nil {
		return fmt.Errorf("goldr routes refs: %w", err)
	}
	rows, err := wiring.RouteSurface(manifest)
	if err != nil {
		return fmt.Errorf("goldr routes refs: %w", err)
	}

	references, err := htmxrefs.Scan([]htmxrefs.Root{
		{Dir: paths.routesDir},
		{Dir: paths.mountsDir, SourcePrefix: "../mounts"},
	}, rows)
	if err != nil {
		return fmt.Errorf("goldr routes refs: %w", err)
	}

	if options.json {
		if err := renderRefsJSON(writer, references); err != nil {
			return fmt.Errorf("goldr routes refs: %w", err)
		}
		return nil
	}
	if err := renderRefsTable(writer, references); err != nil {
		return fmt.Errorf("goldr routes refs: %w", err)
	}
	return nil
}

func renderRefsTable(writer io.Writer, references []htmxrefs.Reference) error {
	table := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "STATUS\tMETHOD\tROUTE\tKIND\tATTRIBUTE\tSOURCE\tVALUE"); err != nil {
		return err
	}
	for _, reference := range references {
		if _, err := fmt.Fprintf(
			table,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			reference.Status,
			reference.Method,
			routeRefRouteText(reference.Route),
			routeRefKindText(reference.Match),
			reference.Attribute,
			routeRefSourceText(reference),
			routeRefValueText(reference.Value),
		); err != nil {
			return err
		}
	}
	return table.Flush()
}

func routeRefRouteText(route string) string {
	if route == "" {
		return "-"
	}
	return route
}

func routeRefKindText(match *htmxrefs.RouteMatch) string {
	if match == nil || match.Kind == "" {
		return "-"
	}
	return match.Kind
}

func routeRefSourceText(reference htmxrefs.Reference) string {
	source := filepath.ToSlash(reference.Source)
	if reference.Line == 0 {
		return source
	}
	return fmt.Sprintf("%s:%d:%d", source, reference.Line, reference.Column)
}

func routeRefValueText(value string) string {
	if strings.ContainsFunc(value, func(r rune) bool {
		return unicode.IsControl(r) || unicode.IsSpace(r)
	}) {
		return strconv.Quote(value)
	}
	return value
}

type routeRefJSONRow struct {
	Status    string             `json:"status"`
	Method    string             `json:"method"`
	Route     string             `json:"route"`
	Kind      string             `json:"kind"`
	Attribute string             `json:"attribute"`
	Source    string             `json:"source"`
	Line      int                `json:"line"`
	Column    int                `json:"column"`
	Value     string             `json:"value"`
	Matched   *routeRefJSONMatch `json:"matched,omitempty"`
}

type routeRefJSONMatch struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Source string `json:"source"`
	Helper string `json:"helper"`
}

func renderRefsJSON(writer io.Writer, references []htmxrefs.Reference) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(routeRefJSONRows(references))
}

func routeRefJSONRows(references []htmxrefs.Reference) []routeRefJSONRow {
	rows := make([]routeRefJSONRow, 0, len(references))
	for _, reference := range references {
		row := routeRefJSONRow{
			Status:    reference.Status,
			Method:    reference.Method,
			Route:     reference.Route,
			Attribute: reference.Attribute,
			Source:    filepath.ToSlash(reference.Source),
			Line:      reference.Line,
			Column:    reference.Column,
			Value:     reference.Value,
		}
		if reference.Match != nil {
			row.Kind = reference.Match.Kind
			row.Matched = &routeRefJSONMatch{
				Path:   reference.Match.Path,
				Kind:   reference.Match.Kind,
				Source: reference.Match.Source,
				Helper: reference.Match.Helper,
			}
		}
		rows = append(rows, row)
	}
	return rows
}
