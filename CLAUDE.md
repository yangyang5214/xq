# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go-based Xueqiu (雪球) portfolio holdings monitoring tool. Fetches current holdings using authentication cookies, compares with previous snapshots, and sends Feishu（飞书）notifications when weight changes exceed configurable thresholds.

## Build & Run

```bash
# Build (cross-compiled for Linux AMD64)
task build
# Or: GOOS=linux GOARCH=amd64 go build -o bin/xq ./cmd/xq

# Run HTTP server (default :8080)
./bin/xq server --addr :8080 --cubes-file cubes.txt --cookies-file cookies.txt

# Run tests
go test ./...
# Run tests with verbose output
go test -v ./...
# Run tests for a specific package
go test ./internal/email
```

## Architecture

**CLI Layer** (`cmd/xq/`)
- Uses Cobra framework for commands
- Currently implements `server` subcommand
- Entry: `main.go` initializes global logger (`runtime.log`), then executes `rootCmd`

**HTTP Server** (`internal/server/`)
- Embedded static web UI via `//go:embed static`
- APIs: `/api/cubes`, `/api/cubes/{symbol}`, `/api/config` (GET/PUT), `/api/notify/run`
- Configuration persistence via `configStore` (`$HOME/.xq_config.json`, override with `XQ_CONFIG` env)
- Scheduled notify loop runs only during trading hours (9:00-15:00 Beijing time)

**Xueqiu Client** (`internal/xueqiu/`)
- `FetchCubeViaAPI()`: HTTP-based via `/cubes/rebalancing/current.json`
- `Client`: HTTP client with cookie auth, rate limiting (400ms), retries
- Cookie format supports full Cookie string or just `xq_a_token=xxx`
- Snapshots stored in `$HOME/.xq_snapshots/{symbol}.json`

**Feishu Notification** (`internal/feishu/`)
- Sends text messages via Feishu webhook API
- Configured via `NotifyConfig.FeishuWebhook`

**Logger Module** (`internal/logger/`)
- Global `logger.Log` object for all output
- Appends to `runtime.log` and stderr

**Data Flow**
1. Server loads cube symbols from `cubes.txt` (one per line, `#` for comments, `symbol name` format)
2. `runNotify()` iterates cubes → fetch via API → compare with snapshot → detect weight changes (abs(diff) >= threshold)
3. Feishu messages sent for: weight threshold breaches, fetch failures, or no-change (when threshold=0)
4. Current snapshot always saved after fetch
5. Detailed logging tracks all changes including those below threshold

## Important Rules (from .cursor/rules)

1. **Always update README.md** after adding/modifying commands, parameters, config, or APIs
2. **Never use `fmt.Printf/Println`** for output - must use `internal/logger.Log` (global logger) consistently
3. Use `log.Fatalf(...)` to exit, otherwise `log.Printf/Println` or logger.Log methods

## Configuration Files

| File | Purpose |
|------|---------|
| `cookies.txt` | Xueqiu authentication cookies (from Get cookies.txt LOCALLY plugin) |
| `cubes.txt` | Line-separated cube symbols, optional `symbol name` format |
| `$HOME/.xq_config.json` | Notify settings (enabled, interval, threshold, feishu_webhook). Override with `XQ_CONFIG` env |
| `$HOME/.xq_snapshots/` | Directory for cube 持仓 snapshots |

## Trading Hours Check

Notifications only execute Mon-Fri 9:00-15:00 Beijing time. See `isTradingTime()` in `server.go`.
