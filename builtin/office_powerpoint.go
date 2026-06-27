package builtin

import (
	"context"
	"fmt"

	"github.com/getkawai/unillm"
	"github.com/yudaprama/tools"
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
func CreatePowerPoint(ctx context.Context, input CreatePowerPointInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
	ppt := presentation.New()
	addSlidesToPres(ppt, input.Slides)

	if err := ppt.SaveToFile(input.Filename); err != nil {
		return unillm.NewTextErrorResponse(fmt.Sprintf("failed to save pptx: %v", err)), nil
	}
	return unillm.NewTextResponse(fmt.Sprintf("PowerPoint presentation created successfully at %s", input.Filename)), nil
}

// UpdatePowerPoint creates a new PowerPoint presentation.
func UpdatePowerPoint(ctx context.Context, input UpdatePowerPointInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
	ppt, err := presentation.Open(input.Filename)
	if err != nil {
		return unillm.NewTextErrorResponse(fmt.Sprintf("failed to open pptx: %v", err)), nil
	}

	addSlidesToPres(ppt, input.Slides)

	if err := ppt.SaveToFile(input.Filename); err != nil {
		return unillm.NewTextErrorResponse(fmt.Sprintf("failed to save updated pptx: %v", err)), nil
	}
	return unillm.NewTextResponse(fmt.Sprintf("PowerPoint presentation updated successfully at %s", input.Filename)), nil
}

// ReadPowerPoint reads content from a PowerPoint presentation.
func ReadPowerPoint(ctx context.Context, input ReadPowerPointInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
	ppt, err := presentation.Open(input.Filename)
	if err != nil {
		return unillm.NewTextErrorResponse(fmt.Sprintf("failed to open pptx: %v", err)), nil
	}

	markdown, err := ppt.ToMarkdownWithImageURLs("")
	if err != nil {
		return unillm.NewTextErrorResponse(fmt.Sprintf("failed to convert to markdown: %v", err)), nil
	}

	return unillm.NewTextResponse(markdown), nil
}

// -- Registration --

// RegisterOfficePowerPoint registers the PowerPoint tools.
func RegisterOfficePowerPoint(registry *tools.ToolRegistry) error {
	createTool := unillm.NewAgentTool(
		"office-powerpoint__create",
		"Create a standard Presentation (.pptx).",
		CreatePowerPoint,
	)
	if err := registry.Register(createTool); err != nil {
		return err
	}

	updateTool := unillm.NewAgentTool(
		"office-powerpoint__update",
		"Update an existing Presentation by appending slides.",
		UpdatePowerPoint,
	)
	if err := registry.Register(updateTool); err != nil {
		return err
	}

	readTool := unillm.NewAgentTool(
		"office-powerpoint__read",
		"Read text content from a Presentation.",
		ReadPowerPoint,
	)
	return registry.Register(readTool)
}
