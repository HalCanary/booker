package dom

// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.

import (
	"golang.org/x/net/html"
)

func FindAllMatchingNodes(node *Node, tag string) []*Node {
	var result []*Node
	var findAllMatchingNodesImpl func(n *Node)
	findAllMatchingNodesImpl = func(n *Node) {
		if n != nil {
			if n.Type == html.ElementNode && n.Data == tag {
				result = append(result, n)
			}
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				findAllMatchingNodesImpl((*Node)(child))
			}
		}
	}
	findAllMatchingNodesImpl(node)
	return result
}

func FindOneMatchingNode(node *Node, tag string) *Node {
	if node.Type == html.ElementNode && node.Data == tag {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		r := FindOneMatchingNode((*Node)(child), tag)
		if r != nil {
			return r
		}
	}
	return nil
}

func FindOneMatchingNode2(node *Node, tag, attributeKey, attributeValue string) *Node {
	if node.Type == html.ElementNode && node.Data == tag {
		for _, attr := range node.Attr {
			if attr.Key == attributeKey && attr.Val == attributeValue {
				return node
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		r := FindOneMatchingNode2((*Node)(child), tag, attributeKey, attributeValue)
		if r != nil {
			return r
		}
	}
	return nil
}
