package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/getkawai/unillm"
	"github.com/yudaprama/tools"
	"github.com/yudaprama/tools/search"
)

// WebSearchInput defines input for web search tool
type WebSearchInput struct {
	Query      string `json:"query" jsonschema:"description=The search query"`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"description=Maximum number of results (default: 10)"`
}

// RegisterWebSearch registers the web search tool
func RegisterWebSearch(registry *tools.ToolRegistry) error {
	searchService := search.NewService()

	tool := unillm.NewParallelAgentTool("web_search",
		"Search the web for current information using Brave Search. Returns real-time search results with titles, URLs, and descriptions.",
		func(ctx context.Context, input WebSearchInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			if input.Query == "" {
				return unillm.NewTextErrorResponse("query parameter is required"), nil
			}

			maxResults := input.MaxResults
			if maxResults <= 0 {
				maxResults = 10
			}

			// Use real Brave Search API
			searchQuery := search.SearchQuery{
				Query:            input.Query,
				SearchCategories: []string{"general"},
				SearchEngines:    []string{},
				SearchTimeRange:  "anytime",
			}

			response, err := searchService.WebSearch(searchQuery)
			if err != nil {
				log.Printf("⚠️  Web search failed: %v", err)
				return unillm.NewTextErrorResponse(fmt.Sprintf("search failed: %v", err)), nil
			}

			// Format results for LLM
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
				return unillm.NewTextErrorResponse(fmt.Sprintf("failed to marshal results: %v", err)), nil
			}

			return unillm.NewTextResponse(string(resultJSON)), nil
		},
	)

	return registry.Register(tool)
}
