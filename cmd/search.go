package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/denysvitali/searxng-mcp/pkg/searxng"
	"github.com/spf13/cobra"
)

var (
	flagLimit     int
	flagTimeRange string
	flagCategory  string
	flagPage      int
)

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Perform a web search using Searxng",
	Long: `Perform a web search query and display results.

This command is useful for testing the Searxng integration and
searching the web directly from the command line.

Examples:
  # Basic search
  searxng-mcp search "golang tutorial"

  # Search with limit
  searxng-mcp search "golang tutorial" --limit 3

  # Search with time range
  searxng-mcp search "golang news" --time-range day

  # Search images
  searxng-mcp search "cats" --category images --limit 10`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

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

		// Build search request
		req := searxng.SearchRequest{
			Query:     query,
			Limit:     flagLimit,
			Page:      flagPage,
			TimeRange: flagTimeRange,
			Category:  flagCategory,
		}

		// Perform search
		ctx := context.Background()
		resp, err := client.Search(ctx, req)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		// Display results
		displayResults(resp)

		return nil
	},
}

func displayResults(resp *searxng.SearchResponse) {
	fmt.Printf("\nQuery: %s\n", resp.Query)
	fmt.Printf("Total results: %d\n\n", resp.NumberOfResults)

	if len(resp.Results) == 0 {
		fmt.Println("No results found.")
		return
	}

	// Use tabwriter for aligned output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	for i, result := range resp.Results {
		fmt.Fprintf(w, "%d. %s\n", i+1, result.Title)
		if result.Content != "" {
			// Truncate content to fit terminal
			content := result.Content
			if len(content) > 80 {
				content = content[:77] + "..."
			}
			fmt.Fprintf(w, "   %s\n", content)
		}
		fmt.Fprintf(w, "   %s\n\n", result.URL)
	}

	// Display suggestions if available
	if len(resp.Suggestions) > 0 {
		fmt.Printf("\nSuggestions: %s\n", resp.Suggestions)
	}

	// Display answers if available
	if len(resp.Answers) > 0 {
		fmt.Printf("\nAnswers:\n")
		for _, answer := range resp.Answers {
			fmt.Printf("  - %s\n", answer)
		}
	}

	// Display corrections if available
	if len(resp.Corrections) > 0 {
		fmt.Printf("\nDid you mean: %s?\n", resp.Corrections)
	}

	// Show pagination info
	resultsPerPage := flagLimit
	if resultsPerPage == 0 {
		resultsPerPage = 5
	}
	currentPage := flagPage
	if currentPage == 0 {
		currentPage = 1
	}

	if resp.NumberOfResults > resultsPerPage*currentPage {
		nextPage := currentPage + 1
		fmt.Printf("\n-- More results available (page %d) --\n", nextPage)
		fmt.Printf("Run: searxng-mcp search %s --page %d\n", strconv.Quote(resp.Query), nextPage)
	}
}

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().IntVarP(&flagLimit, "limit", "l", 5, "Number of results to return (1-20)")
	searchCmd.Flags().StringVar(&flagTimeRange, "time-range", "", "Time range filter: day, month, year")
	searchCmd.Flags().StringVar(&flagCategory, "category", "", "Search category: general, images, videos, etc.")
	searchCmd.Flags().IntVarP(&flagPage, "page", "p", 1, "Page number for pagination")
}
