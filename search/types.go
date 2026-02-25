package search

import (
	"github.com/getkawai/tools/htmltomarkdown"
)

// SearchParams represents optional search parameters
type SearchParams struct {
	SearchCategories []string `json:"searchCategories,omitempty"`
	SearchEngines    []string `json:"searchEngines,omitempty"`
	SearchTimeRange  string   `json:"searchTimeRange,omitempty"`
}

// UniformSearchResult represents a single search result
type UniformSearchResult struct {
	Category      string   `json:"category,omitempty"`
	Content       string   `json:"content"`
	Engines       []string `json:"engines"`
	IframeSrc     string   `json:"iframeSrc,omitempty"`
	ImgSrc        string   `json:"imgSrc,omitempty"`
	ParsedUrl     string   `json:"parsedUrl"`
	PublishedDate string   `json:"publishedDate,omitempty"`
	Score         float64  `json:"score"`
	Thumbnail     string   `json:"thumbnail,omitempty"`
	Title         string   `json:"title"`
	URL           string   `json:"url"`
}

// UniformSearchResponse represents the response from a search query
type UniformSearchResponse struct {
	CostTime      int64                 `json:"costTime"` // milliseconds
	Query         string                `json:"query"`
	ResultNumbers int                   `json:"resultNumbers"`
	Results       []UniformSearchResult `json:"results"`
}

// SearchQuery combines query string with search parameters
type SearchQuery struct {
	Query            string   `json:"query"`
	SearchCategories []string `json:"searchCategories,omitempty"`
	SearchEngines    []string `json:"searchEngines,omitempty"`
	SearchTimeRange  string   `json:"searchTimeRange,omitempty"`
}

// CrawlImplType represents the type of crawler implementation
type CrawlImplType string

const (
	CrawlImplNaive       CrawlImplType = "naive"
	CrawlImplBrowserless CrawlImplType = "browserless"
)

// CrawlSuccessResult is now imported from pkg/htmltomarkdown
type CrawlSuccessResult = htmltomarkdown.CrawlSuccessResult

// CrawlErrorResult represents a failed crawl result
type CrawlErrorResult struct {
	ErrorMessage string `json:"errorMessage"`
	ErrorType    string `json:"errorType"`
	URL          string `json:"url"`
}

// CrawlResult is a union type for crawl results
type CrawlResult struct {
	Success *CrawlSuccessResult `json:"success,omitempty"`
	Error   *CrawlErrorResult   `json:"error,omitempty"`
}

// CrawlPagesRequest represents a request to crawl multiple pages
type CrawlPagesRequest struct {
	URLs  []string        `json:"urls"`
	Impls []CrawlImplType `json:"impls,omitempty"`
}

// CrawlPagesResponse represents the response from crawling multiple pages
type CrawlPagesResponse struct {
	Results []CrawlResult `json:"results"`
}
