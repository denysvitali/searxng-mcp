package searxng

import "time"

// DefaultInstanceURL is the default Searxng instance URL
const DefaultInstanceURL = "https://searxng.example.com"

// Config holds the configuration for the Searxng client
type Config struct {
	// BaseURL is the base URL of the Searxng instance
	BaseURL string

	// Timeout is the HTTP request timeout
	Timeout time.Duration

	// MaxRetries is the maximum number of retries for failed requests
	MaxRetries int

	// UserAgent is the HTTP User-Agent header value
	UserAgent string
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		BaseURL:    DefaultInstanceURL,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		UserAgent:  "searxng-mcp/1.0",
	}
}
