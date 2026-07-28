package starterkit_test

import (
	"strings"
	"testing"

	"github.com/cabroe/automad-mcp-server-golang/internal/starterkit"
)

func TestRenderTree_NestsChildrenUnderDirectories(t *testing.T) {
	entries := []starterkit.TreeEntry{
		{Path: "theme.json", Type: "blob"},
		{Path: "components", Type: "tree"},
		{Path: "components/page.php", Type: "blob"},
		{Path: "components/content.php", Type: "blob"},
	}

	out := starterkit.RenderTree(entries)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d:\n%s", len(lines), out)
	}

	// "components/" should appear before its (indented) children.
	dirIdx := indexOfTrimmed(lines, "components/")
	if dirIdx == -1 {
		t.Fatalf("expected a 'components/' line, got:\n%s", out)
	}
	for _, child := range []string{"content.php", "page.php"} {
		idx := indexOfTrimmed(lines, child)
		if idx == -1 {
			t.Errorf("expected a line for %q, got:\n%s", child, out)
			continue
		}
		if idx <= dirIdx {
			t.Errorf("expected %q to be listed after 'components/', got:\n%s", child, out)
		}
		if !strings.HasPrefix(lines[idx], "  ") {
			t.Errorf("expected %q to be indented as a child of components/, got %q", child, lines[idx])
		}
	}
}

func TestRenderTree_Empty(t *testing.T) {
	if out := starterkit.RenderTree(nil); out != "" {
		t.Errorf("expected empty output for no entries, got %q", out)
	}
}

func indexOfTrimmed(lines []string, want string) int {
	for i, l := range lines {
		if strings.TrimSpace(l) == want {
			return i
		}
	}
	return -1
}
