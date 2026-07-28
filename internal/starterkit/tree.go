package starterkit

import (
	"fmt"
	"sort"
	"strings"
)

// RenderTree renders a flat list of tree entries as an indented tree view,
// e.g.:
//
//	components/
//	  content.php
//	  page.php
//	theme.json
//
// Sorting entries lexicographically by their full path is sufficient to
// produce a correct nested view: since a directory's own path is always a
// prefix of everything inside it, string sort naturally groups a directory
// together with its (indented) children before moving on to the next sibling.
func RenderTree(entries []TreeEntry) string {
	sorted := make([]TreeEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })

	var sb strings.Builder
	for _, e := range sorted {
		depth := strings.Count(e.Path, "/")
		name := e.Path
		if idx := strings.LastIndex(name, "/"); idx != -1 {
			name = name[idx+1:]
		}
		indent := strings.Repeat("  ", depth)
		if e.Type == "tree" {
			fmt.Fprintf(&sb, "%s%s/\n", indent, name)
		} else {
			fmt.Fprintf(&sb, "%s%s\n", indent, name)
		}
	}
	return sb.String()
}
