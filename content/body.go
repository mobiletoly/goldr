// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package content

import (
	"bytes"
	"fmt"

	"github.com/a-h/templ"
	"github.com/yuin/goldmark/v2/extension"
	"github.com/yuin/goldmark/v2/parser"
	markdownhtml "github.com/yuin/goldmark/v2/renderer/html"
)

const maxRenderedMarkdownBytes = 2 * 1024 * 1024

func renderEntryBody(entry loadedEntry) (templ.Component, error) {
	body := entry.body
	if entry.markdown {
		converted, err := renderMarkdown(entry.source, body)
		if err != nil {
			return nil, err
		}
		body = converted
	}
	return templ.Raw(`<article class="goldr-content">` + string(body) + `</article>`), nil
}

func renderMarkdown(source string, markdown []byte) ([]byte, error) {
	document := parser.New(
		parser.WithExtensions(extension.GFMParser),
		parser.WithAutoHeadingID(),
	).Parse(markdown)

	var rendered bytes.Buffer
	if err := markdownhtml.New(
		markdownhtml.WithUnsafe(),
		markdownhtml.WithExtensions(extension.GFMHTMLRenderer),
	).Render(&rendered, markdown, document); err != nil {
		return nil, contentError(source, "render Markdown", err)
	}
	if rendered.Len() > maxRenderedMarkdownBytes {
		return nil, contentError(source, fmt.Sprintf("rendered Markdown exceeds %d byte limit", maxRenderedMarkdownBytes), nil)
	}
	return rendered.Bytes(), nil
}
