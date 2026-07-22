package search

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/yudaprama/tools/htmltomarkdown"
	"github.com/yudaprama/tools/htmltomarkdown/converter"
)

// Crawler handles web page crawling
type Crawler struct {
	httpClient *http.Client
	impls      []CrawlImplType
}

// NewCrawler creates a new crawler instance
func NewCrawler(impls []CrawlImplType) *Crawler {
	if len(impls) == 0 {
		// Default: try Naive only
		impls = []CrawlImplType{CrawlImplNaive}
	}

	return &Crawler{
		httpClient: &http.Client{
			Timeout: 45 * time.Second, // Increased timeout for slow sites
			Transport: &http.Transport{
				MaxIdleConns:          10,
				IdleConnTimeout:       30 * time.Second,
				DisableCompression:    false, // Let Go auto-decompress gzip/deflate
				DisableKeepAlives:     false,
				MaxConnsPerHost:       5,
				ResponseHeaderTimeout: 30 * time.Second,
				ForceAttemptHTTP2:     false, // Stick to HTTP/1.1 for better compatibility
			},
		},
		impls: impls,
	}
}

// CrawlPages crawls multiple URLs concurrently
func (c *Crawler) CrawlPages(urls []string, impls []CrawlImplType) []CrawlResult {
	if len(impls) == 0 {
		impls = c.impls
	}

	results := make([]CrawlResult, len(urls))
	var wg sync.WaitGroup

	// Limit concurrency to 3
	semaphore := make(chan struct{}, 3)

	for i, urlStr := range urls {
		wg.Add(1)
		go func(idx int, u string) {
			defer wg.Done() // ✅ MUST be first defer to ensure it always executes
			defer func() {
				if r := recover(); r != nil {
					slog.Error("crawler panic recovered", "url", u, "recover", r)
					results[idx] = CrawlResult{
						Success: &htmltomarkdown.CrawlSuccessResult{
							URL:     u,
							Content: fmt.Sprintf("panic: %v", r),
						},
						Error: &CrawlErrorResult{
							ErrorMessage: fmt.Sprintf("panic: %v", r),
							ErrorType:    "PANIC",
							URL:          u,
						},
					}
				}
			}()

			semaphore <- struct{}{}        // acquire
			defer func() { <-semaphore }() // release

			results[idx] = c.crawlSingle(u, impls)
		}(i, urlStr)
	}

	wg.Wait()
	return results
}

// crawlSingle crawls a single URL with retry logic
func (c *Crawler) crawlSingle(urlStr string, impls []CrawlImplType) CrawlResult {
	// We only have one implementation now
	result, err := c.crawlNaive(urlStr)
	if err == nil {
		return result
	}

	slog.Warn("all crawlers failed", "url", urlStr, "err", err)

	return CrawlResult{
		Error: &CrawlErrorResult{
			ErrorMessage: fmt.Sprintf("crawler failed: %v", err),
			ErrorType:    "CRAWLER_FAILED",
			URL:          urlStr,
		},
	}
}

// crawlNaive performs naive HTTP crawling using htmltomarkdown.ConvertURL
func (c *Crawler) crawlNaive(urlStr string) (CrawlResult, error) {
	slog.Info("crawling URL", "url", urlStr)

	result, err := htmltomarkdown.ConvertURL(urlStr, c.httpClient,
		converter.WithCrawler("kawai"),
		converter.WithMainContentOnly(),
	)
	if err != nil {
		slog.Warn("crawlNaive failed", "url", urlStr, "err", err)
		return CrawlResult{}, err
	}

	if result.Title == "" {
		slog.Warn("empty title, generating from URL", "url", urlStr)
		result.Title = generateTitleFromURL(urlStr)
	}

	slog.Info("crawlNaive succeeded", "title", result.Title, "website", result.Website, "content_chars", len(result.Content))
	return CrawlResult{
		Success: result,
	}, nil
}

// generateTitleFromURL creates a human-readable title from URL
func generateTitleFromURL(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}

	// Get the path without leading/trailing slashes
	path := strings.Trim(parsedURL.Path, "/")

	// If path is empty, use domain
	if path == "" {
		return parsedURL.Hostname()
	}

	// Split path into segments
	segments := strings.Split(path, "/")

	// Use the last segment as title base
	lastSegment := segments[len(segments)-1]

	// Remove file extensions
	lastSegment = strings.TrimSuffix(lastSegment, ".html")
	lastSegment = strings.TrimSuffix(lastSegment, ".htm")
	lastSegment = strings.TrimSuffix(lastSegment, ".php")

	// Replace hyphens and underscores with spaces
	title := strings.ReplaceAll(lastSegment, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")

	// Capitalize first letter of each word
	words := strings.Fields(title)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	title = strings.Join(words, " ")

	// If title is too long, truncate it
	if len(title) > 80 {
		title = title[:77] + "..."
	}

	return title
}
