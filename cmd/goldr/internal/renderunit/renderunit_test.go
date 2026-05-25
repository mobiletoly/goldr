package renderunit

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/a-h/templ"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/routing"
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
		{Kind: KindLayout, Identifier: "/users", GoFile: "users/layout.go", Message: "missing matching .templ file"},
	}
	if !reflect.DeepEqual(validationErr.Problems, want) {
		t.Fatalf("problems = %#v, want %#v", validationErr.Problems, want)
	}
}

func TestValidateManifestChecksLayoutSignature(t *testing.T) {
	root := t.TempDir()
	writeRenderUnitFile(t, root, "layout.go", `package routes

import (
	"net/http"

	"github.com/a-h/templ"
)

func Layout(r *http.Request) templ.Component { return templ.NopComponent }
`)
	writeRenderUnitFile(t, root, "layout.templ", `package routes

templ LayoutView() {}
`)

	manifest := routing.Manifest{
		Root: root,
		Layouts: []routing.ManifestLayout{
			{
				RoutePrefix: "/",
				Unit:        routing.RenderUnit{GoFile: "layout.go", TemplFile: "layout.templ", HasTempl: true},
			},
		},
	}

	err := ValidateManifest(manifest)
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("ValidateManifest() error = %T, want *ValidationError", err)
	}

	wantMessages := []string{"layouts must use func Layout(*http.Request, goldr.LayoutContext) templ.Component"}
	if len(validationErr.Problems) != len(wantMessages) {
		t.Fatalf("problems = %#v, want %d problems", validationErr.Problems, len(wantMessages))
	}
	for index, want := range wantMessages {
		if validationErr.Problems[index].Message != want {
			t.Fatalf("problem %d message = %q, want %q", index, validationErr.Problems[index].Message, want)
		}
	}
}

func TestValidateManifestAcceptsLayoutSignatures(t *testing.T) {
	tests := []struct {
		name     string
		layoutGo string
	}{
		{
			name: "standard imports",
			layoutGo: `package routes

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, layout goldr.LayoutContext) templ.Component {
	return templ.NopComponent
}
`,
		},
		{
			name: "aliased imports",
			layoutGo: `package routes

import (
	stdhttp "net/http"

	t "github.com/a-h/templ"
	g "github.com/mobiletoly/goldr"
)

func Layout(r *stdhttp.Request, layout g.LayoutContext) t.Component {
	return t.NopComponent
}
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeRenderUnitFile(t, root, "layout.go", test.layoutGo)
			writeRenderUnitFile(t, root, "layout.templ", `package routes

templ LayoutView() {}
`)

			manifest := routing.Manifest{
				Root: root,
				Layouts: []routing.ManifestLayout{
					{
						RoutePrefix: "/",
						Unit:        routing.RenderUnit{GoFile: "layout.go", TemplFile: "layout.templ", HasTempl: true},
					},
				},
			}

			if err := ValidateManifest(manifest); err != nil {
				t.Fatalf("ValidateManifest() error = %v, want nil", err)
			}
		})
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

func writeRenderUnitFile(t *testing.T, root, relPath, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
