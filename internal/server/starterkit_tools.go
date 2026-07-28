// starterkit_tools.go registers MCP tools that expose the official Automad
// Theme Starter Kit repository (github.com/automadcms/automad-theme-starter-kit)
// as a "source of truth" reference for theme development: real templates,
// components, and configuration instead of guessed-at file names.
package server

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cabroe/automad-mcp-server/internal/starterkit"
)

// listFilesInput is the input schema for the list_files tool.
type listFilesInput struct {
	Path string `json:"path,omitempty" jsonschema:"Optional subdirectory to scope the listing to (e.g. 'components'). Leave empty to list the entire repository."`
}

// getFileContentInput is the input schema for the get_file_content tool.
type getFileContentInput struct {
	Path string `json:"path" jsonschema:"File path relative to the repository root, e.g. 'theme.json' or 'components/page.php'. Use list_files or search_code to discover paths."`
}

// getTemplateSnippetInput is the input schema for the get_template_snippet tool.
type getTemplateSnippetInput struct {
	Name string `json:"name,omitempty" jsonschema:"Snippet key (e.g. 'page', 'content', 'pagination', 'pagelist-grid', 'default-template', 'pagelist-template', 'page-not-found', 'functions', 'theme-config'). Leave empty to list all available keys."`
}

// searchCodeInput is the input schema for the search_code tool.
type searchCodeInput struct {
	Query      string   `json:"query" jsonschema:"Literal, case-insensitive text to search for, e.g. '@{' or '<@' for Automad template syntax, a statement name like 'newPagelist', or a theme.json key like 'fieldOrder'."`
	Extensions []string `json:"extensions,omitempty" jsonschema:"Optional list of file extensions to restrict the search to, e.g. ['.php']. Defaults to all supported types."`
}

// getFileURLInput is the input schema for the get_file_url tool.
type getFileURLInput struct {
	Path string `json:"path" jsonschema:"File path relative to the repository root."`
}

// RegisterStarterKitTools adds all Automad Theme Starter Kit tools to the MCP server.
func RegisterStarterKitTools(s *mcp.Server, svc *starterkit.Service) {
	registerListFiles(s, svc)
	registerGetFileContent(s, svc)
	registerGetTemplateSnippet(s, svc)
	registerSearchCode(s, svc)
	registerGetFileURL(s, svc)
}

// registerListFiles registers the list_files tool.
func registerListFiles(s *mcp.Server, svc *starterkit.Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "list_files",
		Description: fmt.Sprintf(`List all files and folders in the Automad Theme Starter Kit repository (%s/%s), recursively, as an indented tree.
Optionally scope the listing to a subdirectory via "path".
Use this to discover which files exist before fetching them with get_file_content.`, starterkit.Owner, starterkit.Repo),
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listFilesInput) (*mcp.CallToolResult, any, error) {
		prefix := ""
		var err error
		if strings.TrimSpace(input.Path) != "" {
			prefix, err = starterkit.NormalizeRepositoryPath(input.Path)
			if err != nil {
				return toolError(fmt.Sprintf("Invalid repository path %q: %v", input.Path, err)), nil, nil
			}
		}

		tree, usedFallback, err := svc.ListFiles(ctx)
		if err != nil {
			return toolError(fmt.Sprintf("Failed to list repository files: %v", err)), nil, nil
		}

		entries := tree.Entries
		if prefix != "" {
			filtered := make([]starterkit.TreeEntry, 0, len(entries))
			for _, e := range entries {
				if e.Path == prefix || strings.HasPrefix(e.Path, prefix+"/") {
					filtered = append(filtered, e)
				}
			}
			entries = filtered
			if len(entries) == 0 {
				return toolText(fmt.Sprintf("No files found under path %q.", input.Path)), nil, nil
			}
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%s/%s (branch: %s) — %d entries\n", starterkit.Owner, starterkit.Repo, svc.Branch(), len(entries)))
		if usedFallback {
			sb.WriteString("⚠️ GitHub API unavailable — showing a bundled fallback listing of the top-level structure, which may be incomplete or outdated.\n")
		}
		if tree.Truncated {
			sb.WriteString("⚠️ GitHub returned a truncated repository tree; this listing may be incomplete.\n")
		}
		sb.WriteString("\n")
		sb.WriteString(starterkit.RenderTree(entries))

		return toolText(sb.String()), nil, nil
	})
}

