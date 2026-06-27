package tools

import (
	"testing"
)

func TestExtractToolCallBlocks_SingleToolCall(t *testing.T) {
	input := `Here is the result:
<tool_call>
{"name": "calculator", "arguments": {"expression": "2+2"}}
</tool_call>
Done!`

	blocks := ExtractToolCallBlocks(input)

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	if blocks[0].JSONContent != `{"name": "calculator", "arguments": {"expression": "2+2"}}` {
		t.Errorf("unexpected JSON content: %s", blocks[0].JSONContent)
	}
}

func TestExtractToolCallBlocks_MultipleToolCalls(t *testing.T) {
	input := `Let me search for that:
<tool_call>
{"name": "web_search", "arguments": {"query": "weather today"}}
</tool_call>
And also calculate:
<tool_call>
{"name": "calculator", "arguments": {"expression": "100/4"}}
</tool_call>
All done.`

	blocks := ExtractToolCallBlocks(input)

	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	if blocks[0].JSONContent != `{"name": "web_search", "arguments": {"query": "weather today"}}` {
		t.Errorf("unexpected JSON content for block 0: %s", blocks[0].JSONContent)
	}

	if blocks[1].JSONContent != `{"name": "calculator", "arguments": {"expression": "100/4"}}` {
		t.Errorf("unexpected JSON content for block 1: %s", blocks[1].JSONContent)
	}
}

func TestExtractToolCallBlocks_NestedJSON(t *testing.T) {
	input := `<tool_call>
{"name": "complex_tool", "arguments": {"nested": {"level1": {"level2": "value"}}, "array": [1, 2, 3]}}
</tool_call>`

	blocks := ExtractToolCallBlocks(input)

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	expected := `{"name": "complex_tool", "arguments": {"nested": {"level1": {"level2": "value"}}, "array": [1, 2, 3]}}`
	if blocks[0].JSONContent != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, blocks[0].JSONContent)
	}
}

func TestExtractToolCallBlocks_CloseTagInsideString(t *testing.T) {
	// This is the critical edge case - </tool_call> appearing inside a JSON string value
	input := `<tool_call>
{"name": "test", "arguments": {"text": "The tag </tool_call> should not break parsing"}}
</tool_call>`

	blocks := ExtractToolCallBlocks(input)

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	expected := `{"name": "test", "arguments": {"text": "The tag </tool_call> should not break parsing"}}`
	if blocks[0].JSONContent != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, blocks[0].JSONContent)
	}
}

func TestExtractToolCallBlocks_MalformedJSON(t *testing.T) {
	input := `<tool_call>
{"name": "broken", "arguments": {"missing": "closing brace"
</tool_call>
<tool_call>
{"name": "valid", "arguments": {"key": "value"}}
</tool_call>`

	blocks := ExtractToolCallBlocks(input)

	// Should only extract the valid one
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block (only valid one), got %d", len(blocks))
	}

	if blocks[0].JSONContent != `{"name": "valid", "arguments": {"key": "value"}}` {
		t.Errorf("unexpected JSON content: %s", blocks[0].JSONContent)
	}
}

func TestExtractToolCallBlocks_NoToolCalls(t *testing.T) {
	input := "Just regular text without any tool calls."

	blocks := ExtractToolCallBlocks(input)

	if len(blocks) != 0 {
		t.Fatalf("expected 0 blocks, got %d", len(blocks))
	}
}

func TestExtractToolCallBlocks_EmptyInput(t *testing.T) {
	blocks := ExtractToolCallBlocks("")

	if len(blocks) != 0 {
		t.Fatalf("expected 0 blocks, got %d", len(blocks))
	}
}

func TestExtractToolCallBlocks_UnclosedTag(t *testing.T) {
	input := `<tool_call>
{"name": "test", "arguments": {}}
No closing tag here`

	blocks := ExtractToolCallBlocks(input)

	if len(blocks) != 0 {
		t.Fatalf("expected 0 blocks (no closing tag), got %d", len(blocks))
	}
}

func TestExtractToolCallBlocks_EscapedQuotesInJSON(t *testing.T) {
	input := `<tool_call>
{"name": "test", "arguments": {"text": "He said \"hello\" to me"}}
</tool_call>`

	blocks := ExtractToolCallBlocks(input)

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	expected := `{"name": "test", "arguments": {"text": "He said \"hello\" to me"}}`
	if blocks[0].JSONContent != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, blocks[0].JSONContent)
	}
}

func TestExtractToolCallBlocks_WhitespaceVariations(t *testing.T) {
	// Various whitespace around JSON
	input := `<tool_call>   
   {"name": "test", "arguments": {"key": "value"}}   
   </tool_call>`

	blocks := ExtractToolCallBlocks(input)

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	expected := `{"name": "test", "arguments": {"key": "value"}}`
	if blocks[0].JSONContent != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, blocks[0].JSONContent)
	}
}

