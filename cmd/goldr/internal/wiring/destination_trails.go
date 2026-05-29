// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package wiring

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
)

type destinationTrailEdge struct {
	sourceRoute string
	name        string
	symbolName  string
	targetRoute string
	trailKey    string
}

func destinationTrailEdges(declarations []routing.ManifestRouteDeclaration) ([]destinationTrailEdge, error) {
	targets := routeTargetsByAPIChain(declarations)
	var edges []destinationTrailEdge
	for _, declaration := range declarations {
		for _, destination := range declaration.Destinations {
			if destination.TrailKey == "" {
				continue
			}
			targetRoute, ok := targets[routeTargetKey(destination.Target)]
			if !ok {
				return nil, fmt.Errorf("%w: destination %q targets unknown route helper %s", ErrAmbiguousURLHelper, destination.Name, strings.Join(destination.Target, "."))
			}
			edges = append(edges, destinationTrailEdge{
				sourceRoute: declaration.Route,
				name:        destination.Name,
				symbolName:  destination.SymbolName,
				targetRoute: targetRoute,
				trailKey:    destination.TrailKey,
			})
		}
	}
	slices.SortFunc(edges, func(a, b destinationTrailEdge) int {
		if a.targetRoute != b.targetRoute {
			return strings.Compare(a.targetRoute, b.targetRoute)
		}
		if a.trailKey != b.trailKey {
			return strings.Compare(a.trailKey, b.trailKey)
		}
		if a.sourceRoute != b.sourceRoute {
			return strings.Compare(a.sourceRoute, b.sourceRoute)
		}
		return strings.Compare(a.name, b.name)
	})
	return edges, nil
}

func destinationTrailKeysByRoute(declarations []routing.ManifestRouteDeclaration) (map[string][]string, error) {
	edges, err := destinationTrailEdges(declarations)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]string)
	for _, edge := range edges {
		keys, err := mergeDestinationTrailKeys(result[edge.targetRoute], []string{edge.trailKey})
		if err != nil {
			return nil, err
		}
		result[edge.targetRoute] = keys
	}
	return result, nil
}

func inboundDestinationTrailEdgesByRoute(declarations []routing.ManifestRouteDeclaration) (map[string][]destinationTrailEdge, error) {
	edges, err := destinationTrailEdges(declarations)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]destinationTrailEdge)
	for _, edge := range edges {
		result[edge.targetRoute] = append(result[edge.targetRoute], edge)
	}
	return result, nil
}

func routeTargetsByAPIChain(declarations []routing.ManifestRouteDeclaration) map[string]string {
	result := make(map[string]string, len(declarations))
	for _, declaration := range declarations {
		result[routeTargetKey(routeAPIChain(declaration.Route))] = declaration.Route
	}
	return result
}

func routeAPIChain(route string) []string {
	if route == "/" {
		return []string{"Root"}
	}
	segments := routeSegments(route)
	chain := make([]string, 0, len(segments))
	for _, segment := range segments {
		if paramName, ok := paramSegmentName(segment); ok {
			chain = append(chain, "By"+exportedSegmentName(paramName))
			continue
		}
		chain = append(chain, exportedSegmentName(segment))
	}
	return chain
}

func routeTargetKey(target []string) string {
	return strings.Join(target, "\x00")
}

func mergeDestinationTrailKeys(existing []string, incoming []string) ([]string, error) {
	if len(incoming) == 0 {
		return existing, nil
	}
	values := slices.Clone(existing)
	seen := make(map[string]bool, len(existing)+len(incoming))
	fieldNames := make(map[string]string, len(existing)+len(incoming))
	for _, key := range existing {
		seen[key] = true
		fieldNames[urlTrailKeyFieldName(key)] = key
	}
	for _, key := range incoming {
		if seen[key] {
			continue
		}
		fieldName := urlTrailKeyFieldName(key)
		if previous, ok := fieldNames[fieldName]; ok {
			return nil, fmt.Errorf("%w: trail keys %q and %q both map to %s", ErrAmbiguousURLHelper, previous, key, fieldName)
		}
		seen[key] = true
		fieldNames[fieldName] = key
		values = append(values, key)
	}
	slices.Sort(values)
	return values, nil
}
