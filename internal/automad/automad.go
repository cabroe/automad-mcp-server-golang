// Package automad bridges to a live Automad v2 instance through its dashboard
// JSON API at /_api. It handles session login, CSRF handling, the multipart
// request envelope v2 expects (__csrf__ + __json__), and the {code,data,error}
// response envelope, and exposes high-level page/media/shared/config/package
// operations guarded by a configurable write policy.
//
// The bridge is only active when AUTOMAD_URL, AUTOMAD_USER and AUTOMAD_PASS are
// all set (see LoadConfig). Otherwise Service.Enabled reports false and every
// tool returns a clear "not configured" message, so the docs/starter-kit/
// instance tools keep working with zero configuration.
//
// # Protocol
//
// /_api is v2's dashboard API, not a public REST API — there are no upstream
// docs for it. Everything below was verified against a live automad/automad:v2
// instance, so verify any new endpoint the same way rather than trusting docs.
//
//   - Auth: POST /_api/session/login (urlencoded name-or-email, password) sets
//     the session cookie Automad-<md5>. v2 answers 200 for *any* credentials, so
//     a rejected login is recognised by a non-empty error and an anonymous
//     session, not by the status code.
//   - CSRF: scraped from the dashboard HTML meta tag <meta name="csrf"
//     content="<64-hex>">. /dashboard 301-redirects, and that one request
//     follows redirects manually — the shared client must not auto-follow, or
//     the login Set-Cookie is lost.
//   - Requests: POST bodies are multipart/form-data carrying __csrf__ and
//     __json__ (a JSON string). Uploads instead use the Dropzone contract
//     (file plus dz* fields, no __json__).
//   - Responses: {code, data, error, ...}. A non-empty error means failure even
//     with HTTP 200. data.message == "No session" means the session died and the
//     call must be retried after re-authenticating. Add/delete/publish answer
//     with redirect/success and no data at all.
//
// # Write invariants
//
// These cost real data to learn; TestLiveBridgeMaturity and TestLiveSharedBridge
// exist to keep them true.
//
//   - Saves are full replacements, for pages and for shared data alike. Both
//     write paths therefore read the stored record first and merge the caller's
//     changes on top. An unmerged write silently drops every field it did not
//     mention — including the theme, without which the site stops rendering
//     once the change is published.
//   - Publishing is verified, never assumed. /page/publish can answer 200 while
//     the page stays a draft, and /page/data serves drafts too, so publication
//     is confirmed via get-publication-state (isPublished). For pages the render
//     cache is then cleared (v2 serves stale HTML for ~120s); for shared data it
//     is not needed, because Shared::publish clears it itself.
//   - A page title change regenerates the slug, and the save's own redirect can
//     report a stale one. Publish by the caller's URL, which stays valid, and
//     take the authoritative slug from the publish response.
//   - v2 reports a page's template as an absolute server path; the get action
//     converts it to the package/name id that create and update accept. Setting
//     a template also writes the page's theme field, so such a page no longer
//     follows site-wide theme changes.
//   - /shared/data answers with every field the active theme supports and fills
//     unset ones with "" (70+ keys against a handful actually stored), so only
//     non-empty values may be carried into a merge — otherwise every default is
//     materialised into the stored file.
//
// # Testing against a real instance
//
// The TestLive* tests in live_test.go skip unless AUTOMAD_URL is set. To run
// them, spin up a disposable instance:
//
//	D=$(mktemp -d); docker run -d --name av2 -p 127.0.0.1:18080:80 -v "$D":/app automad/automad:v2
//	docker exec av2 php automad/console user:create --username admin --password admin12345 --email a@b.co
//	# console-ready != HTTP-ready; wait for a login to answer 200 (else: login EOF)
//	until [ "$(curl -so /dev/null -w '%{http_code}' -X POST http://127.0.0.1:18080/_api/session/login --data 'name-or-email=admin&password=admin12345')" = 200 ]; do sleep 1; done
//	AUTOMAD_URL=http://127.0.0.1:18080 AUTOMAD_USER=admin AUTOMAD_PASS=admin12345 \
//	  go test ./internal/automad -run TestLive -v
//	docker rm -f av2
package automad

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// APIBase is the JSON API base path exposed by Automad v2.
const APIBase = "/_api"

// WriteMode controls how destructive operations are gated.
type WriteMode string

const (
	// WriteReadOnly rejects every write; only read actions are allowed.
	WriteReadOnly WriteMode = "read-only"
	// WriteConfirmDestructive (default) allows non-destructive writes freely
	// but requires a confirm token for destructive actions (delete, move, ...).
	WriteConfirmDestructive WriteMode = "confirm-destructive"
	// WriteUnrestricted allows every action without confirmation.
	WriteUnrestricted WriteMode = "unrestricted"
)

func validWriteMode(m WriteMode) bool {
	switch m {
	case WriteReadOnly, WriteConfirmDestructive, WriteUnrestricted:
		return true
	default:
		return false
	}
}

// Config holds the live-instance connection settings, read from the
// environment by LoadConfig.
type Config struct {
	// URL is the base URL of the Automad instance (no trailing slash).
	URL string
	// Username and Password authenticate against the dashboard.
	Username string
	Password string
	// WriteMode gates destructive operations.
	WriteMode WriteMode
	// RequestTimeout bounds each HTTP request. Zero disables the timeout.
	RequestTimeout time.Duration
}

// Enabled reports whether all three credentials are present, meaning the live
// API bridge can operate.
func (c Config) Enabled() bool {
	return c.URL != "" && c.Username != "" && c.Password != ""
}

// LoadConfig reads the bridge configuration from the environment:
//
//   - AUTOMAD_URL   base URL of the live instance (enables the bridge)
//   - AUTOMAD_USER  dashboard username or email
//   - AUTOMAD_PASS  dashboard password
//   - AUTOMAD_WRITE_MODE  read-only | confirm-destructive (default) | unrestricted
//   - AUTOMAD_REQUEST_TIMEOUT_MS  per-request timeout in ms (default 30000, 0 disables)
//
// An invalid AUTOMAD_WRITE_MODE falls back to confirm-destructive and is
// reported via ConfigWarning so startup never fails on a typo.
func LoadConfig() (Config, string) {
	cfg := Config{
		URL:            strings.TrimRight(strings.TrimSpace(os.Getenv("AUTOMAD_URL")), "/"),
		Username:       strings.TrimSpace(os.Getenv("AUTOMAD_USER")),
		Password:       os.Getenv("AUTOMAD_PASS"),
		WriteMode:      WriteConfirmDestructive,
		RequestTimeout: 30 * time.Second,
	}

	var warning string
	if raw := strings.TrimSpace(os.Getenv("AUTOMAD_WRITE_MODE")); raw != "" {
		if validWriteMode(WriteMode(raw)) {
			cfg.WriteMode = WriteMode(raw)
		} else {
			warning = "invalid AUTOMAD_WRITE_MODE " + strconv.Quote(raw) + "; defaulting to confirm-destructive"
		}
	}

	if raw := strings.TrimSpace(os.Getenv("AUTOMAD_REQUEST_TIMEOUT_MS")); raw != "" {
		if ms, err := strconv.Atoi(raw); err == nil && ms >= 0 {
			cfg.RequestTimeout = time.Duration(ms) * time.Millisecond
		}
	}

	return cfg, warning
}
