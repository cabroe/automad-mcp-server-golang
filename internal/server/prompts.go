package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cabroe/automad-mcp-server/internal/docs"
)

// RegisterPrompts adds all Automad documentation prompts to the MCP server.
func RegisterPrompts(s *mcp.Server, svc *docs.Service) {
	registerExplainConceptPrompt(s, svc)
	registerThemeDevelopmentPrompt(s, svc)
}

// registerExplainConceptPrompt registers a prompt that helps explain Automad concepts.
func registerExplainConceptPrompt(s *mcp.Server, svc *docs.Service) {
	s.AddPrompt(&mcp.Prompt{
		Name:        "explain_concept",
		Description: "Explain an Automad concept or feature using the official documentation. Searches for relevant pages and provides them as context.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "concept",
				Description: "The Automad concept or feature to explain (e.g. 'template inheritance', 'pagelist', 'pipe functions')",
				Required:    true,
			},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		concept, ok := req.Params.Arguments["concept"]
		if !ok || strings.TrimSpace(concept) == "" {
			return nil, fmt.Errorf("concept argument is required")
		}

		// Search for relevant pages.
		results := svc.Search(concept)

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Please explain the Automad concept: **%s**\n\n", concept))

		if len(results) == 0 {
			sb.WriteString("No specific documentation pages were found for this concept. ")
			sb.WriteString("Please explain based on general Automad knowledge, and mention that ")
			sb.WriteString("the user can check https://automad.org for the latest documentation.\n")
		} else {
			sb.WriteString("Use the following Automad documentation pages as your primary source:\n\n")
			limit := 5
			if len(results) < limit {
				limit = len(results)
			}
			for i, r := range results[:limit] {
				sb.WriteString(fmt.Sprintf("%d. **%s** — %s%s\n", i+1, r.DocPage.Value, docs.BaseURL, r.DocPage.URL))
				if r.Snippet != "" {
					sb.WriteString(fmt.Sprintf("   > %s\n", r.Snippet))
				}
			}
			sb.WriteString("\n")
			sb.WriteString("Please:\n")
			sb.WriteString("1. Provide a clear and concise explanation of the concept\n")
			sb.WriteString("2. Include practical examples where helpful\n")
			sb.WriteString("3. Reference the specific documentation URLs listed above\n")
			sb.WriteString("4. Mention related concepts the user might want to explore\n")
		}

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Explain Automad concept: %s", concept),
			Messages: []*mcp.PromptMessage{
				{
					Role:    mcp.Role("user"),
					Content: &mcp.TextContent{Text: sb.String()},
				},
			},
		}, nil
	})
}

// registerThemeDevelopmentPrompt registers a prompt for theme development assistance.
func registerThemeDevelopmentPrompt(s *mcp.Server, svc *docs.Service) {
	s.AddPrompt(&mcp.Prompt{
		Name:        "theme_development",
		Description: "Get guidance on Automad theme development with references to the template language documentation.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "task",
				Description: "Describe what you want to build or achieve in your Automad theme (e.g. 'create a navigation menu', 'display a pagelist with thumbnails')",
				Required:    true,
			},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		task, ok := req.Params.Arguments["task"]
		if !ok || strings.TrimSpace(task) == "" {
			return nil, fmt.Errorf("task argument is required")
		}

		// Fetch key theme development pages.
		keyPages := []string{
			"/developer-guide/building-themes/template-language",
			"/developer-guide/building-themes/template-language/toolbox",
			"/developer-guide/building-themes/template-language/pipe",
			"/developer-guide/building-themes/template-language/variables",
		}

		var contextParts []string
		for _, url := range keyPages {
			page, err := svc.GetPage(url)
			if err != nil {
				continue
			}
			if page.Content != "" {
				contextParts = append(contextParts, fmt.Sprintf("### %s\nURL: %s\n\n%s",
					page.Title, page.FullURL, truncate(page.Content, 800)))
			}
		}

		// Also search for task-specific pages.
		results := svc.Search(task)
		taskResults := []string{}
		for i, r := range results {
			if i >= 3 {
				break
			}
			taskResults = append(taskResults, fmt.Sprintf("- **%s**: %s%s", r.DocPage.Value, docs.BaseURL, r.DocPage.URL))
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("I need help with Automad theme development. My task is:\n\n**%s**\n\n", task))
		sb.WriteString("Please provide step-by-step guidance using Automad's template language.\n\n")

		if len(taskResults) > 0 {
			sb.WriteString("Relevant documentation pages for this task:\n")
			sb.WriteString(strings.Join(taskResults, "\n"))
			sb.WriteString("\n\n")
		}

		if len(contextParts) > 0 {
			sb.WriteString("Key reference documentation:\n\n")
			sb.WriteString(strings.Join(contextParts, "\n\n---\n\n"))
		}

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Theme development guidance: %s", task),
			Messages: []*mcp.PromptMessage{
				{
					Role:    mcp.Role("user"),
					Content: &mcp.TextContent{Text: sb.String()},
				},
			},
		}, nil
	})
}

// truncate shortens a string to at most n runes, appending "…" if truncated.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
