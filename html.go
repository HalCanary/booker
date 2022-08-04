package main

// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.

import (
	"io"
	"strings"

	"golang.org/x/net/html"
)

type Node = html.Node

func Comment(data string) *Node {
	return &Node{Type: html.CommentNode, Data: data}
}

func TextNode(data string) *Node {
	return &Node{Type: html.TextNode, Data: data}
}

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

func Elem(tag string, children ...*Node) *Node {
	return Element(tag, nil, children...)
}

func RenderDoc(w io.Writer, root *Node) error {
	d := Node{Type: html.DocumentNode}
	dt := &Node{Type: html.DoctypeNode, Data: "html"}
	for _, c := range []*Node{dt, TextNode("\n"), root, TextNode("\n")} {
		d.AppendChild(c)
	}
	return html.Render(w, &d)
}
