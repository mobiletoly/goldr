// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package wiring

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"go/token"
	"regexp"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/renderunit"
	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
)

const (
	GeneratedFileName          = "goldr_gen.go"
	InspectorGeneratedFileName = "goldr_gen.go"
)

var (
	ErrInvalidPackageName         = errors.New("invalid generated package name")
	ErrInvalidRouteRootImportPath = errors.New("invalid route root import path")
	ErrAmbiguousRuntimeRoute      = errors.New("ambiguous runtime route")
	ErrAmbiguousPageRoute         = ErrAmbiguousRuntimeRoute

	packageNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

type GenerateOptions struct {
	PackageName         string
	RouteRootImportPath string
	InspectorImportPath string
}

type GeneratedFragmentWrappersFile struct {
	Dir     string
	Content []byte
}

func GenerateManifest(manifest routing.Manifest, options GenerateOptions) ([]byte, error) {
	if !isPackageName(options.PackageName) {
		return nil, fmt.Errorf("%w %q: must be a lowercase Go-safe identifier", ErrInvalidPackageName, options.PackageName)
	}
	if err := renderunit.ValidateManifest(manifest); err != nil {
		return nil, err
	}

	routes, err := runtimeRoutes(manifest)
	if err != nil {
		return nil, err
	}
	rootRoutes := rootRouteDeclarations(manifest.Routes)
	adapterImports, err := routeAdapterImports(manifest.Root, rootRoutes, options.RouteRootImportPath)
	if err != nil {
		return nil, err
	}
	rootLayouts := layoutStack("/", manifest.Layouts)
	additionalPlans := additionalPagePlans(manifest.Layouts, manifest.Middlewares)
	additionalErrorLayouts := layoutStack("/", additionalPageLayouts(manifest.Layouts))
	importPlan, err := routeImports(routes, rootLayouts, additionalPlans, adapterImports, options.RouteRootImportPath)
	if err != nil {
		return nil, err
	}
	inspectorImportPath := options.InspectorImportPath
	if inspectorImportPath == "" {
		inspectorImportPath = defaultInspectorImportPath(options.RouteRootImportPath)
	}
	if inspectorImportPath == "" {
		return nil, ErrInvalidRouteRootImportPath
	}
	inboundDestinations, err := inboundDestinationTrailEdgesByRoute(manifest.Routes)
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	writeGeneratedFileHeader(&buffer, routeSurfaceRows(manifest, routes, inboundDestinations))
	fmt.Fprintf(&buffer, "package %s\n\n", options.PackageName)
	writeImports(&buffer, importPlan.Imports, adapterImports, inspectorImportPath, hasDynamicRoutes(routes), true, true, hasSegmentRoutes(routes) || hasRequestNavRoutes(routes) || hasNonRootAdditionalPagePlan(additionalPlans), hasRequestNavRoutes(routes), true)
	writeTypes(&buffer, true)
	writeManifestValue(&buffer, manifest)
	writeHandler(&buffer, routes, rootLayouts, additionalPlans, additionalErrorLayouts, importPlan.Aliases)
	if err := writeRouteDeclarationReference(&buffer, rootRoutes, options.RouteRootImportPath); err != nil {
		return nil, err
	}
	if err := writeRouteAdapterFunctions(&buffer, manifest.Root, rootRoutes, options.RouteRootImportPath); err != nil {
		return nil, err
	}
	writeFragmentWrapperFunctions(&buffer, rootFragments(fragmentWrappers(manifest)))

	source, err := format.Source(buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated manifest: %w", err)
	}
	return source, nil
}

func hasNonRootAdditionalPagePlan(plans []additionalPagePlan) bool {
	return len(plans) > 1
}

func isPackageName(value string) bool {
	return packageNamePattern.MatchString(value) && !token.Lookup(value).IsKeyword()
}
