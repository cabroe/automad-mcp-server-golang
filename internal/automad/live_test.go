package automad_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cabroe/automad-mcp-server-golang/internal/automad"
)

// TestLiveBridge exercises the full page/media/config/package surface against a
// real Automad v2 instance. It is skipped unless AUTOMAD_URL (plus AUTOMAD_USER
// and AUTOMAD_PASS) point at a reachable instance, so `go test ./...` stays
// hermetic while a developer can run it against a disposable Docker instance:
//
//	AUTOMAD_URL=http://127.0.0.1:18080 AUTOMAD_USER=admin AUTOMAD_PASS=... \
//	  go test ./internal/automad -run TestLiveBridge -v
func TestLiveBridge(t *testing.T) {
	if os.Getenv("AUTOMAD_URL") == "" {
		t.Skip("AUTOMAD_URL not set; skipping live bridge integration test")
	}
	// Default to confirm-destructive so the confirm flow is exercised.
	svc := automad.NewService()
	if !svc.Enabled() {
		t.Fatal("service not enabled; set AUTOMAD_URL, AUTOMAD_USER, AUTOMAD_PASS")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("shared_get", func(t *testing.T) {
		if _, err := svc.Shared(ctx, automad.SharedInput{Action: "get"}); err != nil {
			t.Fatalf("shared get: %v", err)
		}
	})

	t.Run("config_get", func(t *testing.T) {
		if _, err := svc.Config(ctx, automad.ConfigInput{Action: "get"}); err != nil {
			t.Fatalf("config get: %v", err)
		}
	})

	t.Run("packages_list", func(t *testing.T) {
		if _, err := svc.Packages(ctx, automad.PackagesInput{Action: "list_installed"}); err != nil {
			t.Fatalf("packages list_installed: %v", err)
		}
	})

	t.Run("media_list", func(t *testing.T) {
		if _, err := svc.Media(ctx, automad.MediaInput{Action: "list", URL: "/"}); err != nil {
			t.Fatalf("media list: %v", err)
		}
	})

	// Page lifecycle: create -> get -> update -> delete (with confirm flow).
	var pageURL string
	t.Run("page_create", func(t *testing.T) {
		res, err := svc.Pages(ctx, automad.PagesInput{Action: "create", TargetURL: "/", Title: "MCP Smoke"})
		if err != nil {
			t.Fatalf("page create: %v", err)
		}
		m, ok := res.(map[string]any)
		if !ok {
			t.Fatalf("create result not a map: %T", res)
		}
		url, _ := m["url"].(string)
		if url == "" {
			t.Fatalf("create returned no url: %v", m)
		}
		pageURL = url
	})

	t.Run("page_get", func(t *testing.T) {
		if pageURL == "" {
			t.Skip("no page created")
		}
		if _, err := svc.Pages(ctx, automad.PagesInput{Action: "get", URL: pageURL}); err != nil {
			t.Fatalf("page get: %v", err)
		}
	})

	t.Run("page_update", func(t *testing.T) {
		if pageURL == "" {
			t.Skip("no page created")
		}
		// A title change regenerates and publishes the slug, so track the
		// authoritative URL returned by the update for the delete step below.
		res, err := svc.Pages(ctx, automad.PagesInput{Action: "update", URL: pageURL, Title: "MCP Smoke Renamed"})
		if err != nil {
			t.Fatalf("page update: %v", err)
		}
		if u, _ := asMap(res)["url"].(string); u != "" {
			pageURL = u
		}
	})

	t.Run("page_delete_confirm_flow", func(t *testing.T) {
		if pageURL == "" {
			t.Skip("no page created")
		}
		// First call: expect a confirmation request, not a deletion.
		res, err := svc.Pages(ctx, automad.PagesInput{Action: "delete", URL: pageURL})
		if err != nil {
			t.Fatalf("page delete (first call): %v", err)
		}
		m, ok := res.(map[string]any)
		if !ok || m["status"] != "confirmation_required" {
			t.Fatalf("expected confirmation_required, got %v", res)
		}
		token, _ := m["confirm_token"].(string)
		if token == "" {
			t.Fatalf("no confirm_token in %v", m)
		}
		// Second call with the token: expect actual deletion.
		if _, err := svc.Pages(ctx, automad.PagesInput{Action: "delete", URL: pageURL, ConfirmToken: token}); err != nil {
			t.Fatalf("page delete (confirmed): %v", err)
		}
	})
}

// TestLiveBridgeMaturity locks in the behaviors the TS server learned the hard
// way and that this Go port must match: a safe full-replace update (fields the
// caller didn't mention survive), template carry-forward, real publish
// verification (not v2's "publish lie"), and the trash path round-trip. Runs in
// unrestricted write mode so the destructive steps execute without token
// juggling — the confirm flow itself is covered by TestLiveBridge.
func TestLiveBridgeMaturity(t *testing.T) {
	if os.Getenv("AUTOMAD_URL") == "" {
		t.Skip("AUTOMAD_URL not set; skipping live bridge integration test")
	}
	t.Setenv("AUTOMAD_WRITE_MODE", "unrestricted")
	svc := automad.NewService()
	if !svc.Enabled() {
		t.Fatal("service not enabled; set AUTOMAD_URL, AUTOMAD_USER, AUTOMAD_PASS")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	no := false
	// Create an explicit draft so we can prove it is NOT published yet. The
	// page's slug can change when its title changes, so we thread the current
	// URL (`cur`) through the test rather than assuming it stays put.
	created, err := svc.Pages(ctx, automad.PagesInput{Action: "create", TargetURL: "/", Title: "Maturity Draft", Publish: &no})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	cur, _ := asMap(created)["url"].(string)
	if cur == "" {
		t.Fatalf("create returned no url: %v", created)
	}
	t.Cleanup(func() {
		// Best-effort cleanup; the container is disposable anyway.
		_, _ = svc.Pages(context.Background(), automad.PagesInput{Action: "delete", URL: cur})
	})

	t.Run("draft_is_not_published", func(t *testing.T) {
		st, err := svc.Pages(ctx, automad.PagesInput{Action: "publication_state", URL: cur})
		if err != nil {
			t.Fatalf("publication_state: %v", err)
		}
		if pub, _ := asMap(st)["isPublished"].(bool); pub {
			t.Errorf("a draft (publish=false) reports isPublished=true: %v", st)
		}
	})

	// Seed a template (page/add assigns none) plus a custom field the active
	// template does not declare (v2 files it under "unused"). Both must survive
	// later updates: the template carried forward, the field merged.
	if _, err := svc.Pages(ctx, automad.PagesInput{
		Action: "update", URL: cur, Title: "Maturity Draft",
		Template: "automad/standard-lite/page_full_width_left",
		Fields:   map[string]any{"color": "blue"}, Publish: &no,
	}); err != nil {
		t.Fatalf("seed template + custom field: %v", err)
	}

	t.Run("publish_is_verified_not_assumed", func(t *testing.T) {
		// Default publish: publishAndVerify only reports true after confirming
		// isPublished via get-publication-state — not from a bare 200.
		upd, err := svc.Pages(ctx, automad.PagesInput{Action: "update", URL: cur, Title: "Maturity Draft"})
		if err != nil {
			t.Fatalf("update+publish: %v", err)
		}
		if pub, _ := asMap(upd)["published"].(bool); !pub {
			t.Fatalf("update did not confirm publication: %v", upd)
		}
		if u, _ := asMap(upd)["url"].(string); u != "" {
			cur = u
		}
		st, err := svc.Pages(ctx, automad.PagesInput{Action: "publication_state", URL: cur})
		if err != nil {
			t.Fatalf("publication_state: %v", err)
		}
		if pub, _ := asMap(st)["isPublished"].(bool); !pub {
			t.Errorf("page reported published but publication_state says otherwise: %v", st)
		}
	})

	t.Run("publish_survives_slug_change_from_rename", func(t *testing.T) {
		// Renaming regenerates the slug; the save's redirect can report a stale
		// one. Publishing must still succeed (we publish by the original URL and
		// resolve the authoritative slug from the publish response).
		upd, err := svc.Pages(ctx, automad.PagesInput{Action: "update", URL: cur, Title: "Maturity Renamed"})
		if err != nil {
			t.Fatalf("rename update: %v", err)
		}
		if pub, _ := asMap(upd)["published"].(bool); !pub {
			t.Fatalf("publish after rename not confirmed: %v", upd)
		}
		if u, _ := asMap(upd)["url"].(string); u != "" {
			cur = u
		}
	})

	t.Run("update_merges_safely_and_keeps_template", func(t *testing.T) {
		// An update that doesn't mention the custom field must not drop it, and
		// must not reset the template (which would break rendering).
		if _, err := svc.Pages(ctx, automad.PagesInput{Action: "update", URL: cur, Title: "Maturity Renamed"}); err != nil {
			t.Fatalf("merge update: %v", err)
		}
		page, err := svc.Pages(ctx, automad.PagesInput{Action: "get", URL: cur})
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got := pageField(page, "color"); got != "blue" {
			t.Errorf("custom field dropped by update: color=%q, want blue", got)
		}
		tmpl, _ := asMap(page)["template"].(string)
		if tmpl == "" || strings.HasSuffix(tmpl, "/.php") {
			t.Errorf("template was reset by update (would break rendering): %q", tmpl)
		}
	})

	t.Run("trash_roundtrip_uses_trash_path", func(t *testing.T) {
		if _, err := svc.Pages(ctx, automad.PagesInput{Action: "delete", URL: cur}); err != nil {
			t.Fatalf("delete: %v", err)
		}
		list, err := svc.Pages(ctx, automad.PagesInput{Action: "trash_list"})
		if err != nil {
			t.Fatalf("trash_list: %v", err)
		}
		trashPath := findTrashPath(list, cur)
		if trashPath == "" {
			t.Fatalf("deleted page not found in trash (or missing /.trash/ path): %v", list)
		}
		if _, err := svc.Pages(ctx, automad.PagesInput{Action: "trash_restore", URL: trashPath}); err != nil {
			t.Fatalf("trash_restore by path %q: %v", trashPath, err)
		}
	})
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

// pageField returns a page field value from either the template-declared
// "fields" map or v2's "unused" map (where fields the active template does not
// declare are stored).
func pageField(page any, key string) string {
	m := asMap(page)
	if f, ok := m["fields"].(map[string]any); ok {
		if v, ok := f[key].(string); ok {
			return v
		}
	}
	if u, ok := m["unused"].(map[string]any); ok {
		if v, ok := u[key].(string); ok {
			return v
		}
	}
	return ""
}

// findTrashPath returns the "/.trash/..." path for the deleted page whose slug
// matches url, or "" if not present.
func findTrashPath(list any, url string) string {
	items, ok := list.([]any)
	if !ok {
		return ""
	}
	base := url[strings.LastIndex(url, "/")+1:]
	for _, it := range items {
		p, _ := asMap(it)["path"].(string)
		if strings.Contains(p, "/.trash/") && strings.Contains(p, base) {
			return p
		}
	}
	return ""
}
