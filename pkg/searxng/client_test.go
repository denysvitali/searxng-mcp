package searxng

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "default config",
			config:  nil,
			wantErr: false,
		},
		{
			name: "custom config",
			config: &Config{
				BaseURL:    "https://example.com",
				Timeout:    10 * time.Second,
				MaxRetries: 1,
			},
			wantErr: false,
		},
		{
			name: "invalid URL",
			config: &Config{
				BaseURL: ":invalid",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestClient_Search_Basic(t *testing.T) {
	defer gock.OffAll()

	// Mock response
	mockResponse := APIResponse{
		Query:           "golang tutorial",
		NumberOfResults: 100,
		Results: []APIResult{
			{
				URL:     "https://example.com/golang",
				Title:   "Golang Tutorial",
				Content: "Learn Go programming",
			},
			{
				URL:     "https://example.com/go-tips",
				Title:   "Go Tips and Tricks",
				Content: "Best practices for Go",
			},
		},
	}

	gock.New("https://searxng.example.com").
		Get("/search").
		MatchParam("q", "golang tutorial").
		MatchParam("format", "json").
		Reply(200).
		JSON(mockResponse)

	config := DefaultConfig()
	client, err := NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	resp, err := client.Search(ctx, SearchRequest{
		Query: "golang tutorial",
		Limit: 5,
	})

	require.NoError(t, err)
	assert.Equal(t, "golang tutorial", resp.Query)
	assert.Equal(t, 100, resp.NumberOfResults)
	assert.Len(t, resp.Results, 2)
	assert.Equal(t, "Golang Tutorial", resp.Results[0].Title)
	assert.Equal(t, "https://example.com/golang", resp.Results[0].URL)
}

func TestClient_Search_WithFilters(t *testing.T) {
	defer gock.OffAll()

	mockResponse := APIResponse{
		Query:           "golang news",
		NumberOfResults: 50,
		Results: []APIResult{
			{
				URL:     "https://example.com/go-news",
				Title:   "Latest Go News",
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

	config := DefaultConfig()
	client, err := NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	resp, err := client.Search(ctx, SearchRequest{
		Query:     "golang news",
		TimeRange: "day",
		Category:  "news",
		Page:      2,
	})

	require.NoError(t, err)
	assert.Equal(t, "golang news", resp.Query)
	assert.Len(t, resp.Results, 1)
}

func TestClient_Search_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		setupMock  func()
		wantErrMsg string
	}{
		{
			name:       "HTTP 404",
			statusCode: 404,
			setupMock: func() {
				gock.New("https://searxng.example.com").
					Get("/search").
					Persist().
					Reply(404).
					BodyString("Not Found")
			},
			wantErrMsg: "HTTP 404",
		},
		{
			name:       "HTTP 500",
			statusCode: 500,
			setupMock: func() {
				gock.New("https://searxng.example.com").
					Get("/search").
					Persist().
					Reply(500).
					BodyString("Internal Server Error")
			},
			wantErrMsg: "HTTP 500",
		},
		{
			name: "invalid JSON",
			setupMock: func() {
				gock.New("https://searxng.example.com").
					Get("/search").
					Persist().
					Reply(200).
					BodyString("invalid json")
			},
			wantErrMsg: "invalid response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gock.Off()
			defer gock.Off()

			tt.setupMock()

			config := DefaultConfig()
			client, err := NewClient(config)
			require.NoError(t, err)

			ctx := context.Background()
			_, err = client.Search(ctx, SearchRequest{Query: "test"})

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}

func TestClient_Search_Limits(t *testing.T) {
	defer gock.OffAll()

	mockResponse := APIResponse{
		Query:           "test",
		NumberOfResults: 100,
		Results:         []APIResult{},
	}

	gock.New("https://searxng.example.com").
		Get("/search").
		Reply(200).
		JSON(mockResponse)

	config := DefaultConfig()
	client, err := NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test limit > 20 is capped to 20
	_, err = client.Search(ctx, SearchRequest{
		Query: "test",
		Limit: 30,
	})

	require.NoError(t, err)
	assert.True(t, gock.IsDone())
}

func TestClient_Search_Retry(t *testing.T) {
	defer gock.OffAll()

	// First attempt fails, second succeeds
	gock.New("https://searxng.example.com").
		Get("/search").
		MatchParam("q", "test").
		MatchParam("format", "json").
		Times(1).
		Reply(500).
		BodyString("Internal Server Error")

	gock.New("https://searxng.example.com").
		Get("/search").
		MatchParam("q", "test").
		MatchParam("format", "json").
		Times(1).
		Reply(200).
		JSON(APIResponse{
			Query:   "test",
			Results: []APIResult{},
		})

	config := DefaultConfig()
	config.MaxRetries = 3
	client, err := NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	resp, err := client.Search(ctx, SearchRequest{Query: "test"})

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestClient_Search_ContextCancel(t *testing.T) {
	defer gock.OffAll()

	// Create a test server that delays response
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		// Use APIResponse for JSON serialization
		apiResp := APIResponse{
			Query:   "test",
			Results: []APIResult{},
		}
		_ = json.NewEncoder(w).Encode(apiResp)
	}))
	defer ts.Close()

	config := DefaultConfig()
	config.BaseURL = ts.URL
	client, err := NewClient(config)
	require.NoError(t, err)

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = client.Search(ctx, SearchRequest{Query: "test"})
	assert.Error(t, err)
}

func TestParsePublishedDate(t *testing.T) {
	tests := []struct {
		name     string
		dateStr  string
		wantNil  bool
		wantYear int
	}{
		{
			name:     "RFC3339 format",
			dateStr:  "2024-01-15T10:30:00Z",
			wantNil:  false,
			wantYear: 2024,
		},
		{
			name:     "date only",
			dateStr:  "2024-01-15",
			wantNil:  false,
			wantYear: 2024,
		},
		{
			name:    "empty string",
			dateStr: "",
			wantNil: true,
		},
		{
			name:    "invalid format",
			dateStr: "invalid",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePublishedDate(tt.dateStr)
			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.wantYear, result.Year())
			}
		})
	}
}

