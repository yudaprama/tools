package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// DuckDuckGoProvider implements the DuckDuckGo search provider
type DuckDuckGoProvider struct {
	baseURL    string
	httpClient *http.Client
}

// DuckDuckGoRelatedTopic represents a related topic in DuckDuckGo response
type DuckDuckGoRelatedTopic struct {
	FirstURL string `json:"FirstURL"`
	Text     string `json:"Text"`
	Result   string `json:"Result"`
	Icon     struct {
		URL string `json:"URL"`
	} `json:"Icon"`
}

// DuckDuckGoTopicGroup represents a group of related topics
type DuckDuckGoTopicGroup struct {
	Name   string                   `json:"Name"`
	Topics []DuckDuckGoRelatedTopic `json:"Topics"`
}

// DuckDuckGoResponse represents the response from DuckDuckGo Instant Answer API
type DuckDuckGoResponse struct {
	Abstract       string                   `json:"Abstract"`
	AbstractText   string                   `json:"AbstractText"`
	AbstractSource string                   `json:"AbstractSource"`
	AbstractURL    string                   `json:"AbstractURL"`
	Heading        string                   `json:"Heading"`
	RelatedTopics  []json.RawMessage        `json:"RelatedTopics"`
	Results        []DuckDuckGoRelatedTopic `json:"Results"`
}

// NewDuckDuckGoProvider creates a new DuckDuckGo search provider
func NewDuckDuckGoProvider() *DuckDuckGoProvider {
	return &DuckDuckGoProvider{
		baseURL: "https://api.duckduckgo.com",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the provider name
func (p *DuckDuckGoProvider) Name() string {
	return "duckduckgo"
}

// Query performs a search query using DuckDuckGo Instant Answer API
func (p *DuckDuckGoProvider) Query(ctx context.Context, query string, params *SearchParams) (*UniformSearchResponse, error) {
	endpoint := p.baseURL

	// Build query parameters
	queryParams := url.Values{}
	queryParams.Set("q", query)
	queryParams.Set("format", "json")
	queryParams.Set("no_html", "1")
	queryParams.Set("skip_disambig", "1")

	fullURL := fmt.Sprintf("%s?%s", endpoint, queryParams.Encode())

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Veridium/1.0")

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
		return nil, fmt.Errorf("duckduckgo API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var ddgResp DuckDuckGoResponse
	if err := json.NewDecoder(resp.Body).Decode(&ddgResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to uniform format
	results := make([]UniformSearchResult, 0)

	// Add abstract as first result if available
	if ddgResp.AbstractText != "" && ddgResp.AbstractURL != "" {
		parsedURL := ""
		if u, err := url.Parse(ddgResp.AbstractURL); err == nil {
			parsedURL = u.Hostname()
		}

		results = append(results, UniformSearchResult{
			Category:  "general",
			Content:   ddgResp.AbstractText,
			Engines:   []string{"duckduckgo"},
			ParsedUrl: parsedURL,
			Score:     1.0,
			Title:     ddgResp.Heading,
			URL:       ddgResp.AbstractURL,
		})
	}

	// Process RelatedTopics
	for _, topicRaw := range ddgResp.RelatedTopics {
		// Try to parse as a single topic
		var topic DuckDuckGoRelatedTopic
		if err := json.Unmarshal(topicRaw, &topic); err == nil && topic.FirstURL != "" {
			parsedURL := ""
			if u, err := url.Parse(topic.FirstURL); err == nil {
				parsedURL = u.Hostname()
			}

			// Extract title from Text (usually format: "Title Description")
			title := topic.Text
			content := topic.Text
			if len(title) > 100 {
				title = title[:100] + "..."
			}

			results = append(results, UniformSearchResult{
				Category:  "general",
				Content:   content,
				Engines:   []string{"duckduckgo"},
				ParsedUrl: parsedURL,
				Score:     0.8,
				Title:     title,
				URL:       topic.FirstURL,
			})
			continue
		}

		// Try to parse as a topic group
		var topicGroup DuckDuckGoTopicGroup
		if err := json.Unmarshal(topicRaw, &topicGroup); err == nil && len(topicGroup.Topics) > 0 {
			for _, groupTopic := range topicGroup.Topics {
				if groupTopic.FirstURL == "" {
					continue
				}

				parsedURL := ""
				if u, err := url.Parse(groupTopic.FirstURL); err == nil {
					parsedURL = u.Hostname()
				}

				title := groupTopic.Text
				content := groupTopic.Text
				if len(title) > 100 {
					title = title[:100] + "..."
				}

				results = append(results, UniformSearchResult{
					Category:  "general",
					Content:   content,
					Engines:   []string{"duckduckgo"},
					ParsedUrl: parsedURL,
					Score:     0.7,
					Title:     title,
					URL:       groupTopic.FirstURL,
				})
			}
		}
	}

	// Limit to 15 results to match Brave
	if len(results) > 15 {
		results = results[:15]
	}

	return &UniformSearchResponse{
		CostTime:      costTime,
		Query:         query,
		ResultNumbers: len(results),
		Results:       results,
	}, nil
}
