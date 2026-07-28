package starterkit_test

import (
	"context"
	"strings"
	"testing"

	"github.com/cabroe/automad-mcp-server/internal/starterkit"
)

func seededTree() *starterkit.Tree {
	return &starterkit.Tree{
		Entries: []starterkit.TreeEntry{
			{Path: "theme.json", Type: "blob", Size: 100},
			{Path: "default.php", Type: "blob", Size: 50},
			{Path: "components", Type: "tree"},
			{Path: "components/page.php", Type: "blob", Size: 500},
			{Path: "components/content.php", Type: "blob", Size: 400},
			{Path: "icons", Type: "tree"},
			{Path: "icons/star.svg", Type: "blob", Size: 200}, // unsupported extension
		},
	}
}

func TestService_ListFiles_ReturnsSeededTree(t *testing.T) {
	svc := starterkit.NewSeededService(seededTree(), nil)

	tree, usedFallback, err := svc.ListFiles(context.Background())
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if usedFallback {
		t.Error("expected usedFallback=false for a seeded (live) tree")
	}
	if len(tree.Entries) != 7 {
		t.Errorf("expected 7 entries, got %d", len(tree.Entries))
	}
}

func TestService_GetFileContent_CacheHit(t *testing.T) {
	svc := starterkit.NewSeededService(nil, map[string][]byte{
		"theme.json": []byte(`{"name":"Starter Kit"}`),
	})

	content, usedFallback, err := svc.GetFileContent(context.Background(), "theme.json")
	if err != nil {
		t.Fatalf("GetFileContent: %v", err)
	}
	if usedFallback {
		t.Error("expected usedFallback=false for a cache hit")
	}
	if !strings.Contains(string(content), "Starter Kit") {
		t.Errorf("unexpected content: %s", content)
	}
}

func TestService_GetFileContent_NormalizesLeadingSlash(t *testing.T) {
	svc := starterkit.NewSeededService(nil, map[string][]byte{
		"theme.json": []byte(`{}`),
	})

	if _, _, err := svc.GetFileContent(context.Background(), "/theme.json"); err != nil {
		t.Fatalf("expected leading slash to be stripped, got error: %v", err)
	}
}

func TestService_GetFileContent_EmptyPath(t *testing.T) {
	svc := starterkit.NewSeededService(nil, nil)
	if _, _, err := svc.GetFileContent(context.Background(), "   "); err == nil {
		t.Error("expected an error for an empty/whitespace path")
	}
}

func TestService_GetFileContent_UnsupportedExtension(t *testing.T) {
	svc := starterkit.NewSeededService(nil, nil)
	if _, _, err := svc.GetFileContent(context.Background(), "icons/star.svg"); err == nil {
		t.Error("expected an error for an unsupported file extension (.svg)")
	}
}

func TestService_FileURLs(t *testing.T) {
	svc := starterkit.NewSeededService(nil, nil)
	raw, blob, err := svc.FileURLs("theme.json")
	if err != nil {
		t.Fatalf("FileURLs: %v", err)
	}

	if !strings.Contains(raw, "raw.githubusercontent.com/"+starterkit.Owner+"/"+starterkit.Repo) {
		t.Errorf("unexpected raw URL: %s", raw)
	}
	if !strings.Contains(blob, "github.com/"+starterkit.Owner+"/"+starterkit.Repo+"/blob/") {
		t.Errorf("unexpected blob URL: %s", blob)
	}
	if !strings.HasSuffix(raw, "/theme.json") || !strings.HasSuffix(blob, "/theme.json") {
		t.Errorf("expected both URLs to end with /theme.json, got raw=%s blob=%s", raw, blob)
	}
}

func TestService_RejectsUnsafeRepositoryPaths(t *testing.T) {
	svc := starterkit.NewSeededService(nil, nil)
	for _, path := range []string{"../theme.json", "components/../../theme.json", `components\\page.php`, "https://example.com/a.php", "theme.json?ref=x"} {
		if _, _, err := svc.GetFileContent(context.Background(), path); err == nil {
			t.Errorf("GetFileContent(%q) accepted unsafe path", path)
		}
		if _, _, err := svc.FileURLs(path); err == nil {
			t.Errorf("FileURLs(%q) accepted unsafe path", path)
		}
	}
}

func TestService_SearchCode_FindsMatchesInCachedFiles(t *testing.T) {
	svc := starterkit.NewSeededService(seededTree(), map[string][]byte{
		"components/page.php":    []byte("<!DOCTYPE html>\n<html>\n<@ layout.php @>\n"),
		"components/content.php": []byte("<h1>@{ title }</h1>\n"),
	})

	matches, uncached, err := svc.SearchCode(context.Background(), "<@", nil)
	if err != nil {
		t.Fatalf("SearchCode: %v", err)
	}
	if len(matches) != 1 || matches[0].Path != "components/page.php" {
		t.Fatalf("expected 1 match in components/page.php, got %+v", matches)
	}

	// theme.json and default.php are in the tree but not seeded into the
	// file cache, so they should be reported as uncached rather than silently
	// fetched over the network.
	if len(uncached) == 0 {
		t.Error("expected some uncached files to be reported")
	}
}

func TestService_SearchCode_RespectsExtensionFilter(t *testing.T) {
	svc := starterkit.NewSeededService(seededTree(), map[string][]byte{
		"components/page.php": []byte("@{ title }"),
	})

	matches, _, err := svc.SearchCode(context.Background(), "title", []string{".json"})
	if err != nil {
		t.Fatalf("SearchCode: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("expected no matches when filtering to .json only, got %+v", matches)
	}
}

func TestService_SearchCode_EmptyQuery(t *testing.T) {
	svc := starterkit.NewSeededService(seededTree(), nil)
	if _, _, err := svc.SearchCode(context.Background(), "  ", nil); err == nil {
		t.Error("expected an error for an empty query")
	}
}

func TestKnownSnippets_AllResolvable(t *testing.T) {
	for _, m := range starterkit.KnownSnippets() {
		found, ok := starterkit.FindSnippet(m.Key)
		if !ok {
			t.Errorf("FindSnippet(%q) not found even though it's in KnownSnippets()", m.Key)
			continue
		}
		if found.Path != m.Path {
			t.Errorf("FindSnippet(%q).Path = %q, want %q", m.Key, found.Path, m.Path)
		}
	}
}

func TestFindSnippet_Unknown(t *testing.T) {
	if _, ok := starterkit.FindSnippet("does-not-exist"); ok {
		t.Error("expected ok=false for an unknown snippet key")
	}
}

func TestService_CacheStats(t *testing.T) {
	svc := starterkit.NewSeededService(seededTree(), map[string][]byte{
		"theme.json": []byte("{}"),
	})
	stats := svc.CacheStats()
	if !strings.Contains(stats, "tree_cached=true") {
		t.Errorf("expected tree_cached=true, got: %s", stats)
	}
	if !strings.Contains(stats, "files_total=1") {
		t.Errorf("expected files_total=1, got: %s", stats)
	}
}
