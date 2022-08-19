package main

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// Clean up a HTML fragment.
func Cleanup(node *Node) *Node {
	node = cleanupStyle(node)
	return node
}

var whiteSpaceOnly = regexp.MustCompile("^\\pZ*$")
var spaceOnly = regexp.MustCompile("^\\pZs*$")
var semiRegexp = regexp.MustCompile("\\s*;\\s*")

func styler(v string) string {
	var result []string
	for _, term := range semiRegexp.Split(v, -1) {
		switch term {
		case "background-attachment: initial",
			"background-clip: initial",
			"background-image: initial",
			"background-origin: initial",
			"background-position: initial",
			"background-repeat: initial",
			"background-size: initial",
			"break-before: page",
			"margin-bottom: 0in",
			"background: transparent",
			"font-family: Arial",
			"font-family: Segoe UI",
			"font-family: Segoe UI, sans-serif",
			"font-family: Segoe UI, serif",
			"font-style: normal",
			"font-variant: normal",
			"font-weight: normal",
			"margin-bottom: 0",
			"page-break-before: always",
			"text-decoration: none",
			"":
			// do nothing
		case "font-family: Courier New, monospace":
			result = append(result, "font-family:monospace")
		default:
			result = append(result, term)
		}
	}
	return strings.Join(result, ";")
}

func cleanupStyle(node *Node) *Node {
	if node != nil {
		switch node.Type {
		case html.TextNode:
			if node.Data != "" && whiteSpaceOnly.MatchString(node.Data) {
				if !spaceOnly.MatchString(node.Data) {
					node.Data = "\n"
				}
			}
		case html.ElementNode:
			if node.Data == "p" {
				if isWhitespaceOnly(node) {
					node.Remove()
					return nil
				}
				if i := getNodeAttributeIndex(node, "align"); i >= 0 {
					switch node.Attr[i].Val {
					case "left":
						node.Attr = append(node.Attr[:i], node.Attr[i+1:]...)
					}
				}
			}
			if i := getNodeAttributeIndex(node, "style"); i >= 0 {
				v := styler(node.Attr[i].Val)
				if v == "" {
					node.Attr = append(node.Attr[:i], node.Attr[i+1:]...)
				} else {
					node.Attr[i].Val = v
				}
			}
			child := node.FirstChild
			for child != nil {
				next := child.NextSibling
				cleanupStyle((*Node)(child))
				child = next
			}

			if node.Data == "span" && len(node.Attr) == 0 {
				if parent := node.Parent; parent != nil {
					nextSibling := node.NextSibling
					child := node.FirstChild
					for child != nil {
						next := child.NextSibling
						(*html.Node)(node).RemoveChild(child)
						parent.InsertBefore(child, nextSibling)
						child = next
					}
					parent.RemoveChild((*html.Node)(node))
				}
			}
		}
	}
	return node
}

func countChildren(node *Node) int {
	var count int = 0
	if node != nil {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			count++
		}
	}
	return count
}

func getNodeAttributeIndex(node *Node, key string) int {
	if node != nil {
		for idx, attr := range node.Attr {
			if attr.Key == key {
				return idx
			}
		}
	}
	return -1
}

func isWhitespaceOnly(node *Node) bool {
	if node != nil {
		switch node.Type {
		case html.TextNode:
			return whiteSpaceOnly.MatchString(node.Data)
		case html.ElementNode:
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if !isWhitespaceOnly((*Node)(child)) {
					return false
				}
			}
		}
	}
	return true
}
