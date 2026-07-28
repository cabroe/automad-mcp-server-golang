package starterkit

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// userAgent identifies this MCP server in GitHub API requests.
	userAgent = "automad-mcp-server/1.0 (github.com/cabroe/automad-mcp-server)"

	// requestTimeout is the maximum time allowed for a single GitHub API call.
	requestTimeout = 20 * time.Second

	// retryBackoff is how long the client waits before retrying a request
	// that failed with a network error or a 5xx server error.
	retryBackoff = 400 * time.Millisecond
)

// Client is a small, rate-limit-aware GitHub REST API client scoped to a
// single repository (Owner/Repo, see starterkit.go).
type Client struct {
	httpClient *http.Client
	token      string
	branch     string

	mu            sync.Mutex
	rateKnown     bool
	rateRemaining int
	rateResetAt   time.Time
}

// NewClient creates a Client for the Starter Kit repository. If the
// GITHUB_TOKEN (or GH_TOKEN) environment variable is set, it is sent as a
// bearer token, raising the GitHub API rate limit from 60 to 5000
// requests/hour. The branch defaults to "master" (the repository's default
// branch) and can be overridden via AUTOMAD_STARTER_KIT_REF, e.g. to pin a
// tag or another branch.
func NewClient() *Client {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	branch := os.Getenv("AUTOMAD_STARTER_KIT_REF")
	if branch == "" {
		branch = "master"
	}
	return &Client{
		httpClient: &http.Client{Timeout: requestTimeout},
		token:      token,
		branch:     branch,
	}
}

// Branch returns the git ref (branch or tag) this client reads from.
func (c *Client) Branch() string { return c.branch }

// Authenticated reports whether a GitHub token was configured, which is
// useful for surfacing hints ("set GITHUB_TOKEN...") only when it would
// actually help.
func (c *Client) Authenticated() bool { return c.token != "" }

// GetTree fetches the full recursive file/directory listing of the
// repository at the client's branch using the Git Trees API. This is a
// single API call regardless of repository size (as opposed to walking the
// Contents API directory by directory), which matters a lot for staying
// under GitHub's unauthenticated rate limit.
func (c *Client) GetTree(ctx context.Context) (*Tree, error) {
	treeURL := fmt.Sprintf("%s/repos/%s/%s/git/trees/%s?recursive=1", apiBaseURL, Owner, Repo, url.PathEscape(c.branch))

	req, err := c.newRequest(ctx, treeURL)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.statusError(resp, "(repository tree)"); err != nil {
		return nil, err
	}

	var parsed struct {
		Tree      []TreeEntry `json:"tree"`
		Truncated bool        `json:"truncated"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decoding tree response: %w", err)
	}

	return &Tree{
		Entries:   parsed.Tree,
		Truncated: parsed.Truncated,
		FetchedAt: time.Now(),
	}, nil
}

// GetContents fetches and decodes the raw content of a single file via the
// GitHub Contents API.
func (c *Client) GetContents(ctx context.Context, path string) ([]byte, error) {
	contentsURL := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s",
		apiBaseURL, Owner, Repo, escapePath(path), url.QueryEscape(c.branch))

	req, err := c.newRequest(ctx, contentsURL)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.statusError(resp, path); err != nil {
		return nil, err
	}

	var parsed struct {
		Type     string `json:"type"`
		Encoding string `json:"encoding"`
		Content  string `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decoding contents response for %s: %w", path, err)
	}

	if parsed.Type != "" && parsed.Type != "file" {
		return nil, fmt.Errorf("%q is a %s, not a file; use list_files to browse it", path, parsed.Type)
	}
	if parsed.Encoding != "base64" {
		return nil, fmt.Errorf("unsupported content encoding %q for %s", parsed.Encoding, path)
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(parsed.Content, "\n", ""))
	if err != nil {
		return nil, fmt.Errorf("decoding base64 content of %s: %w", path, err)
	}
	return decoded, nil
}

// newRequest builds a GET request with the headers GitHub's API expects.
func (c *Client) newRequest(ctx context.Context, rawURL string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", rawURL, err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", userAgent)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return req, nil
}

// do executes a request, tracking GitHub's rate-limit headers and refusing
// to even attempt a call when a prior response told us the quota is
// exhausted. It retries once on network errors or 5xx responses.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	c.mu.Lock()
	exhausted := c.rateKnown && c.rateRemaining <= 0 && time.Now().Before(c.rateResetAt)
	resetAt := c.rateResetAt
	c.mu.Unlock()
	if exhausted {
		return nil, &RateLimitError{ResetAt: resetAt}
	}

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("requesting %s: %w", req.URL, err)
		} else {
			c.recordRateLimit(resp.Header)
			if resp.StatusCode >= 500 && attempt == 0 {
				resp.Body.Close()
				lastErr = fmt.Errorf("GitHub API returned status %d for %s", resp.StatusCode, req.URL)
			} else {
				return resp, nil
			}
		}

		if attempt == 0 {
			select {
			case <-time.After(retryBackoff):
			case <-req.Context().Done():
				return nil, req.Context().Err()
			}
		}
	}
	return nil, lastErr
}

// recordRateLimit updates the client's view of the GitHub rate limit from
// response headers, so subsequent calls can short-circuit instead of
// wasting the last bit of quota (or waiting out an HTTP timeout) once it's
// known to be exhausted.
func (c *Client) recordRateLimit(h http.Header) {
	remaining := h.Get("X-RateLimit-Remaining")
	reset := h.Get("X-RateLimit-Reset")
	if remaining == "" || reset == "" {
		return
	}
	r, errR := strconv.Atoi(remaining)
	t, errT := strconv.ParseInt(reset, 10, 64)
	if errR != nil || errT != nil {
		return
	}
	c.mu.Lock()
	c.rateRemaining = r
	c.rateResetAt = time.Unix(t, 0)
	c.rateKnown = true
	c.mu.Unlock()
}

// statusError translates a non-2xx GitHub API response into a typed error
// (RateLimitError, NotFoundError, or a generic error with the response body
// for anything else). It returns nil for 2xx responses.
func (c *Client) statusError(resp *http.Response, path string) error {
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return &NotFoundError{Path: path}
	case http.StatusForbidden, http.StatusTooManyRequests:
		c.mu.Lock()
		resetAt := c.rateResetAt
		c.mu.Unlock()
		return &RateLimitError{ResetAt: resetAt}
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("unexpected status %d from GitHub API: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}

// escapePath percent-encodes each path segment individually so that slashes
// keep separating directories instead of being escaped themselves.
func escapePath(path string) string {
	segments := strings.Split(path, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	return strings.Join(segments, "/")
}
