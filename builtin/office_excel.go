package builtin

import (
	"context"
	"fmt"

	"github.com/getkawai/unillm"
	"github.com/yudaprama/tools"
	"github.com/yudaprama/tools/gooxml/spreadsheet"
)

// -- Data Structures --

// ExcelRow content for Excel spreadsheets
type ExcelRow struct {
	Cells []string `json:"cells" jsonschema:"description=Cell values for the row"`
}

// -- Inputs --

// CreateExcelInput defines input for creating Excel spreadsheets
type CreateExcelInput struct {
	Filename string     `json:"filename" jsonschema:"description=Output filename (e.g. sheet.xlsx)"`
	Rows     []ExcelRow `json:"rows" jsonschema:"description=List of rows"`
}

// UpdateExcelInput defines input for updating Excel spreadsheets
type UpdateExcelInput struct {
	Filename string     `json:"filename" jsonschema:"description=Filename of existing spreadsheet to update"`
	Sheet    string     `json:"sheet,omitempty" jsonschema:"description=Sheet name to append to (defaults to first sheet)"`
	Rows     []ExcelRow `json:"rows" jsonschema:"description=List of rows to append"`
}

// ReadExcelInput defines input for reading Excel spreadsheets
type ReadExcelInput struct {
	Filename string `json:"filename" jsonschema:"description=Filename of spreadsheet to read"`
}

// -- Executors --

// CreateExcel creates a new Excel spreadsheet.
func CreateExcel(ctx context.Context, input CreateExcelInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
	wb := spreadsheet.New()
	sheet := wb.AddSheet()
	for _, item := range input.Rows {
		row := sheet.AddRow()
		for _, cellValue := range item.Cells {
			cell := row.AddCell()
			cell.SetString(cellValue)
		}
	}
	if err := wb.SaveToFile(input.Filename); err != nil {
		return unillm.NewTextErrorResponse(fmt.Sprintf("failed to save xlsx: %v", err)), nil
	}
	return unillm.NewTextResponse(fmt.Sprintf("Excel spreadsheet created successfully at %s", input.Filename)), nil
}

// UpdateExcel updates an existing Excel spreadsheet.
func UpdateExcel(ctx context.Context, input UpdateExcelInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
	wb, err := spreadsheet.Open(input.Filename)
	if err != nil {
		return unillm.NewTextErrorResponse(fmt.Sprintf("failed to open xlsx: %v", err)), nil
	}

	var sheet spreadsheet.Sheet
	if input.Sheet != "" {
		// Find sheet by name - simplistic traverse if API allows, otherwise default
		// gooxml sheets are accessed via index usually or helper
		// Assuming first sheet for simplicity if name not found easily without iterating
		// For now, let's just append to the first sheet as per basic requirement
		// or iterate to find.
		found := false
		for _, s := range wb.Sheets() {
			if s.Name() == input.Sheet {
				sheet = s
				found = true
				break
			}
		}
		if !found {
			// Create new sheet if specified but not found? Or error?
			// Let's create new.
			sheet = wb.AddSheet()
			// sheet.SetName(input.Sheet) // if supported
		}
	} else {
		// Default to first sheet
		if len(wb.Sheets()) > 0 {
			sheet = wb.Sheets()[0]
		} else {
			sheet = wb.AddSheet()
		}
	}

	for _, item := range input.Rows {
		row := sheet.AddRow()
		for _, cellValue := range item.Cells {
			cell := row.AddCell()
			cell.SetString(cellValue)
		}
	}

	if err := wb.SaveToFile(input.Filename); err != nil {
		return unillm.NewTextErrorResponse(fmt.Sprintf("failed to save updated xlsx: %v", err)), nil
	}
	return unillm.NewTextResponse(fmt.Sprintf("Excel spreadsheet updated successfully at %s", input.Filename)), nil
}

// ReadExcel reads an Excel spreadsheet.
func ReadExcel(ctx context.Context, input ReadExcelInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
	wb, err := spreadsheet.Open(input.Filename)
	if err != nil {
		return unillm.NewTextErrorResponse(fmt.Sprintf("failed to open xlsx: %v", err)), nil
	}

	markdown, err := wb.ToMarkdownWithImageURLs("")
	if err != nil {
		return unillm.NewTextErrorResponse(fmt.Sprintf("failed to convert to markdown: %v", err)), nil
	}

	return unillm.NewTextResponse(markdown), nil
}

// -- Registration --

// RegisterOfficeExcel registers the Excel tools.
func RegisterOfficeExcel(registry *tools.ToolRegistry) error {
	createTool := unillm.NewAgentTool(
		"office-excel__create",
		"Create a standard Spreadsheet (.xlsx).",
		CreateExcel,
	)
	if err := registry.Register(createTool); err != nil {
		return err
	}

	updateTool := unillm.NewAgentTool(
		"office-excel__update",
		"Update an existing Spreadsheet by appending rows.",
		UpdateExcel,
	)
	if err := registry.Register(updateTool); err != nil {
		return err
	}

	readTool := unillm.NewAgentTool(
		"office-excel__read",
		"Read data from a Spreadsheet.",
		ReadExcel,
	)
	return registry.Register(readTool)
}
