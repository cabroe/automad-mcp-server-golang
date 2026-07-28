# Automad MCP Server

[![Go](https://img.shields.io/badge/Go-1.26-blue.svg)](https://golang.org)
[![MCP SDK](https://img.shields.io/badge/MCP%20SDK-v1.7.0-green.svg)](https://github.com/modelcontextprotocol/go-sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

An MCP (**Model Context Protocol**) server that exposes the [Automad CMS](https://automad.org) documentation, the official [Automad Theme Starter Kit](https://github.com/automadcms/automad-theme-starter-kit) repository, Docker-based Automad instance management, and a live API bridge for operating a running Automad v2 site to AI assistants. Implemented using the official [Go SDK for MCP](https://github.com/modelcontextprotocol/go-sdk).

## Features

### 🔧 Docs Tools

| Tool | Description |
|------|-------------|
| `search_docs` | Full-text search across all Automad documentation pages |
| `get_page` | Fetch the full content of a single documentation page as text |
| `list_pages` | List all documentation pages, optionally filtered by category |

The docs tools live-fetch from automad.org and cache in memory. When the site is unreachable (offline / air-gapped) they transparently fall back to an **embedded snapshot** of the full documentation (see [Technical Details](#technical-details)), so `get_page` and `search_docs` keep working — pages served from the snapshot are clearly marked as offline.

### 🧩 Starter Kit Tools

Live access to [automadcms/automad-theme-starter-kit](https://github.com/automadcms/automad-theme-starter-kit) — the "Source of Truth" for real theme files instead of guessed file names. Details, examples, and typical prompts are available in [SKILL.md](SKILL.md).

| Tool | Description |
|------|-------------|
| `list_files` | List all files/folders in the repository recursively as a tree |
| `get_file_content` | Read the content of a file (`.php`, `.json`, `.md`, `.txt`, `.css`, `.js`) |
| `get_template_snippet` | Retrieve curated, frequently used files (page component, pagination, list grid, theme.json, etc.) with explanations |
| `search_code` | Search cached code for text, e.g., Automad template syntax (`@{ }`, `<@ @>`) or theme.json keys |
| `get_file_url` | Generate Raw and GitHub URLs for any repository file without fetching its content (existence verification may fetch the repository tree) |

### 🐳 Instance Tools

Create and remotely control real Automad sites running in Docker (official `automad/automad:v2` image) — for end-to-end testing instead of just reading about behavior. Requires [Docker](https://www.docker.com/) installed and running. Details, examples, and typical prompts are available in [SKILL.md](SKILL.md).

| Tool | Description |
|------|-------------|
| `create_automad_instance` | Create and start a new Automad instance (auto-assigns a free port if none given) |
| `list_automad_instances` | List every instance managed by this server, with status and port |
| `get_automad_instance` | Get full status/detail for a single instance |
| `set_automad_instance_state` | Start, stop, or restart an instance |
| `remove_automad_instance` | Stop and remove an instance, optionally deleting its data |
| `get_automad_instance_logs` | Fetch recent container logs (e.g. the auto-generated dashboard credentials) |
| `run_automad_console_command` | Run one of Automad's own console commands (`cache:clear`, `cache:purge`, `user:create`, `update`) inside an instance |

Every container these tools create is labeled `managed-by=automad-mcp-server`. Lifecycle tools resolve only label-matched containers and then operate on the verified container ID, so unrelated containers with similar names are not affected.

### 🌐 Live API Bridge Tools

Operate a **running** Automad v2 site through its dashboard JSON API (`/_api`) — create and edit real content, not just read about it. The bridge is active only when `AUTOMAD_URL`, `AUTOMAD_USER`, and `AUTOMAD_PASS` are set; otherwise these tools return a clear "not configured" message and every other tool keeps working. Pairs naturally with the Instance Tools: spin up a disposable site in Docker, then drive it end-to-end.

| Tool | Description |
|------|-------------|
| `automad_pages` | Full page lifecycle: `get`, `list`, `create`, `update`, `delete`, `move`, `duplicate`, `publish`, `discard_draft`, `publication_state`, `breadcrumbs`, `history`, `history_restore`, and trash operations. `update` is a safe full-replace save (reads the current page and merges your changes). |
| `automad_media` | Manage files: `list`, `upload` (base64), `import` (from an http(s) URL), `delete`. |
| `automad_shared` | Read/write site-wide shared data fields: `get`, `set`. |
| `automad_config` | Inspect and control config/cache: `get` (bootstrap/system info), `update`, `cache_clear`, `cache_purge`. |
| `automad_packages` | Manage installed themes/extensions (Composer): `list_installed`, `outdated`, `update`, `update_all`, `uninstall`. |

**Write safety.** `AUTOMAD_WRITE_MODE` gates writes: `read-only` rejects all writes; `confirm-destructive` (default) allows non-destructive writes but requires confirmation for destructive ones — the first call returns a single-use `confirm_token`, and re-running the identical call with it executes the action; `unrestricted` allows everything without confirmation.

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

### Download a prebuilt binary (no Go required)

Each release ships prebuilt binaries for Linux, macOS, and Windows (amd64 and arm64) on the [Releases page](https://github.com/cabroe/automad-mcp-server-golang/releases). Download the archive for your platform, extract it, and place `automad-mcp-server` on your `PATH`.

### Install with Go

Requires Go 1.26+.

```bash
go install github.com/cabroe/automad-mcp-server-golang/cmd/automad-mcp-server@latest
```

The binary is installed as `automad-mcp-server` in `$(go env GOPATH)/bin` (or `GOBIN`, when configured). Ensure that directory is on your `PATH`. `@latest` resolves the newest published release tag; pin a specific version with e.g. `@v1.0.0`.

### Build from source

```bash
git clone https://github.com/cabroe/automad-mcp-server-golang
cd automad-mcp-server-golang
make build
# or directly:
go build -o automad-mcp-server ./cmd/automad-mcp-server
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

#### Optional: Instance Tool Settings

The instance tools work out of the box (Docker required) but can be configured via environment variables:

| Variable | Default | Purpose |
|---|---|---|
| `AUTOMAD_INSTANCES_DIR` | `~/.automad-mcp-server/instances` | Base directory for instance data (each instance gets its own subdirectory) |
| `AUTOMAD_DOCKER_IMAGE` | `automad/automad:v2` | Automad v2-compatible image used for new instances; per-call overrides must match this value |
| `AUTOMAD_STARTER_KIT_REF` | `master` | Simple Git branch or tag without `/`; slash-containing refs are rejected to keep generated file URLs unambiguous |

The server shells out to the `docker` CLI, so it also honors standard Docker environment variables like `DOCKER_HOST`/`DOCKER_CONTEXT` if you want to point it at a remote Docker daemon.

#### Optional: Live API Bridge

To enable the live API bridge tools (`automad_pages`, `automad_media`, `automad_shared`, `automad_config`, `automad_packages`), point them at a running Automad v2 instance and its dashboard credentials:

| Variable | Default | Purpose |
|---|---|---|
| `AUTOMAD_URL` | _(unset)_ | Base URL of the live Automad v2 instance (e.g. `http://127.0.0.1:18080`). Enables the bridge when set together with the two below. |
| `AUTOMAD_USER` | _(unset)_ | Dashboard username or email. |
| `AUTOMAD_PASS` | _(unset)_ | Dashboard password. |
| `AUTOMAD_WRITE_MODE` | `confirm-destructive` | Write policy: `read-only`, `confirm-destructive`, or `unrestricted`. |
| `AUTOMAD_REQUEST_TIMEOUT_MS` | `30000` | Per-request timeout in milliseconds; `0` disables it. |

When the three credentials are absent the bridge stays disabled and its tools return a clear message; all other tools work regardless. The bridge speaks Automad v2's dashboard API (`/_api`): it logs in for a session cookie, scrapes the CSRF token from the dashboard, and sends the multipart `__csrf__` + `__json__` request envelope v2 expects.

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
npx @modelcontextprotocol/inspector go run ./cmd/automad-mcp-server
```

> **Note:** The server communicates using the standardized MCP protocol via stdio (not raw JSON-RPC).
> Always use the MCP Inspector or integrate the server directly into your MCP client (Claude, Cursor, etc.) for testing.

## Project Structure

```
automad-mcp-server-golang/
├── cmd/
│   ├── automad-mcp-server/
│   │   └── main.go             # Installable command entry point
│   └── gen-corpus/
│       └── main.go             # Regenerates the embedded offline docs corpus
├── SKILL.md                    # Documentation + examples for the Starter Kit tools
├── internal/
│   ├── docs/
│   │   ├── cache.go            # In-memory cache with TTL
│   │   ├── fetcher.go          # HTTP client for automad.org
│   │   ├── parser.go           # HTML → structured text
│   │   ├── corpus.go           # Embedded offline docs corpus loader (fallback)
│   │   ├── corpus.json.gz      # Gzip-compressed snapshot of ~100 doc pages
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
│   ├── instances/
│   │   ├── docker.go           # docker CLI wrapper (os/exec, no shell)
│   │   ├── parse.go            # Parses `docker ps` output into Instance values
│   │   ├── validate.go         # Name and console-command validation
│   │   ├── errors.go           # NotFoundError, AlreadyExistsError
│   │   ├── service.go          # Coordinator: Create, List, Get, SetState, Remove, Logs, RunConsoleCommand
│   │   └── types.go            # Instance
│   ├── automad/
│   │   ├── automad.go          # Config + LoadConfig (env), write modes
│   │   ├── auth.go             # Session login + CSRF scrape
│   │   ├── client.go           # /_api HTTP client (multipart envelope, retries, upload)
│   │   ├── guard.go            # Write guard (read-only/confirm-destructive/unrestricted)
│   │   ├── errors.go           # Classified APIError
│   │   ├── service.go          # Coordinator: Shared, Config, Packages
│   │   ├── pages.go            # Page lifecycle (create/update/delete/move/trash/...)
│   │   └── media.go            # File list/upload/import/delete
│   └── server/
│       ├── tools.go              # MCP tool handlers (Docs)
│       ├── starterkit_tools.go   # MCP tool handlers (Starter Kit)
│       ├── instance_tools.go     # MCP tool handlers (Instances)
│       ├── automad_tools.go      # MCP tool handlers (Live API bridge)
│       ├── resources.go          # MCP resource handlers
│       └── prompts.go            # MCP prompt handlers
├── .goreleaser.yaml            # Cross-platform release build config
├── .github/workflows/          # ci.yml (test/build), release.yml (tagged releases)
├── Makefile
└── README.md
```

## Development

```bash
# Run tests
make test

# Run the complete local quality gate
make check

# Build a versioned binary
make build VERSION=1.0.0

# Start directly
make run

# Regenerate the embedded offline docs corpus (needs network access)
make corpus
```

### Releasing

Releases are automated with [GoReleaser](https://goreleaser.com). Pushing a semver tag triggers the [release workflow](.github/workflows/release.yml), which builds cross-platform binaries, generates a changelog, and publishes a GitHub release:

```bash
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
```

`go install ...@latest` starts resolving to the newest tag as soon as one exists. Validate the release config locally before tagging (requires the `goreleaser` CLI):

```bash
make release-check     # goreleaser check
make release-snapshot  # build binaries into ./dist without publishing
```

## Technical Details

- **Transport**: stdio (Standard MCP transport)
- **Cache TTL**: 1 hour (configurable in `docs/service.go` or `starterkit/starterkit.go`)
- **Documentation Strategy**: Live-fetch from automad.org with in-memory cache, backed by an embedded offline corpus (`internal/docs/corpus.json.gz`, gzip-compressed JSON of ~100 parsed pages). When a live fetch fails, `get_page` serves the snapshot (flagged offline, not cached so the live site is retried next time) and `search_docs` ranks on snapshot content when the cache is cold. Regenerate the corpus with `make corpus` (`go run ./cmd/gen-corpus`) when the upstream docs change
- **Starter Kit Strategy**: Live access to the GitHub REST API (Git Trees API for `list_files`, Contents API for `get_file_content`) with in-memory caching, rate limit tracking (`X-RateLimit-*` headers), and embedded fallbacks for curated snippet files during API downtime
- **Cache Warmup**: Concurrently warms sitemap pages and supported Starter Kit files (max 5 parallel, 2 min timeout) on startup so `search_docs`/`search_code` can perform full-text searches from the beginning. Concurrent cache misses for the same page/file are deduplicated.
- **Instance Strategy**: Shells out to the `docker` CLI (no Docker SDK dependency) with argument slices only — never an interpolated shell string. Every container is labeled `managed-by=automad-mcp-server`, and every lifecycle call re-verifies that label before acting, so the server can never affect a container it didn't create. `create_automad_instance` waits for Automad's first-run installation to create the console before returning. `run_automad_console_command` is restricted to Automad's four documented console subcommands rather than arbitrary shell execution. Availability of Docker itself is checked lazily per call, so its absence never affects the docs/Starter Kit tools
- **Live API Bridge Strategy**: Speaks Automad v2's dashboard API at `/_api` (there is no public REST API). It logs in for a session cookie (`Automad-<md5>`), scrapes the CSRF token from the dashboard HTML, and sends the multipart `__csrf__` + `__json__` envelope v2 expects (uploads use the Dropzone contract). Responses use v2's `{code,data,error}` envelope, including its 200-with-`error` quirk and `No session` marker, which trigger automatic CSRF-rescrape / re-login. Destructive writes are gated by a configurable write guard with single-use confirm tokens. The bridge activates only with `AUTOMAD_URL`/`AUTOMAD_USER`/`AUTOMAD_PASS` and is checked lazily, so its absence never affects the other tools. The protocol was verified live against `automad/automad:v2` (see `internal/automad/live_test.go`)
- **MCP SDK**: Official Go SDK v1.7.0

## License

MIT
