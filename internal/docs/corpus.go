package docs

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"io"
	"sync"
)

// corpus.go provides an offline snapshot of the Automad documentation. The
// embedded, gzip-compressed JSON corpus maps each documentation URL to its
// parsed title and content, captured from automad.org. It is used only as a
// fallback when the live site is unreachable (offline / air-gapped), so the
// docs tools keep working — get_page serves the snapshot and search_docs ranks
// on real content instead of titles alone.
//
// Regenerate it with `go run ./cmd/gen-corpus` (or `make corpus`) whenever the
// upstream documentation changes. Snapshot content may lag behind the live
// site; pages served from it are marked Offline so callers can flag them.

//go:embed corpus.json.gz
var corpusGz []byte

// corpusEntry is one page's snapshot: its title and extracted content.
type corpusEntry struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

var (
	corpusOnce sync.Once
	corpusData map[string]corpusEntry
)

// loadCorpus decompresses and decodes the embedded corpus exactly once. A
// malformed or empty corpus yields an empty map rather than an error, so the
// server always starts (offline docs simply stay unavailable).
func loadCorpus() map[string]corpusEntry {
	corpusOnce.Do(func() {
		corpusData = map[string]corpusEntry{}
		if len(corpusGz) == 0 {
			return
		}
		zr, err := gzip.NewReader(bytes.NewReader(corpusGz))
		if err != nil {
			return
		}
		defer zr.Close()
		raw, err := io.ReadAll(zr)
		if err != nil {
			return
		}
		var decoded map[string]corpusEntry
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return
		}
		corpusData = decoded
	})
	return corpusData
}

// corpusPage returns an offline Page snapshot for url, or nil when the corpus
// has no entry for it.
func corpusPage(url string) *Page {
	entry, ok := loadCorpus()[url]
	if !ok {
		return nil
	}
	return &Page{
		Title:   entry.Title,
		URL:     url,
		FullURL: BaseURL + url,
		Content: entry.Content,
		Offline: true,
	}
}

// corpusContent returns the snapshot content for url, or "" when absent. Used
// by Search so ranking works offline without a warmed cache.
func corpusContent(url string) string {
	return loadCorpus()[url].Content
}

// CorpusSize reports how many pages the embedded offline corpus contains.
func CorpusSize() int {
	return len(loadCorpus())
}
