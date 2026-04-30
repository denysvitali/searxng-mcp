# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

- Build: `go build .`
- Run all unit tests: `go test ./...`
- Run a single test: `go test ./pkg/server -run TestName -v`
- Integration tests (hit a real Searxng instance, gated by build tag): `go test -tags=integration ./... -run TestIntegration_` (set `SEARXNG_INSTANCE_URL` to override the default target)
- Lint (matches CI): `golangci-lint run ./...`
- Formatting (CI fails if either reports anything): `gofmt -l .` and `goimports -l .`
- Snapshot release build: `goreleaser build --snapshot --clean`
- Run locally: `go run . serve --instance-url https://your-searxng-instance`

CI runs the `test` matrix against Go 1.24 and 1.25; `lint` and `goreleaser build` run on 1.25.

## Architecture

Single Cobra/Viper binary that exposes a Searxng-backed MCP server. Entry point is `main.go` at the repo root, which calls `cmd.Execute()`.

Layers:

- `cmd/` — Cobra commands. `root.go` wires Viper: flags → env (`SEARXNG_URL`, `SEARXNG_TIMEOUT`, `LOG_LEVEL`) → YAML file at `$HOME/.config/searxng-mcp/config.yaml`, merged in that precedence. Tracing env vars (`SENTRY_DSN`, `SENTRY_TRACES_SAMPLE_RATE`, `OTEL_EXPORTER_OTLP_*`) are bound to Viper and re-exported to `os.Environ` in `initConfig` via `exportToEnv`, because the tracing package reads them directly from the environment — keep that round-trip intact if you add new tracing settings.
- `cmd/serve.go` — builds the Searxng client, initializes tracing, composes `mcpserver.ServerOption`s (notably `tracing.MCPServerOptions`) and starts either stdio (default, for MCP clients) or `StreamableHTTP` transport.
- `pkg/searxng/` — HTTP client for a Searxng instance. `client.go` handles request/response incl. parsing Searxng's tuple-format `unresponsive_engines`; `types.go` defines the domain model; `config.go` holds `BaseURL`/`Timeout`.
- `pkg/server/` — MCP tool layer. `server.go` registers two tools: `searxng_search` (delegates to the Searxng client, results formatted by `formatSearchResults`) and `searxng_read`. `reader.go` does generic HTML→Markdown, while `reader_reddit.go` and `reader_github.go` special-case Reddit threads (via `.json`) and GitHub issues/PRs (via API, combining issue/PR body + comments). `fetchURLContent` dispatches to the right reader based on URL shape.
- `internal/log/` — thin logrus wrapper; `log.Init(level)` is called from `PersistentPreRunE`.
- `internal/tracing/` — opt-in Sentry + OpenTelemetry. `Init` / `Shutdown` are no-ops unless the corresponding env vars are set. `MCPServerOptions(transport)` returns middleware that wraps tool calls; the stdio vs http transport string affects span attributes.
- `testdata/` — recorded JSON fixtures (Searxng response, Reddit thread, GitHub issue/PR + comments) used by reader/client tests. When adding a new special-case reader, add a fixture here and a matching `*_test.go` rather than hitting the network.
- `integration_test.go` at the repo root is behind `//go:build integration` and is skipped by normal `go test ./...`.

## Conventions

- Tool argument parsing in `pkg/server/server.go` uses `map[string]interface{}` type assertions (`float64` for numbers per JSON decoding); follow the same pattern when adding tools.
- New config knobs should be added as a Cobra flag + `viper.BindPFlag` + optional `viper.BindEnv` in `cmd/root.go`, so they work across flags, env, and the YAML config file uniformly.
