package starterkit

import (
	"fmt"
	"sync"
	"time"
)

// cachedTree holds a fetched Tree with its expiry time.
type cachedTree struct {
	tree      *Tree
	expiresAt time.Time
}

// cachedFile holds fetched file content with its expiry time.
type cachedFile struct {
	content   []byte
	expiresAt time.Time
}

// Cache is a thread-safe in-memory cache for the repository tree and
// individual file contents, mirroring the pattern used by the docs package's
// Cache for the Automad documentation site.
type Cache struct {
	mu    sync.RWMutex
	ttl   time.Duration
	tree  *cachedTree
	files map[string]*cachedFile
}

// NewCache creates a Cache with the given TTL for both the tree and file entries.
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		ttl:   ttl,
		files: make(map[string]*cachedFile),
	}
}

// GetTree returns the cached tree, or nil if it isn't cached or has expired.
func (c *Cache) GetTree() *Tree {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.tree == nil || time.Now().After(c.tree.expiresAt) {
		return nil
	}
	return c.tree.tree
}

// SetTree stores the repository tree in the cache.
func (c *Cache) SetTree(t *Tree) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tree = &cachedTree{tree: t, expiresAt: time.Now().Add(c.ttl)}
}

// GetFile returns the cached content for path and true, or (nil, false) if
// it isn't cached or has expired.
func (c *Cache) GetFile(path string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.files[path]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.content, true
}

// SetFile stores a file's content in the cache.
func (c *Cache) SetFile(path string, content []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.files[path] = &cachedFile{content: content, expiresAt: time.Now().Add(c.ttl)}
}

// Stats returns a human-readable summary of the current cache state.
func (c *Cache) Stats() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	treeCached := c.tree != nil && time.Now().Before(c.tree.expiresAt)

	fresh := 0
	now := time.Now()
	for _, f := range c.files {
		if now.Before(f.expiresAt) {
			fresh++
		}
	}
	return fmt.Sprintf("tree_cached=%v, files_total=%d, files_fresh=%d", treeCached, len(c.files), fresh)
}
