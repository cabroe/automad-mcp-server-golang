// automad-mcp-server is an MCP server that exposes the Automad CMS documentation
// to AI assistants via the Model Context Protocol.
//
// It provides:
//   - Tools: search_docs, get_page, list_pages
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

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cabroe/automad-mcp-server/internal/docs"
	mcpserver "github.com/cabroe/automad-mcp-server/internal/server"
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

	// Create the MCP server.
	s := mcp.NewServer(&mcp.Implementation{
		Name:    serverName,
		Version: serverVersion,
	}, &mcp.ServerOptions{
		Instructions: `This MCP server provides access to the official Automad CMS documentation (https://automad.org).

Automad is a flat-file CMS and template engine. This server exposes its documentation through:
- Tools: search_docs, get_page, list_pages
- Resources: automad://docs/sitemap (full documentation structure)
- Prompts: explain_concept, theme_development

Use search_docs to find relevant pages, then get_page to read the full content.`,
		Logger: logger,
	})

	// Register all MCP features.
	mcpserver.RegisterTools(s, svc)
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
