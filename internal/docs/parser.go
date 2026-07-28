package docs

import (
	"strings"

	"golang.org/x/net/html"
)

// Page holds the parsed content of a documentation page.
type Page struct {
	// Title is the page title (from <h1> or <title>).
	Title string
	// URL is the relative URL of the page.
	URL string
	// FullURL is the absolute URL of the page.
	FullURL string
	// Content is the extracted plain-text / markdown-like content.
	Content string
}

// Parse extracts structured text content from raw Automad documentation HTML.
// It targets the main content area and strips away navigation, headers, and scripts.
func Parse(rawHTML, url string) *Page {
	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return &Page{
			URL:     url,
			FullURL: BaseURL + url,
			Title:   "Parse Error",
			Content: "Failed to parse page HTML: " + err.Error(),
		}
	}

	p := &Page{
		URL:     url,
		FullURL: BaseURL + url,
	}

	// Extract title and main content from the HTML tree.
	extractContent(doc, p)

	// Trim excessive whitespace.
	p.Content = normalizeWhitespace(p.Content)

	return p
}

// extractContent walks the HTML node tree and populates the Page fields.
func extractContent(n *html.Node, p *Page) {
	// Extract <title>
	if n.Type == html.ElementNode && n.Data == "title" && p.Title == "" {
		p.Title = strings.TrimSpace(textContent(n))
		// Strip " / Automad" or similar suffixes.
		if idx := strings.LastIndex(p.Title, " / "); idx != -1 {
			p.Title = p.Title[:idx]
		}
	}

	// Skip non-content elements.
	if n.Type == html.ElementNode {
		switch n.Data {
		case "script", "style", "nav", "header", "footer", "noscript":
			return
		}
		// Look for the docs content area.
		if isContentNode(n) {
			p.Content += extractText(n, 0)
			return
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractContent(c, p)
	}
}

// isContentNode returns true if this HTML element is the main documentation content.
func isContentNode(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	cls := getAttr(n, "class")
	id := getAttr(n, "id")
	return strings.Contains(cls, "docs-content") ||
		strings.Contains(cls, "uk-article") ||
		strings.Contains(id, "content") ||
		n.Data == "article" ||
		n.Data == "main"
}

// extractText recursively extracts text from a node, adding markdown-like
// formatting for headings, code blocks, and lists.
func extractText(n *html.Node, depth int) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	if n.Type != html.ElementNode {
		var sb strings.Builder
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			sb.WriteString(extractText(c, depth))
		}
		return sb.String()
	}

	// Skip hidden/decorative elements.
	switch n.Data {
	case "script", "style", "noscript":
		return ""
	}

	var sb strings.Builder
	switch n.Data {
	case "h1":
		sb.WriteString("\n# ")
		sb.WriteString(childText(n))
		sb.WriteString("\n\n")
	case "h2":
		sb.WriteString("\n## ")
		sb.WriteString(childText(n))
		sb.WriteString("\n\n")
	case "h3":
		sb.WriteString("\n### ")
		sb.WriteString(childText(n))
		sb.WriteString("\n\n")
	case "h4", "h5", "h6":
		sb.WriteString("\n#### ")
		sb.WriteString(childText(n))
		sb.WriteString("\n\n")
	case "p":
		sb.WriteString("\n")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			sb.WriteString(extractText(c, depth))
		}
		sb.WriteString("\n")
	case "li":
		sb.WriteString("\n- ")
		sb.WriteString(childText(n))
	case "code":
		sb.WriteString("`")
		sb.WriteString(childText(n))
		sb.WriteString("`")
	case "pre":
		sb.WriteString("\n```\n")
		sb.WriteString(childText(n))
		sb.WriteString("\n```\n")
	case "br":
		sb.WriteString("\n")
	case "a":
		text := childText(n)
		href := getAttr(n, "href")
		if href != "" && text != "" {
			sb.WriteString("[")
			sb.WriteString(text)
			sb.WriteString("](")
			if strings.HasPrefix(href, "/") {
				sb.WriteString(BaseURL)
			}
			sb.WriteString(href)
			sb.WriteString(")")
		} else {
			sb.WriteString(text)
		}
	case "strong", "b":
		sb.WriteString("**")
		sb.WriteString(childText(n))
		sb.WriteString("**")
	case "em", "i":
		sb.WriteString("_")
		sb.WriteString(childText(n))
		sb.WriteString("_")
	default:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			sb.WriteString(extractText(c, depth+1))
		}
	}
	return sb.String()
}

// childText returns all text within a node's children as a single string.
func childText(n *html.Node) string {
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(extractText(c, 0))
	}
	return strings.TrimSpace(sb.String())
}

// textContent returns all text content of a node (recursively).
func textContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(textContent(c))
	}
	return sb.String()
}

// getAttr returns the value of a named attribute on an HTML element.
func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// normalizeWhitespace collapses runs of blank lines and trims space.
func normalizeWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	var result []string
	blankCount := 0
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if trimmed == "" {
			blankCount++
			if blankCount <= 2 {
				result = append(result, "")
			}
		} else {
			blankCount = 0
			result = append(result, trimmed)
		}
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}
