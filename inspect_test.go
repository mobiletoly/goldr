// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestTemplateInspectorRendersOnlyInOverlayMode(t *testing.T) {
	for _, test := range []struct {
		name string
		ctx  context.Context
		want string
	}{
		{name: "default", ctx: context.Background()},
		{name: "off", ctx: WithTemplateInspection(context.Background(), TemplateInspectionOff)},
		{name: "comments", ctx: WithTemplateInspection(context.Background(), TemplateInspectionComments)},
		{
			name: "overlay",
			ctx:  WithTemplateInspection(context.Background(), TemplateInspectionOverlay),
			want: `<script src="/goldr/goldr-template-inspector.js" defer></script>`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var body bytes.Buffer
			if err := TemplateInspector().Render(test.ctx, &body); err != nil {
				t.Fatalf("Render() error = %v, want nil", err)
			}
			if body.String() != test.want {
				t.Fatalf("body = %q, want %q", body.String(), test.want)
			}
		})
	}
}

func TestTemplateInspectionFromContext(t *testing.T) {
	if got := TemplateInspectionFromContext(context.Background()); got != TemplateInspectionOff {
		t.Fatalf("default mode = %d, want %d", got, TemplateInspectionOff)
	}

	ctx := WithTemplateInspection(context.Background(), TemplateInspectionOverlay)
	if got := TemplateInspectionFromContext(ctx); got != TemplateInspectionOverlay {
		t.Fatalf("mode = %d, want %d", got, TemplateInspectionOverlay)
	}
}

func TestTemplateInspectorScriptPathIsGoldrBrowserHelper(t *testing.T) {
	var body bytes.Buffer
	if err := TemplateInspector().Render(WithTemplateInspection(context.Background(), TemplateInspectionOverlay), &body); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}
	if !strings.Contains(body.String(), "/goldr/") {
		t.Fatalf("body = %q, want /goldr/ helper path", body.String())
	}
}

func TestLabeledComponentRendersWrappedComponentWhenInspectionOff(t *testing.T) {
	var body bytes.Buffer
	if err := LabeledComponent("User directory", textComponent("<section>Users</section>")).Render(context.Background(), &body); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}
	if got := body.String(); got != "<section>Users</section>" {
		t.Fatalf("body = %q, want wrapped component output", got)
	}
	if strings.Contains(body.String(), "goldr:start") {
		t.Fatalf("body = %q, want no inspector comments", body.String())
	}
}

func TestLabeledComponentRendersCommentsWhenInspectionEnabled(t *testing.T) {
	for _, test := range []struct {
		name string
		mode TemplateInspectionMode
	}{
		{name: "comments", mode: TemplateInspectionComments},
		{name: "overlay", mode: TemplateInspectionOverlay},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := WithTemplateInspection(context.Background(), test.mode)
			var body bytes.Buffer
			if err := LabeledComponent("User directory", textComponent("<section>Users</section>")).Render(ctx, &body); err != nil {
				t.Fatalf("Render() error = %v, want nil", err)
			}

			want := `<!--goldr:start id=g_component_user_directory kind=component label=User%20directory--><section>Users</section><!--goldr:end id=g_component_user_directory-->`
			if got := body.String(); got != want {
				t.Fatalf("body = %q, want %q", got, want)
			}
		})
	}
}

func TestLabeledComponentEncodesCommentSensitiveLabel(t *testing.T) {
	ctx := WithTemplateInspection(context.Background(), TemplateInspectionComments)
	var body bytes.Buffer
	if err := LabeledComponent("A -- B > C % done", textComponent("ok")).Render(ctx, &body); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	got := body.String()
	for _, want := range []string{
		"id=g_component_a_b_c_done",
		"label=A%20%2D%2D%20B%20%3E%20C%20%25%20done",
		">ok<!--goldr:end id=g_component_a_b_c_done-->",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("body = %q, want %q", got, want)
		}
	}
	if strings.Contains(got, "A -- B > C") {
		t.Fatalf("body = %q, want encoded label in marker", got)
	}
}

func TestLabeledComponentRejectsEmptyLabel(t *testing.T) {
	for _, label := range []string{"", "   "} {
		t.Run("label="+label, func(t *testing.T) {
			var body bytes.Buffer
			err := LabeledComponent(label, textComponent("ok")).Render(context.Background(), &body)
			if !errors.Is(err, errLabeledComponentEmptyLabel) {
				t.Fatalf("Render() error = %v, want %v", err, errLabeledComponentEmptyLabel)
			}
			if body.Len() != 0 {
				t.Fatalf("body = %q, want empty", body.String())
			}
		})
	}
}

func TestLabeledComponentRejectsNilComponent(t *testing.T) {
	var body bytes.Buffer
	err := LabeledComponent("User directory", nil).Render(context.Background(), &body)
	if !errors.Is(err, errLabeledComponentNil) {
		t.Fatalf("Render() error = %v, want %v", err, errLabeledComponentNil)
	}
	if body.Len() != 0 {
		t.Fatalf("body = %q, want empty", body.String())
	}
}

func TestLabeledComponentReturnsWrappedComponentError(t *testing.T) {
	wantErr := errors.New("render failed")
	component := templ.ComponentFunc(func(context.Context, io.Writer) error {
		return wantErr
	})

	var body bytes.Buffer
	err := LabeledComponent("User directory", component).Render(context.Background(), &body)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Render() error = %v, want %v", err, wantErr)
	}
	if body.Len() != 0 {
		t.Fatalf("body = %q, want empty", body.String())
	}
}

func textComponent(body string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, body)
		return err
	})
}
