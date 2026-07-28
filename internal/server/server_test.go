package server_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cabroe/automad-mcp-server/internal/docs"
	mcpserver "github.com/cabroe/automad-mcp-server/internal/server"
)

// newTestServer creates a fully wired MCP server with a pre-seeded docs service
// and returns both the server and a connected client session for testing.
func newTestServer(t *testing.T, seedURL string, seedPage *docs.Page) (*mcp.Server, *mcp.ClientSession) {
	t.Helper()

	var svc *docs.Service
	if seedURL != "" && seedPage != nil {
		svc = docs.NewSeededService(seedURL, seedPage)
	} else {
		svc = docs.NewService()
	}

	s := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "0.0.1"}, nil)
	mcpserver.RegisterTools(s, svc)
	mcpserver.RegisterResources(s, svc)

	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()

	if _, err := s.Connect(ctx, t1, nil); err != nil {
		t.Fatalf("server.Connect: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { session.Close() })

	return s, session
}

// callTool is a small helper that calls a named tool and returns the text content.
func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) (string, bool) {
	t.Helper()

	ctx := context.Background()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool(%q): %v", name, err)
	}

	if len(result.Content) == 0 {
		return "", result.IsError
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	return tc.Text, result.IsError
}

// ── list_pages ────────────────────────────────────────────────────────────────

func TestTool_ListPages_All(t *testing.T) {
	_, session := newTestServer(t, "", nil)
	text, isErr := callTool(t, session, "list_pages", map[string]any{})

	if isErr {
		t.Fatalf("list_pages returned error: %s", text)
	}
	if !strings.Contains(text, "Getting Started") {
		t.Errorf("expected 'Getting Started' in list_pages output, got:\n%s", text)
	}
}

func TestTool_ListPages_FilteredBySection(t *testing.T) {
	_, session := newTestServer(t, "", nil)
	text, isErr := callTool(t, session, "list_pages", map[string]any{"parent": "System"})

	if isErr {
		t.Fatalf("list_pages returned error: %s", text)
	}
	if !strings.Contains(text, "Caching") {
		t.Errorf("expected 'Caching' in System section output, got:\n%s", text)
	}
}

func TestTool_ListPages_UnknownSection(t *testing.T) {
	_, session := newTestServer(t, "", nil)
	text, isErr := callTool(t, session, "list_pages", map[string]any{"parent": "NonExistent99"})

	if isErr {
		t.Fatalf("list_pages returned error for unknown section: %s", text)
	}
	if !strings.Contains(text, "No pages found") {
		t.Errorf("expected 'No pages found' message, got:\n%s", text)
	}
}

func TestTool_ListPages_OutputIsValidJSON(t *testing.T) {
	_, session := newTestServer(t, "", nil)
	text, isErr := callTool(t, session, "list_pages", map[string]any{})

	if isErr {
		t.Fatalf("list_pages error: %s", text)
	}

	// Strip the header line before the JSON array.
	idx := strings.Index(text, "[")
	if idx < 0 {
		t.Fatalf("no JSON array found in output:\n%s", text)
	}
	var pages []map[string]any
	if err := json.Unmarshal([]byte(text[idx:]), &pages); err != nil {
		t.Errorf("output is not valid JSON: %v\nOutput:\n%s", err, text[idx:])
	}
}

// ── search_docs ───────────────────────────────────────────────────────────────

func TestTool_SearchDocs_EmptyQuery(t *testing.T) {
	_, session := newTestServer(t, "", nil)
	_, isErr := callTool(t, session, "search_docs", map[string]any{"query": ""})

	if !isErr {
		t.Error("expected error result for empty query")
	}
}

func TestTool_SearchDocs_NoResults(t *testing.T) {
	_, session := newTestServer(t, "", nil)
	text, isErr := callTool(t, session, "search_docs", map[string]any{"query": "xyzzy-no-match-term-99999"})

	if isErr {
		t.Fatalf("search_docs returned error for no-results query: %s", text)
	}
	if !strings.Contains(text, "No documentation pages found") {
		t.Errorf("expected no-results message, got:\n%s", text)
	}
}

