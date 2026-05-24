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
		{Kind: KindLayout, Identifier: "/users", GoFile: "users/layout.go", Message: "missing matching .templ file"},
		{Kind: KindFragment, Identifier: "/users:row", GoFile: "users/frag_row.go", Message: "missing matching .templ file"},
	}
	if !reflect.DeepEqual(validationErr.Problems, want) {
		t.Fatalf("problems = %#v, want %#v", validationErr.Problems, want)
	}
}

func TestValidateManifestAcceptsPageWithoutTempl(t *testing.T) {
	root := t.TempDir()
	writeRenderUnitFile(t, root, "page.go", `package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(_ *http.Request) goldr.RouteResponse {
	return goldr.Redirect{Location: "/sign-in", Status: http.StatusSeeOther}
}
`)

	manifest := routing.Manifest{
		Root: root,
		Pages: []routing.ManifestPage{
			{
				Route: "/",
				Unit:  routing.RenderUnit{GoFile: "page.go"},
			},
		},
	}

	if err := ValidateManifest(manifest); err != nil {
		t.Fatalf("ValidateManifest() error = %v, want nil", err)
	}
}

func TestValidateManifestChecksPageSignatureWithoutTempl(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name: "invalid signature",
			source: `package routes

import "net/http"

func Page(r *http.Request) string { return "" }
`,
		},
		{
			name: "missing page function",
			source: `package routes

func Helper() {}
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeRenderUnitFile(t, root, "page.go", test.source)

			manifest := routing.Manifest{
				Root: root,
				Pages: []routing.ManifestPage{
					{
						Route: "/",
						Unit:  routing.RenderUnit{GoFile: "page.go"},
					},
				},
			}

			err := ValidateManifest(manifest)
			var validationErr *ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("ValidateManifest() error = %T, want *ValidationError", err)
			}

			want := Problem{
				Kind:       KindPage,
				Identifier: "/",
				GoFile:     "page.go",
				Message:    "page handlers must use func Page(*http.Request) goldr.RouteResponse",
			}
			if len(validationErr.Problems) != 1 || validationErr.Problems[0] != want {
				t.Fatalf("problems = %#v, want %#v", validationErr.Problems, []Problem{want})
			}
		})
	}
}

func TestValidateManifestChecksRenderUnitSignatures(t *testing.T) {
	root := t.TempDir()
	writeRenderUnitFile(t, root, "page.go", `package routes

import "net/http"

func Page(r *http.Request) string { return "" }
`)
	writeRenderUnitFile(t, root, "page.templ", `package routes

templ PageView() {}
`)
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
	writeRenderUnitFile(t, root, "frag_row.go", `package routes

import "net/http"

func FragRow(r *http.Request) string { return "" }
`)
	writeRenderUnitFile(t, root, "frag_row.templ", `package routes

templ FragRowView() {}
`)

	manifest := routing.Manifest{
		Root: root,
		Pages: []routing.ManifestPage{
			{
				Route: "/",
				Unit:  routing.RenderUnit{GoFile: "page.go", TemplFile: "page.templ", HasTempl: true},
			},
		},
		Layouts: []routing.ManifestLayout{
			{
				RoutePrefix: "/",
				Unit:        routing.RenderUnit{GoFile: "layout.go", TemplFile: "layout.templ", HasTempl: true},
			},
		},
		Fragments: []routing.ManifestFragment{
			{
				Name:        "row",
				RoutePrefix: "/",
				Unit:        routing.RenderUnit{GoFile: "frag_row.go", TemplFile: "frag_row.templ", HasTempl: true},
			},
		},
	}

	err := ValidateManifest(manifest)
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("ValidateManifest() error = %T, want *ValidationError", err)
	}

	wantMessages := []string{
		"page handlers must use func Page(*http.Request) goldr.RouteResponse",
		"layouts must use func Layout(*http.Request, goldr.LayoutContext) templ.Component",
		"fragments must use func FragName(*http.Request) goldr.RouteResponse",
	}
	if len(validationErr.Problems) != len(wantMessages) {
		t.Fatalf("problems = %#v, want %d problems", validationErr.Problems, len(wantMessages))
	}
	for index, want := range wantMessages {
		if validationErr.Problems[index].Message != want {
			t.Fatalf("problem %d message = %q, want %q", index, validationErr.Problems[index].Message, want)
		}
	}
}

func TestValidateManifestAcceptsRenderUnitSignatures(t *testing.T) {
	root := t.TempDir()
	writeRenderUnitFile(t, root, "page.go", `package routes

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

func Page(r *http.Request) goldr.RouteResponse {
	return goldr.NewPage(nil, goldr.PageMetadata{})
}
`)
	writeRenderUnitFile(t, root, "page.templ", `package routes

templ PageView() {}
`)
	writeRenderUnitFile(t, root, "layout.go", `package routes

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, layout goldr.LayoutContext) templ.Component {
	return templ.NopComponent
}
`)
	writeRenderUnitFile(t, root, "layout.templ", `package routes

