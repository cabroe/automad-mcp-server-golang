package automad

import (
	"context"
	"slices"
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
	Action string
	Fields map[string]any
	// Publish controls whether set publishes the resulting draft; nil means publish.
	Publish      *bool
	ConfirmToken string
}

var (
	actSharedGet     = action{name: "shared.get", readOnly: true}
	actSharedState   = action{name: "shared.publication_state", readOnly: true}
	actSharedSet     = action{name: "shared.set"}
	actSharedPublish = action{name: "shared.publish"}
	actSharedDiscard = action{name: "shared.discard_draft", destructive: true}
)

// Shared reads or writes Automad's shared (site-wide) data fields. Writes go to
// the draft state, so set publishes by default — an unpublished draft is
// invisible to visitors.
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

	case "publication_state":
		if _, err := s.gate(actSharedState, "/", in.ConfirmToken); err != nil {
			return nil, err
		}
		return s.client.post(ctx, APIBase+"/shared/get-publication-state", map[string]any{})

	case "set":
		if len(in.Fields) == 0 {
			return nil, validationError("fields is required for set (a non-empty object)")
		}
		if pending, err := s.gate(actSharedSet, "/", in.ConfirmToken); err != nil || pending != nil {
			return pending, err
		}
		return s.setShared(ctx, in)

	case "publish":
		if pending, err := s.gate(actSharedPublish, "/", in.ConfirmToken); err != nil || pending != nil {
			return pending, err
		}
		published, warnings := s.publishSharedAndVerify(ctx)
		result := map[string]any{"published": published}
		if len(warnings) > 0 {
			result["warnings"] = warnings
		}
		return result, nil

	case "discard_draft":
		if pending, err := s.gate(actSharedDiscard, "/", in.ConfirmToken); err != nil || pending != nil {
			return pending, err
		}
		return s.client.post(ctx, APIBase+"/shared/discard-draft", map[string]any{})

	default:
		return nil, validationError(
			"unknown shared action %q (want: get, set, publish, discard_draft, publication_state)", in.Action)
	}
}

// setShared writes shared fields as a merge: v2's Shared::save replaces the whole
// draft state, so the stored record is read first and the caller's fields merged
// on top. Without this, every field the caller did not mention is dropped —
// including the theme, which a publish would then make permanent. This mirrors
// updatePage, which solves the same full-replace problem for pages.
func (s *Service) setShared(ctx context.Context, in SharedInput) (any, error) {
	stored, err := s.readStoredShared(ctx)
	if err != nil {
		return nil, err
	}
	data := make(map[string]any, len(stored)+len(in.Fields))
	for k, v := range stored {
		data[k] = v
	}
	for k, v := range in.Fields {
		data[k] = v
	}
	saved, err := s.client.post(ctx, APIBase+"/shared/data", map[string]any{"data": data})
	if err != nil {
		return nil, err
	}
	result := map[string]any{"ok": true, "saved": saved}
	if in.Publish == nil || *in.Publish {
		published, warnings := s.publishSharedAndVerify(ctx)
		result["published"] = published
		if len(warnings) > 0 {
			result["warnings"] = warnings
		}
	}
	return result, nil
}

// readStoredShared returns the shared fields currently stored in the draft state
// (v2 falls back to the published one when there is no draft), which is exactly
// what a save replaces.
func (s *Service) readStoredShared(ctx context.Context) (map[string]any, error) {
	raw, err := s.client.post(ctx, APIBase+"/shared/data", map[string]any{})
	if err != nil {
		return nil, err
	}
	return storedSharedFields(raw), nil
}

// storedSharedFields extracts the stored values from a /shared/data record.
// v2 answers with every field the active theme supports, filling unset ones with
// "" (70+ keys against a handful actually stored), plus the "unused" fields the
// theme does not declare. Empty values are dropped so that merging the read-back
// carries only real content forward rather than materialising every default.
func storedSharedFields(raw any) map[string]any {
	rec, _ := raw.(map[string]any)
	stored := map[string]any{}
	for _, key := range []string{"fields", "unused"} {
		m, _ := rec[key].(map[string]any)
		for k, v := range m {
			if str, ok := v.(string); ok && str == "" {
				continue
			}
			stored[k] = v
		}
	}
	return stored
}

// publishSharedAndVerify publishes the shared draft and confirms it actually
// became published rather than trusting the response — the same lesson page
// publishing taught. Unlike pages no cache clear is needed here: v2's
// Shared::publish clears the render cache itself. It never throws; a draft that
// saved but did not publish is reported, not hidden.
func (s *Service) publishSharedAndVerify(ctx context.Context) (bool, []string) {
	if _, err := s.client.post(ctx, APIBase+"/shared/publish", map[string]any{}); err != nil {
		return false, []string{"saved, but publishing failed (" + err.Error() +
			") — the shared data is still a draft and visitors keep seeing the previous values; retry with the publish action"}
	}
	deadline := time.Now().Add(publishPollTimeout)
	for {
		state, err := s.client.post(ctx, APIBase+"/shared/get-publication-state", map[string]any{})
		if err == nil {
			if m, ok := state.(map[string]any); ok {
				if published, _ := m["isPublished"].(bool); published {
					return true, nil
				}
			}
		}
		if time.Now().After(deadline) {
			break
		}
		select {
		case <-ctx.Done():
			return false, []string{"publish state check was cancelled: " + ctx.Err().Error()}
		case <-time.After(publishPollInterval):
		}
	}
	return false, []string{
		"publishing was accepted but the shared data is not yet reported as published; verify with the publication_state action"}
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
		raw, err := s.client.get(ctx, APIBase+"/app/bootstrap")
		if err != nil {
			return nil, err
		}
		return trimBootstrap(raw), nil
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

// bootstrapNoise lists /app/bootstrap keys that only serve v2's own dashboard UI.
// "text" alone is ~720 translation strings and 86% of the payload; "languages"
// is the language-pack file index. Neither says anything about the site, so both
// are dropped to keep the response usable as tool output.
var bootstrapNoise = []string{"text", "languages"}

// trimBootstrap removes dashboard-only keys from a bootstrap response. Anything
// that is not the expected object is passed through untouched.
func trimBootstrap(raw any) any {
	rec, ok := raw.(map[string]any)
	if !ok {
		return raw
	}

	trimmed := make(map[string]any, len(rec))
	omitted := make([]string, 0, len(bootstrapNoise))
	for k, v := range rec {
		if slices.Contains(bootstrapNoise, k) {
			omitted = append(omitted, k)
			continue
		}
		trimmed[k] = v
	}
	if len(omitted) > 0 {
		slices.Sort(omitted)
		trimmed["omitted"] = omitted
	}
	return trimmed
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
