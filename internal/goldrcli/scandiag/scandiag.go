// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package scandiag

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mobiletoly/goldr/internal/goldrcli/appfs"
	"github.com/mobiletoly/goldr/internal/routing"
)

func Error(routesDir string, err error) error {
	return formatError(routesDir, err, err, func(path string, message string) string {
		return fmt.Sprintf("%s: %s", path, message)
	})
}

func CodeError(routesDir string, err error, code string) error {
	return formatError(routesDir, err, fmt.Errorf("%s %w", code, err), func(path string, message string) string {
		return fmt.Sprintf("%s: %s %s", path, code, message)
	})
}

func formatError(routesDir string, err error, fallback error, line func(path string, message string) string) error {
	var scanErr *routing.ScanError
	if !errors.As(err, &scanErr) {
		return fallback
	}

	messages := make([]string, 0, len(scanErr.Problems))
	for _, problem := range scanErr.Problems {
		messages = append(messages, line(appfs.RouteDiagnosticPath(routesDir, problem.Path), problem.Message))
	}
	return errors.New(strings.Join(messages, "\n"))
}
