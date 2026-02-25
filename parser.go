package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/kawai-network/veridium/pkg/fantasy"
)

// argsToJSON converts map[string]string to JSON string
func argsToJSON(args map[string]string) string {
	if args == nil {
		return "{}"
	}
	b, err := json.Marshal(args)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// ParseToolCalls parses tool calls from LLM response
// Supports multiple formats:
// 1. <tool_call>{"name": "...", "arguments": {...}}</tool_call>
// 2. <tool_name>JSON</tool_name> or <tool_name attr="value">content</tool_name>
// 3. <tool_name {JSON}> or <tool_name {"key": "value"}> (no closing tag)
// 4. {"name": "tool_name", "parameters": {...}} (pure JSON format)
func ParseToolCalls(response string) []fantasy.ToolCall {
	var calls []fantasy.ToolCall

	// Try format 1: <tool_call>{"name": "...", "arguments": {...}}</tool_call>
	calls = parseToolCallFormat(response)
	if len(calls) > 0 {
		return calls
	}

	// Try format 2: <tool_name>...</tool_name> with closing tag
	calls = parseXMLToolFormat(response)
	if len(calls) > 0 {
		return calls
	}

	// Try format 3: <tool_name {JSON}> without closing tag
	calls = parseInlineJSONToolFormat(response)
	if len(calls) > 0 {
		return calls
	}

	// Try format 4: {"name": "tool_name", "parameters": {...}} pure JSON
	calls = parsePureJSONToolFormat(response)
	return calls
}

// parsePureJSONToolFormat parses pure JSON tool call format
func parsePureJSONToolFormat(response string) []fantasy.ToolCall {
	var calls []fantasy.ToolCall
	toolNames := []string{"calculator", "web_search", "web-search", "search"}
	remaining := strings.TrimSpace(response)

	if strings.HasPrefix(remaining, "{") {
		braceCount := 0
		jsonEnd := -1
		for i, ch := range remaining {
			if ch == '{' {
				braceCount++
			} else if ch == '}' {
				braceCount--
				if braceCount == 0 {
					jsonEnd = i + 1
					break
				}
			}
		}

		if jsonEnd > 0 {
			jsonContent := remaining[:jsonEnd]
			var toolCall struct {
				Name       string                 `json:"name"`
				Parameters map[string]interface{} `json:"parameters"`
				Arguments  map[string]interface{} `json:"arguments"`
			}

			if err := json.Unmarshal([]byte(jsonContent), &toolCall); err == nil {
				isKnownTool := false
				for _, tn := range toolNames {
					if toolCall.Name == tn {
						isKnownTool = true
						break
					}
				}

				if isKnownTool && toolCall.Name != "" {
					params := toolCall.Parameters
					if params == nil {
						params = toolCall.Arguments
					}

					args := make(map[string]string)
					for k, v := range params {
						switch val := v.(type) {
						case string:
							args[k] = val
						case float64:
							args[k] = fmt.Sprintf("%v", val)
						case int:
							args[k] = fmt.Sprintf("%d", val)
						default:
							args[k] = fmt.Sprintf("%v", val)
						}
					}

					if len(args) > 0 {
						calls = append(calls, fantasy.ToolCall{
							ID:    uuid.New().String(),
							Name:  toolCall.Name,
							Input: argsToJSON(args),
						})
					}
				}
			}
		}
	}

	return calls
}

// parseInlineJSONToolFormat parses <tool_name {JSON}> format (no closing tag)
func parseInlineJSONToolFormat(response string) []fantasy.ToolCall {
	var calls []fantasy.ToolCall
	toolNames := []string{"calculator", "web_search", "web-search", "search"}

	for _, toolName := range toolNames {
		remaining := response
		for {
			openTagStart := strings.Index(remaining, "<"+toolName)
			if openTagStart == -1 {
				break
			}

			afterToolName := remaining[openTagStart+len("<"+toolName):]
			afterToolName = strings.TrimSpace(afterToolName)

			if !strings.HasPrefix(afterToolName, "{") {
				remaining = remaining[openTagStart+1:]
				continue
			}

			braceCount := 0
			jsonEnd := -1

			for i, ch := range afterToolName {
				if ch == '{' {
					braceCount++
				} else if ch == '}' {
					braceCount--
					if braceCount == 0 {
						jsonEnd = i + 1
						break
					}
				}
			}

			if jsonEnd == -1 {
				remaining = remaining[openTagStart+1:]
				continue
			}

			jsonContent := afterToolName[:jsonEnd]
			var args map[string]string

			var rawArgs map[string]interface{}
			if err := json.Unmarshal([]byte(jsonContent), &rawArgs); err == nil {
				args = make(map[string]string)
				for k, v := range rawArgs {
					switch val := v.(type) {
					case string:
						args[k] = val
					case float64:
						args[k] = fmt.Sprintf("%v", val)
					case int:
						args[k] = fmt.Sprintf("%d", val)
					default:
						args[k] = fmt.Sprintf("%v", val)
					}
				}
			}

			if len(args) > 0 {
				calls = append(calls, fantasy.ToolCall{
					ID:    uuid.New().String(),
					Name:  toolName,
					Input: argsToJSON(args),
				})
			}

			remaining = remaining[openTagStart+len("<"+toolName)+jsonEnd:]
		}
	}

	return calls
}

// parseToolCallFormat parses <tool_call> tags using robust brace-counting extraction.
// This correctly handles:
// - Multiple tool calls in one response
// - Nested JSON objects like {"a": {"b": 1}}
// - Edge cases where </tool_call> appears inside JSON strings
func parseToolCallFormat(response string) []fantasy.ToolCall {
	var calls []fantasy.ToolCall

	// Use the robust extraction that handles nested JSON and edge cases
	blocks := ExtractToolCallBlocks(response)

	for _, block := range blocks {
		var parsed struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}

		if err := json.Unmarshal([]byte(block.JSONContent), &parsed); err == nil && parsed.Name != "" {
			// Convert arguments to map[string]string for compatibility
			args := make(map[string]string)
			for k, v := range parsed.Arguments {
				switch val := v.(type) {
				case string:
					args[k] = val
				case float64:
					args[k] = fmt.Sprintf("%v", val)
				case int:
					args[k] = fmt.Sprintf("%d", val)
				case bool:
					args[k] = fmt.Sprintf("%v", val)
				default:
					// For complex types (nested objects/arrays), marshal back to JSON
					if b, err := json.Marshal(val); err == nil {
						args[k] = string(b)
					} else {
						args[k] = fmt.Sprintf("%v", val)
					}
				}
			}

			calls = append(calls, fantasy.ToolCall{
				ID:    uuid.New().String(),
				Name:  parsed.Name,
				Input: argsToJSON(args),
			})
		}
	}

	return calls
}

