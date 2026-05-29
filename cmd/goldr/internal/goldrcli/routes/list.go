// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/wiring"
	"github.com/urfave/cli/v3"
)

type listOptions struct {
	root  string
	json  bool
	mount string
}

func listCommand() *cli.Command {
	return &cli.Command{
		Name:        "list",
		Usage:       "list route surface",
		UsageText:   "goldr routes list [--app-root <dir>] [--mount <path>] [--json]",
		Description: listDescription,
		Flags: []cli.Flag{
			rootStringFlag(),
			&cli.StringFlag{
				Name:        "mount",
				Usage:       "filter to routes expanded from one app/mounts path",
				Config:      cli.StringConfig{TrimSpace: true},
				HideDefault: true,
			},
			&cli.BoolFlag{
				Name:        listJSONFlag,
				Usage:       "print JSON output",
				HideDefault: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runList(ctx, listOptions{
				root:  cmd.String(appRootFlag),
				json:  cmd.Bool(listJSONFlag),
				mount: cmd.String("mount"),
			}, cmd.Root().Writer)
		},
	}
}

const listDescription = `Prints the pages, layouts, fragments, actions, paths, params, source files, and generated URL helper expressions goldr sees.

Use --json when scripts or agents need a stable route inventory before editing.`

func runList(_ context.Context, options listOptions, writer io.Writer) error {
	_, manifest, err := scanRouteManifest(options.root)
	if err != nil {
		return fmt.Errorf("goldr routes list: %w", err)
	}

	var rows []wiring.RouteSurfaceRow
	if strings.TrimSpace(options.mount) == "" {
		rows, err = wiring.RouteSurface(manifest)
	} else {
		rows, err = wiring.RouteSurfaceWithMountSelections(manifest)
	}
	if err != nil {
		return fmt.Errorf("goldr routes list: %w", err)
	}
	rows = filterRouteSurfaceRowsByMount(rows, options.mount)
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

func filterRouteSurfaceRowsByMount(rows []wiring.RouteSurfaceRow, mount string) []wiring.RouteSurfaceRow {
	mount = strings.TrimSpace(mount)
	if mount == "" {
		return rows
	}

	result := make([]wiring.RouteSurfaceRow, 0, len(rows))
	mountedSourcePrefix := "../mounts/" + mount + "/"
	for _, row := range rows {
		if row.Declaration != nil && row.Declaration.Mount != nil && row.Declaration.Mount.Path == mount {
			result = append(result, row)
			continue
		}
		if row.Kind == wiring.RouteSurfaceKindLayout && strings.HasPrefix(row.Source, mountedSourcePrefix) {
			result = append(result, row)
		}
	}
	return result
}

func renderRouteSurfaceTable(writer io.Writer, rows []wiring.RouteSurfaceRow) error {
	table := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)
	showStatus := routeSurfaceRowsHaveStatus(rows)
	header := "KIND\tMETHOD\tPATH\tPARAMS\tSOURCE\tOWNER\tDECL\tNAME\tTITLE\tLABELS\tHELPER"
	if showStatus {
		header += "\tSTATUS"
	}
	if _, err := fmt.Fprintln(table, header); err != nil {
		return err
	}
	for _, row := range rows {
		line := "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s"
		values := []any{
			row.Kind,
			wiring.FormatRouteSurfaceList(row.Methods),
			row.Path,
			wiring.FormatRouteSurfaceList(row.Params),
			row.Source,
			routeSurfaceDeclarationOwnerText(row.Declaration),
			routeSurfaceDeclarationKindText(row.Declaration),
			routeSurfaceDeclarationNameText(row.Declaration),
			routeSurfaceDeclarationTitleText(row.Declaration),
			routeSurfaceDeclarationLabelsText(row.Declaration),
			routeSurfaceHelperText(row.Helper),
		}
		if showStatus {
			line += "\t%s"
			values = append(values, routeSurfaceStatusText(row.Selection))
		}
		line += "\n"
		if _, err := fmt.Fprintf(
			table,
			line,
			values...,
		); err != nil {
			return err
		}
	}
	return table.Flush()
}

func routeSurfaceRowsHaveStatus(rows []wiring.RouteSurfaceRow) bool {
	return slices.ContainsFunc(rows, func(row wiring.RouteSurfaceRow) bool {
		return row.Selection != ""
	})
}

type routeSurfaceJSONRow struct {
	Kind        string                       `json:"kind"`
	Methods     []string                     `json:"methods"`
	Path        string                       `json:"path"`
	Params      []string                     `json:"params"`
	Source      string                       `json:"source"`
	Helper      string                       `json:"helper"`
	Status      string                       `json:"status,omitempty"`
	Declaration *routeSurfaceJSONDeclaration `json:"declaration,omitempty"`
}

type routeSurfaceJSONDeclaration struct {
	Source       string                        `json:"source"`
	Kind         string                        `json:"kind"`
	Name         string                        `json:"name"`
	Title        string                        `json:"title"`
	Labels       []routeSurfaceJSONLabel       `json:"labels"`
	NavTrails    []string                      `json:"nav_trails,omitempty"`
	Destinations []routeSurfaceJSONDestination `json:"destinations,omitempty"`
	Mount        *routeSurfaceJSONMount        `json:"mount,omitempty"`
	Kit          *routeSurfaceJSONKit          `json:"kit,omitempty"`
	Page         *routeSurfaceJSONPage         `json:"page,omitempty"`
	Fragment     *routeSurfaceJSONFragment     `json:"fragment,omitempty"`
	Action       *routeSurfaceJSONAction       `json:"action,omitempty"`
}

type routeSurfaceJSONLabel struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type routeSurfaceJSONDestination struct {
	Name     string `json:"name"`
	Helper   string `json:"helper"`
	Target   string `json:"target"`
	NavTrail string `json:"nav_trail"`
}

type routeSurfaceJSONKit struct {
	KitType string `json:"kit_type"`
	New     string `json:"new"`
}

type routeSurfaceJSONMount struct {
	Path  string `json:"path"`
	Owner string `json:"owner"`
}

type routeSurfaceJSONPage struct {
	Handler string `json:"handler"`
	Adapter string `json:"adapter"`
}

type routeSurfaceJSONFragment struct {
	Name    string `json:"name"`
	Segment string `json:"segment"`
	Index   bool   `json:"index"`
	Handler string `json:"handler"`
	Adapter string `json:"adapter"`
}

type routeSurfaceJSONAction struct {
	Method  string `json:"method"`
	Name    string `json:"name"`
	Segment string `json:"segment"`
	Index   bool   `json:"index"`
	Writer  bool   `json:"writer"`
	Handler string `json:"handler"`
	Adapter string `json:"adapter"`
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
			Kind:        row.Kind,
			Methods:     routeSurfaceJSONStrings(row.Methods),
			Path:        row.Path,
			Params:      routeSurfaceJSONStrings(row.Params),
			Source:      row.Source,
			Helper:      row.Helper,
			Status:      row.Selection,
			Declaration: routeSurfaceJSONDeclarationFrom(row.Declaration),
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

func routeSurfaceStatusText(status string) string {
	if status == "" {
		return "-"
	}
	return status
}

func routeSurfaceDeclarationKindText(declaration *wiring.RouteDeclarationInfo) string {
	if declaration == nil || declaration.Kind == "" {
		return "-"
	}
	return declaration.Kind
}

func routeSurfaceDeclarationNameText(declaration *wiring.RouteDeclarationInfo) string {
	if declaration == nil || declaration.Name == "" {
		return "-"
	}
	return routeSurfaceDeclarationText(declaration.Name)
}

func routeSurfaceDeclarationTitleText(declaration *wiring.RouteDeclarationInfo) string {
	if declaration == nil || declaration.Title == "" {
		return "-"
	}
	return routeSurfaceDeclarationText(declaration.Title)
}

func routeSurfaceDeclarationLabelsText(declaration *wiring.RouteDeclarationInfo) string {
	if declaration == nil || len(declaration.Labels) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(declaration.Labels))
	for _, label := range declaration.Labels {
		parts = append(parts, routeSurfaceDeclarationText(label.Key)+"="+strconv.Quote(label.Value))
	}
	return strings.Join(parts, ",")
}

func routeSurfaceDeclarationNavTrailsText(declaration *wiring.RouteDeclarationInfo) string {
	if declaration == nil || len(declaration.NavTrails) == 0 {
		return "-"
	}
	return strings.Join(declaration.NavTrails, ",")
}

func routeSurfaceDeclarationOwnerText(declaration *wiring.RouteDeclarationInfo) string {
	if declaration == nil || declaration.Mount == nil || declaration.Mount.Owner == "" {
		return "-"
	}
	return declaration.Mount.Owner
}

func routeSurfaceDeclarationText(value string) string {
	if strings.ContainsFunc(value, unicode.IsControl) {
		return strconv.Quote(value)
	}
	return value
}

func routeSurfaceJSONDeclarationFrom(declaration *wiring.RouteDeclarationInfo) *routeSurfaceJSONDeclaration {
	if declaration == nil {
		return nil
	}
	result := &routeSurfaceJSONDeclaration{
		Source:       declaration.Source,
		Kind:         declaration.Kind,
		Name:         declaration.Name,
		Title:        declaration.Title,
		Labels:       routeSurfaceJSONLabels(declaration.Labels),
		NavTrails:    routeSurfaceJSONStrings(declaration.NavTrails),
		Destinations: routeSurfaceJSONDestinations(declaration.Destinations),
	}
	if declaration.Kit != nil {
		result.Kit = &routeSurfaceJSONKit{
			KitType: declaration.Kit.KitType,
			New:     declaration.Kit.New,
		}
	}
	if declaration.Mount != nil {
		result.Mount = &routeSurfaceJSONMount{
			Path:  declaration.Mount.Path,
			Owner: declaration.Mount.Owner,
		}
	}
	if declaration.Page != nil {
		result.Page = &routeSurfaceJSONPage{
			Handler: declaration.Page.Handler,
			Adapter: declaration.Page.Adapter,
		}
	}
	if declaration.Fragment != nil {
		result.Fragment = &routeSurfaceJSONFragment{
			Name:    declaration.Fragment.Name,
			Segment: declaration.Fragment.Segment,
			Index:   declaration.Fragment.Index,
			Handler: declaration.Fragment.Handler,
			Adapter: declaration.Fragment.Adapter,
		}
	}
	if declaration.Action != nil {
		result.Action = &routeSurfaceJSONAction{
			Method:  declaration.Action.Method,
			Name:    declaration.Action.Name,
			Segment: declaration.Action.Segment,
			Index:   declaration.Action.Index,
			Writer:  declaration.Action.Writer,
			Handler: declaration.Action.Handler,
			Adapter: declaration.Action.Adapter,
		}
	}
	return result
}

func routeSurfaceJSONLabels(labels []wiring.RouteDeclarationLabel) []routeSurfaceJSONLabel {
	if len(labels) == 0 {
		return []routeSurfaceJSONLabel{}
	}
	result := make([]routeSurfaceJSONLabel, len(labels))
	for index, label := range labels {
		result[index] = routeSurfaceJSONLabel{
			Key:   label.Key,
			Value: label.Value,
		}
	}
	return result
}

func routeSurfaceJSONDestinations(destinations []wiring.RouteDeclarationDestination) []routeSurfaceJSONDestination {
	if len(destinations) == 0 {
		return []routeSurfaceJSONDestination{}
	}
	result := make([]routeSurfaceJSONDestination, len(destinations))
	for index, destination := range destinations {
		result[index] = routeSurfaceJSONDestination{
			Name:     destination.Name,
			Helper:   destination.Helper,
			Target:   destination.Target,
			NavTrail: destination.NavTrail,
		}
	}
	return result
}