templ LayoutView() {}
`)
	writeRenderUnitFile(t, root, "frag_row.go", `package routes

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func FragRow(r *http.Request) goldr.RouteResponse {
	return goldr.NewFragment(templ.NopComponent)
}
`)
	writeRenderUnitFile(t, root, "frag_row.templ", `package routes

templ FragRowView() {}
`)

	manifest := routing.Manifest{
		Root: root,
		Pages: []routing.ManifestPage{
			{
				Route: "/",
				Unit:  routing.RenderUnit{GoFile: "page.go", TemplFile: "page.templ", HasTempl: true},
			},
		},
		Layouts: []routing.ManifestLayout{
			{
				RoutePrefix: "/",
				Unit:        routing.RenderUnit{GoFile: "layout.go", TemplFile: "layout.templ", HasTempl: true},
			},
		},
		Fragments: []routing.ManifestFragment{
			{
				Name:        "row",
				RoutePrefix: "/",
				Unit:        routing.RenderUnit{GoFile: "frag_row.go", TemplFile: "frag_row.templ", HasTempl: true},
			},
		},
	}

	if err := ValidateManifest(manifest); err != nil {
		t.Fatalf("ValidateManifest() error = %v, want nil", err)
	}
}

func TestValidateManifestAcceptsAliasedSignatureImports(t *testing.T) {
	root := t.TempDir()
	writeRenderUnitFile(t, root, "page.go", `package routes

import (
	stdhttp "net/http"

	g "github.com/mobiletoly/goldr"
)

func Page(r *stdhttp.Request) g.RouteResponse {
	return g.NewPage(nil, g.PageMetadata{})
}
`)
	writeRenderUnitFile(t, root, "page.templ", `package routes

templ PageView() {}
`)
	writeRenderUnitFile(t, root, "layout.go", `package routes

import (
	stdhttp "net/http"

	t "github.com/a-h/templ"
	g "github.com/mobiletoly/goldr"
)

func Layout(r *stdhttp.Request, layout g.LayoutContext) t.Component {
	return t.NopComponent
}
`)
	writeRenderUnitFile(t, root, "layout.templ", `package routes

templ LayoutView() {}
`)
	writeRenderUnitFile(t, root, "frag_row.go", `package routes

import (
	stdhttp "net/http"

	t "github.com/a-h/templ"
	g "github.com/mobiletoly/goldr"
)

func FragRow(r *stdhttp.Request) g.RouteResponse {
	return g.NewFragment(t.NopComponent)
}
`)
	writeRenderUnitFile(t, root, "frag_row.templ", `package routes

templ FragRowView() {}
`)

	manifest := routing.Manifest{
		Root: root,
		Pages: []routing.ManifestPage{
			{
				Route: "/",
				Unit:  routing.RenderUnit{GoFile: "page.go", TemplFile: "page.templ", HasTempl: true},
			},
		},
		Layouts: []routing.ManifestLayout{
			{
				RoutePrefix: "/",
				Unit:        routing.RenderUnit{GoFile: "layout.go", TemplFile: "layout.templ", HasTempl: true},
			},
		},
		Fragments: []routing.ManifestFragment{
			{
				Name:        "row",
				RoutePrefix: "/",
				Unit:        routing.RenderUnit{GoFile: "frag_row.go", TemplFile: "frag_row.templ", HasTempl: true},
			},
		},
	}

	if err := ValidateManifest(manifest); err != nil {
		t.Fatalf("ValidateManifest() error = %v, want nil", err)
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
