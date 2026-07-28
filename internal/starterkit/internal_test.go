package starterkit

import "testing"

// TestFallbackFilesCoverKnownSnippets ensures every get_template_snippet
// path has a corresponding embedded fallback, so a GitHub API outage never
// breaks get_template_snippet for the curated snippet set.
func TestFallbackFilesCoverKnownSnippets(t *testing.T) {
	for _, m := range knownSnippets {
		if _, ok := fallbackFiles[m.Path]; !ok {
			t.Errorf("snippet %q (%s) has no fallback content in fallbackFiles", m.Key, m.Path)
		}
	}
}

func TestIsSupportedExtension(t *testing.T) {
	cases := map[string]bool{
		"theme.json":           true,
		"components/page.php":  true,
		"README.md":            true,
		"client/index.ts":      false,
		"icons/star.svg":       false,
		"blocks/pagelist/x.js": true,
	}
	for path, want := range cases {
		if got := isSupportedExtension(path); got != want {
			t.Errorf("isSupportedExtension(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestMatchesExtFilter(t *testing.T) {
	if !matchesExtFilter("components/page.php", nil) {
		t.Error("empty filter should match everything")
	}
	if !matchesExtFilter("components/page.php", []string{".php"}) {
		t.Error("expected .php to match components/page.php")
	}
	if !matchesExtFilter("components/page.php", []string{"php"}) {
		t.Error("expected filter without a leading dot to still match")
	}
	if matchesExtFilter("theme.json", []string{".php"}) {
		t.Error("theme.json should not match a .php-only filter")
	}
}

func TestSearchLines_CaseInsensitiveSubstring(t *testing.T) {
	content := []byte("line one\n<@ newPagelist { limit: 8 } @>\nline three\n@{ title }\n")

	matches := searchLines(content, "newpagelist")
	if len(matches) != 1 || matches[0].Line != 2 {
		t.Fatalf("expected 1 match on line 2, got %+v", matches)
	}

	matches = searchLines(content, "@{")
	if len(matches) != 1 || matches[0].Line != 4 {
		t.Fatalf("expected 1 match on line 4 for '@{', got %+v", matches)
	}
}