// parseXMLToolFormat parses <tool_name>...</tool_name> tags
func parseXMLToolFormat(response string) []fantasy.ToolCall {
	var calls []fantasy.ToolCall
	toolNames := []string{"calculator", "web_search", "web-search", "search"}

	for _, toolName := range toolNames {
		closeTag := "</" + toolName + ">"
		remaining := response
		for {
			openTagStart := strings.Index(remaining, "<"+toolName)
			if openTagStart == -1 {
				break
			}

			openTagEnd := strings.Index(remaining[openTagStart:], ">")
			if openTagEnd == -1 {
				break
			}
			openTagEnd += openTagStart

			fullOpenTag := remaining[openTagStart : openTagEnd+1]

			end := strings.Index(remaining[openTagEnd:], closeTag)
			if end == -1 {
				break
			}
			end += openTagEnd

			content := remaining[openTagEnd+1 : end]
			content = strings.TrimSpace(content)

			var parsed struct {
				Name      string            `json:"name"`
				Arguments map[string]string `json:"arguments"`
			}

			if err := json.Unmarshal([]byte(content), &parsed); err == nil {
				calls = append(calls, fantasy.ToolCall{
					ID:    uuid.New().String(),
					Name:  parsed.Name,
					Input: argsToJSON(parsed.Arguments),
				})
			} else {
				args := parseTagAttributes(fullOpenTag, toolName)

				if strings.HasPrefix(content, "{") {
					var jsonArgs map[string]string
					if err := json.Unmarshal([]byte(content), &jsonArgs); err == nil {
						for k, v := range jsonArgs {
							if _, exists := args[k]; !exists {
								args[k] = v
							}
						}
					}
				} else if content != "" && len(args) == 0 {
					if toolName == "calculator" {
						args["expression"] = content
					} else {
						args["query"] = content
					}
				}

				if len(args) > 0 {
					calls = append(calls, fantasy.ToolCall{
						ID:    uuid.New().String(),
						Name:  toolName,
						Input: argsToJSON(args),
					})
				}
			}

			remaining = remaining[end+len(closeTag):]
		}
	}

	return calls
}

