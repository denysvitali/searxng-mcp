package cmd

import (
	"fmt"

	"github.com/denysvitali/searxng-mcp/internal/log"
	"github.com/denysvitali/searxng-mcp/pkg/server"
	"github.com/denysvitali/searxng-mcp/pkg/searxng"
	"github.com/spf13/cobra"
)

var (
	flagTransport string
	flagPort      int
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server",
	Long: `Start the Model Context Protocol server for Searxng.

By default, the server runs in stdio mode, which is suitable for
integration with MCP clients like Claude Code, Cursor, etc.

To run in HTTP mode (useful for development):
  searxng-mcp serve --transport http --port 8080

Examples:
  # Start in stdio mode (default)
  searxng-mcp serve

  # Start in HTTP mode
  searxng-mcp serve --transport http --port 8080`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if flagTransport != "stdio" && flagTransport != "http" {
			return fmt.Errorf("invalid transport: %s (must be 'stdio' or 'http')", flagTransport)
		}
		if flagTransport == "http" && (flagPort < 1 || flagPort > 65535) {
			return fmt.Errorf("invalid port: %d", flagPort)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create Searxng client config
		config := &searxng.Config{
			BaseURL: instanceURL,
			Timeout: timeout,
		}

		// Create Searxng client
		client, err := searxng.NewClient(config)
		if err != nil {
			return fmt.Errorf("failed to create searxng client: %w", err)
		}

		log.WithField("transport", flagTransport).Info("starting MCP server")

		// Create and start server
		srv := server.New(client)

		switch flagTransport {
		case "http":
			addr := fmt.Sprintf(":%d", flagPort)
			log.WithField("address", addr).Info("listening")
			return srv.ServeHTTP(addr)

		default: // stdio
			return srv.ServeStdio()
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringVarP(&flagTransport, "transport", "t", "stdio", "Transport type: stdio or http")
	serveCmd.Flags().IntVarP(&flagPort, "port", "p", 8080, "Port for HTTP transport")
}
