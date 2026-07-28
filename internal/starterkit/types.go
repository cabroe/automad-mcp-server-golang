package starterkit

import (
	"errors"
	"fmt"
	"time"
)

// TreeEntry is a single file or directory in the repository, as returned by
// GitHub's Git Trees API.
type TreeEntry struct {
	// Path is the entry's path relative to the repository root
	// (e.g. "components/page.php").
	Path string `json:"path"`
	// Type is "blob" for a file or "tree" for a directory.
	Type string `json:"type"`
	// Size is the file size in bytes. Zero for directories.
	Size int `json:"size"`
	// SHA is the git blob/tree SHA.
	SHA string `json:"sha"`
}

// IsFile reports whether the entry is a regular file (as opposed to a directory).
func (e TreeEntry) IsFile() bool { return e.Type == "blob" }

// Tree is the full recursive file/directory listing of the repository at a
// given branch.
type Tree struct {
	Entries []TreeEntry
	// Truncated is true if GitHub's API cut off the listing because the
	// repository is too large for a single recursive tree response. The
	// Starter Kit repository is small, so this should never happen in
	// practice, but it's surfaced to callers just in case.
	Truncated bool
	// FetchedAt is when this tree was retrieved (zero value for the bundled
	// fallback tree, which has no meaningful fetch time).
	FetchedAt time.Time
}

// RateLimitError indicates the GitHub API request quota is exhausted.
type RateLimitError struct {
	ResetAt time.Time
}

func (e *RateLimitError) Error() string {
	if e.ResetAt.IsZero() {
		return "GitHub API rate limit exceeded; set the GITHUB_TOKEN environment variable for a higher limit"
	}
	return fmt.Sprintf(
		"GitHub API rate limit exceeded, resets at %s; set the GITHUB_TOKEN environment variable for a higher limit",
		e.ResetAt.Format(time.RFC3339),
	)
}

var (
	ErrGitHubUnavailable = errors.New("GitHub is unavailable")
	ErrTreeTruncated     = errors.New("the GitHub repository tree is truncated")
)

// VerificationUnavailableError means the live repository could not provide a
// complete answer about whether a file exists.
type VerificationUnavailableError struct {
	Path  string
	Cause error
}

func (e *VerificationUnavailableError) Error() string {
	return fmt.Sprintf("existence of repository file %q cannot be verified: %v", e.Path, e.Cause)
}

func (e *VerificationUnavailableError) Unwrap() error { return e.Cause }

// HTTPStatusError represents a non-success GitHub response.
type HTTPStatusError struct {
	Status int
	Body   string
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("unexpected status %d from GitHub API: %s", e.Status, e.Body)
}

// NotFoundError indicates the requested path does not exist in the repository.
type NotFoundError struct {
	Path string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%q not found in %s/%s", e.Path, Owner, Repo)
}
