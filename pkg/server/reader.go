package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/PuerkitoBio/goquery"
	"github.com/denysvitali/searxng-mcp/internal/log"
)

const (
	defaultUserAgent     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	defaultAccept        = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
	defaultAcceptLang    = "en-US,en;q=0.9"
	defaultHTTPTimeout   = 30 * time.Second
	maxHTTPRedirectCount = 10
)

var supportedSchemes = []string{"http", "https"}

// fetchURLContent fetches content from a URL and converts it to Markdown.
func fetchURLContent(ctx context.Context, urlStr string) (string, error) {
	parsedURL, err := validateURL(urlStr)
	if err != nil {
		return "", err
	}

	log.WithField("url", urlStr).Debug("fetching URL")

	client := newHTTPClient()
	if isRedditThreadURL(parsedURL) {
		return fetchRedditContentAsMarkdown(ctx, client, parsedURL)
	}
	if isGitHubIssueOrPRURL(parsedURL) {
		return fetchGitHubContentAsMarkdown(ctx, client, parsedURL)
	}
	if isGitHubRepoURL(parsedURL) {
		return fetchGitHubRepoAsMarkdown(ctx, client, parsedURL)
	}

	return fetchGenericHTMLAsMarkdown(ctx, client, parsedURL.String())
}

func validateURL(urlStr string) (*url.URL, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if !slices.Contains(supportedSchemes, parsedURL.Scheme) {
		return nil, fmt.Errorf("unsupported URL scheme: %s (only http and https are supported)", parsedURL.Scheme)
	}
	return parsedURL, nil
}

func newHTTPClient() *http.Client {
	client := &http.Client{
		Timeout: defaultHTTPTimeout,
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxHTTPRedirectCount {
			return fmt.Errorf("too many redirects")
		}
		return nil
	}
	return client
}

func newRequest(ctx context.Context, urlStr, accept string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept-Language", defaultAcceptLang)
	if accept != "" {
		req.Header.Set("Accept", accept)
	} else {
		req.Header.Set("Accept", defaultAccept)
	}
	return req, nil
}

func fetchGenericHTMLAsMarkdown(ctx context.Context, client *http.Client, urlStr string) (string, error) {
	req, err := newRequest(ctx, urlStr, defaultAccept)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") && !strings.Contains(contentType, "application/xhtml") {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read response body: %w", err)
		}
		return string(body), nil
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}
	doc.Find("script, style, nav, footer, header, aside").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	html, err := doc.Html()
	if err != nil {
		return "", fmt.Errorf("failed to serialize HTML: %w", err)
	}

	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
		),
	)
	markdown, err := conv.ConvertString(html)
	if err != nil {
		return "", fmt.Errorf("failed to convert to Markdown: %w", err)
	}

	return cleanMarkdown(markdown), nil
}

func pathSegments(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, "/")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		segments = append(segments, part)
	}
	return segments
}

// cleanMarkdown cleans up the converted markdown
func cleanMarkdown(markdown string) string {
	lines := strings.Split(markdown, "\n")
	var cleaned []string

	// Remove excessive empty lines
	emptyCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			emptyCount++
			if emptyCount <= 2 {
				cleaned = append(cleaned, "")
			}
		} else {
			emptyCount = 0
			cleaned = append(cleaned, trimmed)
		}
	}

	// Trim leading and trailing empty lines
	for len(cleaned) > 0 && cleaned[0] == "" {
		cleaned = cleaned[1:]
	}
	for len(cleaned) > 0 && cleaned[len(cleaned)-1] == "" {
		cleaned = cleaned[:len(cleaned)-1]
	}

	return strings.Join(cleaned, "\n")
}
