package domutils

import (
	"strings"

	"github.com/JohannesKaufmann/dom"
	"golang.org/x/net/html"
)

// RemoveNavigation removes navigation and other non-content elements
func RemoveNavigation(node *html.Node) {
	if node.Type == html.ElementNode {
		// 1. Remove specific tags
		if isNavigationTag(node.Data) {
			dom.RemoveNode(node)
			return // Node removed, stop processing children
		}

		// 2. Remove by class/id
		if isNavigationClassOrID(node) {
			dom.RemoveNode(node)
			return // Node removed, stop processing children
		}
	}

	// Process children
	for c := node.FirstChild; c != nil; {
		next := c.NextSibling
		RemoveNavigation(c)
		c = next
	}
}

func isNavigationTag(tag string) bool {
	tagsToRemove := []string{
		"nav", "header", "footer", "aside",
		"script", "style", "noscript", "iframe",
		"svg", "button", "input", "form",
	}
	for _, t := range tagsToRemove {
		if tag == t {
			return true
		}
	}
	return false
}

func isNavigationClassOrID(n *html.Node) bool {
	class, _ := dom.GetAttribute(n, "class")
	id, _ := dom.GetAttribute(n, "id")

	// Convert to lower case for case-insensitive matching
	class = strings.ToLower(class)
	id = strings.ToLower(id)

	keywords := []string{
		"nav", "menu", "sidebar", "header", "footer",
		"cookie", "banner", "popup", "modal", "advert",
		"social", "share", "breadcrumb", "pagination",
		"newsletter", "widget",
	}

	for _, kw := range keywords {
		if strings.Contains(class, kw) || strings.Contains(id, kw) {
			// Careful not to remove "features" which might contain "feature"
			// This is a naive heuristic
			return true
		}
	}
	return false
}
