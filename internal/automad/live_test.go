package automad_test

import (
	"context"
	"os"
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
		if _, err := svc.Pages(ctx, automad.PagesInput{Action: "update", URL: pageURL, Title: "MCP Smoke Renamed"}); err != nil {
			t.Fatalf("page update: %v", err)
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
