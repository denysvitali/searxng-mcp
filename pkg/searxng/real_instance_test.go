package searxng

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const testInstanceURLEnv = "SEARXNG_INSTANCE_URL"

// getTestInstanceURL returns the instance URL from environment variable or empty string
func getTestInstanceURL() string {
	return os.Getenv(testInstanceURLEnv)
}

// skipIfNoInstanceURL skips the test if the instance URL environment variable is not set
func skipIfNoInstanceURL(t *testing.T) {
	t.Helper()
	if getTestInstanceURL() == "" {
		t.Skipf("Real instance tests require %s environment variable to be set", testInstanceURLEnv)
	}
}

// TestRealInstance_BasicSearch tests basic connectivity and search functionality
func TestRealInstance_BasicSearch(t *testing.T) {
	skipIfNoInstanceURL(t)

	config := &Config{
		BaseURL: getTestInstanceURL(),
		Timeout: 30,
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	resp, err := client.Search(ctx, SearchRequest{
		Query: "golang",
		Limit: 5,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	// Query should be echoed back
	require.Equal(t, "golang", resp.Query)
}

// TestRealInstance_UnresponsiveEnginesField tests that unresponsive_engines field is handled correctly
func TestRealInstance_UnresponsiveEnginesField(t *testing.T) {
	skipIfNoInstanceURL(t)

	config := &Config{
		BaseURL: getTestInstanceURL(),
		Timeout: 30,
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	resp, err := client.Search(ctx, SearchRequest{
		Query: "test query",
		Limit: 10,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify UnresponsiveEngines is a valid slice (not causing unmarshal errors)
	// This is the key test - if the JSON unmarshal failed, this would have panicked
	for _, ue := range resp.UnresponsiveEngines {
		require.NotEmpty(t, ue.Name)
	}
}

// TestRealInstance_VariousQueries tests multiple different queries
func TestRealInstance_VariousQueries(t *testing.T) {
	skipIfNoInstanceURL(t)

	config := &Config{
		BaseURL: getTestInstanceURL(),
		Timeout: 30,
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()

	queries := []string{"python", "rust programming", "docker container", "kubernetes"}
	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			resp, err := client.Search(ctx, SearchRequest{
				Query: query,
				Limit: 3,
			})
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Equal(t, query, resp.Query)
		})
	}
}

// TestRealInstance_EmptyResults tests a query that returns no results
func TestRealInstance_EmptyResults(t *testing.T) {
	skipIfNoInstanceURL(t)

	config := &Config{
		BaseURL: getTestInstanceURL(),
		Timeout: 30,
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	resp, err := client.Search(ctx, SearchRequest{
		Query: "thisqueryshouldnotmatchanyresults12345xyz",
		Limit: 5,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	// Empty results should still work
	require.Equal(t, 0, len(resp.Results))
}

// TestRealInstance_JSONSearch tests the POST-based SearchJSON method
func TestRealInstance_JSONSearch(t *testing.T) {
	skipIfNoInstanceURL(t)

	config := &Config{
		BaseURL: getTestInstanceURL(),
		Timeout: 30,
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	resp, err := client.SearchJSON(ctx, SearchRequest{
		Query: "web search",
		Limit: 5,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "web search", resp.Query)
}

// TestRealInstance_LargeResultSet tests a query with higher limit
func TestRealInstance_LargeResultSet(t *testing.T) {
	skipIfNoInstanceURL(t)

	config := &Config{
		BaseURL: getTestInstanceURL(),
		Timeout: 30,
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	resp, err := client.Search(ctx, SearchRequest{
		Query: "technology",
		Limit: 20, // Max allowed
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	// Results should not exceed limit
	require.LessOrEqual(t, len(resp.Results), 20)
}

// TestRealInstance_SpecificEngines tests using specific engines
func TestRealInstance_SpecificEngines(t *testing.T) {
	skipIfNoInstanceURL(t)

	config := &Config{
		BaseURL: getTestInstanceURL(),
		Timeout: 30,
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	resp, err := client.Search(ctx, SearchRequest{
		Query:   "wikipedia search",
		Limit:   5,
		Engines: []string{"wikipedia"},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "wikipedia search", resp.Query)
}
