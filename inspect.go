// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import (
	"context"
	"io"

	"github.com/a-h/templ"
)

const templateInspectorScriptPath = "/goldr/goldr-template-inspector.js"

type templateInspectionContextKey struct{}

// TemplateInspectionMode selects development template inspection behavior.
type TemplateInspectionMode uint8

const (
	// TemplateInspectionOff disables template inspection.
	TemplateInspectionOff TemplateInspectionMode = iota
	// TemplateInspectionComments emits inspector comments without browser UI.
	TemplateInspectionComments
	// TemplateInspectionOverlay emits inspector comments and enables the
	// template inspector browser helper.
	TemplateInspectionOverlay
)

// WithTemplateInspection returns a context carrying mode.
func WithTemplateInspection(ctx context.Context, mode TemplateInspectionMode) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, templateInspectionContextKey{}, mode)
}

// TemplateInspectionFromContext returns the template inspection mode in ctx.
func TemplateInspectionFromContext(ctx context.Context) TemplateInspectionMode {
	if ctx == nil {
		return TemplateInspectionOff
	}
	mode, _ := ctx.Value(templateInspectionContextKey{}).(TemplateInspectionMode)
	return mode
}

// TemplateInspector renders the development browser overlay helper in overlay
// mode.
func TemplateInspector() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		if TemplateInspectionFromContext(ctx) != TemplateInspectionOverlay {
			return nil
		}
		_, err := io.WriteString(writer, `<script src="`+templateInspectorScriptPath+`" defer></script>`)
		return err
	})
}
