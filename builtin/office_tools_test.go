package builtin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kawai-network/veridium/pkg/fantasy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOfficeWord(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "office_word_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	filename := filepath.Join(tmpDir, "test.docx")

	// 1. Create
	createInput := `
	{
		"filename": "` + filename + `",
		"elements": [
			{
				"type": "paragraph",
				"paragraph": {
					"type": "heading1",
					"runs": [{"text": "Hello World"}]
				}
			}
		]
	}`
	createTool := fantasy.NewAgentTool("office-word__create", "desc", CreateWord)
	resp, err := createTool.Run(context.Background(), fantasy.ToolCall{Input: createInput})
	require.NoError(t, err)
	assert.False(t, resp.IsError, resp.Content)

	// 2. Read
	readTool := fantasy.NewAgentTool("office-word__read", "desc", ReadWord)
	resp, err = readTool.Run(context.Background(), fantasy.ToolCall{Input: `{"filename": "` + filename + `"}`})
	require.NoError(t, err)
	assert.False(t, resp.IsError)
	assert.Contains(t, resp.Content, "Hello World")

	// 3. Update
	updateInput := `
	{
		"filename": "` + filename + `",
		"elements": [
			{
				"type": "paragraph",
				"paragraph": {
					"runs": [{"text": "Appended Text"}]
				}
			}
		]
	}`
	updateTool := fantasy.NewAgentTool("office-word__update", "desc", UpdateWord)
	resp, err = updateTool.Run(context.Background(), fantasy.ToolCall{Input: updateInput})
	require.NoError(t, err)
	assert.False(t, resp.IsError)

	// 4. Read Loop
	resp, err = readTool.Run(context.Background(), fantasy.ToolCall{Input: `{"filename": "` + filename + `"}`})
	require.NoError(t, err)
	assert.Contains(t, resp.Content, "Appended Text")
}

func TestOfficeExcel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "office_excel_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	filename := filepath.Join(tmpDir, "test.xlsx")

	// 1. Create
	createTool := fantasy.NewAgentTool("office-excel__create", "desc", CreateExcel)
	input := `{"filename": "` + filename + `", "rows": [{"cells": ["A1", "B1"]}]}`
	resp, err := createTool.Run(context.Background(), fantasy.ToolCall{Input: input})
	require.NoError(t, err)
	assert.False(t, resp.IsError)

	// 2. Read
	readTool := fantasy.NewAgentTool("office-excel__read", "desc", ReadExcel)
	resp, err = readTool.Run(context.Background(), fantasy.ToolCall{Input: `{"filename": "` + filename + `"}`})
	require.NoError(t, err)
	assert.Contains(t, resp.Content, "| A1 | B1 |")

	// 3. Update
	updateTool := fantasy.NewAgentTool("office-excel__update", "desc", UpdateExcel)
	input = `{"filename": "` + filename + `", "rows": [{"cells": ["A2", "B2"]}]}`
	resp, err = updateTool.Run(context.Background(), fantasy.ToolCall{Input: input})
	require.NoError(t, err)
	assert.False(t, resp.IsError)

	// 4. Read Loop
	resp, err = readTool.Run(context.Background(), fantasy.ToolCall{Input: `{"filename": "` + filename + `"}`})
	require.NoError(t, err)
	assert.Contains(t, resp.Content, "| A2 | B2 |")
}

func TestOfficePowerPoint(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "office_ppt_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	filename := filepath.Join(tmpDir, "test.pptx")

	// 1. Create
	createTool := fantasy.NewAgentTool("office-powerpoint__create", "desc", CreatePowerPoint)
	input := `{"filename": "` + filename + `", "slides": [{"title": "Slide 1"}]}`
	resp, err := createTool.Run(context.Background(), fantasy.ToolCall{Input: input})
	require.NoError(t, err)
	assert.False(t, resp.IsError)

	// 2. Read
	readTool := fantasy.NewAgentTool("office-powerpoint__read", "desc", ReadPowerPoint)
	resp, err = readTool.Run(context.Background(), fantasy.ToolCall{Input: `{"filename": "` + filename + `"}`})
	require.NoError(t, err)
	assert.Contains(t, resp.Content, "Slide 1")

	// 3. Update
	updateTool := fantasy.NewAgentTool("office-powerpoint__update", "desc", UpdatePowerPoint)
	input = `{"filename": "` + filename + `", "slides": [{"title": "Slide 2"}]}`
	resp, err = updateTool.Run(context.Background(), fantasy.ToolCall{Input: input})
	require.NoError(t, err)
	assert.False(t, resp.IsError)
}
