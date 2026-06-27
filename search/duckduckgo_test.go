package search

import (
	"context"
	"testing"
)

func TestDuckDuckGoProvider_Query(t *testing.T) {
	provider := NewDuckDuckGoProvider()

	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "Simple query",
			query:   "golang programming",
			wantErr: false,
		},
		{
			name:    "AI news query",
			query:   "artificial intelligence news",
			wantErr: false,
		},
		{
			name:    "Empty query",
			query:   "",
			wantErr: false, // DuckDuckGo API accepts empty queries
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			response, err := provider.Query(ctx, tt.query, nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if response == nil {
					t.Error("Query() returned nil response")
					return
				}

				if response.Query != tt.query {
					t.Errorf("Query() response.Query = %v, want %v", response.Query, tt.query)
				}

				t.Logf("Query: %s, Results: %d, CostTime: %dms",
					response.Query, response.ResultNumbers, response.CostTime)

				// Log first few results for debugging
				for i, result := range response.Results {
					if i >= 3 {
						break
					}
					t.Logf("  Result %d: %s - %s", i+1, result.Title, result.URL)
				}
			}
		})
	}
}

func TestDuckDuckGoProvider_Name(t *testing.T) {
	provider := NewDuckDuckGoProvider()
	if name := provider.Name(); name != "duckduckgo" {
		t.Errorf("Name() = %v, want %v", name, "duckduckgo")
	}
}
