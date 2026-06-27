package htmltomarkdown

// CrawlSuccessResult represents a successful crawl result
type CrawlSuccessResult struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	URL     string `json:"url"`
	Website string `json:"website"`
	Crawler string `json:"crawler"` // "jina" or "kawai" (naive)
}
