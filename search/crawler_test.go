package search

import (
	"testing"
)

func TestGenerateTitleFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "Nature article with ID",
			url:      "https://www.nature.com/articles/d41586-025-03936-2",
			expected: "D41586 025 03936 2",
		},
		{
			name:     "TechCrunch article (long)",
			url:      "https://techcrunch.com/2025/12/19/openai-is-reportedly-trying-to-raise-100b-at-an-830b-valuation/",
			expected: "Openai Is Reportedly Trying To Raise 100b At An 830b Valuation",
		},
		{
			name:     "Very long URL",
			url:      "https://example.com/this-is-a-very-long-url-with-many-words-that-should-be-truncated-because-it-exceeds-the-maximum-length",
			expected: "This Is A Very Long Url With Many Words That Should Be Truncated Because It E...",
		},
		{
			name:     "Simple path",
			url:      "https://example.com/about-us",
			expected: "About Us",
		},
		{
			name:     "Root domain",
			url:      "https://example.com",
			expected: "example.com",
		},
		{
			name:     "With .html extension",
			url:      "https://example.com/contact-us.html",
			expected: "Contact Us",
		},
		{
			name:     "Underscore separator",
			url:      "https://example.com/my_awesome_page",
			expected: "My Awesome Page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateTitleFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("generateTitleFromURL(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}
