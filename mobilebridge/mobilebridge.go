package mobilebridge

import (
	"encoding/json"

	"github.com/yudaprama/tools/search"
)

// WebSearchJSON performs a web search and returns JSON result.
func WebSearchJSON(query string) string {
	service := search.NewService()
	resp, err := service.WebSearch(search.SearchQuery{Query: query})
	if err != nil {
		data, _ := json.Marshal(map[string]any{
			"error": err.Error(),
			"query": query,
		})
		return string(data)
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return "{\"error\":\"json_marshal_failed\"}"
	}
	return string(data)
}
