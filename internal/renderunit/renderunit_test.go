package renderunit

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/a-h/templ"

	"github.com/mobiletoly/goldr/internal/routing"
)

func TestValidateManifestAcceptsCompleteRenderUnitPairs(t *testing.T) {
	manifest := routing.Manifest{
		Pages: []routing.ManifestPage{
			{
				Route: "/",
				Unit:  routing.RenderUnit{GoFile: "page.go", TemplFile: "page.templ", HasTempl: true},
			},
		},
		Layouts: []routing.ManifestLayout{
			{
				RoutePrefix: "/users",
				Unit:        routing.RenderUnit{GoFile: "users/layout.go", TemplFile: "users/layout.templ", HasTempl: true},
			},
		},
		Fragments: []routing.ManifestFragment{
			{
				Name:        "row",
				RoutePrefix: "/users",
				Unit:        routing.RenderUnit{GoFile: "users/frag_row.go", TemplFile: "users/frag_row.templ", HasTempl: true},
			},
		},
	}

	if err := ValidateManifest(manifest); err != nil {
		t.Fatalf("ValidateManifest() error = %v, want nil", err)
	}
}

func TestValidateManifestCollectsMissingTemplPairs(t *testing.T) {
	manifest := routing.Manifest{
		Pages: []routing.ManifestPage{
			{
				Route: "/",
				Unit:  routing.RenderUnit{GoFile: "page.go"},
			},
		},
		Layouts: []routing.ManifestLayout{
			{
				RoutePrefix: "/users",
				Unit:        routing.RenderUnit{GoFile: "users/layout.go"},
			},
		},
		Fragments: []routing.ManifestFragment{
			{
				Name:        "row",
				RoutePrefix: "/users",
				Unit:        routing.RenderUnit{GoFile: "users/frag_row.go"},
			},
		},
	}

	err := ValidateManifest(manifest)
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("ValidateManifest() error = %T, want *ValidationError", err)
	}

	want := []Problem{
		{Kind: KindPage, Identifier: "/", GoFile: "page.go", Message: "missing matching .templ file"},
		{Kind: KindLayout, Identifier: "/users", GoFile: "users/layout.go", Message: "missing matching .templ file"},
		{Kind: KindFragment, Identifier: "/users:row", GoFile: "users/frag_row.go", Message: "missing matching .templ file"},
	}
	if !reflect.DeepEqual(validationErr.Problems, want) {
		t.Fatalf("problems = %#v, want %#v", validationErr.Problems, want)
	}
}

func TestRenderWritesComponentOutput(t *testing.T) {
	component := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "hello")
		return err
	})
	var buffer bytes.Buffer

	if err := Render(context.Background(), &buffer, component); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}
	if buffer.String() != "hello" {
		t.Fatalf("output = %q, want %q", buffer.String(), "hello")
	}
}

func TestRenderRejectsNilComponent(t *testing.T) {
	err := Render(context.Background(), io.Discard, nil)
	if !errors.Is(err, ErrNilComponent) {
		t.Fatalf("Render() error = %v, want ErrNilComponent", err)
	}
}

func TestRenderReturnsComponentErrors(t *testing.T) {
	componentErr := errors.New("component failed")
	component := templ.ComponentFunc(func(ctx context.Context, writer io.Writer) error {
		return componentErr
	})

	err := Render(context.Background(), io.Discard, component)
	if !errors.Is(err, componentErr) {
		t.Fatalf("Render() error = %v, want component error", err)
	}
}
