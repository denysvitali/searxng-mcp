package tracing

import (
	"context"
	"os"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnabled_NoEnvVars(t *testing.T) {
	os.Unsetenv("SENTRY_DSN")
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	assert.False(t, Enabled())
}

func TestEnabled_SentryDSN(t *testing.T) {
	t.Setenv("SENTRY_DSN", "https://key@sentry.io/123")
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	assert.True(t, Enabled())
}

func TestEnabled_OTLPEndpoint(t *testing.T) {
	os.Unsetenv("SENTRY_DSN")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	assert.True(t, Enabled())
}

func TestMCPServerOptions_Disabled(t *testing.T) {
	os.Unsetenv("SENTRY_DSN")
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	opts := MCPServerOptions("stdio")
	assert.Nil(t, opts)
}

func TestMCPServerOptions_Enabled(t *testing.T) {
	t.Setenv("SENTRY_DSN", "https://key@sentry.io/123")
	opts := MCPServerOptions("stdio")
	assert.NotEmpty(t, opts)
}

func TestInit_NoopWhenDisabled(t *testing.T) {
	os.Unsetenv("SENTRY_DSN")
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	err := Init(context.Background())
	require.NoError(t, err)
}

func TestNetworkTransport(t *testing.T) {
	assert.Equal(t, "pipe", networkTransport("stdio"))
	assert.Equal(t, "tcp", networkTransport("http"))
	assert.Equal(t, "pipe", networkTransport("unknown"))
}

func TestToolCallMiddleware_PassesThrough(t *testing.T) {
	// Initialize a real TracerProvider so the middleware can create spans
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	err := Init(context.Background())
	require.NoError(t, err)
	defer Shutdown(context.Background()) //nolint:errcheck

	mw := toolCallMiddleware("stdio")

	called := false
	handler := mw(func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		called = true
		return mcp.NewToolResultText("hello"), nil
	})

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "test_tool",
			Arguments: map[string]interface{}{
				"key": "value",
			},
		},
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, called)
	assert.False(t, result.IsError)
}
