package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/denysvitali/searxng-mcp/pkg/searxng"
	"github.com/h2non/gock"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleWebSearch(t *testing.T) {
	defer gock.OffAll()

	// Mock Searxng response using APIResponse (the format the client expects)
	mockResponse := searxng.APIResponse{
		Query:           "golang tutorial",
		NumberOfResults: 100,
		Results: []searxng.APIResult{
			{
				URL:     "https://example.com/golang",
				Title:   "Golang Tutorial",
				Content: "Learn Go programming",
			},
		},
		Suggestions: []string{"golang course"},
	}

	gock.New("https://searxng.example.com").
		Get("/search").
		MatchParam("q", "golang tutorial").
		MatchParam("format", "json").
		Reply(200).
		JSON(mockResponse)

	config := searxng.DefaultConfig()
	client, err := searxng.NewClient(config)
	require.NoError(t, err)

	srv := New(client)

	// Create test request
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"query": "golang tutorial",
				"limit": float64(5),
			},
			Name: "web_search",
		},
	}

	ctx := context.Background()
	result, err := srv.handleWebSearch(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify result is text
	textContent := result.Content[0].(mcp.TextContent)
	assert.Equal(t, "text", textContent.Type)

	var resultMap map[string]interface{}
	err = json.Unmarshal([]byte(textContent.Text), &resultMap)
	require.NoError(t, err)

	assert.Equal(t, "golang tutorial", resultMap["query"])
	assert.Equal(t, float64(100), resultMap["total_results"])

	results := resultMap["results"].([]interface{})
	assert.Len(t, results, 1)

	firstResult := results[0].(map[string]interface{})
	assert.Equal(t, "Golang Tutorial", firstResult["title"])
	assert.Equal(t, "https://example.com/golang", firstResult["url"])
	assert.Equal(t, "Learn Go programming", firstResult["snippet"])

	assert.Equal(t, []interface{}{"golang course"}, resultMap["suggestions"])
}

func TestHandleWebSearch_MissingQuery(t *testing.T) {
	config := searxng.DefaultConfig()
	client, err := searxng.NewClient(config)
	require.NoError(t, err)

	srv := New(client)

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name:    "no query parameter",
			args:    map[string]interface{}{},
			wantErr: true,
			errMsg:  "query is required",
		},
		{
			name: "empty query string",
			args: map[string]interface{}{
				"query": "",
			},
			wantErr: true,
			errMsg:  "query is required",
		},
		{
			name: "query is not a string",
			args: map[string]interface{}{
				"query": 123,
			},
			wantErr: true,
			errMsg:  "query is required",
		},
		{
			name:    "invalid arguments format",
			args:    nil,
			wantErr: true,
			errMsg:  "query is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: tt.args,
					Name:      "web_search",
				},
			}

			ctx := context.Background()
			result, err := srv.handleWebSearch(ctx, request)

			if tt.wantErr {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.IsError)
				textContent := result.Content[0].(mcp.TextContent)
				assert.Contains(t, textContent.Text, tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.False(t, result.IsError)
			}
		})
	}
}

func TestHandleWebSearch_WithFilters(t *testing.T) {
	defer gock.OffAll()

	mockResponse := searxng.APIResponse{
		Query:           "golang news",
		NumberOfResults: 50,
		Results: []searxng.APIResult{
			{
				URL:    "https://example.com/go-news",
				Title:  "Latest Go News",
				Content: "Go 1.22 released",
			},
		},
	}

	gock.New("https://searxng.example.com").
		Get("/search").
		MatchParam("q", "golang news").
		MatchParam("format", "json").
		MatchParam("time_range", "day").
		MatchParam("category", "news").
		MatchParam("pageno", "2").
		Reply(200).
		JSON(mockResponse)

	config := searxng.DefaultConfig()
	client, err := searxng.NewClient(config)
	require.NoError(t, err)

	srv := New(client)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"query":      "golang news",
				"time_range": "day",
				"category":   "news",
				"page":       float64(2),
			},
			Name: "web_search",
		},
	}

	ctx := context.Background()
	result, err := srv.handleWebSearch(ctx, request)

	require.NoError(t, err)
	assert.False(t, result.IsError)
	textContent := result.Content[0].(mcp.TextContent)
	assert.Equal(t, "text", textContent.Type)
}

func TestHandleWebSearch_SearchError(t *testing.T) {
	defer gock.OffAll()

	gock.New("https://searxng.example.com").
		Get("/search").
		MatchParam("q", "test query").
		MatchParam("format", "json").
		Reply(500).
		BodyString("Internal Server Error")

	config := searxng.DefaultConfig()
	client, err := searxng.NewClient(config)
	require.NoError(t, err)

	srv := New(client)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"query": "test query",
			},
			Name: "web_search",
		},
	}

	ctx := context.Background()
	result, err := srv.handleWebSearch(ctx, request)

	require.NoError(t, err)
	assert.True(t, result.IsError)
	textContent := result.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent.Text, "search failed")
}

