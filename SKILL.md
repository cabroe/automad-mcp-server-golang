---
name: automad-mcp-server
description: Use automad-mcp-server's tools whenever working with Automad CMS — searching/reading the official docs, which also work offline from an embedded snapshot (search_docs, get_page, list_pages), consulting the official Theme Starter Kit repository as a source-of-truth reference implementation (list_files, get_file_content, get_template_snippet, search_code, get_file_url), creating and remotely controlling disposable Automad instances in Docker to test a theme or workflow end-to-end (create_automad_instance, list_automad_instances, get_automad_instance, set_automad_instance_state, remove_automad_instance, get_automad_instance_logs, run_automad_console_command), or operating a running Automad v2 site through its dashboard API — creating and editing pages, media, shared data, config/cache, and installed themes/packages (automad_pages, automad_media, automad_shared, automad_config, automad_packages). Use this whenever building, reviewing, debugging, testing, or managing content on an Automad theme or site, instead of guessing from general PHP/CMS knowledge or outdated training data.
---

# Automad MCP Server Tools

`automad-mcp-server` exposes four tool families to AI assistants over MCP:

- **Docs tools** (`search_docs`, `get_page`, `list_pages`) — the official Automad documentation. Live-fetched, with an embedded offline snapshot so they keep working air-gapped.
- **Starter Kit tools** (`list_files`, `get_file_content`, `get_template_snippet`, `search_code`, `get_file_url`) — the official [Theme Starter Kit](https://github.com/automadcms/automad-theme-starter-kit) repository as a **source of truth** for real theme code.
- **Instance tools** (`create_automad_instance`, `list_automad_instances`, `get_automad_instance`, `set_automad_instance_state`, `remove_automad_instance`, `get_automad_instance_logs`, `run_automad_console_command`) — create and remotely control real, disposable Automad sites running in Docker.
- **Live API bridge tools** (`automad_pages`, `automad_media`, `automad_shared`, `automad_config`, `automad_packages`) — operate a running Automad v2 site through its dashboard API: create/edit content, upload media, change shared data and config, and manage installed themes/packages. Requires credentials (see below).

Use the docs tools to understand a concept, the Starter Kit tools to see a
real implementation, the instance tools to spin up and run a disposable site,
and the live API bridge tools to actually create and edit content on it.

> **Note on naming:** the Starter Kit does not have `header.php`/`footer.php`
> files. Its `components/page.php` file combines both into a single "page
> shell" component (doctype, `<head>`, `<body>` wrapper). `get_template_snippet`
> reflects the repository's actual structure — see the table below.

## Installation

This skill requires a running `automad-mcp-server` (Go binary, MCP stdio
transport). Install it with Go, then point any MCP-compatible client at it.

```bash
go install github.com/cabroe/automad-mcp-server-golang/cmd/automad-mcp-server@latest
```

The binary is written to `$(go env GOPATH)/bin` or `GOBIN`. `@latest` resolves the newest release tag; pin a version with e.g. `@v0.1.0`. No Go toolchain? Download a prebuilt binary for your platform from the [Releases page](https://github.com/cabroe/automad-mcp-server-golang/releases) instead.

To build from a clone instead:

```bash
git clone https://github.com/cabroe/automad-mcp-server-golang
cd automad-mcp-server-golang
make build
# or directly:
go build -o automad-mcp-server ./cmd/automad-mcp-server
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

### Optional: Docker, for the instance tools

The docs and Starter Kit tools work with no further setup. The **instance
tools** additionally require [Docker](https://www.docker.com/) (Desktop or
Engine) installed and running — they shell out to the `docker` CLI. If
Docker isn't available, those seven tools return a clear error; the rest of
the server is unaffected.

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

### Optional: enable the live API bridge

The docs, Starter Kit, and instance tools need no credentials. The **live API
bridge tools** additionally require a running Automad v2 instance and its
dashboard credentials. Without these three variables the bridge stays disabled
and its tools return a clear "not configured" message; everything else works.

```json
{
  "mcpServers": {
    "automad-docs": {
      "command": "/absolute/path/to/automad-mcp-server",
      "env": {
        "AUTOMAD_URL": "http://127.0.0.1:8080",
        "AUTOMAD_USER": "admin",
        "AUTOMAD_PASS": "your-password",
        "AUTOMAD_WRITE_MODE": "confirm-destructive"
      }
    }
  }
}
```

`AUTOMAD_WRITE_MODE` gates writes: `read-only` (reject all writes),
`confirm-destructive` (default — non-destructive writes proceed; destructive
ones need a confirm token) or `unrestricted`. `AUTOMAD_REQUEST_TIMEOUT_MS`
(default `30000`) tunes the per-request timeout. Point `AUTOMAD_URL` at an
instance created with the instance tools to drive a disposable site end to end.

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

## Instance tools

Create and control real Automad sites running in Docker (official
`automad/automad:v2` image), for end-to-end testing rather than reading
about behavior. Requires Docker — see Installation above.

**Safety model:** every container these tools create is labeled
`managed-by=automad-mcp-server`; every lifecycle action re-checks that label
before acting, so these tools can never start, stop, or remove a container
they didn't create — even one that happens to share a name. There is no
arbitrary shell-exec tool: `run_automad_console_command` only runs Automad's
own four documented CLI subcommands. Instance data lives under a
server-managed directory, not an arbitrary host path a tool call could specify.

### `create_automad_instance`

Creates and starts a new instance.

**Parameters:** `name` (required), `port` (optional, auto-assigned if omitted), `image` (optional, default `automad/automad:v2`).

**Example call:**

```json
{ "name": "demo-theme" }
```

**Example response:**

```
**demo-theme**
Container: automad-mcp-demo-theme (a1b2c3d4e5f6)
Status: Up 2 seconds
Ports: 0.0.0.0:54231->80/tcp
Dashboard: http://localhost:54231/dashboard
Image: automad/automad:v2
Data directory: /Users/you/.automad-mcp-server/instances/demo-theme
Created: 2026-07-28 16:00:00 +0000 UTC
```

Automad auto-generates a dashboard user on first start — fetch the
credentials right after creating an instance with `get_automad_instance_logs`.

### `list_automad_instances`

Lists every instance this server has created, with status and port. No parameters.

### `get_automad_instance`

Full status/detail for one instance. **Parameters:** `name` (required).

### `set_automad_instance_state`

Starts, stops, or restarts an instance's container.

**Parameters:** `name` (required), `state` (required) — one of `start`, `stop`, `restart`.

### `remove_automad_instance`

Stops and removes an instance's container.

**Parameters:** `name` (required), `delete_data` (optional, default `false`) — set `true` to also permanently delete its data directory; otherwise the data is kept and can be reattached by creating a new instance with the same name.

### `get_automad_instance_logs`

Fetches recent container logs — the primary way to retrieve the
auto-generated dashboard credentials, or to debug a crash-looping container.

**Parameters:** `name` (required), `tail` (optional, default `100`).

### `run_automad_console_command`

Runs one of Automad's own CLI commands (`php automad/console <command>`)
inside a running instance.

**Parameters:** `name` (required), `command` (required) — one of `cache:clear`, `cache:purge`, `user:create`, `update`.

**Example call:**

```json
{ "name": "demo-theme", "command": "user:create" }
```

## Live API bridge tools

Operate a **running** Automad v2 site through its dashboard API (`/_api`) —
create and edit real content rather than only reading about it. Requires the
`AUTOMAD_URL`/`AUTOMAD_USER`/`AUTOMAD_PASS` credentials (see Installation); when
unset, every tool here returns a clear "not configured" message.

**Write safety.** All five tools respect `AUTOMAD_WRITE_MODE`. In the default
`confirm-destructive` mode a destructive action (delete, move, purge, uninstall,
…) first returns a single-use `confirm_token`; re-run the **identical** call with
that token to execute it. Non-destructive writes proceed without confirmation.

### `automad_pages`

Full page lifecycle. **Parameters:** `action` (required) plus action-specific
fields such as `url`, `target_url`, `title`, `template`, `private`, `tags`,
`fields`, `publish`, `layout`, `history_id`, `confirm_token`.

Actions: `get`, `list`, `create`, `update`, `delete`, `move`, `duplicate`,
`publish`, `discard_draft`, `publication_state`, `breadcrumbs`, `history`,
`history_restore`, `trash_list`, `trash_restore`, `trash_permanently_delete`,
`trash_clear`. `update` is a safe full-replace save: it reads the current page
and merges your changes, so partial updates never drop existing fields.

`get` reports `template` as the `package/name` id (e.g.
`automad/standard-lite/home`) that `create`/`update` accept, so a page can be
read, modified, and written back unchanged. Note that setting a `template` also
pins the page to that package's **theme** — Automad's own behaviour, and the
same thing the dashboard's template dropdown does — so such a page no longer
follows a later site-wide theme change.

**Example call:**

```json
{ "action": "create", "target_url": "/", "title": "About Us" }
```

### `automad_media`

Manage files: `list`, `upload` (base64), `import` (from an http(s) URL),
`delete`. Files attach to a page directory (`url`) or the shared collection
when `url` is empty or `/`.

**Parameters:** `action` (required), `url`, `filename`, `mime_type`,
`data_base64`, `import_url`, `confirm_token`.

**Example call:**

```json
{ "action": "import", "import_url": "https://example.com/logo.svg" }
```

### `automad_shared`

Read or write site-wide shared data fields (`sitename`, `theme`, …):
`get`, `set`, `publish`, `discard_draft`, `publication_state`.

`set` is a targeted write: it reads the stored data and merges your fields on
top, so the ones you omit survive. (Automad's own save replaces the whole
record, which would otherwise drop every unmentioned field — including `theme`,
without which the site refuses to render.) Pass an empty string to clear a field.

Shared writes land in a **draft** that visitors do not see, so `set` publishes by
default and reports the *verified* publication state. Pass `publish: false` to
keep the change as a draft, then inspect it with `publication_state` and either
`publish` or `discard_draft` it.

**Parameters:** `action` (required), `fields` (object, required for `set`),
`publish` (default `true`), `confirm_token` (for `discard_draft`).

```json
{ "action": "set", "fields": { "sitename": "My Site" } }
```

### `automad_config`

Inspect and control configuration and cache: `get` (bootstrap/system info),
`update` (write a config section), `cache_clear`, `cache_purge`.

**Parameters:** `action` (required), `type`, `payload`, `confirm_token`.

### `automad_packages`

Manage installed Composer packages (themes and extensions): `list_installed`,
`outdated`, `update`, `update_all`, `uninstall`.

**Parameters:** `action` (required), `package` (required for `update`/`uninstall`),
`confirm_token`.

## Typical prompts

- "Show me how the official Automad Starter Kit implements pagination — I want to copy the real pattern, not invent one."
- "Search the Starter Kit for every place `newPagelist` is used, and explain the parameters."
- "Compare how our theme's page shell differs from `components/page.php` in the Starter Kit."
- "List every file under `blocks/` in the Starter Kit repo."
- "Get the `theme.json` from the Starter Kit and explain what `fieldOrder` and `masks` do."
- "Give me the raw GitHub URL for the pagelist grid component so I can link to it in a PR description."
- "Spin up a fresh Automad instance called `demo` and give me the dashboard login."
- "Restart the `demo` instance and tail its logs."
- "Run the cache-clear console command on `demo`, then remove it entirely including its data."
- "List every Automad instance I currently have running."
- "Create an 'About Us' page under the homepage on my live site and publish it."
- "Import the logo from this URL into my site's media and set the sitename to 'Acme'."
- "Delete the /old-landing page — and yes, confirm it when you get the token."
- "List the themes installed on my live Automad instance and whether any are outdated."

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
- **Instance isolation:** the Docker CLI is always invoked with an argument
  slice, never an interpolated shell string, so instance names/ports/commands
  have no command-injection surface. Combined with the managed-by label check
  on every lifecycle call, the instance tools are scoped to containers they
  created themselves and can't be redirected at arbitrary Docker containers
  on the host.
- **Offline docs:** the docs tools live-fetch automad.org, but a snapshot of
  every documentation page is embedded in the binary. If a fetch fails,
  `get_page` serves the snapshot (marked as offline) and `search_docs` still
  ranks on real content, so the docs work air-gapped. The snapshot can lag
  behind the live site.
- **Live bridge auth:** the bridge speaks Automad v2's dashboard API — it logs
  in for a session cookie, scrapes the CSRF token, and re-authenticates
  automatically when the session expires. Credentials come only from
  environment variables, never from tool arguments.
- **Write guard:** destructive live-API actions require an explicit, single-use
  confirmation token in the default `confirm-destructive` mode, so an agent
  cannot delete or overwrite content in one unattended step.
