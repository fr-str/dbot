package htmlp

import (
	"strings"

	"golang.org/x/net/html"
)

type Matcher func(*Node) bool

// Any always returns true
func Any() Matcher {
	return func(n *Node) bool { return true }
}

// And combines multiple matchers, returning true only if ALL of them match.
func And(matchers ...Matcher) Matcher {
	if len(matchers) == 0 {
		return func(n *Node) bool { return false }
	}
	if len(matchers) == 1 {
		return matchers[0]
	}
	return func(n *Node) bool {
		for i := range matchers {
			if !matchers[i](n) {
				return false
			}
		}
		return true
	}
}

// Or combines multiple matchers, returning true if ANY of them match.
func Or(matchers ...Matcher) Matcher {
	if len(matchers) == 0 {
		return func(n *Node) bool { return false }
	}
	if len(matchers) == 1 {
		return matchers[0]
	}
	return func(n *Node) bool {
		for i := range matchers {
			if matchers[i](n) {
				return true
			}
		}
		return false
	}
}

// ByTag filters by the element name (e.g., "div", "p")
func ByTag(name string) Matcher {
	return func(n *Node) bool {
		return n.Type == html.ElementNode && n.Data == name
	}
}

// ByAttr filters by an attribute key and value (e.g., "id", "main")
func ByAttr(key, val string) Matcher {
	return func(n *Node) bool {
		if n.Type != html.ElementNode {
			return false
		}
		for i := range n.Attr {
			if n.Attr[i].Key == key && n.Attr[i].Val == val {
				return true
			}
		}
		return false
	}
}

// ByAttrValues matches nodes where a specific attribute contains ALL specified values (space-separated).
func ByAttrValues(key string, values ...string) Matcher {
	if len(values) == 0 {
		return func(n *Node) bool { return n.Type == html.ElementNode }
	}

	return func(n *Node) bool {
		if n.Type != html.ElementNode {
			return false
		}

		var attrVal string
		for i := range n.Attr {
			if n.Attr[i].Key == key {
				attrVal = n.Attr[i].Val
				break
			}
		}

		if attrVal == "" {
			return false
		}

		// Reuses the zero-allocation boundary checker
		for _, reqVal := range values {
			if !hasValue(attrVal, reqVal) {
				return false
			}
		}

		return true
	}
}

// hasValue evaluates word boundaries in-place to support space-separated lists without allocating.
func hasValue(attrStr, target string) bool {
	if target == "" {
		return false
	}
	idx := strings.Index(attrStr, target)
	for idx != -1 {
		endIdx := idx + len(target)
		startOK := idx == 0 || attrStr[idx-1] == ' '
		endOK := endIdx == len(attrStr) || attrStr[endIdx] == ' '

		if startOK && endOK {
			return true
		}
		nextIdx := strings.Index(attrStr[idx+1:], target)
		if nextIdx == -1 {
			break
		}
		idx += 1 + nextIdx
	}
	return false
}
