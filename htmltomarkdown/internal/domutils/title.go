package domutils

import (
	"strings"

	"github.com/JohannesKaufmann/dom"
	"golang.org/x/net/html"
)

// GetTitle extracts title from HTML document
// It checks <title>, <meta property="og:title">, <meta name="twitter:title">, and <h1>
func GetTitle(n *html.Node) string {
	// 1. Try <title> tag first (standard)
	var title string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" {
			if n.FirstChild != nil {
				title = n.FirstChild.Data
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	title = strings.TrimSpace(title)
	if title != "" {
		return title
	}

	// 2. Try Open Graph title
	if val := getMetaContent(n, "property", "og:title"); val != "" {
		return val
	}

	// 3. Try Twitter title
	if val := getMetaContent(n, "name", "twitter:title"); val != "" {
		return val
	}

	// 4. Try first <h1> as fallback
	if val := getFirstH1(n); val != "" {
		return val
	}

	// 5. Try meta description as last resort
	if val := getMetaContent(n, "name", "description"); val != "" {
		// Truncate if too long
		if len(val) > 100 {
			return val[:97] + "..."
		}
		return val
	}

	return ""
}

func getMetaContent(n *html.Node, attrKey, attrValue string) string {
	var content string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if content != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "meta" {
			if val, _ := dom.GetAttribute(n, attrKey); val == attrValue {
				content, _ = dom.GetAttribute(n, "content")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return strings.TrimSpace(content)
}

func getFirstH1(n *html.Node) string {
	var h1Text string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if h1Text != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "h1" {
			h1Text = getTextContent(n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return strings.TrimSpace(h1Text)
}

func getTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var text strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text.WriteString(getTextContent(c))
	}
	return text.String()
}
