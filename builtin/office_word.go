package builtin

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/yudaprama/tools/gooxml/color"
	"github.com/yudaprama/tools/gooxml/document"
	"github.com/yudaprama/tools/gooxml/measurement"
	"github.com/yudaprama/tools/gooxml/schema/soo/wml"
)

// -- Data Structures --

// WordRun defines a run of text with functionality.
type WordRun struct {
	Text      string `json:"text" jsonschema:"description=Text content"`
	Bold      bool   `json:"bold,omitempty" jsonschema:"description=Bold text"`
	Italic    bool   `json:"italic,omitempty" jsonschema:"description=Italic text"`
	Size      int    `json:"size,omitempty" jsonschema:"description=Font size in points"`
	Color     string `json:"color,omitempty" jsonschema:"description=Text color in hex (e.g. FF0000)"`
	Highlight string `json:"highlight,omitempty" jsonschema:"enum=yellow,enum=green,enum=cyan,enum=magenta,enum=blue,enum=red,enum=darkBlue,enum=darkCyan,enum=darkGreen,enum=darkMagenta,enum=darkRed,enum=darkYellow,enum=darkGray,enum=lightGray,enum=black,enum=white,enum=none"`
}

// WordParagraph content for Word documents
type WordParagraph struct {
	Type string    `json:"type" jsonschema:"enum=paragraph,enum=heading1,enum=heading2,enum=heading3,enum=bullet,enum=title"`
	Runs []WordRun `json:"runs" jsonschema:"description=Text runs with formatting"`
}

// WordTableCell content for a table cell
type WordTableCell struct {
	Content []WordParagraph `json:"content" jsonschema:"description=Cell content"`
	Width   int             `json:"width,omitempty" jsonschema:"description=Width of the cell in points"`
}

// WordTableRow content for a table row
type WordTableRow struct {
	Cells []WordTableCell `json:"cells" jsonschema:"description=Row cells"`
}

// WordTable content for a table
type WordTable struct {
	Rows []WordTableRow `json:"rows" jsonschema:"description=Table rows"`
}

// WordElement is a polymorphic wrapper for document elements.
type WordElement struct {
	Type      string         `json:"type" jsonschema:"enum=paragraph,enum=table"`
	Paragraph *WordParagraph `json:"paragraph,omitempty"`
	Table     *WordTable     `json:"table,omitempty"`
}

// -- Inputs --

// CreateWordInput input for creating Word documents
type CreateWordInput struct {
	Filename string        `json:"filename" jsonschema:"description=Output filename (e.g. file.docx)"`
	Elements []WordElement `json:"elements" jsonschema:"description=List of elements (paragraphs or tables)"`
}

// UpdateWordInput input for updating Word documents
type UpdateWordInput struct {
	Filename string        `json:"filename" jsonschema:"description=Filename of existing document to update"`
	Elements []WordElement `json:"elements" jsonschema:"description=List of elements to append"`
}

// ReadWordInput input for reading Word documents
type ReadWordInput struct {
	Filename string `json:"filename" jsonschema:"description=Filename of document to read"`
}

// -- Helpers --

func wordParseHexColor(s string) (color.Color, error) {
	var r, g, b uint8
	if len(s) == 6 {
		_, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b)
		if err != nil {
			return color.Color{}, err
		}
		return color.RGB(r, g, b), nil
	}
	return color.Color{}, fmt.Errorf("invalid color format")
}

func addRunsToParagraph(para document.Paragraph, runs []WordRun) {
	for _, runData := range runs {
		run := para.AddRun()
		run.AddText(runData.Text)

		if runData.Bold {
			run.Properties().SetBold(true)
		}
		if runData.Italic {
			run.Properties().SetItalic(true)
		}
		if runData.Size > 0 {
			run.Properties().SetSize(measurement.Distance(runData.Size) * measurement.Point)
		}
		if runData.Color != "" {
			c, err := wordParseHexColor(runData.Color)
			if err == nil {
				run.Properties().SetColor(c)
			}
		}
		if runData.Highlight != "" && runData.Highlight != "none" {
			switch runData.Highlight {
			case "yellow":
				run.Properties().SetHighlight(wml.ST_HighlightColorYellow)
			case "green":
				run.Properties().SetHighlight(wml.ST_HighlightColorGreen)
			case "cyan":
				run.Properties().SetHighlight(wml.ST_HighlightColorCyan)
			case "magenta":
				run.Properties().SetHighlight(wml.ST_HighlightColorMagenta)
			case "blue":
				run.Properties().SetHighlight(wml.ST_HighlightColorBlue)
			case "red":
				run.Properties().SetHighlight(wml.ST_HighlightColorRed)
			}
		}
	}
}

