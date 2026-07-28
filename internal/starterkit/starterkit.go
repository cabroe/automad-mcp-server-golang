// Package starterkit gives MCP tools read access to the official Automad
// Theme Starter Kit repository on GitHub (automadcms/automad-theme-starter-kit).
// It is the "source of truth" reference implementation for Automad theme
// development: real templates, components, and a theme.json that an AI
// assistant can browse, search, and quote instead of guessing from general
// PHP/CMS knowledge.
package starterkit

import "time"

const (
	// Owner is the GitHub organization that owns the Starter Kit repository.
	Owner = "automadcms"
	// Repo is the Starter Kit repository name.
	Repo = "automad-theme-starter-kit"

	apiBaseURL = "https://api.github.com"
	rawBaseURL = "https://raw.githubusercontent.com"

	// DefaultCacheTTL is how long the file tree and individual file contents
	// stay cached before being re-fetched from the GitHub API.
	DefaultCacheTTL = 1 * time.Hour

	// SupportedExtensionsList is a human-readable summary of SupportedExtensions,
	// used in tool descriptions and error messages.
	SupportedExtensionsList = ".php, .json, .md, .txt, .css, .js"
)

// SupportedExtensions are the file types get_file_content, get_template_snippet,
// search_code, and cache warm-up will read. Other files (images, fonts,
// package-lock.json-sized blobs, etc.) are intentionally out of scope: they
// aren't useful as template reference material and would waste GitHub API
// quota and cache memory.
var SupportedExtensions = []string{".php", ".json", ".md", ".txt", ".css", ".js"}