func TestHandleWebRead(t *testing.T) {
	// Create a test server that serves HTML
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<html>
				<head><title>Test Page</title></head>
				<body>
					<h1>Welcome</h1>
					<p>This is a test page with some content.</p>
				</body>
			</html>
		`))
	}))
	defer ts.Close()

	config := searxng.DefaultConfig()
	client, err := searxng.NewClient(config)
	require.NoError(t, err)

	srv := New(client)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"url": ts.URL,
			},
			Name: "web_read",
		},
	}

	ctx := context.Background()
	result, err := srv.handleWebRead(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, result)
	textContent := result.Content[0].(mcp.TextContent)
	assert.Equal(t, "text", textContent.Type)
	assert.Contains(t, textContent.Text, "Welcome")
	assert.Contains(t, textContent.Text, "test page")
}

func TestHandleWebRead_MissingURL(t *testing.T) {
	config := searxng.DefaultConfig()
	client, err := searxng.NewClient(config)
	require.NoError(t, err)

	srv := New(client)

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name:    "no url parameter",
			args:    map[string]interface{}{},
			wantErr: true,
			errMsg:  "url is required",
		},
		{
			name: "empty url string",
			args: map[string]interface{}{
				"url": "",
			},
			wantErr: true,
			errMsg:  "url is required",
		},
		{
			name: "url is not a string",
			args: map[string]interface{}{
				"url": 123,
			},
			wantErr: true,
			errMsg:  "url is required",
		},
		{
			name:    "invalid arguments format",
			args:    nil,
			wantErr: true,
			errMsg:  "url is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: tt.args,
					Name:      "web_read",
				},
			}

			ctx := context.Background()
			result, err := srv.handleWebRead(ctx, request)

			if tt.wantErr {
				require.NoError(t, err)
				assert.True(t, result.IsError)
				textContent := result.Content[0].(mcp.TextContent)
				assert.Contains(t, textContent.Text, tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.False(t, result.IsError)
			}
		})
	}
}

func TestHandleWebRead_InvalidURL(t *testing.T) {
	config := searxng.DefaultConfig()
	client, err := searxng.NewClient(config)
	require.NoError(t, err)

	srv := New(client)

	tests := []struct {
		name   string
		url    string
		errMsg string
	}{
		{
			name:   "invalid URL format",
			url:    ":invalid-url",
			errMsg: "invalid URL",
		},
		{
			name:   "unsupported scheme",
			url:    "ftp://example.com",
			errMsg: "unsupported URL scheme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]interface{}{
						"url": tt.url,
					},
					Name: "web_read",
				},
			}

			ctx := context.Background()
			result, err := srv.handleWebRead(ctx, request)

			require.NoError(t, err)
			assert.True(t, result.IsError)
			textContent := result.Content[0].(mcp.TextContent)
			assert.Contains(t, textContent.Text, tt.errMsg)
		})
	}
}

func TestFormatSearchResults(t *testing.T) {
	date := searxng.SearchResult{
		URL:    "https://example.com/test",
		Title:  "Test Result",
		Content: "Test content",
	}

	resp := &searxng.SearchResponse{
		Query:           "test query",
		NumberOfResults: 100,
		Results:         []searxng.SearchResult{date},
		Suggestions:     []string{"suggestion 1"},
		Answers:         []string{"answer 1"},
		Corrections:     []string{"correction 1"},
	}

	result := formatSearchResults(resp)

	assert.Equal(t, "test query", result["query"])
	assert.Equal(t, float64(100), result["total_results"])
	assert.Equal(t, []interface{}{"suggestion 1"}, result["suggestions"])
	assert.Equal(t, []interface{}{"answer 1"}, result["answers"])
	assert.Equal(t, []interface{}{"correction 1"}, result["corrections"])

	results := result["results"].([]map[string]interface{})
	assert.Len(t, results, 1)
	assert.Equal(t, "Test Result", results[0]["title"])
	assert.Equal(t, "https://example.com/test", results[0]["url"])
	assert.Equal(t, "Test content", results[0]["snippet"])
}

func TestNewServer(t *testing.T) {
	config := searxng.DefaultConfig()
	client, err := searxng.NewClient(config)
	require.NoError(t, err)

	srv := New(client)

	assert.NotNil(t, srv)
	assert.NotNil(t, srv.MCPServer())
}

// Helper function to create a test HTTP server
func newTestServer(html string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
}
