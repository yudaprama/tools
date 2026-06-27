package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/yudaprama/tools/search"
)

// WebSearchInput defines input for web search tool
type WebSearchInput struct {
	Query      string `json:"query" jsonschema:"description=The search query"`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"description=Maximum number of results (default: 10)"`
}

// NewWebSearch creates the web search tool.
func NewWebSearch(_ context.Context) ([]tool.InvokableTool, error) {
	searchService := search.NewService()

	webSearchTool, err := utils.InferTool("web_search",
		"Search the web for current information using Brave Search. Returns real-time search results with titles, URLs, and descriptions.",
		func(ctx context.Context, input *WebSearchInput) (string, error) {
			if input.Query == "" {
				return "", fmt.Errorf("query parameter is required")
			}

			maxResults := input.MaxResults
			if maxResults <= 0 {
				maxResults = 10
			}

			searchQuery := search.SearchQuery{
				Query:            input.Query,
				SearchCategories: []string{"general"},
				SearchEngines:    []string{},
				SearchTimeRange:  "anytime",
			}

			response, err := searchService.WebSearch(searchQuery)
			if err != nil {
				log.Printf("⚠️  Web search failed: %v", err)
				return "", fmt.Errorf("search failed: %v", err)
			}

			results := make([]map[string]interface{}, 0, len(response.Results))
			for i, result := range response.Results {
				if i >= maxResults {
					break
				}
				results = append(results, map[string]interface{}{
					"title":   result.Title,
					"url":     result.URL,
					"snippet": result.Content,
				})
			}

			resultData := map[string]interface{}{
				"query":       input.Query,
				"results":     results,
				"count":       len(results),
				"max_results": maxResults,
			}

			resultJSON, err := json.Marshal(resultData)
			if err != nil {
				return "", fmt.Errorf("failed to marshal results: %v", err)
			}

			return string(resultJSON), nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to infer web_search tool: %w", err)
	}

	return []tool.InvokableTool{webSearchTool}, nil
}
