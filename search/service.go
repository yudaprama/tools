package search

import (
	"context"
	"log"
	"os"
	"strings"
)

// Service provides search and crawl functionality
type Service struct {
	braveProvider      *BraveProvider
	duckduckgoProvider *DuckDuckGoProvider
	crawler            *Crawler
}

// NewService creates a new search service
func NewService() *Service {
	// Create search providers
	braveProvider := NewBraveProvider()
	duckduckgoProvider := NewDuckDuckGoProvider()

	// Get crawler implementations from environment
	crawlerImpls := getCrawlerImplsFromEnv()
	crawler := NewCrawler(crawlerImpls)

	return &Service{
		braveProvider:      braveProvider,
		duckduckgoProvider: duckduckgoProvider,
		crawler:            crawler,
	}
}

// Query performs a search query using Brave with DuckDuckGo fallback
func (s *Service) Query(query string, params *SearchParams) (*UniformSearchResponse, error) {
	ctx := context.Background()

	// Try Brave first
	response, err := s.braveProvider.Query(ctx, query, params)
	if err == nil {
		log.Printf("🔍 Search provider: Brave (query=%s, results=%d)", query, len(response.Results))
		return response, nil
	}

	// Log Brave failure and fallback
	log.Printf("⚠️  Brave search failed: %v. Falling back to DuckDuckGo...", err)

	// If Brave fails, fallback to DuckDuckGo
	response, err = s.duckduckgoProvider.Query(ctx, query, params)
	if err == nil {
		log.Printf("🔍 Search provider: DuckDuckGo (query=%s, results=%d)", query, len(response.Results))
		return response, nil
	}

	// Both providers failed
	log.Printf("❌ All search providers failed for query: %s", query)
	return nil, err
}

// WebSearch performs a web search with retry logic
func (s *Service) WebSearch(query SearchQuery) (*UniformSearchResponse, error) {
	params := &SearchParams{
		SearchCategories: query.SearchCategories,
		SearchEngines:    query.SearchEngines,
		SearchTimeRange:  query.SearchTimeRange,
	}

	// First attempt with all parameters
	data, err := s.Query(query.Query, params)
	if err != nil {
		return nil, err
	}

	// First retry: remove search engine restrictions if no results found
	if len(data.Results) == 0 && len(query.SearchEngines) > 0 {
		paramsExcludeSearchEngines := &SearchParams{
			SearchCategories: query.SearchCategories,
			SearchEngines:    nil,
			SearchTimeRange:  query.SearchTimeRange,
		}
		data, err = s.Query(query.Query, paramsExcludeSearchEngines)
		if err != nil {
			return nil, err
		}
	}

	// Second retry: remove all restrictions if still no results found
	if len(data.Results) == 0 {
		data, err = s.Query(query.Query, nil)
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}

// CrawlPages crawls multiple URLs concurrently
func (s *Service) CrawlPages(req CrawlPagesRequest) (*CrawlPagesResponse, error) {
	results := s.crawler.CrawlPages(req.URLs, req.Impls)
	return &CrawlPagesResponse{Results: results}, nil
}

// getCrawlerImplsFromEnv reads crawler implementations from environment
func getCrawlerImplsFromEnv() []CrawlImplType {
	envStr := os.Getenv("CRAWLER_IMPLS")
	if envStr == "" {
		// Default: try Naive only
		return []CrawlImplType{CrawlImplNaive}
	}

	// Parse comma-separated list
	implStrs := strings.Split(strings.ReplaceAll(envStr, "，", ","), ",")
	var impls []CrawlImplType
	for _, s := range implStrs {
		s = strings.TrimSpace(s)
		if s != "" {
			impls = append(impls, CrawlImplType(s))
		}
	}

	if len(impls) == 0 {
		return []CrawlImplType{CrawlImplNaive}
	}

	return impls
}
