package docs_test

import (
	"testing"

	"github.com/cabroe/automad-mcp-server/internal/docs"
)

func TestSitemap_NotEmpty(t *testing.T) {
	pages := docs.Sitemap()
	if len(pages) == 0 {
		t.Fatal("Sitemap() must not return an empty list")
	}
}

func TestSitemap_AllPagesHaveURL(t *testing.T) {
	for _, p := range docs.Sitemap() {
		if p.URL == "" {
			t.Errorf("page %q has empty URL", p.Value)
		}
	}
}

func TestSitemap_AllPagesHaveTitle(t *testing.T) {
	for _, p := range docs.Sitemap() {
		if p.Value == "" {
			t.Errorf("page with URL %q has empty title", p.URL)
		}
	}
}

func TestSitemap_URLsStartWithSlash(t *testing.T) {
	for _, p := range docs.Sitemap() {
		if len(p.URL) == 0 || p.URL[0] != '/' {
			t.Errorf("page %q has URL %q which does not start with '/'", p.Value, p.URL)
		}
	}
}

func TestSitemap_NoDuplicateURLs(t *testing.T) {
	seen := make(map[string]bool)
	for _, p := range docs.Sitemap() {
		if seen[p.URL] {
			t.Errorf("duplicate URL %q in sitemap", p.URL)
		}
		seen[p.URL] = true
	}
}

func TestSitemapByParent_GroupsCorrectly(t *testing.T) {
	grouped := docs.SitemapByParent()
	if len(grouped) == 0 {
		t.Fatal("SitemapByParent() returned empty map")
	}

	// All pages in the sitemap must appear in exactly one group.
	totalInGroups := 0
	for _, pages := range grouped {
		totalInGroups += len(pages)
	}
	if totalInGroups != len(docs.Sitemap()) {
		t.Errorf("grouped total %d != sitemap total %d", totalInGroups, len(docs.Sitemap()))
	}
}

func TestSitemapByParent_GroupMatchesParentField(t *testing.T) {
	grouped := docs.SitemapByParent()
	for parent, pages := range grouped {
		for _, p := range pages {
			if p.Parent != parent {
				t.Errorf("page %q in group %q but has Parent=%q", p.Value, parent, p.Parent)
			}
		}
	}
}

func TestFindByURL_Found(t *testing.T) {
	// Use the first page from the sitemap so this test never needs updating.
	first := docs.Sitemap()[0]
	got := docs.FindByURL(first.URL)

	if got == nil {
		t.Fatalf("FindByURL(%q) returned nil, expected a page", first.URL)
	}
	if got.URL != first.URL {
		t.Errorf("expected URL %q, got %q", first.URL, got.URL)
	}
}

func TestFindByURL_NotFound(t *testing.T) {
	got := docs.FindByURL("/this-url-does-not-exist-xyz")
	if got != nil {
		t.Errorf("expected nil for unknown URL, got %+v", got)
	}
}

func TestFindByURL_KnownPages(t *testing.T) {
	knownURLs := []string{
		"/getting-started",
		"/user-guide",
		"/developer-guide",
		"/system",
		"/developer-guide/building-themes/template-language",
	}
	for _, url := range knownURLs {
		if docs.FindByURL(url) == nil {
			t.Errorf("FindByURL(%q) returned nil, expected a valid page", url)
		}
	}
}
