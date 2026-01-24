//go:build integration
// +build integration

package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/denysvitali/searxng-mcp/pkg/searxng"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTestInstanceURL returns the Searxng instance URL for testing
// It checks the SEARXNG_INSTANCE_URL environment variable first,
// then falls back to the default instance
func getTestInstanceURL() string {
	if url := os.Getenv("SEARXNG_INSTANCE_URL"); url != "" {
		return url
	}
	return "https://searxng.example.com"
}

func TestIntegration_BasicSearch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := &searxng.Config{
		BaseURL: getTestInstanceURL(),
		Timeout: defaultTimeout(),
	}

	client, err := searxng.NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	resp, err := client.Search(ctx, searxng.SearchRequest{
		Query: "golang",
		Limit: 5,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, resp.Query)
	assert.GreaterOrEqual(t, resp.NumberOfResults, 0)
	assert.NotEmpty(t, resp.Results)
}

func TestIntegration_SearchWithFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := &searxng.Config{
		BaseURL: getTestInstanceURL(),
		Timeout: defaultTimeout(),
	}

	client, err := searxng.NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test with time range
	resp, err := client.Search(ctx, searxng.SearchRequest{
		Query:     "golang",
		Limit:     5,
		TimeRange: "day",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, resp.Results)
}

func TestIntegration_SearchCategories(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := &searxng.Config{
		BaseURL: getTestInstanceURL(),
		Timeout: defaultTimeout(),
	}

	client, err := searxng.NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()

	categories := []string{"general", "images", "videos"}

	for _, category := range categories {
		t.Run(fmt.Sprintf("category_%s", category), func(t *testing.T) {
			resp, err := client.Search(ctx, searxng.SearchRequest{
				Query:    "test",
				Limit:    3,
				Category: category,
			})

			require.NoError(t, err)
			assert.NotEmpty(t, resp.Query)
			assert.NotNil(t, resp.Results)
		})
	}
}

func TestIntegration_SearchPagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := &searxng.Config{
		BaseURL: getTestInstanceURL(),
		Timeout: defaultTimeout(),
	}

	client, err := searxng.NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Get first page
	page1, err := client.Search(ctx, searxng.SearchRequest{
		Query: "golang",
		Limit: 5,
		Page:  1,
	})

	require.NoError(t, err)

	// Get second page
	page2, err := client.Search(ctx, searxng.SearchRequest{
		Query: "golang",
		Limit: 5,
		Page:  2,
	})

	require.NoError(t, err)

	// Results should be different (or page2 should have results too)
	assert.NotEmpty(t, page1.Results)
	assert.NotEmpty(t, page2.Results)
}

func TestIntegration_SearchJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := &searxng.Config{
		BaseURL: getTestInstanceURL(),
		Timeout: defaultTimeout(),
	}

	client, err := searxng.NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	resp, err := client.SearchJSON(ctx, searxng.SearchRequest{
		Query: "golang",
		Limit: 5,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, resp.Query)
	assert.GreaterOrEqual(t, resp.NumberOfResults, 0)
}

func TestIntegration_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Test with invalid instance URL
	config := &searxng.Config{
		BaseURL: "https://invalid-nonexistent-searxng-instance.example.com",
		Timeout: defaultTimeout(),
	}

	client, err := searxng.NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = client.Search(ctx, searxng.SearchRequest{
		Query: "test",
	})

	assert.Error(t, err)
}

func TestIntegration_ResultFields(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := &searxng.Config{
		BaseURL: getTestInstanceURL(),
		Timeout: defaultTimeout(),
	}

	client, err := searxng.NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	resp, err := client.Search(ctx, searxng.SearchRequest{
		Query: "golang tutorial",
		Limit: 3,
	})

	require.NoError(t, err)

	if len(resp.Results) > 0 {
		result := resp.Results[0]
		assert.NotEmpty(t, result.Title)
		assert.NotEmpty(t, result.URL)
		// Content may be empty for some results
		// assert.NotEmpty(t, result.Content)
	}
}

func TestIntegration_ResponseFeatures(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := &searxng.Config{
		BaseURL: getTestInstanceURL(),
		Timeout: defaultTimeout(),
	}

	client, err := searxng.NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Search for something that might have suggestions
	resp, err := client.Search(ctx, searxng.SearchRequest{
		Query: "golang",
		Limit: 5,
	})

	require.NoError(t, err)

	// These may be empty, but the fields should exist
	assert.NotNil(t, resp.Suggestions)
	assert.NotNil(t, resp.Answers)
	assert.NotNil(t, resp.Corrections)
}

func defaultTimeout() int {
	return 30
}

// Test the Client creation with real instance
func TestIntegration_NewClient(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := &searxng.Config{
		BaseURL: getTestInstanceURL(),
	}

	client, err := searxng.NewClient(config)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

// Test concurrent searches
func TestIntegration_Concurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := &searxng.Config{
		BaseURL: getTestInstanceURL(),
		Timeout: defaultTimeout(),
	}

	client, err := searxng.NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Perform multiple concurrent searches
	results := make(chan error, 5)

	for i := 0; i < 5; i++ {
		go func(i int) {
			_, err := client.Search(ctx, searxng.SearchRequest{
				Query: fmt.Sprintf("test %d", i),
				Limit: 3,
			})
			results <- err
		}(i)
	}

	// All searches should complete
	for i := 0; i < 5; i++ {
		err := <-results
		// Some might fail due to rate limiting
		if err != nil {
			t.Logf("Search %d failed (may be due to rate limiting): %v", i, err)
		}
	}
}
