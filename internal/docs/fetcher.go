package docs

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// BaseURL is the root of the Automad documentation website.
	BaseURL = "https://automad.org"

	// userAgent identifies this MCP server in HTTP requests.
	userAgent = "automad-mcp-server/1.0 (github.com/cabroe/automad-mcp-server)"

	// fetchTimeout is the maximum time allowed for a single HTTP fetch.
	fetchTimeout = 15 * time.Second
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
// Returns the raw HTML body as a string.
func (f *Fetcher) Fetch(path string) (string, error) {
	url := BaseURL + path

	req, err := http.NewRequest(http.MethodGet, url, nil)
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading body of %s: %w", url, err)
	}

	return string(body), nil
}
