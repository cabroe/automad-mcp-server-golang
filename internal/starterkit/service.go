package starterkit

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sync/singleflight"
)

// warmFilesConcurrency bounds how many files WarmFiles fetches in parallel.
const warmFilesConcurrency = 5

// maxWarmFileSize skips unusually large files (e.g. package-lock.json) during
// background warm-up, so a handful of big lockfiles don't eat most of the
// unauthenticated rate-limit budget before anything useful is cached. A
// direct get_file_content call for such a file is not affected by this limit.
const maxWarmFileSize = 100 * 1024

// Service is the main entry point for reading the Automad Theme Starter Kit
// repository. It combines the Client and Cache into a single convenient API,
// mirroring the shape of docs.Service.
type Service struct {
	lifecycle context.Context
	client    *Client
	cache     *Cache
	group     singleflight.Group
}

// NewService creates a new Service with default settings.
func NewService() *Service {
	return NewServiceWithContext(context.Background())
}

// NewServiceWithContext creates a service whose shared GitHub requests stop
// when the server lifecycle context is canceled.
func NewServiceWithContext(ctx context.Context) *Service {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Service{
		lifecycle: ctx,
		client:    NewClient(),
		cache:     NewCache(DefaultCacheTTL),
	}
}

// Branch returns the git ref (branch or tag) this service reads from.
func (s *Service) Branch() string { return s.client.Branch() }

// ConfigError reports invalid Starter Kit environment configuration.
func (s *Service) ConfigError() error { return s.client.ConfigError() }

// Authenticated reports whether a GITHUB_TOKEN/GH_TOKEN was configured,
// raising the GitHub API rate limit from 60 to 5000 requests/hour.
func (s *Service) Authenticated() bool { return s.client.Authenticated() }

// isSupportedExtension reports whether path has one of SupportedExtensions.
func isSupportedExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, e := range SupportedExtensions {
		if e == ext {
			return true
		}
	}
	return false
}

// ListFiles returns the full recursive file/directory listing of the
// repository. The second return value reports whether the bundled fallback
// listing was used because the live GitHub API request failed.
func (s *Service) ListFiles(ctx context.Context) (*Tree, bool, error) {
	if err := s.ConfigError(); err != nil {
		return nil, false, err
	}
	if t := s.cache.GetTree(); t != nil {
		return t, false, nil
	}

	resultCh := s.group.DoChan("tree", func() (any, error) {
		if t := s.cache.GetTree(); t != nil {
			return t, nil
		}
		t, err := s.client.GetTree(s.lifecycle)
		if err != nil {
			return nil, err
		}
		s.cache.SetTree(t)
		return t, nil
	})
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	case result := <-resultCh:
		if result.Err != nil {
			if isFallbackEligible(result.Err) {
				return fallbackTree(), true, nil
			}
			return nil, false, result.Err
		}
		return result.Val.(*Tree), false, nil
	}
}

func isFallbackEligible(err error) bool {
	if err == nil {
		return false
	}
	var rateErr *RateLimitError
	if errors.As(err, &rateErr) {
		return true
	}
	var notFound *NotFoundError
	if errors.As(err, &notFound) {
		return false
	}
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) ||
		strings.Contains(err.Error(), "requesting ")
}

// NormalizeRepositoryPath validates and canonicalizes a repository-relative
// path. It rejects traversal, URL-like input, backslashes, and control bytes.
func NormalizeRepositoryPath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("path must not be empty")
	}
	if strings.HasPrefix(value, "//") {
		return "", fmt.Errorf("path must not start with multiple slashes")
	}
	if strings.Contains(value, "\\") || strings.ContainsAny(value, "\x00\r\n") {
		return "", fmt.Errorf("invalid repository path %q", value)
	}
	if u, err := url.Parse(value); err != nil || u.IsAbs() || u.Host != "" || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("path must be a repository-relative file path")
	}
	value = strings.TrimPrefix(value, "/")
	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			return "", fmt.Errorf("repository path must not contain '..'")
		}
	}
	clean := path.Clean(value)
	if clean == "." || clean == "" || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("path must be a repository-relative file path")
	}
	return clean, nil
}

// GetFileContent returns the content of a single file, using the cache when
// possible. The second return value reports whether embedded fallback
// content was used because the live GitHub API request failed; this is only
// possible for the curated set of paths in fallbackFiles.
func (s *Service) GetFileContent(ctx context.Context, path string) ([]byte, bool, error) {
	if err := s.ConfigError(); err != nil {
		return nil, false, err
	}
	path, err := NormalizeRepositoryPath(path)
	if err != nil {
		return nil, false, err
	}
	if !isSupportedExtension(path) {
		return nil, false, fmt.Errorf("unsupported file type %q; supported extensions: %s", filepath.Ext(path), SupportedExtensionsList)
	}

	if content, ok := s.cache.GetFile(path); ok {
		return content, false, nil
	}

	resultCh := s.group.DoChan("file:"+path, func() (any, error) {
		if content, ok := s.cache.GetFile(path); ok {
			return content, nil
		}
		content, err := s.client.GetContents(s.lifecycle, path)
		if err != nil {
			return nil, err
		}
		s.cache.SetFile(path, content)
		return content, nil
	})
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	case result := <-resultCh:
		if result.Err != nil {
			if isFallbackEligible(result.Err) {
				if fb, ok := fallbackFiles[path]; ok {
					return []byte(fb), true, nil
				}
			}
			return nil, false, result.Err
		}
		return result.Val.([]byte), false, nil
	}
}

