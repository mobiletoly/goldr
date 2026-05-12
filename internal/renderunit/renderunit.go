// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package renderunit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/a-h/templ"

	"github.com/mobiletoly/goldr/internal/routing"
)

const (
	KindPage     = "page"
	KindLayout   = "layout"
	KindFragment = "fragment"
)

var ErrNilComponent = errors.New("render templ component: nil component")

type Problem struct {
	Kind       string
	Identifier string
	GoFile     string
	Message    string
}

type ValidationError struct {
	Problems []Problem
}

func (err *ValidationError) Error() string {
	if len(err.Problems) == 0 {
		return "render-unit validation failed"
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "render-unit validation found %d problem(s)", len(err.Problems))
	for _, problem := range err.Problems {
		fmt.Fprintf(&builder, "; %s %s (%s): %s", problem.Kind, problem.Identifier, problem.GoFile, problem.Message)
	}
	return builder.String()
}

func ValidateManifest(manifest routing.Manifest) error {
	var problems []Problem

	for _, page := range manifest.Pages {
		validateUnit(&problems, KindPage, page.Route, page.Unit)
	}
	for _, layout := range manifest.Layouts {
		validateUnit(&problems, KindLayout, layout.RoutePrefix, layout.Unit)
	}
	for _, fragment := range manifest.Fragments {
		validateUnit(&problems, KindFragment, fragment.RoutePrefix+":"+fragment.Name, fragment.Unit)
	}

	if len(problems) > 0 {
		return &ValidationError{Problems: problems}
	}
	return nil
}

func Render(ctx context.Context, writer io.Writer, component templ.Component) error {
	if component == nil {
		return ErrNilComponent
	}
	return component.Render(ctx, writer)
}

func validateUnit(problems *[]Problem, kind, identifier string, unit routing.RenderUnit) {
	if unit.HasTempl && unit.TemplFile != "" {
		return
	}

	*problems = append(*problems, Problem{
		Kind:       kind,
		Identifier: identifier,
		GoFile:     unit.GoFile,
		Message:    "missing matching .templ file",
	})
}