func TestTool_SearchDocs_MatchesTitle(t *testing.T) {
	_, session := newTestServer(t, "", nil)
	text, isErr := callTool(t, session, "search_docs", map[string]any{"query": "pagelist"})

	if isErr {
		t.Fatalf("search_docs error: %s", text)
	}
	if !strings.Contains(strings.ToLower(text), "pagelist") {
		t.Errorf("expected 'pagelist' in results, got:\n%s", text)
	}
}

func TestTool_SearchDocs_ContainsURLs(t *testing.T) {
	_, session := newTestServer(t, "", nil)
	text, isErr := callTool(t, session, "search_docs", map[string]any{"query": "system"})

	if isErr {
		t.Fatalf("search_docs error: %s", text)
	}
	if !strings.Contains(text, "https://automad.org") {
		t.Errorf("expected automad.org URLs in output, got:\n%s", text)
	}
}

// ── get_page ──────────────────────────────────────────────────────────────────

func TestTool_GetPage_EmptyURL(t *testing.T) {
	_, session := newTestServer(t, "", nil)
	_, isErr := callTool(t, session, "get_page", map[string]any{"url": ""})

	if !isErr {
		t.Error("expected error result for empty URL")
	}
}

func TestTool_GetPage_CachedPage(t *testing.T) {
	page := &docs.Page{
		Title:   "Test Page",
		URL:     "/test-page",
		FullURL: docs.BaseURL + "/test-page",
		Content: "This is the test page content with unique term zyxwvut.",
	}
	_, session := newTestServer(t, "/test-page", page)

	text, isErr := callTool(t, session, "get_page", map[string]any{"url": "/test-page"})
	if isErr {
		t.Fatalf("get_page returned error: %s", text)
	}
	if !strings.Contains(text, "Test Page") {
		t.Errorf("expected title in output, got:\n%s", text)
	}
	if !strings.Contains(text, "zyxwvut") {
		t.Errorf("expected unique content term in output, got:\n%s", text)
	}
}

func TestTool_GetPage_ContainsFullURL(t *testing.T) {
	page := &docs.Page{
		Title:   "URL Check",
		URL:     "/url-check",
		FullURL: docs.BaseURL + "/url-check",
		Content: "content",
	}
	_, session := newTestServer(t, "/url-check", page)

	text, isErr := callTool(t, session, "get_page", map[string]any{"url": "/url-check"})
	if isErr {
		t.Fatalf("get_page error: %s", text)
	}
	if !strings.Contains(text, docs.BaseURL) {
		t.Errorf("expected full URL in output, got:\n%s", text)
	}
}

// ── resources ─────────────────────────────────────────────────────────────────

func TestResource_Sitemap(t *testing.T) {
	_, session := newTestServer(t, "", nil)

	ctx := context.Background()
	result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "automad://docs/sitemap",
	})
	if err != nil {
		t.Fatalf("ReadResource(sitemap): %v", err)
	}
	if len(result.Contents) == 0 {
		t.Fatal("expected sitemap contents, got none")
	}

	content := result.Contents[0]
	if content.MIMEType != "application/json" {
		t.Errorf("expected MIME type application/json, got %q", content.MIMEType)
	}
	if content.Text == "" {
		t.Error("expected non-empty sitemap text")
	}

	// Must be valid JSON.
	var parsed map[string]any
	if err := json.Unmarshal([]byte(content.Text), &parsed); err != nil {
		t.Errorf("sitemap resource is not valid JSON: %v\nContent:\n%s", err, content.Text)
	}
}

func TestResource_Sitemap_ContainsKnownSections(t *testing.T) {
	_, session := newTestServer(t, "", nil)

	ctx := context.Background()
	result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "automad://docs/sitemap",
	})
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}

	text := result.Contents[0].Text
	for _, section := range []string{"System", "User Guide", "Developer Guide"} {
		if !strings.Contains(text, section) {
			t.Errorf("expected section %q in sitemap JSON, got:\n%s", section, text)
		}
	}
}
