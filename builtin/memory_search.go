package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/getkawai/unillm"
	"github.com/yudaprama/tools"
)

// MemorySearchInput defines input for memory search tool
type MemorySearchInput struct {
	Query    string `json:"query" jsonschema:"description=What to search in conversation history and stored memories"`
	Category string `json:"category,omitempty" jsonschema:"description=Filter by category: conversation, fact, preference, task, context. Leave empty for all."`
	Limit    int    `json:"limit,omitempty" jsonschema:"description=Maximum number of results (default: 5)"`
}

// MemorySearchResult represents a search result
type MemorySearchResult struct {
	ID         string  `json:"id"`
	Category   string  `json:"category"`
	Title      string  `json:"title"`
	Summary    string  `json:"summary"`
	Similarity float64 `json:"similarity"`
}

// MemorySearcher interface for memory search operations
type MemorySearcher interface {
	SemanticSearch(ctx context.Context, query string, limit int) ([]MemorySearchResult, error)
	SemanticSearchByCategory(ctx context.Context, query, category string, limit int) ([]MemorySearchResult, error)
}

// RegisterMemorySearch registers the memory search tool
func RegisterMemorySearch(registry *tools.ToolRegistry, searcher MemorySearcher) error {
	tool := unillm.NewParallelAgentTool("search_memory",
		`Search past conversations and stored memories about the user. 
Use this tool when you need to:
- Recall previous discussions or decisions
- Remember user preferences or personal information
- Find context from earlier conversations
- Look up tasks or projects mentioned before

The search uses semantic similarity, so you can describe what you're looking for in natural language.`,
		func(ctx context.Context, input MemorySearchInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			if input.Query == "" {
				return unillm.NewTextErrorResponse("query parameter is required"), nil
			}

			limit := input.Limit
			if limit <= 0 {
				limit = 5
			}

			var results []MemorySearchResult
			var err error

			if input.Category != "" {
				results, err = searcher.SemanticSearchByCategory(ctx, input.Query, input.Category, limit)
			} else {
				results, err = searcher.SemanticSearch(ctx, input.Query, limit)
			}

			if err != nil {
				log.Printf("⚠️  Memory search failed: %v", err)
				return unillm.NewTextErrorResponse(fmt.Sprintf("memory search failed: %v", err)), nil
			}

			if len(results) == 0 {
				return unillm.NewTextResponse("No relevant memories found for your query."), nil
			}

			// Format results for LLM
			formattedResults := formatMemorySearchResults(results)

			resultData := map[string]interface{}{
				"query":   input.Query,
				"count":   len(results),
				"results": results,
				"summary": formattedResults,
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

// formatMemorySearchResults formats results for human-readable output
func formatMemorySearchResults(results []MemorySearchResult) string {
	if len(results) == 0 {
		return "No memories found."
	}

	output := fmt.Sprintf("Found %d relevant memories:\n\n", len(results))
	for i, r := range results {
		output += fmt.Sprintf("%d. [%s] %s\n", i+1, r.Category, r.Title)
		if r.Summary != "" {
			output += fmt.Sprintf("   %s\n", r.Summary)
		}
		output += fmt.Sprintf("   (Relevance: %.1f%%)\n\n", r.Similarity*100)
	}

	return output
}

// MemoryServiceAdapter adapts MemoryService to MemorySearcher interface
type MemoryServiceAdapter struct {
	searchFunc           func(ctx context.Context, query string, limit int) ([]MemorySearchResult, error)
	searchByCategoryFunc func(ctx context.Context, query, category string, limit int) ([]MemorySearchResult, error)
}

// NewMemoryServiceAdapter creates a new adapter
func NewMemoryServiceAdapter(
	searchFunc func(ctx context.Context, query string, limit int) ([]MemorySearchResult, error),
	searchByCategoryFunc func(ctx context.Context, query, category string, limit int) ([]MemorySearchResult, error),
) *MemoryServiceAdapter {
	return &MemoryServiceAdapter{
		searchFunc:           searchFunc,
		searchByCategoryFunc: searchByCategoryFunc,
	}
}

func (a *MemoryServiceAdapter) SemanticSearch(ctx context.Context, query string, limit int) ([]MemorySearchResult, error) {
	return a.searchFunc(ctx, query, limit)
}

func (a *MemoryServiceAdapter) SemanticSearchByCategory(ctx context.Context, query, category string, limit int) ([]MemorySearchResult, error) {
	if a.searchByCategoryFunc != nil {
		return a.searchByCategoryFunc(ctx, query, category, limit)
	}
	// Fallback to regular search
	return a.searchFunc(ctx, query, limit)
}
