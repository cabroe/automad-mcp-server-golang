// automad-mcp-server is an MCP server that exposes the Automad CMS documentation
// and the official Automad Theme Starter Kit repository to AI assistants via
// the Model Context Protocol.
//
// It provides:
//   - Docs tools: search_docs, get_page, list_pages
//   - Starter Kit tools: list_files, get_file_content, get_template_snippet, search_code, get_file_url
//   - Resources: automad://docs/sitemap
//   - Prompts: explain_concept, theme_development
//
// Usage (stdio transport, the default for MCP):
//
//	./automad-mcp-server
//
// See README.md for integration instructions.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cabroe/automad-mcp-server/internal/docs"
	mcpserver "github.com/cabroe/automad-mcp-server/internal/server"
	"github.com/cabroe/automad-mcp-server/internal/starterkit"
)

const (
	serverName    = "automad-docs"
	serverVersion = "1.0.0"
)

func main() {
	// Use stderr for logs so as not to pollute the MCP stdio protocol on stdout.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	if err := run(logger); err != nil {
		if isNormalShutdown(err) {
			logger.Info("server shut down", "reason", err)
			return
		}
		logger.Error("server error", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("starting Automad MCP server",
		"name", serverName,
		"version", serverVersion,
		"transport", "stdio",
	)

	// Initialize the documentation service (fetcher + parser + cache).
	svc := docs.NewService()
	logger.Info("documentation service ready",
		"pages", fmt.Sprintf("%d pages in sitemap", len(docs.Sitemap())),
		"cache_ttl", docs.DefaultCacheTTL,
	)

	// Initialize the Starter Kit service (GitHub API client + cache), giving
	// access to the automadcms/automad-theme-starter-kit repository.
	skSvc := starterkit.NewService()
	logger.Info("starter kit service ready",
		"repo", fmt.Sprintf("%s/%s", starterkit.Owner, starterkit.Repo),
		"branch", skSvc.Branch(),
		"authenticated", skSvc.Authenticated(),
		"cache_ttl", starterkit.DefaultCacheTTL,
	)

	// Warm both caches in the background so search_docs / search_code can
	// rank on full content instead of just title/filename matches from the
	// first moment a client searches. This runs concurrently and never
	// blocks server startup or shuts the server down on failure — it's a
	// best-effort optimization, not a prerequisite for serving requests.
	go func() {
		warmCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()

		warmed, err := svc.WarmCache(warmCtx)
		if err != nil {
			logger.Warn("docs cache warm-up finished with errors", "warmed", warmed, "err", err)
			return
		}
		logger.Info("docs cache warm-up complete", "warmed", warmed)
	}()

	go func() {
		warmCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()

		warmed, err := skSvc.WarmFiles(warmCtx)
		if err != nil {
			logger.Warn("starter kit cache warm-up finished with errors", "warmed", warmed, "err", err)
			return
		}
		logger.Info("starter kit cache warm-up complete", "warmed", warmed)
	}()

	// Create the MCP server.
	s := mcp.NewServer(&mcp.Implementation{
		Name:    serverName,
		Version: serverVersion,
	}, &mcp.ServerOptions{
		Instructions: fmt.Sprintf(`This MCP server provides access to the official Automad CMS documentation (https://automad.org)
and the official Automad Theme Starter Kit repository (https://github.com/%s/%s).

Automad is a flat-file CMS and template engine. This server exposes:
- Docs tools: search_docs, get_page, list_pages
- Starter Kit tools: list_files, get_file_content, get_template_snippet, search_code, get_file_url
- Resources: automad://docs/sitemap (full documentation structure)
- Prompts: explain_concept, theme_development

Use search_docs to find relevant documentation pages, then get_page to read the full content.
Use the Starter Kit as the source of truth for theme development: list_files or search_code to
discover real template/component files, get_file_content or get_template_snippet to read them.`, starterkit.Owner, starterkit.Repo),
		Logger: logger,
	})

	// Register all MCP features.
	mcpserver.RegisterTools(s, svc)
	mcpserver.RegisterStarterKitTools(s, skSvc)
	mcpserver.RegisterResources(s, svc)
	mcpserver.RegisterPrompts(s, svc)

	logger.Info("MCP server initialized, listening on stdio")

	// Run using the stdio transport (standard for MCP servers invoked by clients).
	if err := s.Run(ctx, &mcp.StdioTransport{}); err != nil {
		if isNormalShutdown(err) {
			return err // propagate as-is for clean exit
		}
		return fmt.Errorf("server run error: %w", err)
	}

	logger.Info("server shut down gracefully")
	return nil
}

// isNormalShutdown reports whether err represents a routine, expected shutdown
// condition rather than an actual error:
//   - context.Canceled: triggered by SIGINT / SIGTERM
//   - io.EOF: client closed the stdio pipe (normal for short-lived inspector sessions)
//   - "server is closing": the MCP SDK's internal signal that the connection ended
func isNormalShutdown(err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "server is closing") ||
		strings.Contains(msg, "client is closing") ||
		strings.Contains(msg, "EOF")
}
