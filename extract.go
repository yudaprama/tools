package tools

import (
	"encoding/json"
	"strings"
)

// ToolCallBlock represents a raw extracted tool call block
type ToolCallBlock struct {
	StartIndex  int    // Position in original string where <tool_call> starts
	EndIndex    int    // Position in original string where </tool_call> ends
	JSONContent string // The JSON content between tags
}

// ExtractToolCallBlocks extracts all <tool_call>...</tool_call> blocks from text.
// Uses brace counting to correctly handle nested JSON objects and edge cases
// where </tool_call> might appear inside JSON strings.
//
// This is more robust than simple regex or string.Index approaches because:
// 1. It correctly handles nested JSON objects like {"a": {"b": 1}}
// 2. It handles </tool_call> appearing inside JSON string values
// 3. It handles multiple tool calls in a single response
func ExtractToolCallBlocks(text string) []ToolCallBlock {
	var blocks []ToolCallBlock

	const openTag = "<tool_call>"
	const closeTag = "</tool_call>"

	remaining := text
	offset := 0

	for {
		// Find next opening tag
		openIdx := strings.Index(remaining, openTag)
		if openIdx == -1 {
			break
		}

		// Start position after the opening tag
		contentStart := openIdx + len(openTag)
		if contentStart >= len(remaining) {
			break
		}

		content := remaining[contentStart:]

		// Find the JSON content using brace counting
		jsonContent, jsonEndIdx := extractJSONWithBraceCounting(content)
		if jsonEndIdx == -1 {
			// No valid JSON found, skip this opening tag and continue
			remaining = remaining[openIdx+1:]
			offset += openIdx + 1
			continue
		}

		// Now find the closing tag after the JSON
		afterJSON := content[jsonEndIdx:]
		afterJSONTrimmed := strings.TrimSpace(afterJSON)

		if !strings.HasPrefix(afterJSONTrimmed, closeTag) {
			// No closing tag found, skip this opening tag
			remaining = remaining[openIdx+1:]
			offset += openIdx + 1
			continue
		}

		// Calculate position of closing tag
		closeTagStart := jsonEndIdx + (len(afterJSON) - len(afterJSONTrimmed))
		closeTagEnd := closeTagStart + len(closeTag)

		blocks = append(blocks, ToolCallBlock{
			StartIndex:  offset + openIdx,
			EndIndex:    offset + contentStart + closeTagEnd,
			JSONContent: jsonContent,
		})

		// Move past this block
		remaining = content[closeTagEnd:]
		offset += contentStart + closeTagEnd
	}

	return blocks
}

// extractJSONWithBraceCounting extracts a JSON object from the start of text
// using brace counting to handle nested objects correctly.
// Returns the JSON string and the end index (position after the closing brace in original text).
// Returns ("", -1) if no valid JSON object is found.
func extractJSONWithBraceCounting(text string) (string, int) {
	// Find the first '{' character
	startIdx := -1
	for i := 0; i < len(text); i++ {
		if text[i] == '{' {
			startIdx = i
			break
		} else if text[i] != ' ' && text[i] != '\t' && text[i] != '\n' && text[i] != '\r' {
			// Non-whitespace before '{' means no JSON object at start
			return "", -1
		}
	}

	if startIdx == -1 {
		return "", -1
	}

	braceCount := 0
	inString := false
	escapeNext := false

	for i := startIdx; i < len(text); i++ {
		ch := text[i]

		if escapeNext {
			escapeNext = false
			continue
		}

		if ch == '\\' && inString {
			escapeNext = true
			continue
		}

		if ch == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if ch == '{' {
			braceCount++
		} else if ch == '}' {
			braceCount--
			if braceCount == 0 {
				// Found the matching closing brace
				jsonStr := text[startIdx : i+1]
				// Validate it's actually valid JSON
				if json.Valid([]byte(jsonStr)) {
					return jsonStr, i + 1
				}
				return "", -1
			}
		}
	}

	return "", -1
}

// ExtractTextBeforeToolCalls returns the text content before the first tool call
func ExtractTextBeforeToolCalls(text string, blocks []ToolCallBlock) string {
	if len(blocks) == 0 {
		return strings.TrimSpace(text)
	}
	return strings.TrimSpace(text[:blocks[0].StartIndex])
}

// ExtractTextBetweenToolCalls returns text segments between tool calls
func ExtractTextBetweenToolCalls(text string, blocks []ToolCallBlock) []string {
	if len(blocks) <= 1 {
		return nil
	}

	var segments []string
	for i := 0; i < len(blocks)-1; i++ {
		segment := text[blocks[i].EndIndex:blocks[i+1].StartIndex]
		segment = strings.TrimSpace(segment)
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	return segments
}

// ExtractTextAfterToolCalls returns the text content after the last tool call
func ExtractTextAfterToolCalls(text string, blocks []ToolCallBlock) string {
	if len(blocks) == 0 {
		return ""
	}
	lastBlock := blocks[len(blocks)-1]
	if lastBlock.EndIndex >= len(text) {
		return ""
	}
	return strings.TrimSpace(text[lastBlock.EndIndex:])
}

// StripToolCallTags removes all tool call blocks from text, leaving only regular text
func StripToolCallTags(text string) string {
	blocks := ExtractToolCallBlocks(text)
	if len(blocks) == 0 {
		return text
	}

	var result strings.Builder
	lastEnd := 0

	for _, block := range blocks {
		if block.StartIndex > lastEnd {
			result.WriteString(text[lastEnd:block.StartIndex])
		}
		lastEnd = block.EndIndex
	}

	if lastEnd < len(text) {
		result.WriteString(text[lastEnd:])
	}

	// Clean up multiple spaces and trim
	output := result.String()
	for strings.Contains(output, "  ") {
		output = strings.ReplaceAll(output, "  ", " ")
	}
	return strings.TrimSpace(output)
}
