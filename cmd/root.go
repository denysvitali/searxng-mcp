package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/denysvitali/searxng-mcp/internal/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Flags
	flagInstanceURL string
	flagLogLevel    string
	flagTimeout     time.Duration

	// Config values that will be used by subcommands
	instanceURL string
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
		log.Init(viper.GetString("log-level"))

		// Set config values from viper (merges flags, env, config file)
		instanceURL = viper.GetString("instance-url")
		timeout = viper.GetDuration("timeout")

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
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&flagInstanceURL, "instance-url", "", "Searxng instance URL")
	rootCmd.PersistentFlags().StringVar(&flagLogLevel, "log-level", "info", "Log level: debug, info, warn, error")
	rootCmd.PersistentFlags().DurationVar(&flagTimeout, "timeout", 30*time.Second, "Request timeout")

	// Bind flags to viper
	_ = viper.BindPFlag("instance-url", rootCmd.PersistentFlags().Lookup("instance-url"))
	_ = viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
	_ = viper.BindPFlag("timeout", rootCmd.PersistentFlags().Lookup("timeout"))

	// Bind environment variables (legacy support)
	_ = viper.BindEnv("instance-url", "SEARXNG_URL")
	_ = viper.BindEnv("timeout", "SEARXNG_TIMEOUT")
	_ = viper.BindEnv("log-level", "LOG_LEVEL")

	// Tracing env vars — these are read directly by the tracing package,
	// but we also bind them so they can be set in the config file.
	_ = viper.BindEnv("sentry-dsn", "SENTRY_DSN")
	_ = viper.BindEnv("sentry-traces-sample-rate", "SENTRY_TRACES_SAMPLE_RATE")
	_ = viper.BindEnv("otel-exporter-otlp-endpoint", "OTEL_EXPORTER_OTLP_ENDPOINT")
	_ = viper.BindEnv("otel-exporter-otlp-headers", "OTEL_EXPORTER_OTLP_HEADERS")
}

func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.config/searxng-mcp")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "warning: error reading config file: %v\n", err)
		}
	}

	// Export tracing config keys back to env vars so the tracing package
	// (which reads os.Getenv) picks them up from the config file.
	exportToEnv("sentry-dsn", "SENTRY_DSN")
	exportToEnv("sentry-traces-sample-rate", "SENTRY_TRACES_SAMPLE_RATE")
	exportToEnv("otel-exporter-otlp-endpoint", "OTEL_EXPORTER_OTLP_ENDPOINT")
	exportToEnv("otel-exporter-otlp-headers", "OTEL_EXPORTER_OTLP_HEADERS")
}

// exportToEnv sets an environment variable from a viper key if the env var
// is not already set and the viper key has a value.
func exportToEnv(viperKey, envKey string) {
	if os.Getenv(envKey) != "" {
		return // env var already set, don't override
	}
	if v := viper.GetString(viperKey); v != "" {
		os.Setenv(envKey, v)
	}
}
