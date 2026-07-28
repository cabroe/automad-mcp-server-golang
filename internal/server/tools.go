// Package server provides MCP tools, resources, and prompts that expose
// the Automad documentation to MCP-compatible AI assistants.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cabroe/automad-mcp-server-golang/internal/docs"
)

// searchDocsInput is the input schema for the search_docs tool.
type searchDocsInput struct {
	Query string `json:"query" jsonschema:"The search query to find relevant Automad documentation pages"`
}

// getPageInput is the input schema for the get_page tool.
type getPageInput struct {
	URL string `json:"url" jsonschema:"Relative URL of the documentation page (e.g. '/user-guide/creating-pages')"`
}

// listPagesInput is the input schema for the list_pages tool.
type listPagesInput struct {
	Parent string `json:"parent,omitempty" jsonschema:"Optional parent section filter (e.g. 'User Guide' or 'System'). Leave empty to list all pages."`
}

// RegisterTools adds all Automad documentation tools to the MCP server.
func RegisterTools(s *mcp.Server, svc *docs.Service) {
	registerSearchDocs(s, svc)
	registerGetPage(s, svc)
	registerListPages(s, svc)
}

// registerSearchDocs registers the search_docs tool.
func registerSearchDocs(s *mcp.Server, svc *docs.Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "search_docs",
		Description: `Search the Automad documentation for pages matching a query.
Returns a ranked list of matching pages with URL, title, parent section, and a content snippet.
Use this tool to discover which documentation pages are relevant before fetching them with get_page.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input searchDocsInput) (*mcp.CallToolResult, any, error) {
		if strings.TrimSpace(input.Query) == "" {
			return toolError("query must not be empty"), nil, nil
		}

		results := svc.Search(input.Query)
		if len(results) == 0 {
			return toolText(fmt.Sprintf("No documentation pages found for query: %q\n\nTip: Use list_pages to browse all available pages.", input.Query)), nil, nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d matching page(s) for %q:\n\n", len(results), input.Query))

		for i, r := range results {
			if i >= 10 {
				sb.WriteString(fmt.Sprintf("\n…and %d more results. Refine your query or use list_pages.", len(results)-10))
				break
			}
			sb.WriteString(fmt.Sprintf("**%s**\n", r.DocPage.Value))
			sb.WriteString(fmt.Sprintf("URL: %s%s\n", docs.BaseURL, r.DocPage.URL))
			if r.DocPage.Parent != "" {
				sb.WriteString(fmt.Sprintf("Section: %s\n", r.DocPage.Parent))
			}
			if r.Snippet != "" {
				sb.WriteString(fmt.Sprintf("Excerpt: %s\n", r.Snippet))
			}
			sb.WriteString("\n")
		}

		return toolText(sb.String()), nil, nil
	})
}

// registerGetPage registers the get_page tool.
func registerGetPage(s *mcp.Server, svc *docs.Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "get_page",
		Description: `Fetch the full content of a specific Automad documentation page.
Provide the relative URL (e.g. '/user-guide/creating-pages').
Use search_docs or list_pages to discover available URLs first.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input getPageInput) (*mcp.CallToolResult, any, error) {
		if strings.TrimSpace(input.URL) == "" {
			return toolError("url must not be empty"), nil, nil
		}

		// Check whether the URL is in the known sitemap so a fetch failure
		// can tell the caller whether the URL was simply unrecognized (most
		// likely cause of a 404) versus a transient network/parse issue.
		url := docs.NormalizeURL(input.URL)
		if url == "" {
			return toolError("url must be a relative Automad documentation path, not an absolute URL"), nil, nil
		}
		known := docs.FindByURL(url)

		page, err := svc.GetPage(ctx, url)
		if err != nil {
			if known == nil {
				return toolError(fmt.Sprintf("Failed to fetch page %q: %v\n\nThis URL is not in the known Automad documentation sitemap. Use list_pages or search_docs to find a valid URL.", input.URL, err)), nil, nil
			}
			return toolError(fmt.Sprintf("Failed to fetch page %q: %v\n\nUse list_pages to see valid URLs.", input.URL, err)), nil, nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# %s\n\n", page.Title))
		sb.WriteString(fmt.Sprintf("**URL:** %s\n\n", page.FullURL))
		if page.Offline {
			sb.WriteString("> ⚠ Offline snapshot: the live site was unreachable, so this content comes from the embedded corpus and may be out of date.\n\n")
		}
		sb.WriteString("---\n\n")
		sb.WriteString(page.Content)

		return toolText(sb.String()), nil, nil
	})
}

// registerListPages registers the list_pages tool.
func registerListPages(s *mcp.Server, svc *docs.Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "list_pages",
		Description: `List all available Automad documentation pages.
Optionally filter by parent section (e.g. 'User Guide', 'System', 'Pipe', 'Toolbox').
Returns page titles, relative URLs, and parent sections.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listPagesInput) (*mcp.CallToolResult, any, error) {
		pages := svc.ListPages(input.Parent)
		if len(pages) == 0 {
			msg := "No pages found"
			if input.Parent != "" {
				msg = fmt.Sprintf("No pages found in section %q. Use list_pages without a filter to see all sections.", input.Parent)
			}
			return toolText(msg), nil, nil
		}

		type jsonPage struct {
			Title  string `json:"title"`
			URL    string `json:"url"`
			Parent string `json:"parent,omitempty"`
		}
		out := make([]jsonPage, 0, len(pages))
		for _, p := range pages {
			out = append(out, jsonPage{
				Title:  p.Value,
				URL:    docs.BaseURL + p.URL,
				Parent: p.Parent,
			})
		}

		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return toolError("failed to marshal page list: " + err.Error()), nil, nil
		}

		header := fmt.Sprintf("Found %d documentation page(s)", len(pages))
		if input.Parent != "" {
			header += fmt.Sprintf(" in section %q", input.Parent)
		}
		header += ":\n\n"

		return toolText(header + string(data)), nil, nil
	})
}

// toolText creates a successful tool result with text content.
func toolText(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

// toolError creates a tool result that signals an error to the LLM.
func toolError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}
