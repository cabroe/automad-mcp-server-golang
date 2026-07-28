# Automad MCP Server

[![Go](https://img.shields.io/badge/Go-1.26-blue.svg)](https://golang.org)
[![MCP SDK](https://img.shields.io/badge/MCP%20SDK-v1.7.0-green.svg)](https://github.com/modelcontextprotocol/go-sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Ein MCP-Server (**Model Context Protocol**) der die [Automad CMS](https://automad.org) Dokumentation für KI-Assistenten zugänglich macht. Implementiert mit dem offiziellen [Go SDK für MCP](https://github.com/modelcontextprotocol/go-sdk).

## Features

### 🔧 Tools

| Tool | Beschreibung |
|------|-------------|
| `search_docs` | Volltextsuche über alle Automad-Doku-Seiten |
| `get_page` | Seite vollständig abrufen und als Text zurückgeben |
| `list_pages` | Alle Seiten auflisten, optional nach Kategorie filtern |

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

Diesen Prompt an einen KI-Coding-Agenten mit Datei-/Shell-Zugriff geben, um den Server automatisch zu bauen und einzurichten:

```
Baue den automad-mcp-server aus diesem Repo https://github.com/cabroe/automad-mcp-server-golang (`make build` bzw. `go build -o automad-mcp-server .`)
und trage ihn als MCP-Server "automad-docs" mit dem absoluten Pfad zur Binary in die MCP-Konfiguration
meines aktuellen Tools ein (z. B. claude_desktop_config.json, .cursor/mcp.json, .mcp.json oder via
`claude mcp add`). Zeig mir danach die verwendete Konfiguration.
```

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
├── internal/
│   ├── docs/
│   │   ├── cache.go            # In-Memory-Cache mit TTL
│   │   ├── fetcher.go          # HTTP-Client für automad.org
│   │   ├── parser.go           # HTML → strukturierter Text
│   │   ├── service.go          # Kombinierer: GetPage, Search, ListPages
│   │   └── sitemap.go          # Alle bekannten Doku-URLs
│   └── server/
│       ├── tools.go            # MCP Tool-Handler
│       ├── resources.go        # MCP Resource-Handler
│       └── prompts.go          # MCP Prompt-Handler
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
- **Cache-TTL**: 1 Stunde (konfigurierbar in `docs/service.go`)
- **Dokumentations-Strategie**: Live-Fetch von automad.org mit In-Memory-Cache
- **Cache-Warmup**: Beim Start lädt der Server im Hintergrund alle Sitemap-Seiten (max. 5 parallel, 2 Min. Timeout), damit `search_docs` von Anfang an über den vollen Seiteninhalt sucht statt nur über Titel/URL
- **MCP-Version**: 2026-07-28 (via go-sdk v1.7.0)

## Lizenz

MIT
