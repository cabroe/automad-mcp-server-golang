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
