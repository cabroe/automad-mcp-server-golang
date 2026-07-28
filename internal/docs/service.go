package docs

import (
	"fmt"
	"strings"
	"time"
)

// DefaultCacheTTL is the time documentation pages stay cached in memory.
const DefaultCacheTTL = 1 * time.Hour

// Service is the main entry point for fetching Automad documentation.
// It combines the Fetcher, Parser, and Cache into a single convenient API.
type Service struct {
	fetcher *Fetcher
	cache   *Cache
}

// NewService creates a new Service with default settings.
func NewService() *Service {
	return &Service{
		fetcher: NewFetcher(),
		cache:   NewCache(DefaultCacheTTL),
	}
}

// GetPage fetches and parses a documentation page at the given relative URL.
// Results are cached for DefaultCacheTTL. Subsequent calls with the same URL
// return the cached version without any network request.
func (s *Service) GetPage(url string) (*Page, error) {
	// Normalize: ensure leading slash.
	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}

	// Check cache first.
	if cached := s.cache.Get(url); cached != nil {
		return cached, nil
	}

	// Fetch raw HTML.
	rawHTML, err := s.fetcher.Fetch(url)
	if err != nil {
		return nil, fmt.Errorf("fetching page %s: %w", url, err)
	}

	// Parse into a structured Page.
	page := Parse(rawHTML, url)
	s.cache.Set(url, page)

	return page, nil
}

// Search performs a case-insensitive keyword search across all known
// documentation pages. Only pages that have been fetched and cached are
// searched. For a comprehensive search, call WarmCache first.
func (s *Service) Search(query string) []*SearchResult {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}
	keywords := strings.Fields(query)

	var results []*SearchResult
	for _, doc := range Sitemap() {
		cached := s.cache.Get(doc.URL)
		if cached == nil {
			// For pages not yet in cache, match against title/URL only.
			score := matchScore(keywords, strings.ToLower(doc.Value), strings.ToLower(doc.URL), "")
			if score > 0 {
				results = append(results, &SearchResult{
					DocPage: doc,
					Score:   score,
					Snippet: fmt.Sprintf("Section: %s", doc.Parent),
				})
			}
			continue
		}

		contentLower := strings.ToLower(cached.Content)
		score := matchScore(keywords, strings.ToLower(cached.Title), strings.ToLower(doc.URL), contentLower)
		if score > 0 {
			results = append(results, &SearchResult{
				DocPage: doc,
				Score:   score,
				Snippet: extractSnippet(cached.Content, keywords[0]),
			})
		}
	}

	// Sort by score descending (simple insertion sort for small result sets).
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].Score > results[j-1].Score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}

	return results
}

// SearchResult represents a single search hit.
type SearchResult struct {
	DocPage DocPage
	Score   int
	Snippet string
}

// matchScore computes a simple relevance score for a set of keywords against
// the title, URL, and content of a page.
func matchScore(keywords []string, title, url, content string) int {
	score := 0
	for _, kw := range keywords {
		if strings.Contains(title, kw) {
			score += 10 // title match is most valuable
		}
		if strings.Contains(url, kw) {
			score += 5
		}
		if content != "" && strings.Contains(content, kw) {
			score += 1
		}
	}
	return score
}

// extractSnippet returns a ~200-character excerpt of content around the first
// occurrence of the given keyword.
func extractSnippet(content, keyword string) string {
	lower := strings.ToLower(content)
	keyword = strings.ToLower(keyword)

	idx := strings.Index(lower, keyword)
	if idx < 0 {
		// No match in content; return the beginning.
		if len(content) > 200 {
			return content[:200] + "…"
		}
		return content
	}

	start := idx - 80
	if start < 0 {
		start = 0
	}
	end := idx + len(keyword) + 120
	if end > len(content) {
		end = len(content)
	}

	snippet := content[start:end]
	if start > 0 {
		snippet = "…" + snippet
	}
	if end < len(content) {
		snippet += "…"
	}
	return strings.TrimSpace(snippet)
}

// ListPages returns all pages in the sitemap, optionally filtered by parent section.
func (s *Service) ListPages(parent string) []DocPage {
	if parent == "" {
		return Sitemap()
	}
	parentLower := strings.ToLower(parent)
	var filtered []DocPage
	for _, p := range Sitemap() {
		if strings.ToLower(p.Parent) == parentLower {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// CacheStats returns a human-readable summary of the current cache state.
func (s *Service) CacheStats() string {
	return s.cache.Stats()
}