// ValidateFilePath verifies that filePath names an existing file in the live
// repository tree. A fallback listing is not treated as authoritative.
func (s *Service) ValidateFilePath(ctx context.Context, filePath string) error {
	filePath, err := NormalizeRepositoryPath(filePath)
	if err != nil {
		return err
	}
	tree, fallback, err := s.ListFiles(ctx)
	if err != nil {
		return err
	}
	if fallback {
		return fmt.Errorf("GitHub is unavailable; existence of repository file %q cannot be verified against the live repository", filePath)
	}
	for _, entry := range tree.Entries {
		if entry.Path == filePath && entry.IsFile() {
			return nil
		}
	}
	return fmt.Errorf("repository file %q does not exist", filePath)
}

// FileURLs returns the direct raw-content URL and the human-browsable GitHub
// URL for a file, without fetching anything.
func (s *Service) FileURLs(filePath string) (rawURL, blobURL string, err error) {
	if err := s.ConfigError(); err != nil {
		return "", "", err
	}
	filePath, err = NormalizeRepositoryPath(filePath)
	if err != nil {
		return "", "", err
	}
	branch := s.client.Branch()
	rawURL = fmt.Sprintf("%s/%s/%s/%s/%s", rawBaseURL, Owner, Repo, branch, escapePath(filePath))
	blobURL = fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", Owner, Repo, url.PathEscape(branch), escapePath(filePath))
	return rawURL, blobURL, nil
}

// WarmFiles proactively fetches and caches the content of every supported
// file in the repository that isn't already cached, using a small worker
// pool. It is best-effort: a failure to fetch one file does not stop the
// others. It returns the number of files newly cached and a combined error
// describing any failures (nil if all succeeded).
//
// search_code only searches files that are already in the cache, so calling
// WarmFiles — e.g. once in the background on server startup — makes
// search_code results comprehensive instead of depending on which files a
// client happens to have fetched via get_file_content so far.
func (s *Service) WarmFiles(ctx context.Context) (int, error) {
	tree, _, err := s.ListFiles(ctx)
	if err != nil {
		return 0, err
	}

	if tree.Truncated {
		return 0, fmt.Errorf("GitHub returned a truncated repository tree; cache warm-up would be incomplete")
	}

	var (
		wg       sync.WaitGroup
		sem      = make(chan struct{}, warmFilesConcurrency)
		mu       sync.Mutex
		warmed   int
		failures []string
	)

	for _, entry := range tree.Entries {
		if !entry.IsFile() || !isSupportedExtension(entry.Path) {
			continue
		}
		if entry.Size > maxWarmFileSize {
			continue
		}
		if _, ok := s.cache.GetFile(entry.Path); ok {
			continue
		}

		wg.Add(1)
		go func(path string) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			if ctx.Err() != nil {
				return
			}

			if _, usedFallback, err := s.GetFileContent(ctx, path); err != nil {
				mu.Lock()
				failures = append(failures, fmt.Sprintf("%s: %v", path, err))
				mu.Unlock()
				return
			} else if usedFallback {
				// Fallback content isn't a live cache hit; don't count it as
				// "warmed" since it didn't come from (or get stored via) the API.
				return
			}

			mu.Lock()
			warmed++
			mu.Unlock()
		}(entry.Path)
	}

	wg.Wait()

	if len(failures) > 0 {
		return warmed, fmt.Errorf("failed to warm %d file(s): %s", len(failures), strings.Join(failures, "; "))
	}
	return warmed, nil
}

// SearchCode searches the cached content of supported files for a literal,
// case-insensitive substring. Files that aren't cached yet are skipped (and
// returned separately) rather than fetched on demand, so a single search
// can't unexpectedly burn through the GitHub rate limit; call WarmFiles (or
// get_file_content on specific files) first for full coverage.
func (s *Service) SearchCode(ctx context.Context, query string, extFilter []string) (matches []SearchMatch, uncached []string, err error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil, fmt.Errorf("query must not be empty")
	}

	tree, _, err := s.ListFiles(ctx)
	if err != nil {
		return nil, nil, err
	}
	if tree.Truncated {
		return nil, nil, fmt.Errorf("GitHub returned a truncated repository tree; code search would be incomplete")
	}

	for _, entry := range tree.Entries {
		if !entry.IsFile() || !isSupportedExtension(entry.Path) {
			continue
		}
		if !matchesExtFilter(entry.Path, extFilter) {
			continue
		}

		content, ok := s.cache.GetFile(entry.Path)
		if !ok {
			uncached = append(uncached, entry.Path)
			continue
		}

		for _, lm := range searchLines(content, query) {
			matches = append(matches, SearchMatch{Path: entry.Path, Line: lm.Line, Excerpt: lm.Text})
		}
	}

	return matches, uncached, nil
}

// CacheStats returns a human-readable summary of the current cache state.
func (s *Service) CacheStats() string {
	return s.cache.Stats()
}