func appendWordElements(doc *document.Document, elements []WordElement) {
	for _, element := range elements {
		if element.Type == "paragraph" && element.Paragraph != nil {
			pData := element.Paragraph
			var para document.Paragraph

			switch pData.Type {
			case "title":
				para = doc.AddParagraph()
				para.SetStyle("Title")
			case "heading1":
				para = doc.AddParagraph()
				para.SetStyle("Heading1")
			case "heading2":
				para = doc.AddParagraph()
				para.SetStyle("Heading2")
			case "heading3":
				para = doc.AddParagraph()
				para.SetStyle("Heading3")
			case "bullet":
				para = doc.AddParagraph()
				para.SetStyle("ListParagraph")
			default: // paragraph
				para = doc.AddParagraph()
			}
			addRunsToParagraph(para, pData.Runs)

		} else if element.Type == "table" && element.Table != nil {
			tData := element.Table
			tbl := doc.AddTable()
			for _, rData := range tData.Rows {
				row := tbl.AddRow()
				for _, cData := range rData.Cells {
					cell := row.AddCell()
					for _, cellParaData := range cData.Content {
						cellPara := cell.AddParagraph()
						addRunsToParagraph(cellPara, cellParaData.Runs)
					}
				}
			}
		}
	}
}

// -- Executors --

// CreateWord creates a new Word document.
func CreateWord(ctx context.Context, input *CreateWordInput) (string, error) {
	doc := document.New()
	appendWordElements(doc, input.Elements)

	if err := doc.SaveToFile(input.Filename); err != nil {
		return "", fmt.Errorf("failed to save docx: %v", err)
	}
	return fmt.Sprintf("Word document created successfully at %s", input.Filename), nil
}

// UpdateWord updates an existing Word document.
func UpdateWord(ctx context.Context, input *UpdateWordInput) (string, error) {
	doc, err := document.Open(input.Filename)
	if err != nil {
		return "", fmt.Errorf("failed to open docx: %v", err)
	}

	appendWordElements(doc, input.Elements)

	if err := doc.SaveToFile(input.Filename); err != nil {
		return "", fmt.Errorf("failed to save updated docx: %v", err)
	}
	return fmt.Sprintf("Word document updated successfully at %s", input.Filename), nil
}

// ReadWord reads content from a Word document.
func ReadWord(ctx context.Context, input *ReadWordInput) (string, error) {
	doc, err := document.Open(input.Filename)
	if err != nil {
		return "", fmt.Errorf("failed to open docx: %v", err)
	}

	markdown, err := doc.ToMarkdownWithImageURLs("")
	if err != nil {
		return "", fmt.Errorf("failed to convert to markdown: %v", err)
	}

	return markdown, nil
}

// -- Registration --

// NewOfficeWord registers the Word tools.
func NewOfficeWord(ctx context.Context) ([]tool.InvokableTool, error) {
	createTool, err := utils.InferTool(
		"office-word__create",
		"Create a standard Word document (.docx). Supports rich text (bold, italic, color) and tables.",
		CreateWord,
	)
	if err != nil {
		return nil, err
	}

	updateTool, err := utils.InferTool(
		"office-word__update",
		"Update an existing Word document by appending paragraphs or tables to the end.",
		UpdateWord,
	)
	if err != nil {
		return nil, err
	}

	readTool, err := utils.InferTool(
		"office-word__read",
		"Read text content from a Word document, including paragraphs and tables.",
		ReadWord,
	)
	if err != nil {
		return nil, err
	}

	return []tool.InvokableTool{createTool, updateTool, readTool}, nil
}
