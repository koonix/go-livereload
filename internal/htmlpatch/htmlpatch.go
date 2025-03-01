// Copyright 2024 the go-livereload authors.
// SPDX-License-Identifier: Apache-2.0

// Package htmlpatch provides functionality for modifying HTML.
package htmlpatch

import (
	"golang.org/x/net/html"
)

func findOrCreateHtmlTag(doc *html.Node) (htmlTag *html.Node) {

	htmlTag = findFirstTag(doc, "html")

	if htmlTag == nil {
		htmlTag = &html.Node{
			Type: html.ElementNode,
			Data: "html",
		}
		doc.AppendChild(htmlTag)
	}

	doctype := findDoctype(doc)

	if doctype == nil {
		doctype = &html.Node{
			Type: html.DoctypeNode,
			Data: "html",
		}
		prependChild(doc, doctype)
	}

	return htmlTag
}

func findOrCreateHeadTag(htmlTag *html.Node) (headTag *html.Node) {

	headTag = findFirstTag(htmlTag, "head")

	if headTag == nil {
		headTag = &html.Node{
			Type: html.ElementNode,
			Data: "head",
		}
		prependChild(htmlTag, headTag)
	}

	return headTag
}

func findFirstTag(node *html.Node, tagName string) *html.Node {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == tagName {
			return child
		}
	}
	return nil
}

func findDoctype(doc *html.Node) *html.Node {
	for child := doc.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.DoctypeNode {
			return child
		}
	}
	return nil
}

func prependChild(node, child *html.Node) {
	if node.FirstChild == nil {
		node.AppendChild(child)
	} else {
		node.InsertBefore(child, node.FirstChild)
	}
}
