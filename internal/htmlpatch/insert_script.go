// Copyright 2024 the go-livereload authors.
// SPDX-License-Identifier: Apache-2.0

package htmlpatch

import (
	"bytes"
	"fmt"

	"golang.org/x/net/html"
)

// InsertScript returns a copy of inputHTML
// with a script tag inserted at the end of the head tag of the HTML.
func InsertScript(
	inputHTML []byte,
	scriptAttrs []html.Attribute,
	scriptContent string,
) (
	outputHTML []byte,
	err error,
) {

	// Parse the HTML.
	doc, err := html.Parse(bytes.NewReader(inputHTML))
	if err != nil {
		return inputHTML, fmt.Errorf("could not parse HTML: %w", err)
	}

	// Find or create the head tag.
	htmlTag := findOrCreateHtmlTag(doc)
	headTag := findOrCreateHeadTag(htmlTag)

	// Create and insert the script tag.
	headTag.AppendChild(scriptTag(scriptAttrs, scriptContent))

	// Render the modified HTML.
	buf := new(bytes.Buffer)
	err = html.Render(buf, doc)
	if err != nil {
		return inputHTML, fmt.Errorf("error rendering HTML: %v", err)
	}

	return buf.Bytes(), nil
}

func scriptTag(attrs []html.Attribute, content string) *html.Node {
	script := &html.Node{
		Type: html.ElementNode,
		Data: "script",
		Attr: attrs,
	}
	if content != "" {
		script.AppendChild(&html.Node{
			Type: html.TextNode,
			Data: content,
		})
	}
	return script
}
