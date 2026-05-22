// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import (
	"bytes"
	"context"
	"strings"
	"testing"
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
