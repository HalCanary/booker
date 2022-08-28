// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.
package dom

import (
	"encoding/xml"
	"io"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// Alias for golang.org/x/net/html.Node.
type Node html.Node

var whitespaceRegexp = regexp.MustCompile("\\s+")

// Wrapper for golang.org/x/net/html.Parse.
func Parse(source io.Reader) (*Node, error) {
	n, err := html.Parse(source)
	return (*Node)(n), err
}

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
	return node.Append(children...)
}

func (node *Node) Append(children ...*Node) *Node {
	if node != nil && node.Type == html.ElementNode {
		for _, c := range children {
			if c != nil {
				(*html.Node)(node).AppendChild((*html.Node)(c))
			}
		}
	}
	return node
}

func (node *Node) GetFirstChild() *Node {
	return (*Node)(node.FirstChild)
}

func (node *Node) GetNextSibling() *Node {
	return (*Node)(node.NextSibling)
}

func makeAttribute(k, v string) html.Attribute {
	if ns, key, found := strings.Cut(k, ":"); found {
		return html.Attribute{Namespace: ns, Key: key, Val: v}
	} else {
		return html.Attribute{Key: k, Val: v}
	}
}

func (node *Node) AddAttribute(k, v string) {
	if node != nil && node.Type == html.ElementNode {
		node.Attr = append(node.Attr, makeAttribute(k, v))
	}
}

// Return an element with the given children.
func Elem(tag string, children ...*Node) *Node {
	return Element(tag, nil, children...)
}

// Generates HTML5 doc.
func (root *Node) RenderHTML(w io.Writer) error {
	d := html.Node{Type: html.DocumentNode}
	dt := html.Node{Type: html.DoctypeNode, Data: "html"}
	d.AppendChild(&dt)
	d.AppendChild((*html.Node)(root))
	e := html.Render(w, &d)
	w.Write([]byte{'\n'})
	return e
}

// Generates XHTML1 doc.
func (root *Node) RenderXHTMLDoc(w io.Writer) error {
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
					renderXHTML(w, (*Node)(c))
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
func (node *Node) GetAttribute(key string) string {
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
				accumulator.WriteString(root.GetAttribute("alt"))
			}
		}
		for child := root.FirstChild; child != nil; child = child.NextSibling {
			extractTextImpl((*Node)(child), accumulator)
		}
	}
}

// Extract and combine all Text Nodes under given node.
func (root *Node) ExtractText() string {
	var b strings.Builder
	extractTextImpl(root, &b)
	return b.String()
}

// Remove a node from its parent.
func (node *Node) Remove() *Node {
	if node != nil && node.Parent != nil {
		node.Parent.RemoveChild((*html.Node)(node))
	}
	return node
}

func (n *Node) InsertBefore(v, o *Node) {
	(*html.Node)(n).InsertBefore((*html.Node)(v), (*html.Node)(o))
}
func (n *Node) GetParent() *Node {
	return (*Node)(n.Parent)
}
