package docs

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// errRoundTripper makes every HTTP request fail, simulating an offline /
// air-gapped environment where automad.org is unreachable.
type errRoundTripper struct{}

func (errRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("simulated offline: network unreachable")
}

// offlineService builds a Service whose fetcher can never reach the network, so
// only the embedded corpus can satisfy requests.
func offlineService() *Service {
	return &Service{
		lifecycle: context.Background(),
		fetcher:   &Fetcher{client: &http.Client{Transport: errRoundTripper{}}},
		cache:     NewCache(DefaultCacheTTL),
	}
}

func TestCorpusLoaded(t *testing.T) {
	if got := CorpusSize(); got == 0 {
		t.Fatal("embedded corpus is empty; run `go run ./cmd/gen-corpus`")
	}
	page := corpusPage("/")
	if page == nil {
		t.Fatal(`corpusPage("/") = nil, want a snapshot`)
	}
	if !page.Offline {
		t.Error("corpus page should be marked Offline")
	}
	if strings.TrimSpace(page.Content) == "" {
		t.Error("corpus page has empty content")
	}
	if page.FullURL != BaseURL+"/" {
		t.Errorf("FullURL = %q, want %q", page.FullURL, BaseURL+"/")
	}
}

func TestGetPageOfflineFallback(t *testing.T) {
	svc := offlineService()
	page, err := svc.GetPage(context.Background(), "/")
	if err != nil {
		t.Fatalf("GetPage offline: %v", err)
	}
	if !page.Offline {
		t.Error("offline GetPage should return an Offline page")
	}
	if strings.TrimSpace(page.Content) == "" {
		t.Error("offline page content is empty")
	}
	// Offline pages must not be cached, so a later (potentially online) call
	// retries the live site.
	if svc.cache.Get("/") != nil {
		t.Error("offline fallback page should not be cached")
	}
}

func TestGetPageOfflineUnknownURL(t *testing.T) {
	svc := offlineService()
	// A syntactically valid path with no corpus entry cannot be served offline.
	if _, err := svc.GetPage(context.Background(), "/does-not-exist-in-corpus-xyz"); err == nil {
		t.Fatal("expected an error for an unknown URL while offline")
	}
}

func TestSearchUsesCorpusWhenCacheCold(t *testing.T) {
	svc := offlineService() // cache is empty and cannot be warmed
	results := svc.Search("template")
	if len(results) == 0 {
		t.Fatal("expected offline search hits from the corpus")
	}
	// If the corpus were not consulted, cold-cache hits would carry only a
	// "Section: ..." placeholder snippet. A real content excerpt proves the
	// snapshot content was searched.
	foundContentSnippet := false
	for _, r := range results {
		if !strings.HasPrefix(r.Snippet, "Section:") {
			foundContentSnippet = true
			break
		}
	}
	if !foundContentSnippet {
		t.Error("offline search did not rank on corpus content (no content snippets)")
	}
}
