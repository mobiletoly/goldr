// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package content

import (
	"bytes"
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func TestCheckRejectsInvalidTrees(t *testing.T) {
	tests := []struct {
		name  string
		files fstest.MapFS
		want  string
	}{
		{"root entry", entryFS(".", `{"title":"Root"}`, "body.html", "root"), "filesystem root"},
		{"missing metadata", entryFS("about", "", "body.html", "body"), "missing page.json"},
		{"missing body", entryFS("about", `{"title":"About"}`, "", ""), "exactly one"},
		{"ambiguous body", fstest.MapFS{
			"about/page.json": {Data: []byte(`{"title":"About"}`)},
			"about/body.html": {Data: []byte("html")},
			"about/body.md":   {Data: []byte("markdown")},
		}, "exactly one"},
		{"unknown metadata", entryFS("about", `{"title":"About","extra":"x"}`, "body.html", "body"), "unknown metadata field extra"},
		{"duplicate metadata", entryFS("about", `{"title":"One","title":"Two"}`, "body.html", "body"), "duplicate metadata field title"},
		{"null metadata", entryFS("about", `{"title":null}`, "body.html", "body"), "title must be a string"},
		{"wrong metadata type", entryFS("about", `{"title":42}`, "body.html", "body"), "title must be a string"},
		{"nonobject metadata", entryFS("about", `[]`, "body.html", "body"), "one JSON object"},
		{"malformed metadata", entryFS("about", `{"title":`, "body.html", "body"), "decode metadata field title"},
		{"trailing metadata", entryFS("about", `{"title":"About"} {}`, "body.html", "body"), "trailing JSON value"},
		{"blank title", entryFS("about", `{"title":"  "}`, "body.html", "body"), "nonblank"},
		{"invalid directory", entryFS("Bad", `{"title":"Bad"}`, "body.html", "body"), "Bad"},
		{"double hyphen directory", entryFS("not--valid", `{"title":"Bad"}`, "body.html", "body"), "directory name"},
		{"empty body", entryFS("about", `{"title":"About"}`, "body.html", " \n\t"), "whitespace-only"},
		{"metadata invalid UTF-8", entryFSBytes("about", []byte{'{', '"', 't', 'i', 't', 'l', 'e', '"', ':', '"', 0xff, '"', '}'}, "body.html", []byte("body")), "valid UTF-8"},
		{"body invalid UTF-8", entryFSBytes("about", []byte(`{"title":"About"}`), "body.html", []byte{0xff}), "valid UTF-8"},
		{"symlink", fstest.MapFS{
			"about/page.json": {Data: []byte(`{"title":"About"}`)},
			"about/body.html": {Data: []byte("target"), Mode: 0o777 | fs.ModeSymlink},
		}, "symbolic links"},
		{"special file", fstest.MapFS{
			"about/page.json": {Data: []byte(`{"title":"About"}`)},
			"about/body.html": {Data: []byte("body")},
			"about/socket":    {Mode: fs.ModeSocket},
		}, "special files"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := Check(test.files)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Check() error = %v, want containing %q", err, test.want)
			}
			if _, newErr := New(Config{FS: test.files}); newErr == nil {
				t.Fatal("New() error = nil, want invalid-tree error")
			}
		})
	}
}

func TestCheckEnforcesInputLimits(t *testing.T) {
	metadata := bytes.Repeat([]byte(" "), maxMetadataBytes+1)
	files := entryFSBytes("about", metadata, "body.html", []byte("body"))
	if err := Check(files); err == nil || !strings.Contains(err.Error(), fmt.Sprintf("%d byte limit", maxMetadataBytes)) {
		t.Fatalf("metadata limit error = %v", err)
	}

	body := bytes.Repeat([]byte("a"), maxBodyBytes+1)
	files = entryFSBytes("about", []byte(`{"title":"About"}`), "body.html", body)
	if err := Check(files); err == nil || !strings.Contains(err.Error(), fmt.Sprintf("%d byte limit", maxBodyBytes)) {
		t.Fatalf("body limit error = %v", err)
	}

	body = bytes.Repeat([]byte("a"), maxBodyBytes)
	files = entryFSBytes("about", []byte(`{"title":"About"}`), "body.html", body)
	if err := Check(files); err != nil {
		t.Fatalf("exact body limit error = %v, want nil", err)
	}
}

