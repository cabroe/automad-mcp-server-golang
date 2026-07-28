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
func (c *Cache) Get(url string) *Page {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.pages[url]
	if !ok {
		return nil
	}
	if time.Now().After(entry.expiresAt) {
		return nil
	}
	return entry.content
}

// Set stores a page in the cache.
func (c *Cache) Set(url string, page *Page) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pages[url] = &cachedPage{
		content:   page,
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
