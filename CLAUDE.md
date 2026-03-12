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
- Configuration loaded from `.env` file (default current directory, override with `XQ_ENV` env)
- Scheduled notify loop runs only during trading hours (9:00-15:00 Beijing time)

**Xueqiu Client** (`internal/xueqiu/`)
- `FetchCubeViaAPI()`: HTTP-based via `/cubes/rebalancing/current.json`
- `Client`: HTTP client with cookie auth, rate limiting (400ms), retries
- Cookie format supports full Cookie string or just `xq_a_token=xxx`
- Snapshots stored in `$HOME/.xq_snapshots/{symbol}.json`

**Feishu Notification** (`internal/feishu/`)
- Sends text messages via Feishu app API
- Configured via `NotifyConfig` (XQ_FEISHU_* env vars)

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
4. **Run `task build` after code changes** - ensure the project compiles successfully before marking a task complete

## Configuration Files

| File | Purpose |
|------|---------|
| `cookies.txt` | Xueqiu authentication cookies (from Get cookies.txt LOCALLY plugin) |
| `cubes.txt` | Line-separated cube symbols, optional `symbol name` format |
| `.env` | Notify settings (XQ_NOTIFY_ENABLED, XQ_INTERVAL_MINUTES, XQ_WEIGHT_THRESHOLD, XQ_FEISHU_*). Use `XQ_ENV` env var to override path |
| `$HOME/.xq_snapshots/` | Directory for cube 持仓 snapshots |

**Environment Variables**
- `XQ_NOTIFY_ENABLED`: Enable/disable notifications (true/false)
- `XQ_INTERVAL_MINUTES`: Check interval in minutes
- `XQ_WEIGHT_THRESHOLD`: Weight change threshold percentage
- `XQ_FEISHU_APP_ID`: Feishu app ID
- `XQ_FEISHU_APP_SECRET`: Feishu app secret
- `XQ_FEISHU_RECEIVE_ID`: Feishu receiver ID (user or chat)
- `XQ_FEISHU_RECEIVE_TYPE`: Receiver type (open_id/user_id/union_id/chat_id)
- `XQ_ENV`: Path to .env file (default: `.env`)

## Trading Hours Check

Notifications only execute Mon-Fri 9:00-15:00 Beijing time. See `isTradingTime()` in `server.go`.
