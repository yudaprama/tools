package builtin

import (
	"context"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yudaprama/tools"
	"github.com/getkawai/unillm"
	pdfcreator "github.com/kawai-network/x/pdf/creator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPDFOperations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pdf_builtin_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.ToSlash(filepath.Join(tmpDir, "input.pdf"))
	outputPath := filepath.ToSlash(filepath.Join(tmpDir, "output.pdf"))

	require.NoError(t, createTestPDF(inputPath, "Hello PDF world. Hello again."))

	matches, err := doPDFSearch("Hello", inputPath, "*")
	require.NoError(t, err)
	require.Contains(t, matches, 1)
	require.GreaterOrEqual(t, len(matches[1].Indexes), 2)

	texts, err := doPDFExtract(inputPath, "1")
	require.NoError(t, err)
	require.Contains(t, texts, 1)
	assert.Contains(t, texts[1], "Hello PDF world")

	_, err = doPDFSearchReplace(PDFSearchReplaceInput{
		Pattern:     "Hello",
		Replacement: "Hi",
		Pages:       "*",
		InputPath:   inputPath,
		OutputPath:  outputPath,
	})
	require.NoError(t, err)

	after, err := doPDFExtract(outputPath, "*")
	require.NoError(t, err)
	require.Contains(t, after, 1)
	assert.Contains(t, after[1], "Hi PDF world")
	assert.NotContains(t, after[1], "Hello PDF world")
}

func TestPDFMergeSplitPageInfoMetadataAndImages(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pdf_builtin_advanced_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	inputA := filepath.ToSlash(filepath.Join(tmpDir, "input_a.pdf"))
	inputB := filepath.ToSlash(filepath.Join(tmpDir, "input_b.pdf"))
	merged := filepath.ToSlash(filepath.Join(tmpDir, "merged.pdf"))
	metadataOut := filepath.ToSlash(filepath.Join(tmpDir, "metadata.pdf"))
	splitDir := filepath.ToSlash(filepath.Join(tmpDir, "split"))
	imgInput := filepath.ToSlash(filepath.Join(tmpDir, "with_image.pdf"))
	imgOutDir := filepath.ToSlash(filepath.Join(tmpDir, "images"))

	require.NoError(t, createTestPDFWithPages(inputA, []string{"Page A1", "Page A2"}))
	require.NoError(t, createTestPDFWithPages(inputB, []string{"Page B1"}))

	mergeResult, err := doPDFMerge(PDFMergeInput{
		InputPaths: []string{inputA, inputB},
		OutputPath: merged,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, mergeResult.FileCount)
	assert.Equal(t, 3, mergeResult.PageCount)

	pageInfo, err := doPDFPageInfo(merged)
	require.NoError(t, err)
	assert.Equal(t, 3, pageInfo.PageCount)
	require.Len(t, pageInfo.Pages, 3)
	assert.True(t, pageInfo.Pages[0].HasMediaBox)
	assert.Greater(t, pageInfo.Pages[0].Width, 0.0)
	assert.Greater(t, pageInfo.Pages[0].Height, 0.0)

	splitResult, err := doPDFSplit(PDFSplitInput{
		InputPath: merged,
		OutputDir: splitDir,
		Ranges:    "1-2,3",
	})
	require.NoError(t, err)
	require.Len(t, splitResult.OutputPaths, 2)

	firstSplitInfo, err := doPDFPageInfo(splitResult.OutputPaths[0])
	require.NoError(t, err)
	assert.Equal(t, 2, firstSplitInfo.PageCount)
	secondSplitInfo, err := doPDFPageInfo(splitResult.OutputPaths[1])
	require.NoError(t, err)
	assert.Equal(t, 1, secondSplitInfo.PageCount)

	metadataResult, err := doPDFMetadataSet(PDFMetadataSetInput{
		InputPath:  merged,
		OutputPath: metadataOut,
		Metadata: map[string]string{
			"Title":  "Test Title",
			"Author": "Test Author",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "Test Title", metadataResult.Metadata["Title"])
	assert.Equal(t, "Test Author", metadataResult.Metadata["Author"])

	meta, err := readPDFMetadata(metadataOut)
	require.NoError(t, err)
	assert.Equal(t, "Test Title", meta["Title"])
	assert.Equal(t, "Test Author", meta["Author"])

	require.NoError(t, createTestPDFWithImage(imgInput))
	imagesResult, err := doPDFExtractImages(PDFExtractImagesInput{
		InputPath: imgInput,
		OutputDir: imgOutDir,
		Format:    "png",
	})
	require.NoError(t, err)
	require.NotEmpty(t, imagesResult.Images)
	for _, img := range imagesResult.Images {
		_, statErr := os.Stat(img.Path)
		require.NoError(t, statErr)
	}
}

func TestParsePDFPages(t *testing.T) {
	pages, err := parsePDFPages("3,1,2,2")
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, pages)

	pages, err = parsePDFPages("*")
	require.NoError(t, err)
	assert.Nil(t, pages)

	_, err = parsePDFPages("0")
	require.Error(t, err)

	_, err = parsePDFPages("a")
	require.Error(t, err)
}

func TestParsePDFPageSpec(t *testing.T) {
	pages, err := parsePDFPageSpec("1-3,5", 5)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3, 5}, pages)

	pages, err = parsePDFPageSpec("*", 3)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, pages)

	_, err = parsePDFPageSpec("2-1", 3)
	require.Error(t, err)

	_, err = parsePDFPageSpec("6", 5)
	require.Error(t, err)
}

