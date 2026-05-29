// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package templscan

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseFileCollectsElementAttributes(t *testing.T) {
	path := writeTemplFile(t, `package routes

templ PageView() {
	<section>
		<form
			method="post"
			hx-post={ urls.Users.Create.Path() }
			data-hx-delete="/users/7"
		>
			<button hx-get="/users/table">Load</button>
		</form>
		<script hx-post="/ignored-by-browser"></script>
		if active {
			<a hx-put="/users/7">Save</a>
		} else {
			<a hx-patch={ patchPath }>Patch</a>
		}
	</section>
}
`)

	file, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	var got []Attribute
	for _, attribute := range file.Attributes {
		if strings.Contains(attribute.Name, "hx-") {
			got = append(got, attribute)
		}
	}
	want := []Attribute{
		{Element: "form", Name: "hx-post", Kind: AttributeKindExpression, Value: "urls.Users.Create.Path()", Line: 6, Column: 3},
		{Element: "form", Name: "data-hx-delete", Kind: AttributeKindConstant, Value: "/users/7", Line: 7, Column: 3},
		{Element: "button", Name: "hx-get", Kind: AttributeKindConstant, Value: "/users/table", Line: 9, Column: 11},
		{Element: "script", Name: "hx-post", Kind: AttributeKindConstant, Value: "/ignored-by-browser", Line: 11, Column: 10},
		{Element: "a", Name: "hx-put", Kind: AttributeKindConstant, Value: "/users/7", Line: 13, Column: 6},
		{Element: "a", Name: "hx-patch", Kind: AttributeKindExpression, Value: "patchPath", Line: 15, Column: 6},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("attributes = %#v, want %#v", got, want)
	}
}

func TestParseFileIgnoresCommentsAndText(t *testing.T) {
	path := writeTemplFile(t, `package routes

templ PageView() {
	<section>
		<!-- <button hx-get="/comment">Ignored</button> -->
		<p>hx-post="/text"</p>
		<button hx-get="/real">Load</button>
	</section>
}
`)

	file, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	var got []Attribute
	for _, attribute := range file.Attributes {
		if strings.Contains(attribute.Name, "hx-") {
			got = append(got, attribute)
		}
	}
	want := []Attribute{
		{Element: "button", Name: "hx-get", Kind: AttributeKindConstant, Value: "/real", Line: 6, Column: 10},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("attributes = %#v, want %#v", got, want)
	}
}

func TestScanDirSortsTemplFilesAndSkipsNonTempl(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "b.templ", `package routes

templ B() { <button hx-get="/b">B</button> }
`)
	writeFile(t, root, "a.templ", `package routes

templ A() { <button hx-get="/a">A</button> }
`)
	writeFile(t, root, "ignore.go", "package routes\n")

	files, err := ScanDir(root)
	if err != nil {
		t.Fatalf("ScanDir() error = %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("ScanDir() returned %d files, want 2: %#v", len(files), files)
	}
	if filepath.Base(files[0].Path) != "a.templ" || filepath.Base(files[1].Path) != "b.templ" {
		t.Fatalf("files = %#v, want sorted .templ files", files)
	}
}

func TestParseFileReportsTemplParserErrors(t *testing.T) {
	path := writeTemplFile(t, "package routes\n\ntempl Broken() { <div> }\n")

	_, err := ParseFile(path)
	if err == nil {
		t.Fatal("ParseFile() error = nil, want parse error")
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("ParseFile() error = %v, want path", err)
	}
}

func writeTemplFile(t *testing.T, source string) string {
	t.Helper()

	root := t.TempDir()
	path := filepath.Join(root, "page.templ")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func writeFile(t *testing.T, root string, name string, source string) {
	t.Helper()

	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
