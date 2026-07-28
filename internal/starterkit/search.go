package starterkit

import (
	"path/filepath"
	"strings"
)

// SearchMatch is a single line match found by Service.SearchCode.
type SearchMatch struct {
	// Path is the file the match was found in.
	Path string
	// Line is the 1-based line number of the match.
	Line int
	// Excerpt is the (trimmed) matching line.
	Excerpt string
}

// lineMatch is an intermediate result from searchLines, before the file
// path is known to the caller.
type lineMatch struct {
	Line int
	Text string
}

// matchesExtFilter reports whether path's extension is in filter. An empty
// filter matches everything.
func matchesExtFilter(path string, filter []string) bool {
	if len(filter) == 0 {
		return true
	}
	ext := strings.ToLower(filepath.Ext(path))
	for _, f := range filter {
		f = strings.ToLower(strings.TrimSpace(f))
		if !strings.HasPrefix(f, ".") {
			f = "." + f
		}
		if f == ext {
			return true
		}
	}
	return false
}

// searchLines returns every line in content that contains query
// (case-insensitive substring match), with 1-based line numbers.
//
// A literal substring search — rather than treating query as a regular
// expression — is deliberate: Automad's template syntax uses characters
// like "@{", "}", and "<@" that are regex metacharacters (or would need
// escaping), so a regex search would make the two example queries from the
// tool's own description ("@{ }" and "<@ @>") awkward or surprising to use.
func searchLines(content []byte, query string) []lineMatch {
	lowerQuery := strings.ToLower(query)
	var matches []lineMatch
	for i, line := range strings.Split(string(content), "\n") {
		if strings.Contains(strings.ToLower(line), lowerQuery) {
			matches = append(matches, lineMatch{Line: i + 1, Text: strings.TrimSpace(line)})
		}
	}
	return matches
}
