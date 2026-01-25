package searxng

import "time"

// SearchRequest represents a search request to Searxng
type SearchRequest struct {
	Query     string   // Search query
	Limit     int      // Default: 5, Max: 20
	Page      int      // Default: 1
	TimeRange string   // "day", "month", "year"
	Category  string   // "general", "images", "videos", etc.
	Language  string   // Language code (e.g., "en", "fr")
	Engines   []string // Specific engines to use
}

// APIRequest is the API request format (exported for testing)
type APIRequest struct {
	Query     string   `json:"q"`
	Category  string   `json:"category,omitempty"`
	Engines   []string `json:"engines,omitempty"`
	Language  string   `json:"language,omitempty"`
	Pageno    int      `json:"pageno,omitempty"`
	TimeRange string   `json:"time_range,omitempty"`
	Format    string   `json:"format"`
}

// SearchResult represents a single search result from Searxng
type SearchResult struct {
	URL           string
	Title         string
	Content       string
	PublishedDate *time.Time
	Engine        string
	Category      string
	Score         float64
	Thumbnail     string
	ImageSrc      string
	Engines       []string
	Positions     []int
}

// APIResult is the API result format (exported for testing)
type APIResult struct {
	URL           string   `json:"url"`
	Title         string   `json:"title"`
	Content       string   `json:"content"`
	PublishedDate string   `json:"publishedDate,omitempty"`
	Engine        string   `json:"engine,omitempty"`
	Category      string   `json:"category,omitempty"`
	Score         float64  `json:"score,omitempty"`
	Thumbnail     string   `json:"thumbnail,omitempty"`
	ImgSrc        string   `json:"img_src,omitempty"`
	Engines       []string `json:"engines,omitempty"`
	Positions     []int    `json:"positions,omitempty"`
}

// Infobox represents an infobox result from Searxng
type Infobox struct {
	Content       string                `json:"content"`
	Engine        string                `json:"engine"`
	Attribution   string                `json:"attribution"`
	Images        []InfoboxImage        `json:"images"`
	Label         string                `json:"label"`
	RelatedTopics []InfoboxRelatedTopic `json:"relatedTopics"`
	Urls          []InfoboxURL          `json:"urls"`
}

// InfoboxImage represents an image in an infobox
type InfoboxImage struct {
	URL          string `json:"url"`
	Alt          string `json:"alt"`
	ThumbnailURL string `json:"thumbnailUrl,omitempty"`
}

// InfoboxRelatedTopic represents a related topic in an infobox
type InfoboxRelatedTopic struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// InfoboxURL represents a URL in an infobox
type InfoboxURL struct {
	URL      string `json:"url"`
	Title    string `json:"title"`
	Engine   string `json:"engine,omitempty"`
	Category string `json:"category,omitempty"`
	ImgSrc   string `json:"img_src,omitempty"`
}

// UnresponsiveEngine represents an engine that failed to respond
type UnresponsiveEngine struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}

// SearchResponse represents the complete search response from Searxng
type SearchResponse struct {
	Query               string
	NumberOfResults     int
	Results             []SearchResult
	Answers             []string
	Corrections         []string
	Infoboxes           []Infobox
	Suggestions         []string
	UnresponsiveEngines []UnresponsiveEngine
}

// APIResponse is the API response format (exported for testing)
type APIResponse struct {
	Query               string               `json:"query"`
	NumberOfResults     int                  `json:"number_of_results"`
	Results             []APIResult          `json:"results"`
	Answers             []string             `json:"answers"`
	Corrections         []string             `json:"corrections"`
	Infoboxes           []Infobox            `json:"infoboxes"`
	Suggestions         []string             `json:"suggestions"`
	UnresponsiveEngines []UnresponsiveEngine `json:"unresponsive_engines"`
}

// parsePublishedDate parses a published date string
func parsePublishedDate(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}

	// Try common date formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return &t
		}
	}

	return nil
}

// toSearchResult converts an API result to a SearchResult
func toSearchResult(r APIResult) SearchResult {
	return SearchResult{
		URL:           r.URL,
		Title:         r.Title,
		Content:       r.Content,
		PublishedDate: parsePublishedDate(r.PublishedDate),
		Engine:        r.Engine,
		Category:      r.Category,
		Score:         r.Score,
		Thumbnail:     r.Thumbnail,
		ImageSrc:      r.ImgSrc,
		Engines:       r.Engines,
		Positions:     r.Positions,
	}
}

// toSearchResponse converts an API response to a SearchResponse
func toSearchResponse(r APIResponse) SearchResponse {
	results := make([]SearchResult, len(r.Results))
	for i, result := range r.Results {
		results[i] = toSearchResult(result)
	}

	return SearchResponse{
		Query:               r.Query,
		NumberOfResults:     r.NumberOfResults,
		Results:             results,
		Answers:             r.Answers,
		Corrections:         r.Corrections,
		Infoboxes:           r.Infoboxes,
		Suggestions:         r.Suggestions,
		UnresponsiveEngines: r.UnresponsiveEngines,
	}
}
