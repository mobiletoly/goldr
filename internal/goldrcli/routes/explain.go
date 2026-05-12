// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/mobiletoly/goldr/internal/goldrcli/ansi"
	"github.com/mobiletoly/goldr/internal/wiring"
	"github.com/urfave/cli/v3"
)

const explainMethodFlag = "method"

type explainOptions struct {
	root   string
	method string
	target string
}

func explainCommand() *cli.Command {
	return &cli.Command{
		Name:      "explain",
		Usage:     "explain which route matches a URL or path",
		UsageText: "goldr routes explain [--root <dir>] [--method <method>] <url-or-path>",
		Flags: []cli.Flag{
			rootStringFlag(),
			&cli.StringFlag{
				Name:        explainMethodFlag,
				Value:       "GET",
				Usage:       "HTTP method to explain",
				HideDefault: false,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() != 1 {
				return fmt.Errorf("goldr routes explain: expected one URL or path")
			}
			return runExplain(ctx, explainOptions{
				root:   cmd.String(rootFlag),
				method: cmd.String(explainMethodFlag),
				target: cmd.Args().First(),
			}, cmd.Root().Writer)
		},
	}
}

func runExplain(_ context.Context, options explainOptions, writer io.Writer) error {
	escapedPath, err := explainTargetPath(options.target)
	if err != nil {
		return fmt.Errorf("goldr routes explain: %w", err)
	}

	paths, manifest, err := scanRouteManifest(options.root)
	if err != nil {
		return fmt.Errorf("goldr routes explain: %w", err)
	}

	explanation, err := wiring.ExplainRoute(manifest, options.method, escapedPath)
	if err != nil {
		return fmt.Errorf("goldr routes explain: %w", err)
	}

	switch explanation.Status {
	case wiring.RouteExplainStatusMatched:
		if err := renderExplainOutput(writer, paths.routesDir, explanation, ansi.ForWriter(writer)); err != nil {
			return fmt.Errorf("goldr routes explain: %w", err)
		}
		return nil
	case wiring.RouteExplainStatusMethodNotAllowed:
		return fmt.Errorf(
			"goldr routes explain: %s %s: method not allowed (allowed: %s)",
			explanation.Method,
			explanation.Path,
			wiring.FormatRouteSurfaceList(explanation.AllowedMethods),
		)
	default:
		return fmt.Errorf("goldr routes explain: %s %s: no route matches path", explanation.Method, explanation.Path)
	}
}

func explainTargetPath(target string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", fmt.Errorf("missing URL or path")
	}

	parsed, err := url.Parse(target)
	if err != nil {
		return "", fmt.Errorf("invalid URL or path %q: %w", target, err)
	}
	if parsed.Opaque != "" {
		return "", fmt.Errorf("invalid URL or path %q", target)
	}

	escapedPath := parsed.EscapedPath()
	if escapedPath == "" && parsed.IsAbs() && parsed.Host != "" {
		escapedPath = "/"
	}
	if !strings.HasPrefix(escapedPath, "/") {
		return "", fmt.Errorf("URL or path must start with /")
	}
	return escapedPath, nil
}

func renderExplainOutput(writer io.Writer, routesDir string, explanation wiring.RouteExplanation, style ansi.Style) error {
	if _, err := fmt.Fprintf(writer, "%s  %s\n\n", explanation.Path, explanation.Method); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(writer, style.Bold("MATCH")); err != nil {
		return err
	}

	match := explanation.Match
	if _, err := fmt.Fprintf(writer, "  %s %s\n", explainKindText(match.Kind, style), match.Route); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "  %-8s %s\n", "source", explainSourceText(routesDir, match)); err != nil {
		return err
	}
	if len(match.Params) > 0 {
		if _, err := fmt.Fprintf(writer, "  %-8s %s\n", "params", explainParamsText(match.Params)); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(writer); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(writer, style.Bold("LAYOUT STACK")); err != nil {
		return err
	}
	if match.Kind != wiring.RouteSurfaceKindPage {
		_, err := fmt.Fprintln(writer, "  not layout-wrapped")
		return err
	}
	if len(match.Layouts) == 0 {
		_, err := fmt.Fprintln(writer, "  none")
		return err
	}

	width := explainLayoutPrefixWidth(match.Layouts)
	for _, layout := range match.Layouts {
		if _, err := fmt.Fprintf(
			writer,
			"  %-*s %s\n",
			width,
			layout.RoutePrefix,
			routeSourceDisplayPath(routesDir, layout.Source),
		); err != nil {
			return err
		}
	}
	return nil
}

func explainKindText(kind string, style ansi.Style) string {
	text := fmt.Sprintf("%-8s", kind)
	switch kind {
	case wiring.RouteSurfaceKindPage:
		return style.Green(text)
	case wiring.RouteSurfaceKindFragment:
		return style.Yellow(text)
	case wiring.RouteSurfaceKindAction:
		return style.Magenta(text)
	default:
		return text
	}
}

func explainSourceText(routesDir string, match wiring.RouteExplanationMatch) string {
	source := routeSourceDisplayPath(routesDir, match.Source)
	if match.Function == "" {
		return source
	}
	return fmt.Sprintf("%s (%s)", source, match.Function)
}

func explainParamsText(params []wiring.RouteExplanationParam) string {
	parts := make([]string, 0, len(params))
	for _, param := range params {
		parts = append(parts, fmt.Sprintf("%s = %s", param.Name, strconv.Quote(param.Value)))
	}
	return strings.Join(parts, ", ")
}

func explainLayoutPrefixWidth(layouts []wiring.RouteExplanationLayout) int {
	width := 0
	for _, layout := range layouts {
		width = max(width, len(layout.RoutePrefix))
	}
	return width
}