func TestRegisterPDFTools(t *testing.T) {
	registry := tools.NewToolRegistry()
	require.NoError(t, RegisterPDF(registry))

	for _, name := range []string{
		"pdf_search_replace",
		"pdf_search_text",
		"pdf_extract_text",
		"pdf_merge",
		"pdf_split",
		"pdf_page_info",
		"pdf_metadata_get",
		"pdf_metadata_set",
		"pdf_extract_images",
	} {
		_, ok := registry.Get(name)
		require.True(t, ok, "tool %s should be registered", name)
	}
}

func TestPDFToolsRun(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pdf_builtin_run_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.ToSlash(filepath.Join(tmpDir, "input.pdf"))
	outputPath := filepath.ToSlash(filepath.Join(tmpDir, "output.pdf"))
	require.NoError(t, createTestPDF(inputPath, "Search me please"))

	registry := tools.NewToolRegistry()
	require.NoError(t, RegisterPDF(registry))

	searchTool, ok := registry.Get("pdf_search_text")
	require.True(t, ok)
	resp, err := searchTool.Run(context.Background(), unillm.ToolCall{
		Name:  "pdf_search_text",
		Input: `{"pattern":"Search","inputPath":"` + inputPath + `"}`,
	})
	require.NoError(t, err)
	assert.False(t, resp.IsError)
	assert.True(t, strings.Contains(resp.Content, "Search"))

	extractTool, ok := registry.Get("pdf_extract_text")
	require.True(t, ok)
	resp, err = extractTool.Run(context.Background(), unillm.ToolCall{
		Name:  "pdf_extract_text",
		Input: `{"inputPath":"` + inputPath + `"}`,
	})
	require.NoError(t, err)
	assert.False(t, resp.IsError)
	assert.True(t, strings.Contains(resp.Content, "Search me please"))

	replaceTool, ok := registry.Get("pdf_search_replace")
	require.True(t, ok)
	resp, err = replaceTool.Run(context.Background(), unillm.ToolCall{
		Name: "pdf_search_replace",
		Input: `{"pattern":"Search","replacement":"Find","inputPath":"` + inputPath +
			`","outputPath":"` + outputPath + `"}`,
	})
	require.NoError(t, err)
	assert.False(t, resp.IsError)

	after, err := doPDFExtract(outputPath, "*")
	require.NoError(t, err)
	assert.Contains(t, after[1], "Find me please")
}

func createTestPDF(path, text string) error {
	c := pdfcreator.New()
	p := c.NewParagraph(text)
	if err := c.Draw(p); err != nil {
		return err
	}
	return c.WriteToFile(path)
}

func createTestPDFWithPages(path string, texts []string) error {
	if len(texts) == 0 {
		texts = []string{""}
	}

	c := pdfcreator.New()
	for i, text := range texts {
		if i > 0 {
			c.NewPage()
		}
		p := c.NewParagraph(text)
		if err := c.Draw(p); err != nil {
			return err
		}
	}
	return c.WriteToFile(path)
}

func createTestPDFWithImage(path string) error {
	c := pdfcreator.New()

	img := image.NewRGBA(image.Rect(0, 0, 24, 24))
	for y := 0; y < 24; y++ {
		for x := 0; x < 24; x++ {
			img.Set(x, y, color.RGBA{R: uint8(10 * x), G: uint8(10 * y), B: 120, A: 255})
		}
	}

	pdfImg, err := c.NewImageFromGoImage(img)
	if err != nil {
		return err
	}
	pdfImg.ScaleToWidth(120)
	if err := c.Draw(pdfImg); err != nil {
		return err
	}

	return c.WriteToFile(path)
}
