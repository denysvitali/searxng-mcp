package searxng

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/denysvitali/searxng-mcp/internal/log"
	"github.com/sirupsen/logrus"
)

var (
	ErrInvalidURL      = errors.New("invalid searxng instance URL")
	ErrRequestFailed   = errors.New("search request failed")
	ErrInvalidResponse = errors.New("invalid response from searxng")
	ErrTimeout         = errors.New("request timeout")
)

// rateLimiter implements a simple rate limiter using a token bucket
type rateLimiter struct {
	mu         sync.Mutex
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
}

// newRateLimiter creates a new rate limiter
// maxTokens: maximum number of tokens
// refillRate: time to add one token
func newRateLimiter(maxTokens int, refillRate time.Duration) *rateLimiter {
	return &rateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// wait waits until a token is available
func (rl *rateLimiter) wait(ctx context.Context) error {
	for {
		rl.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(rl.lastRefill)

		// Refill tokens based on elapsed time
		tokensToAdd := int(elapsed / rl.refillRate)
		if tokensToAdd > 0 {
			rl.tokens = min(rl.maxTokens, rl.tokens+tokensToAdd)
			rl.lastRefill = now
		}

		if rl.tokens > 0 {
			rl.tokens--
			rl.mu.Unlock()
			return nil
		}

		rl.mu.Unlock()

		// Wait for next refill or context cancellation
		select {
		case <-time.After(rl.refillRate):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Client is a Searxng API client
type Client struct {
	config      *Config
	httpClient  *http.Client
	rateLimiter *rateLimiter
}

// NewClient creates a new Searxng client
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Validate base URL
	if _, err := url.Parse(config.BaseURL); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidURL, err)
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		rateLimiter: newRateLimiter(10, 100*time.Millisecond), // 10 req/s limit
	}, nil
}

// Search performs a search query against Searxng
func (c *Client) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	// Apply defaults
	if req.Limit <= 0 {
		req.Limit = 5
	}
	if req.Limit > 20 {
		req.Limit = 20
	}
	if req.Page <= 0 {
		req.Page = 1
	}

	// Rate limiting
	if err := c.rateLimiter.wait(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTimeout, err)
	}

	log.WithFields(logrus.Fields{
		"query": req.Query,
		"limit": req.Limit,
		"page":  req.Page,
	}).Debug("performing search")

	// Build API request URL
	apiURL, err := c.buildSearchURL(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build search URL: %w", err)
	}

	// Perform request with retries
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			log.WithField("attempt", attempt).Debug("retrying search request")
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		var resp *SearchResponse
		resp, lastErr = c.doSearchRequest(ctx, apiURL)
		if lastErr == nil {
			return resp, nil
		}

		// Don't retry context errors or 4xx errors
		if errors.Is(lastErr, context.Canceled) || errors.Is(lastErr, context.DeadlineExceeded) {
			return nil, lastErr
		}
	}

	return nil, fmt.Errorf("%w: %w", ErrRequestFailed, lastErr)
}

// buildSearchURL builds the search API URL
func (c *Client) buildSearchURL(req SearchRequest) (string, error) {
	baseURL, err := url.Parse(c.config.BaseURL)
	if err != nil {
		return "", err
	}

	// Build search URL: /search?q=...&format=json
	searchPath, _ := url.Parse("/search")
	apiURL := baseURL.ResolveReference(searchPath)

	queryParams := url.Values{}
	queryParams.Set("q", req.Query)
	queryParams.Set("format", "json")

	if req.Category != "" {
		queryParams.Set("category", req.Category)
	}
	if req.Language != "" {
		queryParams.Set("language", req.Language)
	}
	if req.Page > 1 {
		queryParams.Set("pageno", strconv.Itoa(req.Page))
	}
	if req.TimeRange != "" {
		queryParams.Set("time_range", req.TimeRange)
	}

	for _, engine := range req.Engines {
		queryParams.Add("engines", engine)
	}

	return apiURL.String() + "?" + queryParams.Encode(), nil
}

// doSearchRequest performs the actual HTTP request
func (c *Client) doSearchRequest(ctx context.Context, searchURL string) (*SearchResponse, error) {
	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if c.config.UserAgent != "" {
		httpReq.Header.Set("User-Agent", c.config.UserAgent)
	}
	httpReq.Header.Set("Accept", "application/json")

	// Execute request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Check status code
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", httpResp.StatusCode, string(body))
	}

	// Parse response
	var apiResp APIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidResponse, err)
	}

	resp := toSearchResponse(apiResp)
	return &resp, nil
}

// SearchJSON performs a search using POST with JSON body
func (c *Client) SearchJSON(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	// Apply defaults
	if req.Limit <= 0 {
		req.Limit = 5
	}
	if req.Limit > 20 {
		req.Limit = 20
	}
	if req.Page <= 0 {
		req.Page = 1
	}

	// Rate limiting
	if err := c.rateLimiter.wait(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTimeout, err)
	}

	log.WithFields(logrus.Fields{
		"query": req.Query,
		"limit": req.Limit,
		"page":  req.Page,
	}).Debug("performing JSON search")

	// Build API request URL
	baseURL, err := url.Parse(c.config.BaseURL)
	if err != nil {
		return nil, err
	}

	searchPath, _ := url.Parse("/search")
	apiURL := baseURL.ResolveReference(searchPath).String()

	// Build JSON request body
	apiReq := APIRequest{
		Query:     req.Query,
		Category:  req.Category,
		Engines:   req.Engines,
		Language:  req.Language,
		Pageno:    req.Page,
		TimeRange: req.TimeRange,
		Format:    "json",
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Perform request with retries
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			log.WithField("attempt", attempt).Debug("retrying JSON search request")
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		var resp *SearchResponse
		resp, lastErr = c.doSearchJSONRequest(ctx, apiURL, body)
		if lastErr == nil {
			return resp, nil
		}

		// Don't retry context errors or 4xx errors
		if errors.Is(lastErr, context.Canceled) || errors.Is(lastErr, context.DeadlineExceeded) {
			return nil, lastErr
		}
	}

	return nil, fmt.Errorf("%w: %w", ErrRequestFailed, lastErr)
}

// doSearchJSONRequest performs the actual HTTP POST request
func (c *Client) doSearchJSONRequest(ctx context.Context, apiURL string, body []byte) (*SearchResponse, error) {
	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if c.config.UserAgent != "" {
		httpReq.Header.Set("User-Agent", c.config.UserAgent)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Check status code
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", httpResp.StatusCode, string(body))
	}

	// Parse response
	var apiResp APIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidResponse, err)
	}

	resp := toSearchResponse(apiResp)
	return &resp, nil
}
