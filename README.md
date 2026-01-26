# searxng-mcp

MCP (Model Context Protocol) server for Searxng - enables AI assistants to search the web and read webpage content through Searxng instances.

## Overview

This MCP server provides two tools for AI assistants:

- **web_search**: Search the web using Searxng and return structured results
- **web_read**: Fetch and convert webpage content from URLs to Markdown

## Installation

### Go Install (Recommended)

Install the latest version using Go:

```bash
go install github.com/denysvitali/searxng-mcp@latest
```

This will install the `searxng-mcp` binary to your `GOPATH/bin` directory (typically `~/go/bin/`). Make sure this directory is in your `PATH`.

### Pre-built Binaries

Download pre-built binaries for your platform from the [GitHub Releases](https://github.com/denysvitali/searxng-mcp/releases) page.

Download the appropriate archive for your OS and architecture, extract it, and place the binary in your PATH.

### From Source

```bash
git clone https://github.com/denysvitali/searxng-mcp.git
cd searxng-mcp
go build -o searxng-mcp ./cmd/searxng-mcp
sudo cp searxng-mcp /usr/local/bin/
```

## MCP Configuration

### Claude Code

Add to your `~/.claude/code/mcp.json`:

```json
{
  "mcpServers": {
    "searxng": {
      "command": "searxng-mcp",
      "args": ["serve", "--searxng-url", "https://your-searxng-instance.example.com"]
    }
  }
}
```

## Tool Reference

### web_search

Search the web using Searxng and return limited results.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | The search query string |
| `limit` | number | No | Number of results (default: 5, min: 1, max: 20) |
| `time_range` | string | No | Filter by time: "day", "month", "year" |
| `category` | string | No | Search category: "general", "images", "videos", "news", "map", "music", "it", "science" |
| `page` | number | No | Page number for pagination (default: 1) |

**Example:**

```json
{
  "query": "Go MCP server tutorial",
  "limit": 10,
  "time_range": "month",
  "category": "general"
}
```

### web_read

Fetch and read content from a URL, converting HTML to Markdown.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | Yes | The URL to fetch and read |

**Example:**

```json
{
  "url": "https://example.com/article"
}
```

## Configuration

### Command Line Options

| Flag | Env Variable | Default | Description |
|------|--------------|---------|-------------|
| `--searxng-url` | `SEARXNG_URL` | `https://searxng.example.com` | Searxng instance base URL |
| `--log-level` | `LOG_LEVEL` | `info` | Log level: debug, info, warn, error |
| `--timeout` | `SEARXNG_TIMEOUT` | `30s` | HTTP request timeout |

### Environment Variables

- `SEARXNG_URL` - Base URL of your Searxng instance
- `SEARXNG_TIMEOUT` - Request timeout (e.g., "30s", "1m")
- `LOG_LEVEL` - Logging level (debug, info, warn, error)

### Examples

Using environment variables:

```bash
export SEARXNG_URL="https://searxng.example.com"
export SEARXNG_TIMEOUT="60s"
export LOG_LEVEL="debug"
searxng-mcp serve
```

Using command line flags:

```bash
searxng-mcp serve \
  --searxng-url "https://searxng.example.com" \
  --timeout 60s \
  --log-level debug
```

## Setting Up Searxng

If you don't have a Searxng instance, you can run one using Docker:

```bash
docker run -d \
  --name searxng \
  -p 8080:8080 \
  -v ./searxng:/etc/searxng \
  searxng/searxng
```

Then configure the base URL in the MCP server:

```bash
searxng-mcp serve --searxng-url "http://localhost:8080"
```

## License

MIT
