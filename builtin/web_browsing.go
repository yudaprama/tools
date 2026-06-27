package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/yudaprama/tools/search"
)

// ============================================================================
// Input Types for unillm.NewAgentTool
// ============================================================================

// WebSearchInput defines input for web search tool
type WebBrowsingSearchInput struct {
	Query            string   `json:"query" jsonschema:"description=The search query string"`
	SearchCategories []string `json:"searchCategories,omitempty" jsonschema:"description=Search categories: general&#44; images&#44; news&#44; science&#44; videos"`
	SearchEngines    []string `json:"searchEngines,omitempty" jsonschema:"description=Search engines: google&#44; bing&#44; duckduckgo&#44; brave&#44; wikipedia&#44; github&#44; arxiv"`
	SearchTimeRange  string   `json:"searchTimeRange,omitempty" jsonschema:"description=Time range filter: anytime&#44; day&#44; week&#44; month&#44; year"`
}

// CrawlSingleInput defines input for crawl single page tool
type CrawlSingleInput struct {
	URL string `json:"url" jsonschema:"description=The URL of the webpage to crawl"`
}

// CrawlMultiInput defines input for crawl multiple pages tool
type CrawlMultiInput struct {
	URLs []string `json:"urls" jsonschema:"description=The URLs of the webpages to crawl"`
}

// WebBrowsingService wraps search.Service to provide lobe-web-browsing compatible responses
type WebBrowsingService struct {
	searchService *search.Service
}

// NewWebBrowsingService creates a new web browsing service
func NewWebBrowsingService() *WebBrowsingService {
	return &WebBrowsingService{
		searchService: search.NewService(),
	}
}

// ============================================================================
// Response Types (matching frontend expected format)
// ============================================================================

// Note: UniformSearchResult and UniformSearchResponse are imported from internal/search
// to avoid duplication. They match the frontend UniformSearchResult interface.

// CrawlData matches frontend expected crawl data format
type CrawlData struct {
	Content     string `json:"content"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// CrawlResult matches frontend CrawlResult interface
type CrawlResult struct {
	OriginalUrl string    `json:"originalUrl"`
	Crawler     string    `json:"crawler"`
	Data        CrawlData `json:"data"`
}

// CrawlPluginState matches frontend CrawlPluginState interface
type CrawlPluginState struct {
	Results []CrawlResult `json:"results"`
}

// ============================================================================
// Service Methods
// ============================================================================

// Search performs web search and returns frontend-compatible response
func (s *WebBrowsingService) Search(query string, categories, engines []string, timeRange string) (*search.UniformSearchResponse, error) {
	startTime := time.Now()

	searchQuery := search.SearchQuery{
		Query:            query,
		SearchCategories: categories,
		SearchEngines:    engines,
		SearchTimeRange:  timeRange,
	}

	response, err := s.searchService.WebSearch(searchQuery)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// No transformation needed - response is already in the correct format
	// Just update the cost time
	response.CostTime = time.Since(startTime).Milliseconds()

	return response, nil
}

// CrawlSinglePage crawls a single URL and returns frontend-compatible response
func (s *WebBrowsingService) CrawlSinglePage(url string) (*CrawlPluginState, error) {
	return s.CrawlMultiPages([]string{url})
}

// CrawlMultiPages crawls multiple URLs and returns frontend-compatible response
func (s *WebBrowsingService) CrawlMultiPages(urls []string) (*CrawlPluginState, error) {
	response, err := s.searchService.CrawlPages(search.CrawlPagesRequest{
		URLs: urls,
	})
	if err != nil {
		return nil, fmt.Errorf("crawl failed: %w", err)
	}

	// Transform to frontend format
	results := make([]CrawlResult, 0, len(response.Results))
	for i, r := range response.Results {
		originalUrl := ""
		if i < len(urls) {
			originalUrl = urls[i]
		}

		if r.Error != nil {
			// Skip error results - don't include them in tool result
			log.Printf("⚠️ Skipping failed crawl result for URL: %s (Error: %s)", originalUrl, r.Error.ErrorMessage)
			continue
		} else if r.Success != nil {
			// Use crawler label from result: "jina" for Jina, "kawai" for naive
			crawlerLabel := r.Success.Crawler
			if crawlerLabel == "" {
				crawlerLabel = "kawai"
			}
			results = append(results, CrawlResult{
				OriginalUrl: originalUrl,
				Crawler:     crawlerLabel,
				Data: CrawlData{
					Content:     r.Success.Content,
					URL:         r.Success.URL,
					Title:       r.Success.Title,
					Description: "",
				},
			})
		}
	}

	return &CrawlPluginState{
		Results: results,
	}, nil
}

// ============================================================================
// Tool Registration
// ============================================================================

// NewWebBrowsing creates the lobe-web-browsing tools (search, crawlSinglePage, crawlMultiPages).
func NewWebBrowsing(_ context.Context) ([]tool.InvokableTool, error) {
	service := NewWebBrowsingService()

	searchTool, err := utils.InferTool("lobe-web-browsing__search",
		"Search the web for information. Returns a list of search results with title, content, and URL.",
		func(ctx context.Context, input *WebBrowsingSearchInput) (string, error) {
			if input.Query == "" {
				return "", fmt.Errorf("query is required")
			}

			timeRange := input.SearchTimeRange
			if timeRange == "" {
				timeRange = "anytime"
			}

			response, err := service.Search(input.Query, input.SearchCategories, input.SearchEngines, timeRange)
			if err != nil {
				return "", err
			}

			resultJSON, err := json.Marshal(response)
			if err != nil {
				return "", fmt.Errorf("failed to marshal response: %v", err)
			}

			log.Printf("🔍 Web search completed: query=%s, results=%d", input.Query, len(response.Results))
			return string(resultJSON), nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to infer search tool: %w", err)
	}

	crawlSingleTool, err := utils.InferTool("lobe-web-browsing__crawlSinglePage",
		"Retrieve content from a specific webpage. Returns the page title, content, URL and website.",
		func(ctx context.Context, input *CrawlSingleInput) (string, error) {
			if input.URL == "" {
				return "", fmt.Errorf("url is required")
			}

			response, err := service.CrawlSinglePage(input.URL)
			if err != nil {
				return "", err
			}

			resultJSON, err := json.Marshal(response)
			if err != nil {
				return "", fmt.Errorf("failed to marshal response: %v", err)
			}

			log.Printf("🌐 Crawled single page: url=%s", input.URL)
			return string(resultJSON), nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to infer crawlSinglePage tool: %w", err)
	}

	crawlMultiTool, err := utils.InferTool("lobe-web-browsing__crawlMultiPages",
		"Retrieve content from multiple webpages simultaneously. Returns an array of page results.",
		func(ctx context.Context, input *CrawlMultiInput) (string, error) {
			if len(input.URLs) == 0 {
				return "", fmt.Errorf("at least one URL is required")
			}

			response, err := service.CrawlMultiPages(input.URLs)
			if err != nil {
				return "", err
			}

			resultJSON, err := json.Marshal(response)
			if err != nil {
				return "", fmt.Errorf("failed to marshal response: %v", err)
			}

			log.Printf("🌐 Crawled %d pages", len(input.URLs))
			return string(resultJSON), nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to infer crawlMultiPages tool: %w", err)
	}

	return []tool.InvokableTool{searchTool, crawlSingleTool, crawlMultiTool}, nil
}