// registerGetFileContent registers the get_file_content tool.
func registerGetFileContent(s *mcp.Server, svc *starterkit.Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "get_file_content",
		Description: `Read the full content of a specific file in the Automad Theme Starter Kit repository.
Supported file types: .php, .json, .md, .txt, .css, .js.
Use list_files or search_code to discover paths first.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input getFileContentInput) (*mcp.CallToolResult, any, error) {
		normalizedPath, err := starterkit.NormalizeRepositoryPath(input.Path)
		if err != nil {
			return toolError(fmt.Sprintf("Invalid file path %q: %v", input.Path, err)), nil, nil
		}

		content, usedFallback, err := svc.GetFileContent(ctx, normalizedPath)
		if err != nil {
			return toolError(fmt.Sprintf("Failed to fetch %q: %v\n\nUse list_files to see valid paths.", input.Path, err)), nil, nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# %s\n\n", normalizedPath))
		rawURL, blobURL, err := svc.FileURLs(normalizedPath)
		if err != nil {
			return toolError(fmt.Sprintf("Invalid file path %q: %v", input.Path, err)), nil, nil
		}
		sb.WriteString(fmt.Sprintf("Raw: %s\nGitHub: %s\n\n", rawURL, blobURL))
		if usedFallback {
			sb.WriteString("⚠️ GitHub API unavailable — showing bundled fallback content, which may be outdated.\n\n")
		}
		sb.WriteString("---\n\n")
		sb.WriteString(fenceBlock(normalizedPath, string(content)))

		return toolText(sb.String()), nil, nil
	})
}

// registerGetTemplateSnippet registers the get_template_snippet tool.
func registerGetTemplateSnippet(s *mcp.Server, svc *starterkit.Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "get_template_snippet",
		Description: `Return a well-known, commonly reused Automad Theme Starter Kit file, together with an
explanation of what it does. Leave "name" empty to list all available snippet keys.

Note: the Starter Kit combines what other themes might split into "header.php"/"footer.php" into a single
"page" component (components/page.php) covering the full HTML document shell.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input getTemplateSnippetInput) (*mcp.CallToolResult, any, error) {
		name := strings.TrimSpace(input.Name)

		if name == "" {
			var sb strings.Builder
			sb.WriteString("Available template snippets:\n\n")
			for _, m := range starterkit.KnownSnippets() {
				sb.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", m.Key, m.Path, m.Description))
			}
			return toolText(sb.String()), nil, nil
		}

		meta, ok := starterkit.FindSnippet(name)
		if !ok {
			keys := make([]string, 0, len(starterkit.KnownSnippets()))
			for _, m := range starterkit.KnownSnippets() {
				keys = append(keys, m.Key)
			}
			return toolError(fmt.Sprintf("Unknown snippet %q. Known snippets: %s", name, strings.Join(keys, ", "))), nil, nil
		}

		content, usedFallback, err := svc.GetFileContent(ctx, meta.Path)
		if err != nil {
			return toolError(fmt.Sprintf("Failed to fetch snippet %q (%s): %v", name, meta.Path, err)), nil, nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# %s\n\n%s\n\n**File:** %s\n\n", meta.Title, meta.Description, meta.Path))
		if usedFallback {
			sb.WriteString("⚠️ GitHub API unavailable — showing bundled fallback content, which may be outdated.\n\n")
		}
		sb.WriteString(fenceBlock(meta.Path, string(content)))

		return toolText(sb.String()), nil, nil
	})
}

