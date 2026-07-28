# CLAUDE.md

Guidance for working in this repo. Keep it lean; update it when architecture changes.

## What this is

An MCP server (Go, official `modelcontextprotocol/go-sdk`) exposing Automad CMS to AI assistants over stdio. Four capability groups:

- **Docs** — live-fetch + parse automad.org, with an embedded offline corpus fallback (`internal/docs`)
- **Starter Kit** — GitHub API access to `automadcms/automad-theme-starter-kit` (`internal/starterkit`)
- **Instances** — create/control Automad in Docker via the `docker` CLI (`internal/instances`)
- **Live API bridge** — operate a running Automad v2 site through its `/_api` dashboard API (`internal/automad`)

## Layout

- `cmd/automad-mcp-server/main.go` — entry point; builds services, registers tools, runs stdio transport.
- `internal/<domain>/service.go` — coordinator per domain. `internal/server/*_tools.go` — MCP tool handlers only.
- Each domain owns its own package; tools live in `internal/server` and call the service.

## Conventions

- Services **never fail at startup**. Missing prerequisites (network, Docker, credentials) are checked lazily per call so unrelated tools keep working. Follow this for anything new.
- Tool handlers return `(*mcp.CallToolResult, any, error)`; use `toolText`/`toolError` (never a Go `error` for user-facing failures). `jsonResult` renders bridge results.
- Config comes from env vars, validated in `LoadConfig`; invalid values warn and fall back, never panic.
- Go 1.22+ style: `for range n` for plain counting loops (enforced rule).
- No new external deps without cause — the bridge uses only stdlib.

## Commands

```bash
make check      # gofmt + vet + test + race + build (the gate; run before done)
make test       # tests only
make build
make corpus     # regenerate internal/docs/corpus.json.gz from the live site (needs network)
make release-check     # validate .goreleaser.yaml (needs the goreleaser CLI)
make release-snapshot  # local cross-platform build into ./dist, unpublished
```

## Releasing

GoReleaser + `.github/workflows/release.yml`, triggered by pushing a `v*` tag (`git tag -a v0.1.0 -m v0.1.0 && git push origin v0.1.0`). Builds linux/darwin/windows × amd64/arm64, a changelog, and a GitHub release; `go install ...@latest` resolves the newest tag once one exists. The release ldflag `-X main.version={{.Version}}` matches `main.version` (and CI), so the binary reports the tag. Validate config changes with `make release-check` before tagging.

## Offline docs corpus (`internal/docs`)

`get_page`/`search_docs` live-fetch automad.org. When a fetch fails they fall back to `corpus.json.gz` (gzip JSON of `url -> {title, content}`, embedded via `go:embed` in `corpus.go`):

- `GetPage` returns the snapshot with `Page.Offline = true` and does **not** cache it (so the live site is retried next call). `get_page` prints an offline banner.
- `Search` ranks on snapshot content when the cache is cold, so offline search stays comprehensive without a warmed cache.
- Regenerate after upstream doc changes: `make corpus`. The generator (`cmd/gen-corpus`) fetches every `Sitemap()` URL with the real `Fetcher`/`Parse`, writing sorted JSON for minimal diffs. It refuses to write an empty corpus.
- The embed needs the file to exist to compile; if you ever delete it, recreate a placeholder with `printf '{}' | gzip -c > internal/docs/corpus.json.gz` before regenerating.

## Live API bridge (`internal/automad`)

Talks to Automad v2's **dashboard** JSON API at `/_api` (not a public REST API — the public REST docs don't exist). Protocol, verified live against `automad/automad:v2`:

- **Auth**: `POST /_api/session/login` (urlencoded `name-or-email`,`password`) → session cookie `Automad-<md5>`. v2 returns 200 for *any* credentials, so a rejected login carries a non-empty `error` and an anonymous session.
- **CSRF**: scraped from the dashboard HTML `<meta name="csrf" content="<64-hex>">`. `/dashboard` 301-redirects; the client follows redirects manually here (the shared client does **not** auto-follow, to preserve the login `Set-Cookie`).
- **Request envelope**: POST bodies are `multipart/form-data` with `__csrf__` + `__json__` (JSON string). Uploads use the Dropzone contract (`file`, `dz*` fields, no `__json__`).
- **Response envelope**: `{code, data, error, ...}`. Non-empty `error` (even with HTTP 200) is a failure. `data.message == "No session"` means the session died → re-auth. Add/delete/publish answer with `redirect`/`success` and no `data`.
- **Write guard** (`AUTOMAD_WRITE_MODE`): `read-only` | `confirm-destructive` (default) | `unrestricted`. Destructive actions in the default mode return a single-use `confirm_token`; the caller re-runs the identical call with it to execute.
- **Page write invariants** (learned live, covered by `TestLiveBridgeMaturity`):
  - `update` is a **full-replace save**: read the current page (`fields` + `unused`), merge the caller's changes, and carry the template forward — v2 rejects a save without a title and resets any field/template not in the payload.
  - **Publish is verified, never assumed:** `/page/publish` can 200 while the page stays a draft, and `/page/data` serves drafts. Confirm via `/page/get-publication-state` (`isPublished`), then clear the render cache (v2 serves stale HTML for ~120s).
  - A title change **regenerates the slug** and the save's `redirect` can report a stale one. Publish by the caller's URL (still valid) and take the authoritative slug from the publish response's `redirect`.

Env: `AUTOMAD_URL`, `AUTOMAD_USER`, `AUTOMAD_PASS` enable the bridge; `AUTOMAD_WRITE_MODE`, `AUTOMAD_REQUEST_TIMEOUT_MS` tune it.

### Testing the bridge against a real instance

`TestLiveBridge` and `TestLiveBridgeMaturity` (`internal/automad/live_test.go`) skip unless `AUTOMAD_URL` is set. To run them, spin up a disposable instance:

```bash
D=$(mktemp -d); docker run -d --name av2 -p 127.0.0.1:18080:80 -v "$D":/app automad/automad:v2
# wait for first-run install, then create a known user:
docker exec av2 php automad/console user:create --username admin --password admin12345 --email a@b.co
# console-ready != HTTP-ready; wait for login to 200 before running (else: login EOF)
until [ "$(curl -so /dev/null -w '%{http_code}' -X POST http://127.0.0.1:18080/_api/session/login --data 'name-or-email=admin&password=admin12345')" = 200 ]; do sleep 1; done
AUTOMAD_URL=http://127.0.0.1:18080 AUTOMAD_USER=admin AUTOMAD_PASS=admin12345 \
  go test ./internal/automad -run TestLiveBridge -v
docker rm -f av2   # cleanup
```

When adding/changing a `/_api` endpoint, verify the exact path and payload against a live instance before trusting docs.
