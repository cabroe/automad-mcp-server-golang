# Automad MCP Server

[![Go](https://img.shields.io/badge/Go-1.26-blue.svg)](https://golang.org)
[![MCP SDK](https://img.shields.io/badge/MCP%20SDK-v1.7.0-green.svg)](https://github.com/modelcontextprotocol/go-sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

An MCP (**Model Context Protocol**) server that exposes the [Automad CMS](https://automad.org) documentation and the official [Automad Theme Starter Kit](https://github.com/automadcms/automad-theme-starter-kit) repository to AI assistants. Implemented using the official [Go SDK for MCP](https://github.com/modelcontextprotocol/go-sdk).

## Features

### 🔧 Docs Tools

| Tool | Description |
|------|-------------|
| `search_docs` | Full-text search across all Automad documentation pages |
| `get_page` | Fetch the full content of a single documentation page as text |
| `list_pages` | List all documentation pages, optionally filtered by category |

### 🧩 Starter Kit Tools

Live access to [automadcms/automad-theme-starter-kit](https://github.com/automadcms/automad-theme-starter-kit) — the "Source of Truth" for real theme files instead of guessed file names. Details, examples, and typical prompts are available in [SKILL.md](SKILL.md).

| Tool | Description |
|------|-------------|
| `list_files` | List all files/folders in the repository recursively as a tree |
| `get_file_content` | Read the content of a file (`.php`, `.json`, `.md`, `.txt`, `.css`, `.js`) |
| `get_template_snippet` | Retrieve curated, frequently used files (page component, pagination, list grid, theme.json, etc.) with explanations |
| `search_code` | Search cached code for text, e.g., Automad template syntax (`@{ }`, `<@ @>`) or theme.json keys |
| `get_file_url` | Generate Raw and GitHub URLs for a file, without fetching it |

### 📦 Resources

| URI | Description |
|-----|-------------|
| `automad://docs/sitemap` | Full documentation structure as JSON |

### 💬 Prompts

| Prompt | Description |
|--------|-------------|
| `explain_concept` | Explain an Automad concept using the official documentation |
| `theme_development` | Guided theme development assistance with documentation context |

## Installation

### Prerequisites

- Go 1.26+

### Build

```bash
git clone https://github.com/cabroe/automad-mcp-server-golang
cd automad-mcp-server-golang
make build
# or directly:
go build -o automad-mcp-server .
```

### For AI Agents (Claude Code, Cursor Agent, etc.)

Installation instructions, tool references with examples, and typical prompts for AI agents are documented in [SKILL.md](SKILL.md).

## Configuration

### Claude Desktop

Add the following to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "automad-docs": {
      "command": "/absolute/path/to/automad-mcp-server"
    }
  }
}
```

#### Optional: Increase GitHub Rate Limit

The Starter Kit tools access the GitHub REST API unauthenticated (limited to 60 requests/hour). If needed, set a `GITHUB_TOKEN` (Read-only, public repos) to increase the limit to 5000 requests/hour:

```json
{
  "mcpServers": {
    "automad-docs": {
      "command": "/absolute/path/to/automad-mcp-server",
      "env": { "GITHUB_TOKEN": "ghp_..." }
    }
  }
}
```

### Cursor

In `.cursor/mcp.json` or globally in `~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "automad-docs": {
      "command": "/absolute/path/to/automad-mcp-server",
      "args": []
    }
  }
}
```

### VS Code (with Copilot / MCP Extension)

```json
{
  "mcp": {
    "servers": {
      "automad-docs": {
        "type": "stdio",
        "command": "/absolute/path/to/automad-mcp-server"
      }
    }
  }
}
```

## Usage

### Testing with MCP Inspector

```bash
# Starts the interactive inspector in the browser
npx @modelcontextprotocol/inspector go run .
```

> **Note:** The server communicates using the standardized MCP protocol via stdio (not raw JSON-RPC).
> Always use the MCP Inspector or integrate the server directly into your MCP client (Claude, Cursor, etc.) for testing.

## Project Structure

```
automad-mcp-server-golang/
├── main.go                     # Entry point
├── SKILL.md                    # Documentation + examples for the Starter Kit tools
├── internal/
│   ├── docs/
│   │   ├── cache.go            # In-memory cache with TTL
│   │   ├── fetcher.go          # HTTP client for automad.org
│   │   ├── parser.go           # HTML → structured text
│   │   ├── service.go          # Coordinator: GetPage, Search, ListPages, WarmCache
│   │   └── sitemap.go          # All known sitemap URLs
│   ├── starterkit/
│   │   ├── client.go           # GitHub API client (Trees + Contents), rate-limit tracking
│   │   ├── cache.go            # In-memory cache with TTL (Tree + files)
│   │   ├── fallback.go         # Embedded fallback content for API downtime
│   │   ├── search.go           # Search helpers for search_code
│   │   ├── snippets.go         # Registry for get_template_snippet
│   │   ├── tree.go             # Tree rendering for list_files
│   │   ├── service.go          # Coordinator: ListFiles, GetFileContent, SearchCode, WarmFiles
│   │   └── types.go            # Tree/TreeEntry, error types
│   └── server/
│       ├── tools.go              # MCP tool handlers (Docs)
│       ├── starterkit_tools.go   # MCP tool handlers (Starter Kit)
│       ├── resources.go          # MCP resource handlers
│       └── prompts.go            # MCP prompt handlers
├── Makefile
└── README.md
```

## Development

```bash
# Run tests
make test

# Build binary
make build

# Start directly
make run
```

## Technical Details

- **Transport**: stdio (Standard MCP transport)
- **Cache TTL**: 1 hour (configurable in `docs/service.go` or `starterkit/starterkit.go`)
- **Documentation Strategy**: Live-fetch from automad.org with in-memory cache
- **Starter Kit Strategy**: Live access to the GitHub REST API (Git Trees API for `list_files`, Contents API for `get_file_content`) with in-memory caching, rate limit tracking (`X-RateLimit-*` headers), and embedded fallbacks for curated snippet files during API downtime
- **Cache Warmup**: Concurrently warms sitemap pages and supported Starter Kit files (max 5 parallel, 2 min timeout) on startup so `search_docs`/`search_code` can perform full-text searches from the beginning
- **MCP Version**: 2026-07-28 (via go-sdk v1.7.0)

## License

MIT
