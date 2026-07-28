# Automad MCP Server

[![Go](https://img.shields.io/badge/Go-1.26-blue.svg)](https://golang.org)
[![MCP SDK](https://img.shields.io/badge/MCP%20SDK-v1.7.0-green.svg)](https://github.com/modelcontextprotocol/go-sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Ein MCP-Server (**Model Context Protocol**) der die [Automad CMS](https://automad.org) Dokumentation sowie das offizielle [Automad Theme Starter Kit](https://github.com/automadcms/automad-theme-starter-kit) Repository für KI-Assistenten zugänglich macht. Implementiert mit dem offiziellen [Go SDK für MCP](https://github.com/modelcontextprotocol/go-sdk).

## Features

### 🔧 Docs-Tools

| Tool | Beschreibung |
|------|-------------|
| `search_docs` | Volltextsuche über alle Automad-Doku-Seiten |
| `get_page` | Seite vollständig abrufen und als Text zurückgeben |
| `list_pages` | Alle Seiten auflisten, optional nach Kategorie filtern |

### 🧩 Starter-Kit-Tools

Greifen live auf [automadcms/automad-theme-starter-kit](https://github.com/automadcms/automad-theme-starter-kit) zu — die "Source of Truth" für reale Theme-Dateien statt geratener Dateinamen. Details, Beispiele und typische Prompts: [SKILL.md](SKILL.md).

| Tool | Beschreibung |
|------|-------------|
| `list_files` | Listet alle Dateien/Ordner des Repos rekursiv als Baum auf |
| `get_file_content` | Liest den Inhalt einer Datei (`.php`, `.json`, `.md`, `.txt`, `.css`, `.js`) |
| `get_template_snippet` | Liefert kuratierte, häufig genutzte Dateien (page-Komponente, Pagination, Pagelist-Grid, theme.json, …) mit Erklärung |
| `search_code` | Durchsucht den gecachten Code nach Text, z. B. Automad-Syntax (`@{ }`, `<@ @>`) oder theme.json-Keys |
| `get_file_url` | Erzeugt Raw- und GitHub-URL für eine Datei, ohne sie abzurufen |

### 📦 Resources

| URI | Beschreibung |
|-----|-------------|
| `automad://docs/sitemap` | Vollständige Dokumentationsstruktur als JSON |

### 💬 Prompts

| Prompt | Beschreibung |
|--------|-------------|
| `explain_concept` | Erkläre ein Automad-Konzept anhand der offiziellen Docs |
| `theme_development` | Hilfe bei der Theme-Entwicklung mit Docs-Kontext |

## Installation

### Voraussetzungen

- Go 1.26+


### Bauen

```bash
git clone https://github.com/cabroe/automad-mcp-server-golang
cd automad-mcp-server-golang
make build
# oder direkt:
go build -o automad-mcp-server .
```

### Für KI-Agenten (Claude Code, Cursor Agent, etc.)

Installationsanleitung, Tool-Referenz mit Beispielen und typische Prompts für KI-Agenten stehen in [SKILL.md](SKILL.md).

## Konfiguration

### Claude Desktop

Füge folgendes zu `~/Library/Application Support/Claude/claude_desktop_config.json` hinzu:

```json
{
  "mcpServers": {
    "automad-docs": {
      "command": "/absoluter/pfad/zu/automad-mcp-server"
    }
  }
}
```

#### Optional: GitHub-Rate-Limit erhöhen

Die Starter-Kit-Tools nutzen die GitHub REST API unauthentifiziert (60 Requests/Stunde). Bei Bedarf per `GITHUB_TOKEN` (Read-only, public repo) auf 5000 Requests/Stunde erhöhen:

```json
{
  "mcpServers": {
    "automad-docs": {
      "command": "/absoluter/pfad/zu/automad-mcp-server",
      "env": { "GITHUB_TOKEN": "ghp_..." }
    }
  }
}
```

### Cursor

In `.cursor/mcp.json` oder global in `~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "automad-docs": {
      "command": "/absoluter/pfad/zu/automad-mcp-server",
      "args": []
    }
  }
}
```

### VS Code (mit Copilot / MCP-Erweiterung)

```json
{
  "mcp": {
    "servers": {
      "automad-docs": {
        "type": "stdio",
        "command": "/absoluter/pfad/zu/automad-mcp-server"
      }
    }
  }
}
```

## Nutzung

### Mit MCP Inspector testen

```bash
# Startet den interaktiven Inspector im Browser
npx @modelcontextprotocol/inspector go run .
```

> **Hinweis:** Der Server kommuniziert über das standardisierte MCP-Protokoll via stdio (kein rohes JSON-RPC).
> Verwende zum Testen immer den MCP Inspector oder binde ihn direkt in deinen MCP-Client (Claude, Cursor etc.) ein.


## Projektstruktur

```
automad-mcp-server-golang/
├── main.go                     # Einstiegspunkt
├── SKILL.md                    # Doku + Beispiele für die Starter-Kit-Tools
├── internal/
│   ├── docs/
│   │   ├── cache.go            # In-Memory-Cache mit TTL
│   │   ├── fetcher.go          # HTTP-Client für automad.org
│   │   ├── parser.go           # HTML → strukturierter Text
│   │   ├── service.go          # Kombinierer: GetPage, Search, ListPages, WarmCache
│   │   └── sitemap.go          # Alle bekannten Doku-URLs
│   ├── starterkit/
│   │   ├── client.go           # GitHub-API-Client (Trees + Contents), Rate-Limit-Tracking
│   │   ├── cache.go            # In-Memory-Cache mit TTL (Tree + Dateien)
│   │   ├── fallback.go         # Eingebettete Fallback-Inhalte für den API-Ausfall
│   │   ├── search.go           # Such-Helper für search_code
│   │   ├── snippets.go         # Registry für get_template_snippet
│   │   ├── tree.go             # Baum-Rendering für list_files
│   │   ├── service.go          # Kombinierer: ListFiles, GetFileContent, SearchCode, WarmFiles
│   │   └── types.go            # Tree/TreeEntry, Fehlertypen
│   └── server/
│       ├── tools.go              # MCP Tool-Handler (Docs)
│       ├── starterkit_tools.go   # MCP Tool-Handler (Starter Kit)
│       ├── resources.go          # MCP Resource-Handler
│       └── prompts.go            # MCP Prompt-Handler
├── Makefile
└── README.md
```

## Entwicklung

```bash
# Tests ausführen
make test

# Binary bauen
make build

# Direkt starten
make run
```

## Technische Details

- **Transport**: stdio (Standard-MCP-Transport)
- **Cache-TTL**: 1 Stunde (konfigurierbar in `docs/service.go` bzw. `starterkit/starterkit.go`)
- **Dokumentations-Strategie**: Live-Fetch von automad.org mit In-Memory-Cache
- **Starter-Kit-Strategie**: Live-Zugriff auf die GitHub REST API (Git Trees API für `list_files`, Contents API für `get_file_content`), mit In-Memory-Cache, Rate-Limit-Tracking (`X-RateLimit-*`-Header) und eingebetteten Fallback-Inhalten für die kuratierten Snippet-Dateien, falls die API nicht erreichbar ist
- **Cache-Warmup**: Beim Start lädt der Server im Hintergrund alle Sitemap-Seiten sowie alle unterstützten Starter-Kit-Dateien (je max. 5 parallel, 2 Min. Timeout), damit `search_docs`/`search_code` von Anfang an über den vollen Inhalt suchen statt nur über Titel/Dateiname
- **MCP-Version**: 2026-07-28 (via go-sdk v1.7.0)

## Lizenz

MIT
