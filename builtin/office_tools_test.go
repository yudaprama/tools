package builtin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOfficeWord(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "office_word_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	filename := filepath.ToSlash(filepath.Join(tmpDir, "test.docx"))
	ctx := context.Background()

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
	createTool, err := utils.InferTool("office-word__create", "desc", CreateWord)
	require.NoError(t, err)
	content, err := createTool.InvokableRun(ctx, createInput)
	require.NoError(t, err, content)

	// 2. Read
	readTool, err := utils.InferTool("office-word__read", "desc", ReadWord)
	require.NoError(t, err)
	content, err = readTool.InvokableRun(ctx, `{"filename": "`+filename+`"}`)
	require.NoError(t, err)
	assert.Contains(t, content, "Hello World")

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
	updateTool, err := utils.InferTool("office-word__update", "desc", UpdateWord)
	require.NoError(t, err)
	_, err = updateTool.InvokableRun(ctx, updateInput)
	require.NoError(t, err)

	// 4. Read Loop
	content, err = readTool.InvokableRun(ctx, `{"filename": "`+filename+`"}`)
	require.NoError(t, err)
	assert.Contains(t, content, "Appended Text")
}

func TestOfficeExcel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "office_excel_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	filename := filepath.ToSlash(filepath.Join(tmpDir, "test.xlsx"))
	ctx := context.Background()

	// 1. Create
	createTool, err := utils.InferTool("office-excel__create", "desc", CreateExcel)
	require.NoError(t, err)
	input := `{"filename": "` + filename + `", "rows": [{"cells": ["A1", "B1"]}]}`
	content, err := createTool.InvokableRun(ctx, input)
	require.NoError(t, err, content)

	// 2. Read
	readTool, err := utils.InferTool("office-excel__read", "desc", ReadExcel)
	require.NoError(t, err)
	content, err = readTool.InvokableRun(ctx, `{"filename": "`+filename+`"}`)
	require.NoError(t, err)
	assert.Contains(t, content, "| A1 | B1 |")

	// 3. Update
	updateTool, err := utils.InferTool("office-excel__update", "desc", UpdateExcel)
	require.NoError(t, err)
	input = `{"filename": "` + filename + `", "rows": [{"cells": ["A2", "B2"]}]}`
	_, err = updateTool.InvokableRun(ctx, input)
	require.NoError(t, err)

	// 4. Read Loop
	content, err = readTool.InvokableRun(ctx, `{"filename": "`+filename+`"}`)
	require.NoError(t, err)
	assert.Contains(t, content, "| A2 | B2 |")
}

func TestOfficePowerPoint(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "office_ppt_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	filename := filepath.ToSlash(filepath.Join(tmpDir, "test.pptx"))
	ctx := context.Background()

	// 1. Create
	createTool, err := utils.InferTool("office-powerpoint__create", "desc", CreatePowerPoint)
	require.NoError(t, err)
	input := `{"filename": "` + filename + `", "slides": [{"title": "Slide 1"}]}`
	content, err := createTool.InvokableRun(ctx, input)
	require.NoError(t, err, content)

	// 2. Read
	readTool, err := utils.InferTool("office-powerpoint__read", "desc", ReadPowerPoint)
	require.NoError(t, err)
	content, err = readTool.InvokableRun(ctx, `{"filename": "`+filename+`"}`)
	require.NoError(t, err)
	assert.Contains(t, content, "Slide 1")

	// 3. Update
	updateTool, err := utils.InferTool("office-powerpoint__update", "desc", UpdatePowerPoint)
	require.NoError(t, err)
	input = `{"filename": "` + filename + `", "slides": [{"title": "Slide 2"}]}`
	_, err = updateTool.InvokableRun(ctx, input)
	require.NoError(t, err)
}
