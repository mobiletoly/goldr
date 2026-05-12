// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/mobiletoly/goldr/internal/goldrcli/ansi"
	"github.com/mobiletoly/goldr/internal/wiring"
	"github.com/urfave/cli/v3"
)

type layoutsOptions struct {
	root string
}

func layoutsCommand() *cli.Command {
	return &cli.Command{
		Name:      "layouts",
		Usage:     "show route layout map",
		UsageText: "goldr routes layouts [--root <dir>]",
		Flags: []cli.Flag{
			rootStringFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runLayouts(ctx, layoutsOptions{
				root: cmd.String(rootFlag),
			}, cmd.Root().Writer)
		},
	}
}

func runLayouts(_ context.Context, options layoutsOptions, writer io.Writer) error {
	paths, manifest, err := scanRouteManifest(options.root)
	if err != nil {
		return fmt.Errorf("goldr routes layouts: %w", err)
	}

	layoutMap, err := wiring.BuildRouteLayoutMap(manifest)
	if err != nil {
		return fmt.Errorf("goldr routes layouts: %w", err)
	}
	if err := renderLayoutMap(writer, paths.routesDir, layoutMap, ansi.ForWriter(writer)); err != nil {
		return fmt.Errorf("goldr routes layouts: %w", err)
	}
	return nil
}

func renderLayoutMap(writer io.Writer, routesDir string, layoutMap wiring.RouteLayoutMap, style ansi.Style) error {
	if layoutMap.Root == nil {
		return nil
	}

	if _, err := fmt.Fprintln(writer, style.Bold("Layout map")); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(writer); err != nil {
		return err
	}
	rootPath := cwdRelativePath(routesDir)
	if !strings.HasSuffix(rootPath, "/") {
		rootPath += "/"
	}
	if _, err := fmt.Fprintln(writer, rootPath); err != nil {
		return err
	}
	if err := renderLayoutMapNode(writer, routesDir, layoutMap.Root, "", true, style); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(writer); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(writer, "Rule:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(writer, "  pages inherit every layout above them"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(writer, "  fragments and actions do not inherit layouts"); err != nil {
		return err
	}
	return nil
}

func renderLayoutMapNode(writer io.Writer, routesDir string, node *wiring.RouteLayoutMapNode, prefix string, last bool, style ansi.Style) error {
	if _, err := fmt.Fprintf(writer, "%s%s%s\n", prefix, layoutMapConnector(last), layoutMapNodeText(routesDir, node, style)); err != nil {
		return err
	}

	childPrefix := prefix + layoutMapChildPrefix(last)
	entryCount := len(node.Pages) + len(node.Children) + len(node.Fragments) + len(node.Actions)
	entryIndex := 0
	for _, page := range node.Pages {
		entryIndex++
		text := layoutMapRouteText("page", page.Methods, page.Route, page.Params, routeSourceDisplayPath(routesDir, page.Source), style)
		if err := renderLayoutMapLeaf(writer, childPrefix, text, entryIndex == entryCount); err != nil {
			return err
		}
	}
	for _, child := range node.Children {
		entryIndex++
		if err := renderLayoutMapNode(writer, routesDir, child, childPrefix, entryIndex == entryCount, style); err != nil {
			return err
		}
	}
	for _, fragment := range node.Fragments {
		entryIndex++
		text := layoutMapRouteText("fragment (not wrapped)", fragment.Methods, fragment.Route, fragment.Params, routeSourceDisplayPath(routesDir, fragment.Source), style)
		if err := renderLayoutMapLeaf(writer, childPrefix, text, entryIndex == entryCount); err != nil {
			return err
		}
	}
	for _, action := range node.Actions {
		entryIndex++
		if err := renderLayoutMapLeaf(writer, childPrefix, layoutMapActionText(routesDir, action, style), entryIndex == entryCount); err != nil {
			return err
		}
	}
	return nil
}

func renderLayoutMapLeaf(writer io.Writer, prefix string, text string, last bool) error {
	_, err := fmt.Fprintf(writer, "%s%s%s\n", prefix, layoutMapConnector(last), text)
	return err
}

func layoutMapConnector(last bool) string {
	if last {
		return "└─ "
	}
	return "├─ "
}

func layoutMapChildPrefix(last bool) string {
	if last {
		return "   "
	}
	return "│  "
}

func layoutMapNodeText(routesDir string, node *wiring.RouteLayoutMapNode, style ansi.Style) string {
	name := node.Name
	if name != "/" {
		name += "/"
	}
	if node.Layout == nil {
		return style.Bold(name)
	}
	return fmt.Sprintf("%s  %s: %s", style.Bold(name), style.Cyan("layout"), routeSourceDisplayPath(routesDir, node.Layout.Source))
}

func layoutMapActionText(routesDir string, action wiring.RouteLayoutMapAction, style ansi.Style) string {
	source := fmt.Sprintf("%s (%s)", routeSourceDisplayPath(routesDir, action.Source), action.Function)
	return layoutMapRouteText("action (not wrapped)", action.Methods, action.Route, action.Params, source, style)
}

func layoutMapRouteText(kind string, methods []string, route string, params []string, source string, style ansi.Style) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "%s: %s %s", layoutMapKindText(kind, style), wiring.FormatRouteSurfaceList(methods), route)
	if len(params) > 0 {
		fmt.Fprintf(&builder, "  %s %s", style.Dim("params:"), wiring.FormatRouteSurfaceList(params))
	}
	fmt.Fprintf(&builder, "  %s", source)
	return builder.String()
}

func layoutMapKindText(kind string, style ansi.Style) string {
	switch kind {
	case "page":
		return style.Green(kind)
	case "fragment (not wrapped)":
		return style.Yellow(kind)
	case "action (not wrapped)":
		return style.Magenta(kind)
	default:
		return kind
	}
}
