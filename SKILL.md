---
name: automad-mcp-server
description: Use automad-mcp-server's tools whenever working with Automad CMS — searching/reading the official docs, which also work offline from an embedded snapshot (search_docs, get_page, list_pages), consulting the official Theme Starter Kit repository as a source-of-truth reference implementation (list_files, get_file_content, get_template_snippet, search_code, get_file_url), creating and remotely controlling disposable Automad instances in Docker to test a theme or workflow end-to-end (create_automad_instance, list_automad_instances, get_automad_instance, set_automad_instance_state, remove_automad_instance, get_automad_instance_logs, run_automad_console_command), or operating a running Automad v2 site through its dashboard API — creating and editing pages, media, shared data, config/cache, and installed themes/packages (automad_pages, automad_media, automad_shared, automad_config, automad_packages). Use this whenever building, reviewing, debugging, testing, or managing content on an Automad theme or site, instead of guessing from general PHP/CMS knowledge or outdated training data.
---

# Automad MCP Server Tools

Use these instead of guessing Automad behaviour from general PHP/CMS knowledge.
Every tool's own description carries its parameters; this file covers only what
you cannot see from a tool signature. README.md has the full reference.

## Tool families

- **Docs** — `search_docs`, `get_page`, `list_pages`. The official
  documentation, live-fetched from automad.org.
- **Starter Kit** — `list_files`, `get_file_content`, `get_template_snippet`,
  `search_code`, `get_file_url`. The official
  [Theme Starter Kit](https://github.com/automadcms/automad-theme-starter-kit)
  as the source of truth for real theme code.
- **Instances** — `create_automad_instance`, `list_automad_instances`,
  `get_automad_instance`, `set_automad_instance_state`,
  `remove_automad_instance`, `get_automad_instance_logs`,
  `run_automad_console_command`. Real, disposable Automad sites in Docker.
- **Live API bridge** — `automad_pages`, `automad_media`, `automad_shared`,
  `automad_config`, `automad_packages`. Operate a running Automad v2 site.

## Configuration

Docs and Starter Kit tools need nothing. `GITHUB_TOKEN` raises the GitHub rate
limit (60/h unauthenticated, shared across all tools for that IP). Instance
tools need Docker. The live bridge activates only with `AUTOMAD_URL`,
`AUTOMAD_USER` and `AUTOMAD_PASS`; `AUTOMAD_WRITE_MODE` and
`AUTOMAD_REQUEST_TIMEOUT_MS` tune it. Unconfigured tools say so clearly and
never break the others.

## What will surprise you otherwise

- **Write safety.** In the default `confirm-destructive` mode a destructive
  live-API action first returns a single-use `confirm_token`; re-run the
  **identical** call with it to execute. `read-only` rejects all writes,
  `unrestricted` skips confirmation.
- **`automad_pages` update merges.** It reads the page and merges your changes,
  so partial updates never drop fields. `get` returns `template` as the
  `package/name` id that `update` accepts — but setting a template also pins the
  page to that package's theme, so it stops following site-wide theme changes.
- **`automad_shared` set merges and publishes.** Fields you omit survive. Shared
  writes land in a draft, so `set` publishes by default; pass `publish: false`
  to keep a draft, then use `publication_state`, `publish` or `discard_draft`.
- **Docs and Starter Kit work offline.** A snapshot of every documentation page
  and the most-used theme files is embedded in the binary. Offline answers are
  marked and can lag behind upstream.
- **`search_code` is literal, not regex.** Automad's template syntax (`@{`, `<@`)
  is full of regex metacharacters, so queries match as typed.
- **Instance tools only touch their own containers.** Every lifecycle call
  checks the managed-by label, so they cannot be pointed at other Docker
  containers on the host.
