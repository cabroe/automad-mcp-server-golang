package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cabroe/automad-mcp-server/internal/docs"
)

const (
	// sitemapURI is the MCP resource URI for the documentation sitemap.
	sitemapURI = "automad://docs/sitemap"
)

// RegisterResources adds all Automad documentation resources to the MCP server.
func RegisterResources(s *mcp.Server, svc *docs.Service) {
	registerSitemapResource(s, svc)
}

// registerSitemapResource exposes the full Automad documentation sitemap as an MCP resource.
func registerSitemapResource(s *mcp.Server, svc *docs.Service) {
	s.AddResource(&mcp.Resource{
		URI:         sitemapURI,
		Name:        "Automad Documentation Sitemap",
		Description: "The complete list of all Automad documentation pages, organized by section. Use this to discover available documentation URLs.",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		grouped := docs.SitemapByParent()

		type pageEntry struct {
			Title string `json:"title"`
			URL   string `json:"url"`
		}
		type section struct {
			Pages []pageEntry `json:"pages"`
		}

		result := make(map[string]section)
		for parent, pages := range grouped {
			key := parent
			if key == "" {
				key = "Root"
			}
			entries := make([]pageEntry, 0, len(pages))
			for _, p := range pages {
				entries = append(entries, pageEntry{
					Title: p.Value,
					URL:   docs.BaseURL + p.URL,
				})
			}
			result[key] = section{Pages: entries}
		}

		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("serializing sitemap: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      sitemapURI,
					MIMEType: "application/json",
					Text:     strings.TrimSpace(string(data)),
				},
			},
		}, nil
	})
}
