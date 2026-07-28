package docs

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// BaseURL is the root of the Automad documentation website.
	BaseURL = "https://automad.org"

	// userAgent identifies this MCP server in HTTP requests.
	userAgent = "automad-mcp-server/1.0 (github.com/cabroe/automad-mcp-server-golang)"

	// fetchTimeout is the maximum time allowed for a single HTTP fetch.
	fetchTimeout = 15 * time.Second

	// maxResponseSize bounds documentation HTML retained in memory.
	maxResponseSize = 5 * 1024 * 1024
)

// Fetcher fetches raw HTML from automad.org with a shared HTTP client.
type Fetcher struct {
	client *http.Client
}

// NewFetcher creates a Fetcher with sensible timeouts.
func NewFetcher() *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: fetchTimeout,
		},
	}
}

// Fetch retrieves the HTML for a given relative path (e.g. "/user-guide").
// Returns the raw HTML body as a string. The provided context governs
// cancellation and deadlines for the underlying HTTP request, so a caller
// (e.g. an MCP tool handler) can abort a slow fetch when its own request
// context is cancelled instead of always waiting out the fixed client timeout.
func (f *Fetcher) Fetch(ctx context.Context, path string) (string, error) {
	url := BaseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request for %s: %w", url, err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching %s: unexpected status %d", url, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !strings.HasPrefix(contentType, "text/html") && !strings.HasPrefix(contentType, "application/xhtml+xml") {
		return "", fmt.Errorf("fetching %s: unexpected content type %q", url, contentType)
	}
	if resp.ContentLength > maxResponseSize {
		return "", fmt.Errorf("response from %s exceeds maximum size of %d bytes", url, maxResponseSize)
	}

	limited := io.LimitReader(resp.Body, maxResponseSize+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("reading body of %s: %w", url, err)
	}
	if len(body) > maxResponseSize {
		return "", fmt.Errorf("response from %s exceeds maximum size of %d bytes", url, maxResponseSize)
	}

	return string(body), nil
}
