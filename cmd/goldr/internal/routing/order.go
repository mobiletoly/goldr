// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"cmp"
	"strings"
)

func compareRouteOrder(routeA, fileA, routeB, fileB string) int {
	if result := cmp.Compare(routeA, routeB); result != 0 {
		return result
	}
	if isMountSource(fileA) != isMountSource(fileB) {
		if isMountSource(fileA) {
			return 1
		}
		return -1
	}
	return cmp.Compare(fileA, fileB)
}

func isMountSource(file string) bool {
	return strings.HasPrefix(file, "../mounts/")
}

func compareFragmentOrder(prefixA, nameA, fileA, prefixB, nameB, fileB string) int {
	if result := cmp.Compare(prefixA, prefixB); result != 0 {
		return result
	}
	if result := cmp.Compare(nameA, nameB); result != 0 {
		return result
	}
	return cmp.Compare(fileA, fileB)
}

func compareActionOrder(routeA, methodA, functionA, routeB, methodB, functionB string) int {
	if result := cmp.Compare(routeA, routeB); result != 0 {
		return result
	}
	if result := cmp.Compare(methodA, methodB); result != 0 {
		return result
	}
	return cmp.Compare(functionA, functionB)
}
