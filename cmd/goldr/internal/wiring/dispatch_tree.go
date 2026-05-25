// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package wiring

import (
	"slices"
	"strconv"
	"strings"
)

type dispatchNode struct {
	name           string
	segment        string
	depth          int
	path           *runtimePath
	staticChildren map[string]*dispatchNode
	dynamicChild   *dispatchNode
}

func buildDispatchTree(paths []runtimePath) *dispatchNode {
	root := &dispatchNode{
		name:           "goldrDispatchRoot",
		staticChildren: make(map[string]*dispatchNode),
	}
	for _, item := range paths {
		routePath := item
		node := root
		for _, segment := range routePath.segments {
			if isParamSegment(segment) {
				if node.dynamicChild == nil {
					node.dynamicChild = &dispatchNode{
						segment:        segment,
						depth:          node.depth + 1,
						staticChildren: make(map[string]*dispatchNode),
					}
				}
				node = node.dynamicChild
				continue
			}
			child := node.staticChildren[segment]
			if child == nil {
				child = &dispatchNode{
					segment:        segment,
					depth:          node.depth + 1,
					staticChildren: make(map[string]*dispatchNode),
				}
				node.staticChildren[segment] = child
			}
			node = child
		}
		node.path = &routePath
	}
	assignDispatchNodeNames(root)
	return root
}

func assignDispatchNodeNames(node *dispatchNode) {
	children := sortedStaticChildren(node)
	baseCounts := make(map[string]int, len(children))
	for _, child := range children {
		baseCounts[dispatchStaticNodeName(node.name, child.segment)]++
	}

	usedNames := make(map[string]int, len(children)+1)
	for _, child := range children {
		name := dispatchStaticNodeName(node.name, child.segment)
		if baseCounts[name] > 1 {
			name += "Segment" + dispatchLiteralSegmentName(child.segment)
		}
		if count := usedNames[name]; count > 0 {
			name += strconv.Itoa(count + 1)
		}
		usedNames[name]++
		child.node.name = name
		assignDispatchNodeNames(child.node)
	}

	if node.dynamicChild != nil {
		node.dynamicChild.name = node.name + "Param" + dispatchSegmentName(node.dynamicChild.segment)
		assignDispatchNodeNames(node.dynamicChild)
	}
}

func dispatchStaticNodeName(parentName string, segment string) string {
	return parentName + "Static" + dispatchSegmentName(segment)
}

func dispatchSegmentName(segment string) string {
	name := exportedSegmentName(segment)
	if name == "" {
		return "Value"
	}
	return name
}

func dispatchLiteralSegmentName(segment string) string {
	var builder strings.Builder
	uppercaseNext := true
	for _, r := range segment {
		switch {
		case r >= 'a' && r <= 'z':
			if uppercaseNext {
				builder.WriteRune(r - 'a' + 'A')
			} else {
				builder.WriteRune(r)
			}
			uppercaseNext = false
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
			uppercaseNext = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			uppercaseNext = false
		case r == '-':
			builder.WriteString("Dash")
			uppercaseNext = true
		case r == '_':
			builder.WriteString("Underscore")
			uppercaseNext = true
		default:
			builder.WriteString("U")
			builder.WriteString(strconv.FormatInt(int64(r), 16))
			uppercaseNext = true
		}
	}
	if builder.Len() == 0 {
		return "Value"
	}
	return builder.String()
}

type dispatchStaticChild struct {
	segment string
	node    *dispatchNode
}

func sortedStaticChildren(node *dispatchNode) []dispatchStaticChild {
	children := make([]dispatchStaticChild, 0, len(node.staticChildren))
	for segment, child := range node.staticChildren {
		children = append(children, dispatchStaticChild{segment: segment, node: child})
	}
	slices.SortFunc(children, func(a, b dispatchStaticChild) int {
		return strings.Compare(a.segment, b.segment)
	})
	return children
}
