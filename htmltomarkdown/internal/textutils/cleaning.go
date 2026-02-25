package textutils

import (
	"strings"
)

// CleanMarkdownContent cleans up markdown content
func CleanMarkdownContent(content string) string {
	lines := strings.Split(content, "\n")
	var cleaned []string
	emptyCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip excessive empty lines (max 2 consecutive)
		if trimmed == "" {
			emptyCount++
			if emptyCount <= 2 {
				cleaned = append(cleaned, "")
			}
			continue
		}
		emptyCount = 0

		// Skip common noise patterns
		if IsNoisePattern(trimmed) {
			continue
		}

		cleaned = append(cleaned, line)
	}

	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

// IsNoisePattern checks if a line is common web noise
func IsNoisePattern(line string) bool {
	lower := strings.ToLower(line)

	noisePatterns := []string{
		"skip to content",
		"skip to main",
		"cookie",
		"we use cookies",
		"accept all",
		"reject all",
		"privacy policy",
		"terms of service",
		"subscribe to",
		"sign up for",
		"newsletter",
		"advertisement",
		"sponsored",
		"loading...",
		"please wait",
		"javascript is required",
		"enable javascript",
	}

	for _, pattern := range noisePatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}