// registerSearchCode registers the search_code tool.
func registerSearchCode(s *mcp.Server, svc *starterkit.Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "search_code",
		Description: `Search the Automad Theme Starter Kit source for a literal, case-insensitive string — e.g.
Automad template syntax like "@{" or "<@", statement names like "newPagelist", or theme.json configuration
keys like "fieldOrder". Search runs over the server's warmed cache of file contents, so results reflect a
recent snapshot of the repository rather than always the very latest commit.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input searchCodeInput) (*mcp.CallToolResult, any, error) {
		if strings.TrimSpace(input.Query) == "" {
			return toolError("query must not be empty"), nil, nil
		}

		matches, uncached, err := svc.SearchCode(ctx, input.Query, input.Extensions)
		if err != nil {
			return toolError(fmt.Sprintf("search_code failed: %v", err)), nil, nil
		}

		if len(matches) == 0 {
			msg := fmt.Sprintf("No matches for %q.", input.Query)
			if len(uncached) > 0 {
				msg += fmt.Sprintf("\n\nNote: %d file(s) aren't cached yet and were skipped. Call get_file_content on them directly, or retry shortly while the background cache warm-up completes.", len(uncached))
			}
			return toolText(msg), nil, nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d match(es) for %q:\n\n", len(matches), input.Query))

		const limit = 25
		for i, m := range matches {
			if i >= limit {
				sb.WriteString(fmt.Sprintf("\n…and %d more matches. Narrow your query or extensions.\n", len(matches)-limit))
				break
			}
			block := fenceBlock(m.Path, m.Excerpt)
			sb.WriteString(fmt.Sprintf("**%s:%d**\n%s\n", m.Path, m.Line, block))
		}
		if len(uncached) > 0 {
			sb.WriteString(fmt.Sprintf("\nNote: %d file(s) were skipped because they aren't cached yet.\n", len(uncached)))
		}

		return toolText(sb.String()), nil, nil
	})
}

// registerGetFileURL registers the get_file_url tool.
func registerGetFileURL(s *mcp.Server, svc *starterkit.Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "get_file_url",
		Description: `Generate direct URLs (raw content and GitHub web view) for a file in the Automad Theme Starter Kit
repository, without fetching its content.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input getFileURLInput) (*mcp.CallToolResult, any, error) {
		if strings.TrimSpace(input.Path) == "" {
			return toolError("path must not be empty"), nil, nil
		}

		verificationNote := ""
		if err := svc.ValidateFilePath(ctx, input.Path); err != nil {
			var unavailable *starterkit.VerificationUnavailableError
			if errors.As(err, &unavailable) {
				verificationNote = fmt.Sprintf("\n\n⚠️ %v.", unavailable)
			} else {
				return toolError(fmt.Sprintf("Invalid file path %q: %v", input.Path, err)), nil, nil
			}
		}
		rawURL, blobURL, err := svc.FileURLs(input.Path)
		if err != nil {
			return toolError(fmt.Sprintf("Invalid file path %q: %v", input.Path, err)), nil, nil
		}
		msg := fmt.Sprintf("Raw:    %s\nGitHub: %s%s", rawURL, blobURL, verificationNote)
		return toolText(msg), nil, nil
	})
}

// fenceBlock wraps content in a Markdown code fence, using a language hint
// derived from the file's extension so tool output renders with syntax
// highlighting in clients that support it.
func fenceBlock(path, content string) string {
	lang := ""
	switch {
	case strings.HasSuffix(path, ".php"):
		lang = "php"
	case strings.HasSuffix(path, ".json"):
		lang = "json"
	case strings.HasSuffix(path, ".md"):
		lang = "markdown"
	case strings.HasSuffix(path, ".js"):
		lang = "javascript"
	case strings.HasSuffix(path, ".css"):
		lang = "css"
	}
	fence := "```"
	for strings.Contains(content, fence) {
		fence += "`"
	}
	return fmt.Sprintf("%s%s\n%s\n%s\n", fence, lang, content, fence)
}
