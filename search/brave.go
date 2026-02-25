package search

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// BraveProvider implements the Brave search provider
type BraveProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// BraveWebResult represents a single web search result from Brave
type BraveWebResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// BraveResponse represents the response from Brave Search API
type BraveResponse struct {
	Web struct {
		Results []BraveWebResult `json:"results"`
	} `json:"web"`
}

// NewBraveProvider creates a new Brave search provider
func NewBraveProvider() *BraveProvider {
	apiKey := "BSApaMPDAzviYP-OUpmwkMSSWUNgJBo"

	return &BraveProvider{
		apiKey:  apiKey,
		baseURL: "https://api.search.brave.com/res/v1",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the provider name
func (p *BraveProvider) Name() string {
	return "brave"
}

// Query performs a search query using Brave Search API
func (p *BraveProvider) Query(ctx context.Context, query string, params *SearchParams) (*UniformSearchResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("BRAVE_SEARCH_API_KEY environment variable not set")
	}

	endpoint := fmt.Sprintf("%s/web/search", p.baseURL)

	// Build query parameters
	queryParams := url.Values{}
	queryParams.Set("q", query)
	queryParams.Set("count", "15")
	queryParams.Set("result_filter", "web")

	// Add time range if specified
	if params != nil && params.SearchTimeRange != "" && params.SearchTimeRange != "anytime" {
		freshness := mapTimeRange(params.SearchTimeRange)
		if freshness != "" {
			queryParams.Set("freshness", freshness)
		}
	}

	fullURL := fmt.Sprintf("%s?%s", endpoint, queryParams.Encode())

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("X-Subscription-Token", p.apiKey)

	// Execute request
	startTime := time.Now()
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	costTime := time.Since(startTime).Milliseconds()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("brave API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Handle gzip encoding
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer func() { _ = gzReader.Close() }()
		reader = gzReader
	}

	// Parse response
	var braveResp BraveResponse
	if err := json.NewDecoder(reader).Decode(&braveResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to uniform format
	results := make([]UniformSearchResult, 0, len(braveResp.Web.Results))
	for _, result := range braveResp.Web.Results {
		parsedURL := ""
		if u, err := url.Parse(result.URL); err == nil {
			parsedURL = u.Hostname()
		}

		results = append(results, UniformSearchResult{
			Category:  "general",
			Content:   result.Description,
			Engines:   []string{"brave"},
			ParsedUrl: parsedURL,
			Score:     1.0,
			Title:     result.Title,
			URL:       result.URL,
		})
	}

	return &UniformSearchResponse{
		CostTime:      costTime,
		Query:         query,
		ResultNumbers: len(results),
		Results:       results,
	}, nil
}

// mapTimeRange maps frontend time range to Brave API freshness parameter
func mapTimeRange(timeRange string) string {
	mapping := map[string]string{
		"day":   "pd", // past day
		"week":  "pw", // past week
		"month": "pm", // past month
		"year":  "py", // past year
	}
	return mapping[timeRange]
}
