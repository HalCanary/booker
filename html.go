package main

// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.

import (
	"encoding/xml"
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
		node.Attr = append(node.Attr, makeAttribute(k, v))
	}
	return Append(node, children...)
}

func Append(node *Node, children ...*Node) *Node {
	if node != nil && node.Type == html.ElementNode {
		for _, c := range children {
			if c != nil {
				node.AppendChild(c)
			}
		}
	}
	return node
}

func makeAttribute(k, v string) html.Attribute {
	if ns, key, found := strings.Cut(k, ":"); found {
		return html.Attribute{Namespace: ns, Key: key, Val: v}
	} else {
		return html.Attribute{Key: k, Val: v}
	}
}

func AddAttribute(node *Node, k, v string) {
	if node != nil && node.Type == html.ElementNode {
		node.Attr = append(node.Attr, makeAttribute(k, v))
	}
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

// Generates XHTML1 doc.
func RenderXHTMLDoc(w io.Writer, root *Node) error {
	if root == nil || w == nil {
		return nil
	}
	w.Write([]byte(xml.Header))
	cw := checkedWriter{Writer: w}
	renderXHTML(&cw, root)
	cw.Write([]byte{'\n'})
	return cw.Error
}

type checkedWriter struct {
	io.Writer
	Error error
}

func (w *checkedWriter) Write(b []byte) {
	if w.Error == nil {
		_, w.Error = w.Writer.Write(b)
	}
}

var xhtmlattribs = map[string]struct{}{
	"alt":        struct{}{},
	"border":     struct{}{},
	"class":      struct{}{},
	"content":    struct{}{},
	"dir":        struct{}{},
	"href":       struct{}{},
	"http-equiv": struct{}{},
	"id":         struct{}{},
	"lang":       struct{}{},
	"name":       struct{}{},
	"src":        struct{}{},
	"style":      struct{}{},
	"title":      struct{}{},
	"type":       struct{}{},
	"xmlns":      struct{}{},
}

func renderXHTML(w *checkedWriter, node *Node) {
	switch node.Type {
	case html.ElementNode:
		w.Write([]byte{'<'})
		w.Write([]byte(node.Data))
		for _, attr := range node.Attr {
			ok := attr.Namespace != ""
			if !ok {
				_, ok = xhtmlattribs[attr.Key]
			}
			if ok {
				w.Write([]byte{' '})
				if attr.Namespace != "" {
					w.Write([]byte(attr.Namespace))
					w.Write([]byte{':'})
				}
				w.Write([]byte(attr.Key))
				w.Write([]byte{'=', '"'})
				w.Write([]byte(html.EscapeString(attr.Val)))
				w.Write([]byte{'"'})
			}
		}
		if node.FirstChild == nil {
			w.Write([]byte{'/', '>'})
		} else {
			w.Write([]byte{'>'})
			for c := node.FirstChild; c != nil; c = c.NextSibling {
				if w.Error == nil {
					renderXHTML(w, c)
				}
			}
			w.Write([]byte{'<', '/'})
			w.Write([]byte(node.Data))
			w.Write([]byte{'>'})
		}
	case html.TextNode:
		w.Write([]byte(html.EscapeString(node.Data)))
	case html.CommentNode:
		w.Write([]byte{'<', '!', '-', '-'})
		w.Write([]byte(node.Data))
		w.Write([]byte{'-', '-', '>'})
	}
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
