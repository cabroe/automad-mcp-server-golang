package automad

import (
	"context"
	"strings"
	"time"
)

// Service is the high-level entry point for the live Automad v2 bridge. It ties
// the HTTP client to the write guard and exposes page/media/shared/config/
// package operations. It is safe for concurrent use.
type Service struct {
	cfg     Config
	client  *Client
	guard   *writeGuard
	warning string
}

// NewService builds the bridge from the environment (see LoadConfig). When the
// three credentials are absent the returned Service reports Enabled() == false
// and every operation returns a clear UNSUPPORTED error, so the rest of the
// server keeps working with no configuration.
func NewService() *Service {
	cfg, warning := LoadConfig()
	s := &Service{cfg: cfg, guard: newWriteGuard(cfg.WriteMode), warning: warning}
	if cfg.Enabled() {
		s.client = newClient(cfg)
	}
	return s
}

// Enabled reports whether the live API bridge is configured.
func (s *Service) Enabled() bool { return s.cfg.Enabled() }

// WriteMode returns the configured write policy.
func (s *Service) WriteMode() WriteMode { return s.cfg.WriteMode }

// URL returns the configured instance base URL (empty when not configured).
func (s *Service) URL() string { return s.cfg.URL }

// ConfigWarning returns a non-fatal configuration warning, or "".
func (s *Service) ConfigWarning() string { return s.warning }

func (s *Service) ensureEnabled() error {
	if !s.cfg.Enabled() {
		return newError(CodeUnsupported,
			"live API bridge is not configured; set AUTOMAD_URL, AUTOMAD_USER and AUTOMAD_PASS to enable it")
	}
	return nil
}

// gate runs the write guard for act on target. It returns (proceed, pending
// result, error): when pending is non-nil the caller must return it to the user
// as a confirmation request; when err is non-nil the action is forbidden.
func (s *Service) gate(act action, target, confirmToken string) (map[string]any, error) {
	per := s.guard.check(act, target, confirmToken)
	if per.forbidden != "" {
		return nil, newError(CodeForbidden, "%s", per.forbidden)
	}
	if per.pending {
		return pendingResult(per), nil
	}
	return nil, nil
}

func pendingResult(p permit) map[string]any {
	return map[string]any{
		"status":        "confirmation_required",
		"action":        p.action,
		"target":        p.target,
		"confirm_token": p.confirmToken,
		"expires_at":    p.expiresAt.UTC().Format(time.RFC3339),
		"message": "This is a destructive action. Re-run the exact same call with this confirm_token " +
			"to proceed. The token is single-use and expires in 5 minutes.",
	}
}

// --- Shared (global) data -------------------------------------------------

// SharedInput selects a shared-data action.
type SharedInput struct {
	Action       string
	Fields       map[string]any
	ConfirmToken string
}

var (
	actSharedGet = action{name: "shared.get", readOnly: true}
	actSharedSet = action{name: "shared.set"}
)

// Shared reads or writes Automad's shared (site-wide) data fields.
func (s *Service) Shared(ctx context.Context, in SharedInput) (any, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	switch in.Action {
	case "get":
		if _, err := s.gate(actSharedGet, "/", in.ConfirmToken); err != nil {
			return nil, err
		}
		return s.client.post(ctx, APIBase+"/shared/data", map[string]any{})
	case "set":
		if len(in.Fields) == 0 {
			return nil, validationError("fields is required for set (a non-empty object)")
		}
		if pending, err := s.gate(actSharedSet, "/", in.ConfirmToken); err != nil || pending != nil {
			return pending, err
		}
		return s.client.post(ctx, APIBase+"/shared/data", map[string]any{"data": in.Fields})
	default:
		return nil, validationError("unknown shared action %q (want: get, set)", in.Action)
	}
}

// --- Config & cache -------------------------------------------------------

// ConfigInput selects a config/cache action.
type ConfigInput struct {
	Action       string
	Type         string
	Payload      map[string]any
	ConfirmToken string
}

