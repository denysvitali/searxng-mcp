package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/denysvitali/searxng-mcp/internal/log"
	"github.com/spf13/cobra"
)

var (
	// Flags
	flagInstanceURL string
	flagLogLevel    string
	flagTimeout     time.Duration

	// Config values that will be used by subcommands
	instanceURL string
	logLevel    string
	timeout     time.Duration
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "searxng-mcp",
	Short: "A Model Context Protocol server for Searxng web search",
	Long: `Searxng MCP Server - A Model Context Protocol server that enables
AI agents to search and navigate the web using Searxng instances.

This server provides two main tools:
  - web_search: Search the web and return limited results
  - web_read: Fetch and read content from URLs, converting HTML to Markdown`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize logger
		log.Init(flagLogLevel)

		// Set config values from flags
		instanceURL = flagInstanceURL
		logLevel = flagLogLevel
		timeout = flagTimeout

		if instanceURL == "" {
			return fmt.Errorf("instance URL cannot be empty")
		}

		if timeout == 0 {
			timeout = 30 * time.Second
		}

		log.WithField("instance_url", instanceURL).Debug("using searxng instance")
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagInstanceURL, "instance-url", "", "Searxng instance URL")
	rootCmd.MarkPersistentFlagRequired("instance-url")
	rootCmd.PersistentFlags().StringVar(&flagLogLevel, "log-level", "info", "Log level: debug, info, warn, error")
	rootCmd.PersistentFlags().DurationVar(&flagTimeout, "timeout", 30*time.Second, "Request timeout")
}