// parseTagAttributes extracts attributes from XML-like opening tag
func parseTagAttributes(tag string, toolName string) map[string]string {
	args := make(map[string]string)

	inner := strings.TrimPrefix(tag, "<"+toolName)
	inner = strings.TrimSuffix(inner, ">")
	inner = strings.TrimSpace(inner)

	if inner == "" {
		return args
	}

	for len(inner) > 0 {
		inner = strings.TrimSpace(inner)
		if inner == "" {
			break
		}

		eqIdx := strings.Index(inner, "=")
		if eqIdx == -1 {
			break
		}

		attrName := strings.TrimSpace(inner[:eqIdx])
		inner = inner[eqIdx+1:]
		inner = strings.TrimSpace(inner)

		if len(inner) == 0 {
			break
		}

		quote := inner[0]
		if quote != '"' && quote != '\'' {
			spaceIdx := strings.Index(inner, " ")
			if spaceIdx == -1 {
				args[attrName] = inner
				break
			}
			args[attrName] = inner[:spaceIdx]
			inner = inner[spaceIdx:]
			continue
		}

		inner = inner[1:]
		closeIdx := strings.Index(inner, string(quote))
		if closeIdx == -1 {
			break
		}

		args[attrName] = inner[:closeIdx]
		inner = inner[closeIdx+1:]
	}

	return args
}

// FormatToolsJSON formats tool definitions as JSON string
func FormatToolsJSON(toolDefs []map[string]any) string {
	if len(toolDefs) == 0 {
		return ""
	}

	toolsJSON, err := json.MarshalIndent(toolDefs, "", "  ")
	if err != nil {
		return ""
	}
	return string(toolsJSON)
}

// BuildSystemPrompt builds a system prompt with tool definitions
func BuildSystemPrompt(basePrompt string, toolsJSON string) string {
	if toolsJSON == "" {
		return basePrompt
	}

	return fmt.Sprintf(`%s

# Available Tools

You have access to the following tools:

%s

# Tool Usage Instructions

IMPORTANT: When you need to use a tool, you MUST use EXACTLY this format:

<tool_call>
{"name": "TOOL_NAME", "arguments": {"param1": "value1", "param2": "value2"}}
</tool_call>

Example for web_search:
<tool_call>
{"name": "web_search", "arguments": {"query": "latest AI news", "max_results": "5"}}
</tool_call>

Example for calculator:
<tool_call>
{"name": "calculator", "arguments": {"expression": "2 + 2 * 3"}}
</tool_call>

Rules:
1. ALWAYS use <tool_call> tags - other formats will NOT work
2. Use "arguments" (not "parameters") for the parameter object
3. All argument values must be strings (e.g., "5" not 5)
4. Wait for tool results before providing your final answer
5. After receiving results, synthesize them into a helpful response`, basePrompt, toolsJSON)
}
