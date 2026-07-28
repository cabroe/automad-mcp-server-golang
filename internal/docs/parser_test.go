package docs_test

import (
	"strings"
	"testing"

	"github.com/cabroe/automad-mcp-server-golang/internal/docs"
)

// htmlWithMain wraps content in a minimal Automad-like HTML page.
func htmlWithMain(title, body string) string {
	return `<!DOCTYPE html><html><head><title>` + title + ` / Automad</title></head>` +
		`<body><main class="docs-content">` + body + `</main></body></html>`
}

func TestParse_Title(t *testing.T) {
	raw := htmlWithMain("Getting Started", "<h1>Getting Started</h1><p>Introduction text.</p>")
	page := docs.Parse(raw, "/getting-started")

	if page.Title != "Getting Started" {
		t.Errorf("expected title %q, got %q", "Getting Started", page.Title)
	}
}

func TestParse_URL(t *testing.T) {
	raw := htmlWithMain("Test", "<p>Hello</p>")
	page := docs.Parse(raw, "/test-url")

	if page.URL != "/test-url" {
		t.Errorf("expected URL %q, got %q", "/test-url", page.URL)
	}
	if page.FullURL != docs.BaseURL+"/test-url" {
		t.Errorf("expected FullURL %q, got %q", docs.BaseURL+"/test-url", page.FullURL)
	}
}

func TestParse_HeadingFormatting(t *testing.T) {
	raw := htmlWithMain("H-Test", `
		<main class="docs-content">
			<h1>Main Title</h1>
			<h2>Subsection</h2>
			<h3>Sub-subsection</h3>
		</main>`)
	page := docs.Parse(raw, "/h-test")

	if !strings.Contains(page.Content, "# Main Title") {
		t.Errorf("expected h1 formatted as '# Main Title', got:\n%s", page.Content)
	}
	if !strings.Contains(page.Content, "## Subsection") {
		t.Errorf("expected h2 formatted as '## Subsection', got:\n%s", page.Content)
	}
	if !strings.Contains(page.Content, "### Sub-subsection") {
		t.Errorf("expected h3 formatted as '### Sub-subsection', got:\n%s", page.Content)
	}
}

func TestParse_CodeFormatting(t *testing.T) {
	raw := htmlWithMain("Code", `<main class="docs-content"><p>Use <code>pagelist</code> here.</p></main>`)
	page := docs.Parse(raw, "/code")

	if !strings.Contains(page.Content, "`pagelist`") {
		t.Errorf("expected inline code with backticks, got:\n%s", page.Content)
	}
}

func TestParse_PreFormatting(t *testing.T) {
	raw := htmlWithMain("Pre", `<main class="docs-content"><pre>some code block</pre></main>`)
	page := docs.Parse(raw, "/pre")

	if !strings.Contains(page.Content, "```") {
		t.Errorf("expected fenced code block, got:\n%s", page.Content)
	}
	if !strings.Contains(page.Content, "some code block") {
		t.Errorf("expected code block content, got:\n%s", page.Content)
	}
}

func TestParse_ScriptStripped(t *testing.T) {
	raw := htmlWithMain("Scripts", `<main class="docs-content"><script>alert("xss")</script><p>Visible</p></main>`)
	page := docs.Parse(raw, "/scripts")

	if strings.Contains(page.Content, "alert") {
		t.Errorf("script content should be stripped, got:\n%s", page.Content)
	}
	if !strings.Contains(page.Content, "Visible") {
		t.Errorf("expected paragraph text, got:\n%s", page.Content)
	}
}

func TestParse_InvalidHTML(t *testing.T) {
	// golang.org/x/net/html is very permissive, so this should not crash.
	page := docs.Parse("<<<<not valid>>>>", "/broken")
	if page == nil {
		t.Fatal("Parse should never return nil")
	}
}

func TestParse_EmptyHTML(t *testing.T) {
	page := docs.Parse("", "/empty")
	if page == nil {
		t.Fatal("Parse should never return nil for empty input")
	}
	if page.URL != "/empty" {
		t.Errorf("expected URL %q, got %q", "/empty", page.URL)
	}
}

func TestParse_TitleStripsAutomadSuffix(t *testing.T) {
	raw := `<html><head><title>User Guide / Automad</title></head><body></body></html>`
	page := docs.Parse(raw, "/user-guide")

	if page.Title != "User Guide" {
		t.Errorf("expected suffix stripped, got %q", page.Title)
	}
}

func TestParse_WhitespaceNormalized(t *testing.T) {
	raw := htmlWithMain("WS", `<main class="docs-content"><p>Line one</p>


		<p>Line two</p></main>`)
	page := docs.Parse(raw, "/ws")

	// Should not have more than 2 consecutive blank lines.
	if strings.Contains(page.Content, "\n\n\n\n") {
		t.Errorf("excessive blank lines not collapsed:\n%q", page.Content)
	}
}