func TestToSearchResult(t *testing.T) {
	apiResult := APIResult{
		URL:           "https://example.com",
		Title:         "Test Title",
		Content:       "Test Content",
		PublishedDate: "2024-01-15",
		Engine:        "google",
		Category:      "general",
		Score:         9.5,
		Thumbnail:     "https://example.com/thumb.jpg",
		Engines:       []string{"google", "bing"},
		Positions:     []int{1, 2},
	}

	result := toSearchResult(apiResult)

	assert.Equal(t, "https://example.com", result.URL)
	assert.Equal(t, "Test Title", result.Title)
	assert.Equal(t, "Test Content", result.Content)
	assert.NotNil(t, result.PublishedDate)
	assert.Equal(t, 2024, result.PublishedDate.Year())
	assert.Equal(t, "google", result.Engine)
	assert.Equal(t, "general", result.Category)
	assert.Equal(t, 9.5, result.Score)
	assert.Equal(t, "https://example.com/thumb.jpg", result.Thumbnail)
	assert.Equal(t, []string{"google", "bing"}, result.Engines)
	assert.Equal(t, []int{1, 2}, result.Positions)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "https://searxng.example.com", config.BaseURL)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, "searxng-mcp/1.0", config.UserAgent)
}

func TestRateLimiter(t *testing.T) {
	rl := newRateLimiter(5, 10*time.Millisecond)
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < 7; i++ {
		err := rl.wait(ctx)
		assert.NoError(t, err)
	}
	elapsed := time.Since(start)

	// Should have waited for at least one refill (10ms)
	assert.GreaterOrEqual(t, elapsed, 10*time.Millisecond)
}

func TestClient_SearchJSON(t *testing.T) {
	defer gock.OffAll()

	mockResponse := APIResponse{
		Query:           "json search",
		NumberOfResults: 10,
		Results: []APIResult{
			{
				URL:     "https://example.com/json",
				Title:   "JSON Result",
				Content: "Content",
			},
		},
	}

	gock.New("https://searxng.example.com").
		Post("/search").
		Reply(200).
		JSON(mockResponse)

	config := DefaultConfig()
	client, err := NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	resp, err := client.SearchJSON(ctx, SearchRequest{
		Query: "json search",
	})

	require.NoError(t, err)
	assert.Equal(t, "json search", resp.Query)
	assert.Len(t, resp.Results, 1)
}
