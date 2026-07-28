// Package docs provides types and logic to fetch, parse, and cache
// pages from the Automad documentation website (https://automad.org).
package docs

import (
	"fmt"
	"sync"
	"time"
)

// cachedPage holds a fetched and parsed page with its expiry time.
type cachedPage struct {
	content   *Page
	expiresAt time.Time
}

// Cache is a thread-safe in-memory cache for fetched documentation pages.
type Cache struct {
	mu    sync.RWMutex
	pages map[string]*cachedPage
	ttl   time.Duration
}

// NewCache creates a new Cache with the given TTL for cached entries.
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		pages: make(map[string]*cachedPage),
		ttl:   ttl,
	}
}

// Get retrieves a cached page by URL. Returns nil if not found or expired.
// Expired entries are removed lazily. Returned pages are copies so callers
// cannot mutate the cached value.
func (c *Cache) Get(url string) *Page {
	c.mu.RLock()
	entry, ok := c.pages[url]
	if !ok {
		c.mu.RUnlock()
		return nil
	}
	expired := time.Now().After(entry.expiresAt)
	if !expired {
		page := *entry.content
		c.mu.RUnlock()
		return &page
	}
	c.mu.RUnlock()

	c.mu.Lock()
	if current, exists := c.pages[url]; exists && time.Now().After(current.expiresAt) {
		delete(c.pages, url)
	}
	c.mu.Unlock()
	return nil
}

// Set stores a page in the cache.
func (c *Cache) Set(url string, page *Page) {
	c.mu.Lock()
	defer c.mu.Unlock()

	copy := *page
	c.pages[url] = &cachedPage{
		content:   &copy,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Invalidate removes a page from the cache.
func (c *Cache) Invalidate(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.pages, url)
}

// Stats returns cache statistics.
func (c *Cache) Stats() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := len(c.pages)
	expired := 0
	now := time.Now()
	for _, entry := range c.pages {
		if now.After(entry.expiresAt) {
			expired++
		}
	}
	return fmt.Sprintf("total=%d, expired=%d, fresh=%d", total, expired, total-expired)
}