var (
	actConfigGet   = action{name: "config.get", readOnly: true}
	actConfigWrite = action{name: "config.update"}
	actCacheClear  = action{name: "config.cache_clear"}
	actCachePurge  = action{name: "config.cache_purge", destructive: true}
)

// Config reads bootstrap info, updates runtime config, or clears/purges caches.
func (s *Service) Config(ctx context.Context, in ConfigInput) (any, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	switch in.Action {
	case "get":
		if _, err := s.gate(actConfigGet, "/", in.ConfirmToken); err != nil {
			return nil, err
		}
		return s.client.get(ctx, APIBase+"/app/bootstrap")
	case "update":
		if strings.TrimSpace(in.Type) == "" {
			return nil, validationError("type is required for update")
		}
		if pending, err := s.gate(actConfigWrite, in.Type, in.ConfirmToken); err != nil || pending != nil {
			return pending, err
		}
		payload := map[string]any{"type": in.Type}
		for k, v := range in.Payload {
			payload[k] = v
		}
		return s.client.post(ctx, APIBase+"/config/update", payload)
	case "cache_clear":
		if pending, err := s.gate(actCacheClear, "/", in.ConfirmToken); err != nil || pending != nil {
			return pending, err
		}
		return s.client.post(ctx, APIBase+"/cache/clear", map[string]any{})
	case "cache_purge":
		if pending, err := s.gate(actCachePurge, "/", in.ConfirmToken); err != nil || pending != nil {
			return pending, err
		}
		return s.client.post(ctx, APIBase+"/cache/purge", map[string]any{})
	default:
		return nil, validationError("unknown config action %q (want: get, update, cache_clear, cache_purge)", in.Action)
	}
}

// --- Package manager (themes/extensions) ----------------------------------

// PackagesInput selects a package-manager action.
type PackagesInput struct {
	Action       string
	Package      string
	ConfirmToken string
}

var (
	actPkgList      = action{name: "packages.list_installed", readOnly: true}
	actPkgOutdated  = action{name: "packages.outdated", readOnly: true}
	actPkgUpdate    = action{name: "packages.update"}
	actPkgUpdateAll = action{name: "packages.update_all"}
	actPkgUninstall = action{name: "packages.uninstall", destructive: true}
)

// Packages manages installed Composer packages (themes and extensions).
func (s *Service) Packages(ctx context.Context, in PackagesInput) (any, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	switch in.Action {
	case "list_installed":
		if _, err := s.gate(actPkgList, "/", in.ConfirmToken); err != nil {
			return nil, err
		}
		return s.client.post(ctx, APIBase+"/package-manager/get-package-collection", map[string]any{})
	case "outdated":
		if _, err := s.gate(actPkgOutdated, "/", in.ConfirmToken); err != nil {
			return nil, err
		}
		return s.client.post(ctx, APIBase+"/package-manager/get-outdated", map[string]any{})
	case "update":
		if strings.TrimSpace(in.Package) == "" {
			return nil, validationError("package is required for update")
		}
		if pending, err := s.gate(actPkgUpdate, in.Package, in.ConfirmToken); err != nil || pending != nil {
			return pending, err
		}
		return s.client.post(ctx, APIBase+"/package-manager/update", map[string]any{"package": in.Package})
	case "update_all":
		if pending, err := s.gate(actPkgUpdateAll, "/", in.ConfirmToken); err != nil || pending != nil {
			return pending, err
		}
		return s.client.post(ctx, APIBase+"/package-manager/update-all", map[string]any{})
	case "uninstall":
		if strings.TrimSpace(in.Package) == "" {
			return nil, validationError("package is required for uninstall")
		}
		if pending, err := s.gate(actPkgUninstall, in.Package, in.ConfirmToken); err != nil || pending != nil {
			return pending, err
		}
		return s.client.post(ctx, APIBase+"/package-manager/remove", map[string]any{"package": in.Package})
	default:
		return nil, validationError("unknown packages action %q (want: list_installed, outdated, update, update_all, uninstall)", in.Action)
	}
}
