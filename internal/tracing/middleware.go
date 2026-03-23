package tracing

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const tracerName = "mcp.server"

// MCPServerOptions returns mcpserver.ServerOptions that instrument MCP
// tool calls with OpenTelemetry spans following the Sentry MCP tracing
// conventions (https://develop.sentry.dev/sdk/foundations/client/integrations/mcp/tracing/).
//
// Returns nil when tracing is not enabled.
func MCPServerOptions(transport string) []mcpserver.ServerOption {
	if !Enabled() {
		return nil
	}
	return []mcpserver.ServerOption{
		mcpserver.WithToolHandlerMiddleware(toolCallMiddleware(transport)),
		mcpserver.WithHooks(initializeHooks(transport)),
	}
}

// toolCallMiddleware creates spans for MCP tools/call requests.
func toolCallMiddleware(transport string) mcpserver.ToolHandlerMiddleware {
	tracer := otel.Tracer(tracerName)
	netTransport := networkTransport(transport)

	return func(next mcpserver.ToolHandlerFunc) mcpserver.ToolHandlerFunc {
		return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			toolName := request.Params.Name

			ctx, span := tracer.Start(ctx,
				fmt.Sprintf("tools/call %s", toolName),
				oteltrace.WithSpanKind(oteltrace.SpanKindServer),
				oteltrace.WithAttributes(
					attribute.String("mcp.method.name", "tools/call"),
					attribute.String("mcp.tool.name", toolName),
					attribute.String("mcp.transport", transport),
					attribute.String("network.transport", netTransport),
					attribute.String("network.protocol.version", "2.0"),
				),
			)
			defer span.End()

			// Session ID from HTTP header (available in HTTP/SSE transport)
			if sessionID := request.Header.Get("X-MCP-Session-Id"); sessionID != "" {
				span.SetAttributes(attribute.String("mcp.session.id", sessionID))
			}

			// Tool arguments as span attributes
			if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
				for k, v := range args {
					span.SetAttributes(attribute.String(
						fmt.Sprintf("mcp.request.argument.%s", k),
						fmt.Sprintf("%v", v),
					))
				}
			}

			result, err := next(ctx, request)

			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return result, err
			}

			if result != nil {
				span.SetAttributes(
					attribute.Bool("mcp.tool.result.is_error", result.IsError),
					attribute.Int("mcp.tool.result.content_count", len(result.Content)),
				)

				if contentJSON, jsonErr := json.Marshal(result.Content); jsonErr == nil {
					// Cap serialized content to avoid excessively large span attributes
					s := string(contentJSON)
					if len(s) > 4096 {
						s = s[:4096]
					}
					span.SetAttributes(attribute.String("mcp.tool.result.content", s))
				}

				if result.IsError {
					span.SetStatus(codes.Error, "tool returned error")
				}
			}

			return result, err
		}
	}
}

// initializeHooks creates a Hooks that records a span for the MCP initialize handshake.
func initializeHooks(transport string) *mcpserver.Hooks {
	tracer := otel.Tracer(tracerName)
	netTransport := networkTransport(transport)

	hooks := &mcpserver.Hooks{}
	hooks.AddAfterInitialize(func(ctx context.Context, id any, req *mcp.InitializeRequest, res *mcp.InitializeResult) {
		_, span := tracer.Start(ctx,
			"initialize",
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
			oteltrace.WithAttributes(
				attribute.String("mcp.method.name", "initialize"),
				attribute.String("mcp.transport", transport),
				attribute.String("network.transport", netTransport),
				attribute.String("network.protocol.version", "2.0"),
			),
		)
		defer span.End()

		if req != nil {
			if req.Params.ClientInfo.Name != "" {
				span.SetAttributes(attribute.String("mcp.client.name", req.Params.ClientInfo.Name))
			}
			if req.Params.ClientInfo.Version != "" {
				span.SetAttributes(attribute.String("mcp.client.version", req.Params.ClientInfo.Version))
			}
			if req.Params.ProtocolVersion != "" {
				span.SetAttributes(attribute.String("mcp.protocol.version", req.Params.ProtocolVersion))
			}
		}

		if res != nil {
			if res.ServerInfo.Name != "" {
				span.SetAttributes(attribute.String("mcp.server.name", res.ServerInfo.Name))
			}
			if res.ServerInfo.Version != "" {
				span.SetAttributes(attribute.String("mcp.server.version", res.ServerInfo.Version))
			}
			if res.ProtocolVersion != "" {
				span.SetAttributes(attribute.String("mcp.protocol.version", res.ProtocolVersion))
			}
		}

		if id != nil {
			span.SetAttributes(attribute.String("mcp.request.id", fmt.Sprintf("%v", id)))
		}
	})

	return hooks
}

func networkTransport(transport string) string {
	switch transport {
	case "stdio":
		return "pipe"
	case "http":
		return "tcp"
	default:
		return "pipe"
	}
}
