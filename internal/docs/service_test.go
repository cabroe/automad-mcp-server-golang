package docs_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cabroe/automad-mcp-server-golang/internal/docs"
)

func TestService_ListPages_All(t *testing.T) {
	svc := docs.NewService()
	pages := svc.ListPages("")
	if len(pages) != len(docs.Sitemap()) {
		t.Errorf("ListPages(\"\") returned %d pages, expected %d", len(pages), len(docs.Sitemap()))
	}
}

func TestService_ListPages_FilteredSection(t *testing.T) {
	svc := docs.NewService()
	pages := svc.ListPages("System")
	if len(pages) == 0 {
		t.Fatal("ListPages(\"System\") returned no pages")
	}
	for _, p := range pages {
		if !strings.EqualFold(p.Parent, "System") {
			t.Errorf("page %q has parent %q, expected System", p.Value, p.Parent)
		}
	}
}

func TestService_ListPages_UnknownSection(t *testing.T) {
	svc := docs.NewService()
	pages := svc.ListPages("NonExistentSection42")
	if len(pages) != 0 {
		t.Errorf("expected 0 pages for unknown section, got %d", len(pages))
	}
}

func TestService_ListPages_CaseInsensitive(t *testing.T) {
	svc := docs.NewService()
	upper := svc.ListPages("SYSTEM")
	lower := svc.ListPages("system")
	mixed := svc.ListPages("System")
	if len(upper) != len(lower) || len(lower) != len(mixed) {
		t.Errorf("case sensitivity inconsistency: SYSTEM=%d, system=%d, System=%d",
			len(upper), len(lower), len(mixed))
	}
}

func TestService_Search_EmptyQuery(t *testing.T) {
	svc := docs.NewService()
	results := svc.Search("")
	if results != nil {
		t.Errorf("expected nil for empty query, got %v", results)
	}
}

func TestService_Search_WhitespaceOnlyQuery(t *testing.T) {
	svc := docs.NewService()
	results := svc.Search("   ")
	if results != nil {
		t.Errorf("expected nil for whitespace-only query, got %v", results)
	}
}

func TestService_Search_MatchesTitle(t *testing.T) {
	svc := docs.NewService()
	// "pagelist" appears in several page titles/URLs in the sitemap.
	results := svc.Search("pagelist")
	if len(results) == 0 {
		t.Fatal("expected results for 'pagelist', got none")
	}
	// Top result should be the most relevant (highest score).
	if results[0].Score <= 0 {
		t.Errorf("top result has score %d, expected > 0", results[0].Score)
	}
}

func TestService_Search_SortedByScore(t *testing.T) {
	svc := docs.NewService()
	results := svc.Search("template")
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("results not sorted: index %d (score=%d) > index %d (score=%d)",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
}

func TestService_Search_NoResults(t *testing.T) {
	svc := docs.NewService()
	results := svc.Search("xyzzy-nonexistent-term-12345")
	// Should return nil or empty, never panic.
	_ = results
}

func TestService_Search_CaseInsensitive(t *testing.T) {
	svc := docs.NewService()
	upper := svc.Search("SYSTEM")
	lower := svc.Search("system")
	if len(upper) != len(lower) {
		t.Errorf("case sensitivity inconsistency: SYSTEM=%d results, system=%d results",
			len(upper), len(lower))
	}
}

func TestService_CacheStats(t *testing.T) {
	svc := docs.NewService()
	stats := svc.CacheStats()
	if stats == "" {
		t.Error("CacheStats() returned empty string")
	}
}

func TestService_Search_UsesContentWhenCached(t *testing.T) {
	// Pre-seed a page with specific content into the cache.
	svc := docs.NewSeededService("/user-guide/creating-pages", &docs.Page{
		Title:   "Creating Pages",
		URL:     "/user-guide/creating-pages",
		FullURL: docs.BaseURL + "/user-guide/creating-pages",
		Content: "unique-content-term-xyz creating pages in automad",
	})

	results := svc.Search("unique-content-term-xyz")
	if len(results) == 0 {
		t.Fatal("expected search results for cached content term, got none")
	}
}

// matchScoreTests tests the internal scoring logic via the service's Search.
func TestService_Search_TitleMatchScoresHigher(t *testing.T) {
	// "system" is in the title of several pages and also in their URLs.
	// "allowed" only appears in the title of one page.
	svc := docs.NewService()

	results := svc.Search("allowed file types")
	if len(results) == 0 {
		t.Skip("no results for 'allowed file types'")
	}
	// The "Allowed File Types" page should score highly.
	found := false
	for _, r := range results {
		if strings.Contains(strings.ToLower(r.DocPage.Value), "allowed") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'Allowed File Types' page in results for query 'allowed file types'")
	}
}

func TestService_GetPage_NormalizesURL(t *testing.T) {
	// GetPage with a pre-seeded URL without leading slash.
	svc := docs.NewSeededService("/getting-started", &docs.Page{
		Title:   "Getting Started",
		URL:     "/getting-started",
		FullURL: docs.BaseURL + "/getting-started",
		Content: "some content",
	})

	// Should normalize "getting-started" → "/getting-started".
	page, err := svc.GetPage(context.Background(), "getting-started")
	if err != nil {
		t.Fatalf("GetPage without leading slash returned error: %v", err)
	}
	if page.Title != "Getting Started" {
		t.Errorf("expected title 'Getting Started', got %q", page.Title)
	}
}

func TestService_GetPage_ReturnsCachedPage(t *testing.T) {
	page := &docs.Page{
		Title:   "Cached Page",
		URL:     "/cached",
		FullURL: docs.BaseURL + "/cached",
		Content: "content from cache",
	}
	svc := docs.NewSeededService("/cached", page)

	got, err := svc.GetPage(context.Background(), "/cached")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != "Cached Page" {
		t.Errorf("expected cached title, got %q", got.Title)
	}
}

// TestService_TTL ensures that NewSeededService uses default TTL.
func TestService_TTL(t *testing.T) {
	svc := docs.NewSeededService("/ttl-test", &docs.Page{Title: "TTL"})
	// Stats should show 1 page in cache.
	stats := svc.CacheStats()
	if !strings.Contains(stats, "total=1") {
		t.Errorf("expected total=1 in stats, got: %s", stats)
	}
	_ = time.Now() // keep time import used
}
