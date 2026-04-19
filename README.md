# mihomo-cli-tui

Interactive terminal UI for [mihomo-cli](https://github.com/taoshop/mihomo-cli) — standalone extension plugin.

## Overview

Standalone Bubble Tea TUI that connects to `mihomo-cli serve` via HTTP API. Provides visual proxy node management, connection monitoring, and traffic statistics.

## Prerequisites

- mihomo-cli >= 0.2.0 with `serve` command running
- Go 1.21+ (for building from source)

## Installation

### Via mihomo-cli extension manager (recommended)

```bash
mihomo-cli ext install tui
```

### Build from source

```bash
git clone https://github.com/taoshop/mihomo-cli-extension-tui.git
cd mihomo-cli-extension-tui
make build
```

### Cross-compile for all platforms

```bash
make build-all
```

## Usage

1. Start mihomo-cli API server:

   ```bash
   mihomo-cli serve
   ```

2. Launch TUI:

   ```bash
   # Via extension manager
   mihomo-cli ui

   # Or standalone binary
   ./mihomo-cli-tui
   ```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `MIHOMO_CLI_API_ADDR` | API endpoint | `http://127.0.0.1:8080` |
| `MIHOMO_CLI_API_KEY` | API key (if serve uses `--api-key`) | "" |

## Key Bindings

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate |
| `Enter` | Confirm / Switch node |
| `r` | Refresh data |
| `1`–`5` | Switch tabs |
| `Tab` | Next tab |
| `q` / `ESC` / `Ctrl+C` | Exit |

## Tabs

1. **Status** — Service status, PID, uptime
2. **Nodes** — Proxy node list, switch active node
3. **Rules** — Routing rules
4. **Logs** — Recent log entries (last 50 lines)
5. **Stats** — Upload/download traffic

## Architecture

```
┌──────────────────────────────────────┐
│       mihomo-cli serve (core)        │
│  /api/v1/status  /api/v1/nodes       │
│  /api/v1/rules   /api/v1/logs        │
└──────────────────────────────────────┘
                   │ HTTP
                   ▼
┌──────────────────────────────────────┐
│     mihomo-cli-tui (extension)       │
│  Bubble Tea TUI + HTTP client        │
└──────────────────────────────────────┘
```

## Project Structure

```
.
├── main.go          # TUI entry point + Bubble Tea implementation
├── go.mod           # Go module (standalone, no core dependency)
├── go.sum
├── manifest.yaml    # Extension manifest (for ext install)
├── Makefile         # Build automation
├── LICENSE          # Apache License 2.0
├── README.md        # This file
└── .gitignore
```

## Development

```bash
# Build
make build

# Run standalone (requires mihomo-cli serve)
./mihomo-cli-tui

# Run standalone with custom API address
MIHOMO_CLI_API_ADDR=http://127.0.0.1:9090 ./mihomo-cli-tui
```

## License

Apache License 2.0 — see [LICENSE](LICENSE) for details.
