// Command gen-corpus regenerates the embedded offline documentation corpus at
// internal/docs/corpus.json.gz. It fetches every page in the Automad sitemap
// from automad.org, extracts its content with the same parser the server uses,
// and writes a gzip-compressed JSON map of url -> {title, content}.
//
// Run it whenever the upstream documentation changes:
//
//	go run ./cmd/gen-corpus            # writes internal/docs/corpus.json.gz
//	go run ./cmd/gen-corpus -o path.gz # custom output path
//
// Requires network access to automad.org.
package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/cabroe/automad-mcp-server-golang/internal/docs"
)

const concurrency = 5

type entry struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func main() {
	out := flag.String("o", "internal/docs/corpus.json.gz", "output path for the gzip-compressed corpus")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fetcher := docs.NewFetcher()
	pages := docs.Sitemap()

	var (
		wg      sync.WaitGroup
		sem     = make(chan struct{}, concurrency)
		mu      sync.Mutex
		corpus  = make(map[string]entry, len(pages))
		failed  []string
		fetched int
	)

	for _, doc := range pages {
		url := docs.NormalizeURL(doc.URL)
		if url == "" {
			continue
		}
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			html, err := fetcher.Fetch(ctx, url)
			if err != nil {
				mu.Lock()
				failed = append(failed, fmt.Sprintf("%s: %v", url, err))
				mu.Unlock()
				return
			}
			page := docs.Parse(html, url)
			mu.Lock()
			corpus[url] = entry{Title: page.Title, Content: page.Content}
			fetched++
			mu.Unlock()
		}(url)
	}
	wg.Wait()

	if len(corpus) == 0 {
		fmt.Fprintln(os.Stderr, "error: fetched no pages; refusing to write an empty corpus")
		os.Exit(1)
	}

	if err := writeCorpus(*out, corpus); err != nil {
		fmt.Fprintf(os.Stderr, "error writing corpus: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("wrote %s: %d pages\n", *out, len(corpus))
	if len(failed) > 0 {
		sort.Strings(failed)
		fmt.Fprintf(os.Stderr, "warning: %d page(s) failed and were omitted:\n", len(failed))
		for _, f := range failed {
			fmt.Fprintf(os.Stderr, "  - %s\n", f)
		}
	}
}

func writeCorpus(path string, corpus map[string]entry) error {
	// Marshal deterministically so regenerations produce minimal diffs.
	raw, err := json.MarshalIndent(corpus, "", "  ")
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return err
	}
	if _, err := zw.Write(raw); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
