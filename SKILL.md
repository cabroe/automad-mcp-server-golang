---
name: automad-starter-kit-mcp
description: Access the official Automad Theme Starter Kit GitHub repository (automadcms/automad-theme-starter-kit) through the automad-mcp-server's list_files, get_file_content, get_template_snippet, search_code, and get_file_url tools. Use this whenever building, reviewing, or debugging an Automad theme and you need the Starter Kit's real file structure, an authoritative reference implementation of a component (page shell, content block, pagination, pagelist grid), or example usage of Automad template syntax (@{ }, <@ @>) as source of truth — instead of guessing from general PHP/CMS knowledge or from outdated training data.
---

# Automad Starter Kit MCP Tools

`automad-mcp-server` exposes the official Automad Theme Starter Kit
(https://github.com/automadcms/automad-theme-starter-kit) as five MCP tools,
in addition to its existing Automad documentation tools (`search_docs`,
`get_page`, `list_pages`). Use the Starter Kit tools as the **source of
truth** for what real Automad theme code looks like — the documentation
explains concepts, the Starter Kit shows a working implementation.

> **Note on naming:** the Starter Kit does not have `header.php`/`footer.php`
> files. Its `components/page.php` file combines both into a single "page
> shell" component (doctype, `<head>`, `<body>` wrapper). `get_template_snippet`
> reflects the repository's actual structure — see the table below.

## Installation

This skill requires a running `automad-mcp-server` (Go binary, MCP stdio
transport). Build it once, then point any MCP-compatible client at it.

```bash
git clone https://github.com/cabroe/automad-mcp-server-golang
cd automad-mcp-server-golang
make build
# or directly:
go build -o automad-mcp-server .
```

Register it with your MCP client, e.g. Claude Desktop
(`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "automad-docs": {
      "command": "/absolute/path/to/automad-mcp-server"
    }
  }
}
```

The same binary also works with Cursor, VS Code, and any other MCP stdio
client — see the main [README.md](README.md) for those configs.

### Optional: raise the GitHub API rate limit

By default, the Starter Kit tools call GitHub's REST API unauthenticated
(60 requests/hour, shared across all tools for that IP). If you hit rate
limits, set a token with read-only public-repo access before starting the
server:

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

With a token, the limit rises to 5000 requests/hour. The server also warms
its cache of Starter Kit files in the background on startup and caches
everything it fetches for one hour, so a normal session makes very few live
API calls regardless.

## Tools

### `list_files`

Recursively lists all files and folders in the repository as an indented tree.

**Parameters:** `path` (optional) — scope the listing to a subdirectory, e.g. `"components"`.

**Example call:**

```json
{ "path": "components" }
```

**Example response:**

```
automadcms/automad-theme-starter-kit (branch: master) — 4 entries

components/
  content.php
  page.php
  pagination.php
```

### `get_file_content`

Reads the full content of a specific file. Supported types: `.php`, `.json`,
`.md`, `.txt`, `.css`, `.js`.

**Parameters:** `path` (required), e.g. `"theme.json"` or `"components/page.php"`.

**Example call:**

```json
{ "path": "theme.json" }
```

Returns the file content plus its raw and GitHub URLs. If the live GitHub
API request fails (rate limit, network outage) and the path is one of the
files covered by `get_template_snippet` below, a bundled fallback copy is
returned instead, clearly marked with a ⚠️ warning.

### `get_template_snippet`

Returns one of a curated set of frequently-referenced Starter Kit files,
along with an explanation of what it does. Leave `name` empty to list all keys.

| Key | File | What it is |
|---|---|---|
| `page` | `components/page.php` | Base HTML document shell (doctype, `<head>`, `<body>`) — the header+footer equivalent |
| `content` | `components/content.php` | Title, date/tags subtitle, and the `+main` block editor field |
| `pagination` | `components/pagination.php` | Prev/next + numbered pagination links |
| `pagelist-grid` | `blocks/pagelist/grid.php` | Renders a pagelist as a grid of preview cards |
| `default-template` | `default.php` | Minimal template: just includes `components/page.php` |
| `pagelist-template` | `pagelist.php` | Template with a filterable, paginated pagelist |
| `page-not-found` | `page_not_found.php` | 404 template |
| `functions` | `lib/functions.php` | Example custom PHP helper function (`icon()`) |
| `theme-config` | `theme.json` | Dashboard field order, labels, select options, masks, tooltips |

**Example call:**

```json
{ "name": "pagination" }
```

### `search_code`

Searches the cached content of supported files for a literal,
case-insensitive substring — e.g. Automad template syntax (`@{`, `<@`),
statement names (`newPagelist`, `queryStringMerge`), or `theme.json` keys
(`fieldOrder`).

**Parameters:** `query` (required), `extensions` (optional, e.g. `[".php"]`).

**Example call:**

```json
{ "query": "newPagelist" }
```

**Example response:**

```
Found 1 match(es) for "newPagelist":

**pagelist.php:24**
```
<@ newPagelist {
```
```

Search only covers files already in the server's cache (warmed in the
background on startup). If a query returns fewer results than expected right
after server startup, the note in the response will say how many files were
skipped as "not yet cached" — retry in a few seconds, or call
`get_file_content` on the specific file first.

### `get_file_url`

Generates the raw-content URL and the human-browsable GitHub URL for a file,
without fetching its content.

**Parameters:** `path` (required).

**Example call:**

```json
{ "path": "components/page.php" }
```

**Example response:**

```
Raw:    https://raw.githubusercontent.com/automadcms/automad-theme-starter-kit/master/components/page.php
GitHub: https://github.com/automadcms/automad-theme-starter-kit/blob/master/components/page.php
```

## Typical prompts

- "Show me how the official Automad Starter Kit implements pagination — I want to copy the real pattern, not invent one."
- "Search the Starter Kit for every place `newPagelist` is used, and explain the parameters."
- "Compare how our theme's page shell differs from `components/page.php` in the Starter Kit."
- "List every file under `blocks/` in the Starter Kit repo."
- "Get the `theme.json` from the Starter Kit and explain what `fieldOrder` and `masks` do."
- "Give me the raw GitHub URL for the pagelist grid component so I can link to it in a PR description."

## Design notes

- **Rate limiting:** the client tracks GitHub's `X-RateLimit-*` response
  headers and refuses new requests once the quota is known to be exhausted
  (returning a clear error with the reset time) instead of burning further
  calls on requests that would just fail.
- **Caching:** the file tree (one API call via the Git Trees API) and
  individual file contents are cached for 1 hour. A background warm-up on
  server startup pre-fetches all supported files (bounded to 5 concurrent
  requests, skipping files over 100 KB) so `search_code` has full coverage
  quickly without a client needing to fetch every file first.
- **Fallbacks:** the 9 files covered by `get_template_snippet`, plus the
  top-level repository structure, are embedded in the server binary and used
  automatically if a live GitHub API call fails. Fallback responses are
  always marked with a ⚠️ warning since they can lag behind upstream.
- **Search is literal, not regex:** Automad's template syntax uses `{`, `}`,
  `<`, `>` — all regex metacharacters — so `search_code` does a plain
  case-insensitive substring match instead, matching queries like `@{` and
  `<@` exactly as typed.