func TestTrustedHTMLAndMarkdownPassThrough(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		body     string
		contains []string
	}{
		{
			name:     "HTML is inserted without validation or normalization",
			filename: "body.html",
			body:     `<script data-token="trusted">window.example = true</script><p ID="One" id="Two"><span>unclosed`,
			contains: []string{
				`<article class="goldr-content">`,
				`<script data-token="trusted">window.example = true</script><p ID="One" id="Two"><span>unclosed`,
				`</article>`,
			},
		},
		{
			name:     "Markdown emits raw HTML and dangerous destinations",
			filename: "body.md",
			body:     "<section data-block=\"trusted\">block</section>\n\nInline <span onclick=\"trusted()\">HTML</span> and [link](javascript:trusted()).\n",
			contains: []string{
				`<article class="goldr-content">`,
				`<section data-block="trusted">block</section>`,
				`Inline <span onclick="trusted()">HTML</span> and <a href="javascript:trusted()">link</a>.`,
				`</article>`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pages, err := New(Config{FS: entryFS("about", `{"title":"About"}`, test.filename, test.body)})
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			body := renderPage(t, mustPage(t, pages, "/about"))
			for _, want := range test.contains {
				if !strings.Contains(body, want) {
					t.Fatalf("body missing %q: %s", want, body)
				}
			}
		})
	}
}

func TestMarkdownRendersGFM(t *testing.T) {
	pages, err := New(Config{FS: entryFS("about", `{"title":"About"}`, "body.md", `| Feature | State |
| --- | --- |
| Content pages | Ready |

- [x] Trusted HTML
- [ ] Automatic publishing

~~Obsolete guidance~~

https://example.com/docs
`)})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	body := renderPage(t, mustPage(t, pages, "/about"))
	for _, want := range []string{
		"<table>",
		"<th>Feature</th>",
		"<input ",
		`type="checkbox"`,
		`disabled=""`,
		"Trusted HTML",
		"Automatic publishing",
		"<del>Obsolete guidance</del>",
		`<a href="https://example.com/docs">https://example.com/docs</a>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q: %s", want, body)
		}
	}
	if got := strings.Count(body, `type="checkbox"`); got != 2 {
		t.Fatalf("checkbox count = %d, want 2: %s", got, body)
	}
	if got := strings.Count(body, `disabled=""`); got != 2 {
		t.Fatalf("disabled checkbox count = %d, want 2: %s", got, body)
	}
	if got := strings.Count(body, `checked=""`); got != 1 {
		t.Fatalf("checked checkbox count = %d, want 1: %s", got, body)
	}
}

func TestMarkdownHeadingsReceiveGoldmarkIDs(t *testing.T) {
	pages, err := New(Config{FS: entryFS("about", `{"title":"About"}`, "body.md", `# Install Goldr

Install Goldr
---

## What's New?!

## !!!

## !!!
`)})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	body := renderPage(t, mustPage(t, pages, "/about"))
	for _, want := range []string{
		`<h1 id="install-goldr">Install Goldr</h1>`,
		`<h2 id="install-goldr-1">Install Goldr</h2>`,
		`<h2 id="whats-new">`,
		`<h2 id="heading">!!!</h2>`,
		`<h2 id="heading-1">!!!</h2>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q: %s", want, body)
		}
	}
}

func TestRenderedMarkdownLimitIsSourceLocal(t *testing.T) {
	const sensitive = "sensitive-body-marker"
	body := append([]byte(sensitive+"\n\n"), bytes.Repeat([]byte("&"), maxBodyBytes-len(sensitive)-2)...)
	err := Check(entryFSBytes("about", []byte(`{"title":"About"}`), "body.md", body))
	if err == nil || !strings.Contains(err.Error(), "about/body.md") || !strings.Contains(err.Error(), fmt.Sprintf("rendered Markdown exceeds %d byte limit", maxRenderedMarkdownBytes)) {
		t.Fatalf("Check() error = %v, want source-local rendered Markdown limit error", err)
	}
	if strings.Contains(err.Error(), sensitive) {
		t.Fatalf("Check() error exposes body text: %v", err)
	}
}

func entryFS(directory, metadata, bodyName, body string) fstest.MapFS {
	return entryFSBytes(directory, []byte(metadata), bodyName, []byte(body))
}

func entryFSBytes(directory string, metadata []byte, bodyName string, body []byte) fstest.MapFS {
	files := make(fstest.MapFS)
	if len(metadata) > 0 {
		name := "page.json"
		if directory != "." {
			name = directory + "/" + name
		}
		files[name] = &fstest.MapFile{Data: metadata}
	}
	if bodyName != "" {
		name := bodyName
		if directory != "." {
			name = directory + "/" + name
		}
		files[name] = &fstest.MapFile{Data: body}
	}
	return files
}
