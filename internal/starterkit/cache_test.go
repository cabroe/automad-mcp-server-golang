package starterkit_test

import (
	"strings"
	"testing"
	"time"

	"github.com/cabroe/automad-mcp-server/internal/starterkit"
)

func TestCache_TreeMissByDefault(t *testing.T) {
	c := starterkit.NewCache(time.Hour)
	if got := c.GetTree(); got != nil {
		t.Errorf("expected nil tree from empty cache, got %+v", got)
	}
}

func TestCache_SetAndGetTree(t *testing.T) {
	c := starterkit.NewCache(time.Hour)
	tree := &starterkit.Tree{Entries: []starterkit.TreeEntry{{Path: "theme.json", Type: "blob"}}}

	c.SetTree(tree)

	got := c.GetTree()
	if got == nil {
		t.Fatal("expected cached tree, got nil")
	}
	if len(got.Entries) != 1 || got.Entries[0].Path != "theme.json" {
		t.Errorf("unexpected cached tree contents: %+v", got.Entries)
	}
}

func TestCache_TreeExpires(t *testing.T) {
	c := starterkit.NewCache(1 * time.Millisecond)
	c.SetTree(&starterkit.Tree{Entries: []starterkit.TreeEntry{{Path: "x", Type: "blob"}}})

	time.Sleep(5 * time.Millisecond)

	if got := c.GetTree(); got != nil {
		t.Errorf("expected expired tree to be nil, got %+v", got)
	}
}

func TestCache_SetAndGetFile(t *testing.T) {
	c := starterkit.NewCache(time.Hour)
	c.SetFile("theme.json", []byte(`{"name":"Starter Kit"}`))

	content, ok := c.GetFile("theme.json")
	if !ok {
		t.Fatal("expected cache hit for theme.json")
	}
	if !strings.Contains(string(content), "Starter Kit") {
		t.Errorf("unexpected cached content: %s", content)
	}
}

func TestCache_FileMiss(t *testing.T) {
	c := starterkit.NewCache(time.Hour)
	if _, ok := c.GetFile("does-not-exist.php"); ok {
		t.Error("expected cache miss for unseeded path")
	}
}

func TestCache_FileExpires(t *testing.T) {
	c := starterkit.NewCache(1 * time.Millisecond)
	c.SetFile("theme.json", []byte("{}"))

	time.Sleep(5 * time.Millisecond)

	if _, ok := c.GetFile("theme.json"); ok {
		t.Error("expected expired file entry to be a cache miss")
	}
}

func TestCache_Stats(t *testing.T) {
	c := starterkit.NewCache(time.Hour)
	c.SetTree(&starterkit.Tree{Entries: []starterkit.TreeEntry{{Path: "x", Type: "blob"}}})
	c.SetFile("a.php", []byte("<?php"))
	c.SetFile("b.php", []byte("<?php"))

	stats := c.Stats()
	if !strings.Contains(stats, "tree_cached=true") {
		t.Errorf("expected tree_cached=true in stats, got: %s", stats)
	}
	if !strings.Contains(stats, "files_total=2") {
		t.Errorf("expected files_total=2 in stats, got: %s", stats)
	}
}
