package ebook

// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.

import (
	"regexp"
	"strings"

	"github.com/HalCanary/facility/dom"
	"golang.org/x/net/html"
)

type Node = dom.Node

// Clean up a HTML fragment.
func Cleanup(node *Node) *Node {
	node = cleanupStyle(node)
	cleanupLinks(node)
	cleanupTables(node)
	cleanupCenter(node)
	cleanupDoubled(node)
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

func cleanupCenter(node *Node) {
	if node != nil && node.Type == html.ElementNode {
		if node.Data == "center" {
			node.Data = "div"
			if i := getNodeAttributeIndex(node, "class"); i >= 0 {
				node.Attr[i].Val = node.Attr[i].Val + " mid"
			} else {
				node.AddAttribute("class", "mid")
			}
		}
		if node.Data == "big" {
			node.Data = "span"
			node.AddAttribute("style", "font-size:larger")
		}
		c := node.GetFirstChild()
		for c != nil {
			next := c.GetNextSibling()
			cleanupCenter(c)
			c = next
		}
	}
}

func cleanupDoubled(node *Node) {
	if node.Type == html.ElementNode {
		data := node.Data
		for c := node.GetFirstChild(); c != nil; {
			next := c.GetNextSibling()
			cleanupDoubled(c)
			if data == "ul" && c.Type == html.ElementNode && c.Data == data {
				(*Node)(c).Remove()
				for c2 := c.GetFirstChild(); c2 != nil; {
					n2 := c2.GetNextSibling()
					node.InsertBefore(c2.Remove(), next)
					c2 = n2
				}
			}
			c = next
		}
	}
}

func cleanupTables(node *Node) {
	if node != nil && node.Type == html.ElementNode {
		if i := getNodeAttributeIndex(node, "border"); i >= 0 {
			v := node.Attr[i].Val
			if v != "1" && v != "" {
				if v == "none" {
					node.Attr[i].Val = ""
				} else {
					node.Attr[i].Val = "1"
				}
			}
		}

		c := node.GetFirstChild()
		for c != nil {
			next := c.GetNextSibling()
			cleanupTables(c)
			c = next
		}
		if node.FirstChild == nil {
			switch node.Data {
			case "tbody", "dd", "dl":
				node.Remove()
			}
		}
	}
}

func cleanupLinks(node *Node) {
	if node != nil && node.Type == html.ElementNode {
		if i := getNodeAttributeIndex(node, "href"); i >= 0 {
			if strings.HasPrefix(node.Attr[i].Val, "/") {
				node.Attr = append(node.Attr[:i], node.Attr[i+1:]...)
			}
		}
		for c := node.GetFirstChild(); c != nil; c = c.GetNextSibling() {
			cleanupLinks(c)
		}
	}
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
			child := node.GetFirstChild()
			for child != nil {
				next := child.GetNextSibling()
				cleanupStyle(child)
				child = next
			}

			if node.Data == "span" && len(node.Attr) == 0 {
				if parent := node.GetParent(); parent != nil {
					nextSibling := node.GetNextSibling()
					child := node.GetFirstChild()
					for child != nil {
						next := child.GetNextSibling()
						child.Remove()
						parent.InsertBefore(child, nextSibling)
						child = next
					}
					node.Remove()
				}
			}
			if node.Data == "img" {
				if i := getNodeAttributeIndex(node, "src"); i >= 0 {
					if node.Attr[i].Val == "" {
						node.Attr[i].Val = "data:null;,"
					}
				} else {
					node.Attr = append(node.Attr, html.Attribute{Key: "src", Val: "data:null;,"})
				}
			}
		}
	}
	return node
}

func countChildren(node *Node) int {
	var count int = 0
	if node != nil {
		for child := node.GetFirstChild(); child != nil; child = child.GetNextSibling() {
			count++
		}
	}
	return count
}

func getNodeAttributeIndex(node *Node, key string) int {
	if node != nil {
		for idx, attr := range node.Attr {
			if attr.Namespace == "" && attr.Key == key {
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
			for child := node.GetFirstChild(); child != nil; child = child.GetNextSibling() {
				if !isWhitespaceOnly(child) {
					return false
				}
			}
		}
	}
	return true
}
