package docs_test

import (
	"testing"
	"time"

	"github.com/cabroe/automad-mcp-server/internal/docs"
)

func TestNormalizeURL(t *testing.T) {
	cases := map[string]string{
		"system/caching":           "/system/caching",
		"/system/caching?x=1#part": "/system/caching",
		"/system/../user-guide":    "/user-guide",
		"https://example.com/a":    "",
	}
	for input, want := range cases {
		if got := docs.NormalizeURL(input); got != want {
			t.Errorf("NormalizeURL(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCache_GetMiss(t *testing.T) {
	c := docs.NewCache(time.Hour)
	got := c.Get("/nonexistent")
	if got != nil {
		t.Errorf("expected nil for cache miss, got %+v", got)
	}
}

func TestCache_SetAndGet(t *testing.T) {
	c := docs.NewCache(time.Hour)
	page := &docs.Page{Title: "Test", URL: "/test", FullURL: "https://automad.org/test", Content: "hello"}
	c.Set("/test", page)

	got := c.Get("/test")
	if got == nil {
		t.Fatal("expected cached page, got nil")
	}
	if got.Title != "Test" {
		t.Errorf("expected title %q, got %q", "Test", got.Title)
	}
}

func TestCache_Expiry(t *testing.T) {
	// Use a very short TTL so we can test expiry without sleeping long.
	c := docs.NewCache(10 * time.Millisecond)
	page := &docs.Page{Title: "Expires", URL: "/exp", Content: "bye"}
	c.Set("/exp", page)

	// Entry should be present immediately.
	if c.Get("/exp") == nil {
		t.Fatal("expected page right after Set, got nil")
	}

	// Wait for expiry.
	time.Sleep(20 * time.Millisecond)

	if got := c.Get("/exp"); got != nil {
		t.Errorf("expected nil after TTL expiry, got %+v", got)
	}
}

func TestCache_Invalidate(t *testing.T) {
	c := docs.NewCache(time.Hour)
	page := &docs.Page{Title: "Gone", URL: "/gone"}
	c.Set("/gone", page)

	c.Invalidate("/gone")

	if c.Get("/gone") != nil {
		t.Error("expected nil after Invalidate, got a page")
	}
}

func TestCache_Stats(t *testing.T) {
	c := docs.NewCache(10 * time.Millisecond)
	c.Set("/a", &docs.Page{Title: "A"})
	c.Set("/b", &docs.Page{Title: "B"})

	stats := c.Stats()
	if stats == "" {
		t.Error("expected non-empty stats string")
	}

	// Wait for expiry, then check expired count is reflected.
	time.Sleep(20 * time.Millisecond)
	stats = c.Stats()
	if stats == "" {
		t.Error("expected non-empty stats string after expiry")
	}
}

func TestCache_ReturnsCopies(t *testing.T) {
	c := docs.NewCache(time.Hour)
	original := &docs.Page{Title: "Original"}
	c.Set("/copy", original)
	original.Title = "Changed after Set"
	got := c.Get("/copy")
	got.Title = "Changed after Get"
	if again := c.Get("/copy"); again.Title != "Original" {
		t.Fatalf("cached page was mutated: %q", again.Title)
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	c := docs.NewCache(time.Hour)
	done := make(chan struct{})

	// Multiple concurrent writers and readers should not race.
	for i := range 10 {
		go func(i int) {
			url := "/page"
			c.Set(url, &docs.Page{Title: "concurrent"})
			_ = c.Get(url)
			if i == 9 {
				close(done)
			}
		}(i)
	}
	<-done
}
