package docs

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// DefaultCacheTTL is the time documentation pages stay cached in memory.
const DefaultCacheTTL = 1 * time.Hour

// Service is the main entry point for fetching Automad documentation.
// It combines the Fetcher, Parser, and Cache into a single convenient API.
type Service struct {
	lifecycle context.Context
	fetcher   *Fetcher
	cache     *Cache
	group     singleflight.Group
}

// NewService creates a new Service with default settings.
func NewService() *Service {
	return NewServiceWithContext(context.Background())
}

// NewServiceWithContext creates a service whose shared fetches stop when the
// server lifecycle context is canceled.
func NewServiceWithContext(ctx context.Context) *Service {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Service{
		lifecycle: ctx,
		fetcher:   NewFetcher(),
		cache:     NewCache(DefaultCacheTTL),
	}
}

// NormalizeURL canonicalizes a documentation path for sitemap lookup,
// fetching, and cache keys. Query strings and fragments are discarded.
func NormalizeURL(value string) string {
	value = strings.TrimSpace(value)
	if strings.ContainsAny(value, "\\\x00\r\n") {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return ""
	}
	if parsed.Path == "" && (parsed.RawQuery != "" || parsed.Fragment != "") {
		return ""
	}
	value = parsed.Path
	if value == "" {
		return "/"
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	clean := path.Clean(value)
	if clean == "." {
		return "/"
	}
	return clean
}

// GetPage fetches and parses a documentation page at the given relative URL.
// Results are cached for DefaultCacheTTL. Subsequent calls with the same URL
// return the cached version without any network request. The provided
// context is forwarded to the underlying HTTP request, so cancelling it
// (e.g. because the calling MCP client disconnected) aborts the fetch
// instead of blocking until the fixed fetch timeout elapses.
func (s *Service) GetPage(ctx context.Context, url string) (*Page, error) {
	if err := s.lifecycle.Err(); err != nil {
		return nil, err
	}
	url = NormalizeURL(url)
	if url == "" {
		return nil, fmt.Errorf("documentation URL must be a relative path")
	}

	// Check cache first.
	if cached := s.cache.Get(url); cached != nil {
		return cached, nil
	}

	// Deduplicate concurrent cache misses while allowing each waiter to cancel
	// independently. The shared fetch is bounded by the HTTP client timeout.
	resultCh := s.group.DoChan(url, func() (any, error) {
		if cached := s.cache.Get(url); cached != nil {
			return cached, nil
		}

		rawHTML, err := s.fetcher.Fetch(s.lifecycle, url)
		if err != nil {
			return nil, fmt.Errorf("fetching page %s: %w", url, err)
		}

		page := Parse(rawHTML, url)
		s.cache.Set(url, page)
		return page, nil
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultCh:
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if result.Err != nil {
			return nil, result.Err
		}
		return result.Val.(*Page), nil
	}
}

// warmCacheConcurrency bounds how many documentation pages WarmCache fetches
// in parallel, so a full warm-up does not fire ~100 simultaneous requests at
// automad.org.
const warmCacheConcurrency = 5

// WarmCache proactively fetches and caches every page in the sitemap that
// isn't already cached, using a small worker pool. It is best-effort: a
// failure to fetch one page does not stop the others. It returns the number
// of pages newly cached and a combined error describing any failures (nil if
// all succeeded or everything was already cached).
//
// Search only ranks by full content for pages that are already in the
// cache (uncached pages are matched on title/URL alone), so calling
// WarmCache — e.g. once in the background on server startup — makes
// search_docs results comprehensive instead of depending on which pages a
// client happens to have fetched via get_page so far.
func (s *Service) WarmCache(ctx context.Context) (int, error) {
	pages := Sitemap()

	var (
		wg       sync.WaitGroup
		sem      = make(chan struct{}, warmCacheConcurrency)
		mu       sync.Mutex
		warmed   int
		failures []string
	)

	for _, doc := range pages {
		if s.cache.Get(doc.URL) != nil {
			continue // already cached, nothing to do
		}

		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			if ctx.Err() != nil {
				return
			}

			if _, err := s.GetPage(ctx, url); err != nil {
				mu.Lock()
				failures = append(failures, fmt.Sprintf("%s: %v", url, err))
				mu.Unlock()
				return
			}

			mu.Lock()
			warmed++
			mu.Unlock()
		}(doc.URL)
	}

	wg.Wait()

	if len(failures) > 0 {
		return warmed, fmt.Errorf("failed to warm %d page(s): %s", len(failures), strings.Join(failures, "; "))
	}
	return warmed, nil
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

	// Sort by score descending, preserving relative order for ties.
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

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

func runesEqual(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// extractSnippet returns a ~200-character excerpt of content around the first
// occurrence of the given keyword.
func extractSnippet(content, keyword string) string {
	runes := []rune(content)
	lowerRunes := []rune(strings.ToLower(content))
	keywordRunes := []rune(strings.ToLower(keyword))
	idx := -1
	for i := 0; i+len(keywordRunes) <= len(lowerRunes); i++ {
		if runesEqual(lowerRunes[i:i+len(keywordRunes)], keywordRunes) {
			idx = i
			break
		}
	}
	if idx < 0 {
		if len(runes) > 200 {
			return string(runes[:200]) + "…"
		}
		return content
	}

	start := idx - 80
	if start < 0 {
		start = 0
	}
	end := idx + len(keywordRunes) + 120
	if end > len(runes) {
		end = len(runes)
	}

	snippet := string(runes[start:end])
	if start > 0 {
		snippet = "…" + snippet
	}
	if end < len(runes) {
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
