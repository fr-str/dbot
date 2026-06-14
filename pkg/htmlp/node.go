package htmlp

import (
	"io"
	"iter"
	"slices"
	"strings"

	"golang.org/x/net/html"
)

type Node html.Node

func Parse(r io.Reader) (*Node, error) {
	n, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	return (*Node)(n), err
}

func (n *Node) ChildNodes() iter.Seq[*Node] {
	if n == nil {
		return nil
	}

	return func(yield func(*Node) bool) {
		for c := (*Node)(n.FirstChild); c != nil && yield(c); c = (*Node)(c.NextSibling) {
		}
	}
}

// Find recursively searches for the first node that satisfies the given matcher.
func (n *Node) Find(match Matcher) *Node {
	if n == nil {
		return nil
	}

	if n.Type == html.ElementNode && match(n) {
		return (*Node)(n)
	}

	for c := range n.ChildNodes() {
		res := c.Find(match)
		if res != nil {
			return res
		}
	}
	return nil
}

// FindAll collects all nodes that satisfy the given matcher using the append pattern.
func (n *Node) FindAll(match Matcher) []*Node {
	if n == nil {
		return nil
	}

	return slices.Collect(n.FindSeq(match))
}

// FindSeq returns a lazy iterator for matching nodes.
func (n *Node) FindSeq(match Matcher) iter.Seq[*Node] {
	if n == nil {
		return nil
	}

	return func(yield func(*Node) bool) {
		// Skip the root node
		for c := range n.ChildNodes() {
			if !c.walkNodes(yield, match) {
				return
			}
		}
	}
}

func (n *Node) walkNodes(yield func(*Node) bool, match Matcher) bool {
	if n == nil {
		return true
	}

	if n.Type == html.ElementNode && match(n) {
		if !yield(n) {
			return false
		}
	}

	for c := range n.ChildNodes() {
		if !c.walkNodes(yield, match) {
			return false
		}
	}
	return true
}

func (n *Node) GetAttr(key string) string {
	if n == nil || n.Type != html.ElementNode {
		return ""
	}

	for i := range n.Attr {
		if n.Attr[i].Key == key {
			return n.Attr[i].Val
		}
	}

	return ""
}

func (n *Node) Text() string {
	if n == nil {
		return ""
	}
	var sb strings.Builder
	n.writeText(&sb)
	return strings.TrimSpace(sb.String())
}

func (n *Node) writeText(sb *strings.Builder) {
	if n.Type == html.TextNode {
		sb.WriteString(n.Data)
	}

	for c := range n.ChildNodes() {
		c.writeText(sb)
	}
}
