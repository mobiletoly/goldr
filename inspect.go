// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/a-h/templ"
)

const templateInspectorScriptPath = "/goldr/goldr-template-inspector.js"

var (
	errLabeledComponentEmptyLabel = errors.New("goldr labeled component: empty label")
	errLabeledComponentNil        = errors.New("goldr labeled component: nil component")
)

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

// LabeledComponent labels one existing templ component render for template
// inspection.
//
// With template inspection off, LabeledComponent renders exactly the wrapped
// component. With comments or overlay mode enabled, it emits paired inspector
// comments around the wrapped component.
func LabeledComponent(label string, component templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		if strings.TrimSpace(label) == "" {
			return errLabeledComponentEmptyLabel
		}
		if component == nil {
			return errLabeledComponentNil
		}
		if TemplateInspectionFromContext(ctx) == TemplateInspectionOff {
			return component.Render(ctx, writer)
		}

		id := labeledComponentMarkerID(label)
		start := "<!--goldr:start id=" + id +
			" kind=component" +
			" label=" + inspectorCommentValue(label) +
			"-->"
		end := "<!--goldr:end id=" + id + "-->"

		if _, err := io.WriteString(writer, start); err != nil {
			return err
		}
		if err := component.Render(ctx, writer); err != nil {
			return err
		}
		_, err := io.WriteString(writer, end)
		return err
	})
}

func inspectorCommentValue(value string) string {
	value = strings.ReplaceAll(value, "%", "%25")
	value = strings.ReplaceAll(value, " ", "%20")
	value = strings.ReplaceAll(value, "--", "%2D%2D")
	value = strings.ReplaceAll(value, ">", "%3E")
	return value
}

func labeledComponentMarkerID(label string) string {
	var builder strings.Builder
	lastUnderscore := false
	for index := 0; index < len(label); index++ {
		char := label[index]
		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteByte(char)
			lastUnderscore = false
		case char >= 'A' && char <= 'Z':
			builder.WriteByte(char + ('a' - 'A'))
			lastUnderscore = false
		case char >= '0' && char <= '9':
			builder.WriteByte(char)
			lastUnderscore = false
		default:
			if builder.Len() > 0 && !lastUnderscore {
				builder.WriteByte('_')
				lastUnderscore = true
			}
		}
	}

	slug := strings.Trim(builder.String(), "_")
	if slug == "" {
		return "g_component"
	}
	return "g_component_" + slug
}
