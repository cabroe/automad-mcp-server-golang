# CLAUDE.md

An MCP server (Go, official `modelcontextprotocol/go-sdk`) that exposes Automad CMS to AI assistants over stdio. Four capability groups, one package each:

- `internal/docs` — automad.org documentation, live-fetched with an embedded offline fallback
- `internal/starterkit` — the `automadcms/automad-theme-starter-kit` repo via the GitHub API
- `internal/instances` — disposable Automad instances via the `docker` CLI
- `internal/automad` — live bridge to a running Automad v2 site's `/_api` dashboard API

MCP tool handlers live in `internal/server/*_tools.go` and call the service of their domain. `cmd/automad-mcp-server` wires it together.

Run `make check` (gofmt + vet + test + race + build) before calling work done.

The Automad v2 `/_api` protocol is undocumented upstream; what we know was verified against a live instance and is written down in the `internal/automad` package doc, with the invariants covered by the `TestLive*` tests in `live_test.go` (they skip unless `AUTOMAD_URL` is set).
