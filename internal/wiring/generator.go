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

	"github.com/mobiletoly/goldr/internal/renderunit"
	"github.com/mobiletoly/goldr/internal/routing"
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
	imports, err := routeImports(routes, options.RouteRootImportPath)
	if err != nil {
		return nil, err
	}
	inspectorImportPath := options.InspectorImportPath
	if inspectorImportPath == "" {
		inspectorImportPath = defaultInspectorImportPath(options.RouteRootImportPath)
	}
	if inspectorImportPath == "" && len(routes) > 0 {
		return nil, ErrInvalidRouteRootImportPath
	}

	var buffer bytes.Buffer
	writeGeneratedFileHeader(&buffer, routeSurfaceRows(manifest, routes))
	fmt.Fprintf(&buffer, "package %s\n\n", options.PackageName)
	needsRouteRenderer := hasRenderRoutes(routes) || hasActionRoutes(routes)
	writeImports(&buffer, imports, inspectorImportPath, hasDynamicRoutes(routes), len(routes) > 0, needsRouteRenderer, hasSegmentRoutes(routes), len(routes) > 0)
	writeTypes(&buffer, len(routes) > 0)
	writeManifestValue(&buffer, manifest)
	if len(routes) > 0 {
		writeHandler(&buffer, routes)
	}
	writeFragmentWrapperFunctions(&buffer, rootFragments(manifest.Fragments))

	source, err := format.Source(buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated manifest: %w", err)
	}
	return source, nil
}

func isPackageName(value string) bool {
	return packageNamePattern.MatchString(value) && !token.Lookup(value).IsKeyword()
}
