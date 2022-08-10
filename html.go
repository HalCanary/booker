package main

// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.

import (
	"io"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

type Node = html.Node

var whitespaceRegexp = regexp.MustCompile("\\s+")

// Return a HTML comment with the given data.
func Comment(data string) *Node {
	if data == "" {
		return nil
	}
	return &Node{Type: html.CommentNode, Data: data}
}

// Return a HTML node with the given text.
func TextNode(data string) *Node {
	return &Node{Type: html.TextNode, Data: data}
}

// Return an element with given attributes and children.
func Element(tag string, attributes map[string]string, children ...*Node) *Node {
	node := &Node{Type: html.ElementNode, Data: tag}
	for k, v := range attributes {
		if ns, key, found := strings.Cut(k, ":"); found {
			node.Attr = append(node.Attr, html.Attribute{Namespace: ns, Key: key, Val: v})
		} else {
			node.Attr = append(node.Attr, html.Attribute{Key: k, Val: v})
		}
	}
	for _, c := range children {
		if c != nil {
			node.AppendChild(c)
		}
	}
	return node
}

// Return an element with the given children.
func Elem(tag string, children ...*Node) *Node {
	return Element(tag, nil, children...)
}

// Generates HTML5 doc.
func RenderDoc(w io.Writer, root *Node) error {
	d := Node{Type: html.DocumentNode}
	dt := Node{Type: html.DoctypeNode, Data: "html"}
	d.AppendChild(&dt)
	d.AppendChild(root)
	e := html.Render(w, &d)
	w.Write([]byte{'\n'})
	return e
}

// Find the matching attributes, ignoring namespace.
func GetAttribute(node *Node, key string) string {
	if node != nil {
		for _, attr := range node.Attr {
			if attr.Key == key {
				return attr.Val
			}
		}
	}
	return ""
}

func extractTextImpl(root *Node, accumulator *strings.Builder) {
	if root != nil {
		if root.Type == html.TextNode {
			accumulator.WriteString(whitespaceRegexp.ReplaceAllString(root.Data, " "))
			return
		}
		if root.Type == html.ElementNode {
			switch root.Data {
			case "br":
				accumulator.WriteString("\n")
			case "hr":
				accumulator.WriteString("\n* * *\n")
			case "p":
				accumulator.WriteString("\n\n")
			case "img":
				accumulator.WriteString(GetAttribute(root, "alt"))
			}
		}
		for child := root.FirstChild; child != nil; child = child.NextSibling {
			extractTextImpl(child, accumulator)
		}
	}
}

// Extract and combine all Text Nodes under given node.
func ExtractText(root *Node) string {
	var b strings.Builder
	extractTextImpl(root, &b)
	return b.String()
}

// Remove a node from it's parent.
func Remove(node *Node) *Node {
	if node != nil && node.Parent != nil {
		node.Parent.RemoveChild(node)
	}
	return node
}
