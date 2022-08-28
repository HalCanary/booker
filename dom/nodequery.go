package dom

// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.

import (
	"golang.org/x/net/html"
)

func (node *Node) FindAllMatchingNodes(tag string) []*Node {
	var result []*Node
	var findAllMatchingNodesImpl func(n *Node)
	findAllMatchingNodesImpl = func(n *Node) {
		if n != nil {
			if n.Type == html.ElementNode && n.Data == tag {
				result = append(result, n)
			}
			for child := n.GetFirstChild(); child != nil; child = child.GetNextSibling() {
				findAllMatchingNodesImpl(child)
			}
		}
	}
	findAllMatchingNodesImpl(node)
	return result
}

func (node *Node) FindOneMatchingNode(tag string) *Node {
	if node.Type == html.ElementNode && node.Data == tag {
		return node
	}
	for child := node.GetFirstChild(); child != nil; child = child.GetNextSibling() {
		r := child.FindOneMatchingNode(tag)
		if r != nil {
			return r
		}
	}
	return nil
}

func (node *Node) FindOneMatchingNode2(tag, attributeKey, attributeValue string) *Node {
	if node.Type == html.ElementNode && node.Data == tag {
		for _, attr := range node.Attr {
			if attr.Key == attributeKey && attr.Val == attributeValue {
				return node
			}
		}
	}
	for child := node.GetFirstChild(); child != nil; child = child.GetNextSibling() {
		r := child.FindOneMatchingNode2(tag, attributeKey, attributeValue)
		if r != nil {
			return r
		}
	}
	return nil
}
