package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/denysvitali/searxng-mcp/internal/log"
	"github.com/denysvitali/searxng-mcp/pkg/searxng"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
)

// Server wraps the MCP server and Searxng client
type Server struct {
	mcpServer     *mcpserver.MCPServer
	searxngClient *searxng.Client
}

// New creates a new MCP server
func New(client *searxng.Client) *Server {
	s := &Server{
		searxngClient: client,
	}

	// Create MCP server
	mcpServer := mcpserver.NewMCPServer(
		"searxng-mcp",
		"1.0.0",
		mcpserver.WithToolCapabilities(true),
	)

	s.mcpServer = mcpServer

	// Register tools
	s.registerTools()

	return s
}

// registerTools registers all available tools
func (s *Server) registerTools() {
	// Register web_search tool
	webSearchTool := mcp.Tool{
		Name:        "web_search",
		Description: "Search the web and return limited results. Useful for finding current information, facts, and online resources.",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"query"},
			Properties: map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query string",
				},
				"limit": map[string]interface{}{
					"type":        "number",
					"description": "Number of results to return (default: 5, min: 1, max: 20)",
					"minimum":     1,
					"maximum":     20,
				},
				"time_range": map[string]interface{}{
					"type":        "string",
					"description": "Filter results by time period: 'day', 'month', or 'year'",
					"enum":        []string{"day", "month", "year"},
				},
				"category": map[string]interface{}{
					"type":        "string",
					"description": "Search category: 'general' (default), 'images', 'videos', 'news', 'map', 'music', 'it', 'science'",
				},
				"page": map[string]interface{}{
					"type":        "number",
					"description": "Page number for pagination (default: 1)",
					"minimum":     1,
				},
			},
		},
	}
	s.mcpServer.AddTool(webSearchTool, s.handleWebSearch)

	// Register web_read tool
	webReadTool := mcp.Tool{
		Name:        "web_read",
		Description: "Fetch and read content from a URL, converting HTML to Markdown. Useful for extracting readable text from web pages.",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"url"},
			Properties: map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "The URL to fetch and read",
				},
			},
		},
	}
	s.mcpServer.AddTool(webReadTool, s.handleWebRead)
}

// handleWebSearch handles the web_search tool call
func (s *Server) handleWebSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.WithField("request", request).Debug("handling web_search")

	// Parse arguments
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}

	// Extract query (required)
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	// Build search request
	req := searxng.SearchRequest{
		Query: query,
	}

	// Extract optional parameters
	if limit, ok := args["limit"].(float64); ok {
		req.Limit = int(limit)
	}
	if timeRange, ok := args["time_range"].(string); ok {
		req.TimeRange = timeRange
	}
	if category, ok := args["category"].(string); ok {
		req.Category = category
	}
	if page, ok := args["page"].(float64); ok {
		req.Page = int(page)
	}

	log.WithField("request", req).Debug("searching")

	// Perform search
	resp, err := s.searxngClient.Search(ctx, req)
	if err != nil {
		log.WithFields(logrus.Fields{"error": err}).Error("search failed")
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	// Format results as JSON
	resultJSON, err := json.MarshalIndent(formatSearchResults(resp), "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to format results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleWebRead handles the web_read tool call
func (s *Server) handleWebRead(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.WithField("request", request).Debug("handling web_read")

	// Parse arguments
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}

	// Extract URL (required)
	url, ok := args["url"].(string)
	if !ok || url == "" {
		return mcp.NewToolResultError("url is required"), nil
	}

	log.WithField("url", url).Debug("reading URL")

	// Fetch and parse the URL
	content, err := fetchURLContent(ctx, url)
	if err != nil {
		log.WithFields(logrus.Fields{"error": err}).Error("fetch URL failed")
		return mcp.NewToolResultError(fmt.Sprintf("failed to fetch URL: %v", err)), nil
	}

	return mcp.NewToolResultText(content), nil
}

// ServeStdio runs the server in stdio mode
func (s *Server) ServeStdio() error {
	log.Info("starting MCP server in stdio mode")
	return mcpserver.ServeStdio(s.mcpServer)
}

// ServeHTTP runs the server in HTTP mode using StreamableHTTP
func (s *Server) ServeHTTP(addr string) error {
	log.WithField("address", addr).Info("starting MCP server in HTTP mode")

	httpServer := mcpserver.NewStreamableHTTPServer(s.mcpServer)
	return httpServer.Start(addr)
}

// MCPServer returns the underlying MCP server for advanced usage
func (s *Server) MCPServer() *mcpserver.MCPServer {
	return s.mcpServer
}

// formatSearchResults formats the search response for JSON output
func formatSearchResults(resp *searxng.SearchResponse) map[string]interface{} {
	results := make([]map[string]interface{}, len(resp.Results))
	for i, r := range resp.Results {
		results[i] = map[string]interface{}{
			"title":   r.Title,
			"url":     r.URL,
			"snippet": r.Content,
		}
		if r.PublishedDate != nil {
			results[i]["published_date"] = r.PublishedDate.Format("2006-01-02")
		}
	}

	output := map[string]interface{}{
		"query":         resp.Query,
		"total_results": float64(resp.NumberOfResults),
		"results":       results,
	}

	if len(resp.Suggestions) > 0 {
		suggestions := make([]interface{}, len(resp.Suggestions))
		for i, s := range resp.Suggestions {
			suggestions[i] = s
		}
		output["suggestions"] = suggestions
	}

	if len(resp.Answers) > 0 {
		answers := make([]interface{}, len(resp.Answers))
		for i, a := range resp.Answers {
			answers[i] = a
		}
		output["answers"] = answers
	}

	if len(resp.Corrections) > 0 {
		corrections := make([]interface{}, len(resp.Corrections))
		for i, c := range resp.Corrections {
			corrections[i] = c
		}
		output["corrections"] = corrections
	}

	return output
}