func TestExtractTextBeforeToolCalls(t *testing.T) {
	input := `Here is some text before.
<tool_call>
{"name": "test", "arguments": {}}
</tool_call>`

	blocks := ExtractToolCallBlocks(input)
	text := ExtractTextBeforeToolCalls(input, blocks)

	expected := "Here is some text before."
	if text != expected {
		t.Errorf("expected: %q, got: %q", expected, text)
	}
}

func TestExtractTextAfterToolCalls(t *testing.T) {
	input := `<tool_call>
{"name": "test", "arguments": {}}
</tool_call>
And here is text after.`

	blocks := ExtractToolCallBlocks(input)
	text := ExtractTextAfterToolCalls(input, blocks)

	expected := "And here is text after."
	if text != expected {
		t.Errorf("expected: %q, got: %q", expected, text)
	}
}

func TestStripToolCallTags(t *testing.T) {
	input := `Before text.
<tool_call>
{"name": "test", "arguments": {}}
</tool_call>
Middle text.
<tool_call>
{"name": "another", "arguments": {"key": "val"}}
</tool_call>
After text.`

	result := StripToolCallTags(input)
	// The function preserves newlines around the stripped blocks
	expected := "Before text.\n\nMiddle text.\n\nAfter text."

	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}

func TestExtractToolCallBlocks_InterleavedText(t *testing.T) {
	input := `Let me help you with that.
<tool_call>
{"name": "search", "arguments": {"query": "first query"}}
</tool_call>
Found some results. Now let me search more:
<tool_call>
{"name": "search", "arguments": {"query": "second query"}}
</tool_call>
Here are all the results combined.`

	blocks := ExtractToolCallBlocks(input)

	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	textBefore := ExtractTextBeforeToolCalls(input, blocks)
	if textBefore != "Let me help you with that." {
		t.Errorf("unexpected text before: %q", textBefore)
	}

	textBetween := ExtractTextBetweenToolCalls(input, blocks)
	if len(textBetween) != 1 || textBetween[0] != "Found some results. Now let me search more:" {
		t.Errorf("unexpected text between: %v", textBetween)
	}

	textAfter := ExtractTextAfterToolCalls(input, blocks)
	if textAfter != "Here are all the results combined." {
		t.Errorf("unexpected text after: %q", textAfter)
	}
}

func TestExtractJSONWithBraceCounting_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantJSON string
		wantIdx  int
	}{
		{
			name:     "simple object",
			input:    `{"key": "value"}`,
			wantJSON: `{"key": "value"}`,
			wantIdx:  16,
		},
		{
			name:     "nested object",
			input:    `{"outer": {"inner": "value"}}`,
			wantJSON: `{"outer": {"inner": "value"}}`,
			wantIdx:  29,
		},
		{
			name:     "with trailing content",
			input:    `{"key": "value"} extra stuff`,
			wantJSON: `{"key": "value"}`,
			wantIdx:  16,
		},
		{
			name:     "empty object",
			input:    `{}`,
			wantJSON: `{}`,
			wantIdx:  2,
		},
		{
			name:     "with whitespace prefix",
			input:    `   {"key": "value"}`,
			wantJSON: `{"key": "value"}`,
			wantIdx:  19, // Index after the closing brace in original string (3 spaces + 16 chars)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotJSON, gotIdx := extractJSONWithBraceCounting(tt.input)
			if gotJSON != tt.wantJSON {
				t.Errorf("JSON = %q, want %q", gotJSON, tt.wantJSON)
			}
			if gotIdx != tt.wantIdx {
				t.Errorf("idx = %d, want %d", gotIdx, tt.wantIdx)
			}
		})
	}
}

func TestExtractJSONWithBraceCounting_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantOK  bool
	}{
		{
			name:   "unclosed brace",
			input:  `{"key": "value"`,
			wantOK: false,
		},
		{
			name:   "not starting with brace",
			input:  `"just a string"`,
			wantOK: false,
		},
		{
			name:   "empty string",
			input:  ``,
			wantOK: false,
		},
		{
			name:   "only whitespace",
			input:  `   `,
			wantOK: false,
		},
		{
			name:   "brace inside string",
			input:  `{"text": "has } inside"}`,
			wantOK: true,
		},
		{
			name:   "escaped quote",
			input:  `{"text": "has \"quote\" inside"}`,
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, idx := extractJSONWithBraceCounting(tt.input)
			gotOK := idx != -1
			if gotOK != tt.wantOK {
				t.Errorf("ok = %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}

