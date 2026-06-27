package builtin

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/yudaprama/tools/gooxml/measurement"
	"github.com/yudaprama/tools/gooxml/presentation"
)

// -- Data Structures --

// PowerPointSlide content for PowerPoint presentations
type PowerPointSlide struct {
	Title string   `json:"title,omitempty" jsonschema:"description=Title for the slide"`
	Items []string `json:"items,omitempty" jsonschema:"description=Bullet points for the slide body"`
}

// -- Inputs --

// CreatePowerPointInput defines input for creating PowerPoint presentations
type CreatePowerPointInput struct {
	Filename string            `json:"filename" jsonschema:"description=Output filename (e.g. pres.pptx)"`
	Slides   []PowerPointSlide `json:"slides" jsonschema:"description=List of slides"`
}

// UpdatePowerPointInput defines input for creating PowerPoint presentations
type UpdatePowerPointInput struct {
	Filename string            `json:"filename" jsonschema:"description=Filename of existing presentation to update"`
	Slides   []PowerPointSlide `json:"slides" jsonschema:"description=List of slides to append"`
}

// ReadPowerPointInput defines input for reading PowerPoint presentations
type ReadPowerPointInput struct {
	Filename string `json:"filename" jsonschema:"description=Filename of presentation to read"`
}

// -- Helpers --

func addSlidesToPres(ppt *presentation.Presentation, slides []PowerPointSlide) {
	for _, item := range slides {
		slide := ppt.AddSlide()

		// Title
		if item.Title != "" {
			textBox := slide.AddTextBox()
			textBox.Properties().SetPosition(0, 0)
			textBox.Properties().SetSize(600*measurement.Point, 100*measurement.Point) // Rough positioning
			para := textBox.AddParagraph()
			run := para.AddRun()
			run.SetText(item.Title)
			run.Properties().SetSize(24)
		}

		// Bullet items
		if len(item.Items) > 0 {
			textBox := slide.AddTextBox()
			textBox.Properties().SetPosition(50*measurement.Point, 150*measurement.Point)
			textBox.Properties().SetSize(500*measurement.Point, 400*measurement.Point) // Rough positioning
			for _, bulletPoint := range item.Items {
				para := textBox.AddParagraph()
				para.Properties().SetBulletChar("•")
				run := para.AddRun()
				run.SetText(bulletPoint)
			}
		}
	}
}

// -- Executors --

// CreatePowerPoint creates a new PowerPoint presentation.
func CreatePowerPoint(ctx context.Context, input *CreatePowerPointInput) (string, error) {
	ppt := presentation.New()
	addSlidesToPres(ppt, input.Slides)

	if err := ppt.SaveToFile(input.Filename); err != nil {
		return "", fmt.Errorf("failed to save pptx: %v", err)
	}
	return fmt.Sprintf("PowerPoint presentation created successfully at %s", input.Filename), nil
}

// UpdatePowerPoint creates a new PowerPoint presentation.
func UpdatePowerPoint(ctx context.Context, input *UpdatePowerPointInput) (string, error) {
	ppt, err := presentation.Open(input.Filename)
	if err != nil {
		return "", fmt.Errorf("failed to open pptx: %v", err)
	}

	addSlidesToPres(ppt, input.Slides)

	if err := ppt.SaveToFile(input.Filename); err != nil {
		return "", fmt.Errorf("failed to save updated pptx: %v", err)
	}
	return fmt.Sprintf("PowerPoint presentation updated successfully at %s", input.Filename), nil
}

// ReadPowerPoint reads content from a PowerPoint presentation.
func ReadPowerPoint(ctx context.Context, input *ReadPowerPointInput) (string, error) {
	ppt, err := presentation.Open(input.Filename)
	if err != nil {
		return "", fmt.Errorf("failed to open pptx: %v", err)
	}

	markdown, err := ppt.ToMarkdownWithImageURLs("")
	if err != nil {
		return "", fmt.Errorf("failed to convert to markdown: %v", err)
	}

	return markdown, nil
}

// -- Registration --

// NewOfficePowerPoint registers the PowerPoint tools.
func NewOfficePowerPoint(ctx context.Context) ([]tool.InvokableTool, error) {
	createTool, err := utils.InferTool(
		"office-powerpoint__create",
		"Create a standard Presentation (.pptx).",
		CreatePowerPoint,
	)
	if err != nil {
		return nil, err
	}

	updateTool, err := utils.InferTool(
		"office-powerpoint__update",
		"Update an existing Presentation by appending slides.",
		UpdatePowerPoint,
	)
	if err != nil {
		return nil, err
	}

	readTool, err := utils.InferTool(
		"office-powerpoint__read",
		"Read text content from a Presentation.",
		ReadPowerPoint,
	)
	if err != nil {
		return nil, err
	}

	return []tool.InvokableTool{createTool, updateTool, readTool}, nil
}
