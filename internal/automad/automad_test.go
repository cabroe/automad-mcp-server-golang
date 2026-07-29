package automad

import (
	"strings"
	"testing"
)

func TestExtractCSRF(t *testing.T) {
	const tok = "8ac5ff1a7e4a86dfb3d839fe0bb45a8cb0831b2eafe61e8f05a9ecfd80d1bacd"
	cases := map[string]string{
		`<meta name="csrf" content="` + tok + `">`: tok,
		`<meta content="` + tok + `" name="csrf">`: tok,
		`<meta name='csrf' content='` + tok + `'>`: tok,
		`<meta name="robots" content="noindex">`:   "",
		`<meta name="csrf" content="tooshort">`:    "",
	}
	for html, want := range cases {
		if got := extractCSRF(html); got != want {
			t.Errorf("extractCSRF(%q) = %q, want %q", html, got, want)
		}
	}
}

func TestTemplateIDFromPath(t *testing.T) {
	cases := map[string]string{
		"/app/packages/automad/standard-lite/home.php": "automad/standard-lite/home",
		"/app/packages/mcp/cafe/home.php":              "mcp/cafe/home",
		"/app/packages/automad/standard-lite/.php":     "", // no template selected
		"":                    "",
		"/no/marker/here.php": "",
	}
	for in, want := range cases {
		if got := templateIDFromPath(in); got != want {
			t.Errorf("templateIDFromPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWithTemplateID(t *testing.T) {
	record := map[string]any{
		"url":      "/sign-in",
		"template": "/app/packages/automad/standard-lite/page_full_width_centered.php",
		"fields":   map[string]any{"title": "Sign In"},
	}
	out, ok := withTemplateID(record).(map[string]any)
	if !ok {
		t.Fatalf("withTemplateID did not return a record: %T", out)
	}
	if got := out["template"]; got != "automad/standard-lite/page_full_width_centered" {
		t.Errorf("template = %v, want the id form", got)
	}
	if out["url"] != "/sign-in" || out["fields"] == nil {
		t.Errorf("other keys were lost: %v", out)
	}
	// The input record must not be mutated.
	if !strings.HasPrefix(record["template"].(string), "/app/") {
		t.Errorf("input record was mutated: %v", record["template"])
	}
	// Records without a usable template are passed through untouched.
	if got := withTemplateID(map[string]any{"url": "/x"}); asMapT(t, got)["template"] != nil {
		t.Errorf("a record without a template gained one: %v", got)
	}
	if got := withTemplateID([]any{1}); len(got.([]any)) != 1 {
		t.Errorf("a non-record payload was rewritten: %v", got)
	}
}

func TestStoredSharedFields(t *testing.T) {
	// /shared/data answers with every supported field, unset ones as "", plus
	// the theme-independent "unused" ones. Only stored values may be carried
	// into a merge-save; the empty defaults must not materialise.
	record := map[string]any{
		"fields": map[string]any{
			"sitename":      "New Project",
			"theme":         "automad/standard-lite",
			"metaTitle":     "",
			"customCSS":     "",
			"checkboxHide":  false,
			"numberMaxPage": float64(3),
		},
		"unused": map[string]any{"legacyField": "keep me", "legacyEmpty": ""},
	}
	got := storedSharedFields(record)
	want := map[string]any{
		"sitename":      "New Project",
		"theme":         "automad/standard-lite",
		"checkboxHide":  false,
		"numberMaxPage": float64(3),
		"legacyField":   "keep me",
	}
	if len(got) != len(want) {
		t.Fatalf("stored fields = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("stored[%q] = %v, want %v", k, got[k], v)
		}
	}
	// v2 encodes an empty "unused" set as a JSON array, not an object.
	if got := storedSharedFields(map[string]any{"fields": map[string]any{"theme": "t"}, "unused": []any{}}); len(got) != 1 {
		t.Errorf("empty unused array mishandled: %v", got)
	}
	if got := storedSharedFields("not a record"); len(got) != 0 {
		t.Errorf("non-record payload = %v, want empty", got)
	}
}

func asMapT(t *testing.T, v any) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("not a map: %T", v)
	}
	return m
}

func TestSlugFromResult(t *testing.T) {
	if got := slugFromResult(map[string]any{"redirect": "page?url=%2Fblog%2Fpost"}); got != "/blog/post" {
		t.Errorf("redirect slug = %q, want /blog/post", got)
	}
	if got := slugFromResult(map[string]any{"slug": "about"}); got != "/about" {
		t.Errorf("slug field = %q, want /about", got)
	}
	if got := slugFromResult(map[string]any{"success": "ok"}); got != "" {
		t.Errorf("no slug = %q, want empty", got)
	}
}

func TestUnwrap(t *testing.T) {
	t.Run("data payload", func(t *testing.T) {
		v, err := unwrap([]byte(`{"code":200,"data":{"x":1}}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m, _ := v.(map[string]any)
		if m["x"] != float64(1) {
			t.Errorf("got %v", v)
		}
	})
	t.Run("error field", func(t *testing.T) {
		_, err := unwrap([]byte(`{"code":200,"error":"Title missing!"}`))
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("empty error is success", func(t *testing.T) {
		v, err := unwrap([]byte(`{"code":200,"error":"","reload":true}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m, _ := v.(map[string]any)
		if m["reload"] != true {
			t.Errorf("got %v", v)
		}
	})
	t.Run("redirect without data", func(t *testing.T) {
		v, err := unwrap([]byte(`{"code":200,"redirect":"page?url=%2F"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m, _ := v.(map[string]any)
		if m["redirect"] != "page?url=%2F" {
			t.Errorf("got %v", v)
		}
	})
	t.Run("no session marker", func(t *testing.T) {
		_, err := unwrap([]byte(`{"code":200,"data":{"message":"No session"}}`))
		apiErr, ok := err.(*APIError)
		if !ok || apiErr.Code != CodeAuth {
			t.Fatalf("want AUTH error, got %v", err)
		}
	})
}

func TestEnvelopeIsNoSession(t *testing.T) {
	if !envelopeIsNoSession([]byte(`{"code":200,"data":{"message":"No session"}}`)) {
		t.Error("expected No session detected")
	}
	if envelopeIsNoSession([]byte(`{"code":200,"data":{"message":"hi"}}`)) {
		t.Error("false positive")
	}
}

func TestWriteGuardConfirmFlow(t *testing.T) {
	g := newWriteGuard(WriteConfirmDestructive)
	del := action{name: "pages.delete", destructive: true}

	// Read actions always pass.
	if p := g.check(action{name: "pages.get", readOnly: true}, "/x", ""); !p.allowed {
		t.Fatal("read action should be allowed")
	}
	// Non-destructive write passes without a token.
	if p := g.check(action{name: "pages.update"}, "/x", ""); !p.allowed {
		t.Fatal("non-destructive write should be allowed")
	}
	// Destructive write first returns pending with a token.
	p := g.check(del, "/x", "")
	if !p.pending || p.confirmToken == "" {
		t.Fatalf("expected pending permit with token, got %+v", p)
	}
	// A wrong token is rejected.
	if r := g.check(del, "/x", "wrong"); r.forbidden == "" {
		t.Fatal("wrong token should be forbidden")
	}
	// Re-issue and confirm with the fresh token.
	p2 := g.check(del, "/x", "")
	if ok := g.check(del, "/x", p2.confirmToken); !ok.allowed {
		t.Fatalf("valid token should allow, got %+v", ok)
	}
	// The token is single-use.
	if again := g.check(del, "/x", p2.confirmToken); again.forbidden == "" {
		t.Fatal("token should be single-use")
	}
}

func TestWriteGuardReadOnlyMode(t *testing.T) {
	g := newWriteGuard(WriteReadOnly)
	if p := g.check(action{name: "pages.update"}, "/x", ""); p.forbidden == "" {
		t.Fatal("read-only mode should forbid writes")
	}
	if p := g.check(action{name: "pages.get", readOnly: true}, "/x", ""); !p.allowed {
		t.Fatal("read-only mode should allow reads")
	}
}

func TestWriteGuardUnrestricted(t *testing.T) {
	g := newWriteGuard(WriteUnrestricted)
	if p := g.check(action{name: "pages.delete", destructive: true}, "/x", ""); !p.allowed {
		t.Fatal("unrestricted mode should allow destructive writes without a token")
	}
}

func TestValidateImportURL(t *testing.T) {
	if err := validateImportURL("https://example.com/a.png"); err != nil {
		t.Errorf("valid https rejected: %v", err)
	}
	if err := validateImportURL("ftp://example.com/a.png"); err == nil {
		t.Error("ftp should be rejected")
	}
	if err := validateImportURL(""); err == nil {
		t.Error("empty should be rejected")
	}
}

func TestValidateFilename(t *testing.T) {
	for _, bad := range []string{"", "a/b.png", `a\b.png`, "../etc"} {
		if err := validateFilename(bad); err == nil {
			t.Errorf("filename %q should be rejected", bad)
		}
	}
	if err := validateFilename("logo.png"); err != nil {
		t.Errorf("valid filename rejected: %v", err)
	}
}

func TestTrashPath(t *testing.T) {
	if _, err := trashPath("/.trash/old-page", "trash_restore"); err != nil {
		t.Errorf("valid trash path rejected: %v", err)
	}
	if _, err := trashPath("/live-page", "trash_restore"); err == nil {
		t.Error("non-trash path should be rejected")
	}
}

func TestLoadConfigDisabledByDefault(t *testing.T) {
	t.Setenv("AUTOMAD_URL", "")
	t.Setenv("AUTOMAD_USER", "")
	t.Setenv("AUTOMAD_PASS", "")
	cfg, _ := LoadConfig()
	if cfg.Enabled() {
		t.Error("bridge should be disabled without credentials")
	}
	if cfg.WriteMode != WriteConfirmDestructive {
		t.Errorf("default write mode = %q, want confirm-destructive", cfg.WriteMode)
	}
}

// TestTrimBootstrapDropsDashboardOnlyKeys pins that config get stays usable as
// tool output: v2's bootstrap ships ~720 UI translation strings under "text",
// which was 86% of a 51 KB response and said nothing about the site.
func TestTrimBootstrapDropsDashboardOnlyKeys(t *testing.T) {
	raw := map[string]any{
		"version":   "2.0.0-beta.51",
		"sitename":  "New Project",
		"dashboard": "/dashboard",
		"text":      map[string]any{"aboutAutomad": "About Automad"},
		"languages": map[string]any{"German": "/lib/german.json"},
	}

	trimmed, ok := trimBootstrap(raw).(map[string]any)
	if !ok {
		t.Fatal("trimBootstrap did not return an object")
	}

	for _, key := range []string{"text", "languages"} {
		if _, present := trimmed[key]; present {
			t.Errorf("key %q should have been dropped", key)
		}
	}
	for _, key := range []string{"version", "sitename", "dashboard"} {
		if _, present := trimmed[key]; !present {
			t.Errorf("key %q should have been kept", key)
		}
	}

	omitted, _ := trimmed["omitted"].([]string)
	if len(omitted) != 2 || omitted[0] != "languages" || omitted[1] != "text" {
		t.Errorf("omitted = %v, want [languages text]", omitted)
	}
}

func TestTrimBootstrapPassesThroughNonObjects(t *testing.T) {
	if got := trimBootstrap("not an object"); got != "not an object" {
		t.Errorf("non-object payload was altered: %v", got)
	}
}
