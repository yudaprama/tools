package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/yudaprama/tools"
	"github.com/getkawai/unillm"
	"github.com/kawai-network/x/pdf/core"
	pdfextractor "github.com/kawai-network/x/pdf/extractor"
	pdfmodel "github.com/kawai-network/x/pdf/model"
)

var pdfWriterMetadataMu sync.Mutex

// PDFSearchReplaceInput defines input for PDF search and replace.
type PDFSearchReplaceInput struct {
	Pattern     string `json:"pattern" jsonschema:"description=Text pattern to search in PDF"`
	Replacement string `json:"replacement" jsonschema:"description=Replacement text"`
	Pages       string `json:"pages,omitempty" jsonschema:"description=Comma-separated page numbers (e.g. 1&#44;2) or '*' for all pages. Defaults to '*'"`
	InputPath   string `json:"inputPath" jsonschema:"description=Input PDF path"`
	OutputPath  string `json:"outputPath" jsonschema:"description=Output PDF path"`
}

// PDFSearchTextInput defines input for PDF text search.
type PDFSearchTextInput struct {
	Pattern   string `json:"pattern" jsonschema:"description=Text pattern to search in PDF"`
	Pages     string `json:"pages,omitempty" jsonschema:"description=Comma-separated page numbers (e.g. 1&#44;2) or '*' for all pages. Defaults to '*'"`
	InputPath string `json:"inputPath" jsonschema:"description=Input PDF path"`
}

// PDFExtractTextInput defines input for PDF text extraction.
type PDFExtractTextInput struct {
	Pages     string `json:"pages,omitempty" jsonschema:"description=Comma-separated page numbers (e.g. 1&#44;2) or '*' for all pages. Defaults to '*'"`
	InputPath string `json:"inputPath" jsonschema:"description=Input PDF path"`
}

// PDFMergeInput defines input for merging PDF files.
type PDFMergeInput struct {
	InputPaths []string `json:"inputPaths" jsonschema:"description=List of input PDF paths to merge in order"`
	OutputPath string   `json:"outputPath" jsonschema:"description=Output PDF path"`
}

// PDFSplitInput defines input for splitting PDF files.
type PDFSplitInput struct {
	InputPath string `json:"inputPath" jsonschema:"description=Input PDF path"`
	OutputDir string `json:"outputDir" jsonschema:"description=Output directory for split PDF files"`
	Ranges    string `json:"ranges,omitempty" jsonschema:"description=Comma-separated page ranges (e.g. 1-2&#44;3&#44;4-5). Defaults to one output per page"`
}

// PDFPageInfoInput defines input for page information extraction.
type PDFPageInfoInput struct {
	InputPath string `json:"inputPath" jsonschema:"description=Input PDF path"`
}

// PDFMetadataGetInput defines input for reading PDF metadata.
type PDFMetadataGetInput struct {
	InputPath string `json:"inputPath" jsonschema:"description=Input PDF path"`
}

// PDFMetadataSetInput defines input for writing PDF metadata.
type PDFMetadataSetInput struct {
	InputPath  string            `json:"inputPath" jsonschema:"description=Input PDF path"`
	OutputPath string            `json:"outputPath" jsonschema:"description=Output PDF path"`
	Metadata   map[string]string `json:"metadata" jsonschema:"description=Metadata fields to set: Title&#44; Author&#44; Subject&#44; Keywords&#44; Creator&#44; Producer"`
}

// PDFExtractImagesInput defines input for extracting images from PDF pages.
type PDFExtractImagesInput struct {
	InputPath string `json:"inputPath" jsonschema:"description=Input PDF path"`
	OutputDir string `json:"outputDir" jsonschema:"description=Output directory for extracted images"`
	Pages     string `json:"pages,omitempty" jsonschema:"description=Page selection like '*' or 1-3&#44;5. Defaults to '*'"`
	Format    string `json:"format,omitempty" jsonschema:"description=Output image format: png or jpg. Defaults to png"`
}

// RegisterPDF registers PDF tools backed by github.com/kawai-network/x/pdf.
func RegisterPDF(registry *tools.ToolRegistry) error {
	searchReplaceTool := unillm.NewParallelAgentTool("pdf_search_replace",
		"Search and replace text in PDF files using kawai-network/x/pdf",
		func(ctx context.Context, input PDFSearchReplaceInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			matches, err := doPDFSearchReplace(input)
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("pdf_search_replace failed: %v", err)), nil
			}
			return unillm.NewTextResponse(
				fmt.Sprintf(
					"Replaced %q with %q. Matches found on %d page(s). Output: %s",
					input.Pattern,
					input.Replacement,
					len(matches),
					input.OutputPath,
				),
			), nil
		},
	)
	if err := registry.Register(searchReplaceTool); err != nil {
		return err
	}

	searchTool := unillm.NewParallelAgentTool("pdf_search_text",
		"Search text in PDF files and return page-level match information",
		func(ctx context.Context, input PDFSearchTextInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			matches, err := doPDFSearch(input.Pattern, input.InputPath, input.Pages)
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("pdf_search_text failed: %v", err)), nil
			}
			payload, err := json.Marshal(map[string]any{
				"pattern": input.Pattern,
				"matches": matches,
			})
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("failed to encode response: %v", err)), nil
			}
			return unillm.NewTextResponse(string(payload)), nil
		},
	)
	if err := registry.Register(searchTool); err != nil {
		return err
	}

	extractTool := unillm.NewParallelAgentTool("pdf_extract_text",
		"Extract text from PDF pages",
		func(ctx context.Context, input PDFExtractTextInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			texts, err := doPDFExtract(input.InputPath, input.Pages)
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("pdf_extract_text failed: %v", err)), nil
			}
			payload, err := json.Marshal(map[string]any{"pages": texts})
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("failed to encode response: %v", err)), nil
			}
			return unillm.NewTextResponse(string(payload)), nil
		},
	)
	if err := registry.Register(extractTool); err != nil {
		return err
	}

	mergeTool := unillm.NewParallelAgentTool("pdf_merge",
		"Merge multiple PDF files into one output PDF",
		func(ctx context.Context, input PDFMergeInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			result, err := doPDFMerge(input)
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("pdf_merge failed: %v", err)), nil
			}
			payload, err := json.Marshal(result)
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("failed to encode response: %v", err)), nil
			}
			return unillm.NewTextResponse(string(payload)), nil
		},
	)
	if err := registry.Register(mergeTool); err != nil {
		return err
	}

	splitTool := unillm.NewParallelAgentTool("pdf_split",
		"Split a PDF into multiple output PDFs",
		func(ctx context.Context, input PDFSplitInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			result, err := doPDFSplit(input)
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("pdf_split failed: %v", err)), nil
			}
			payload, err := json.Marshal(result)
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("failed to encode response: %v", err)), nil
			}
			return unillm.NewTextResponse(string(payload)), nil
		},
	)
	if err := registry.Register(splitTool); err != nil {
		return err
	}

	pageInfoTool := unillm.NewParallelAgentTool("pdf_page_info",
		"Get PDF page count and page-level size/rotation information",
		func(ctx context.Context, input PDFPageInfoInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			result, err := doPDFPageInfo(input.InputPath)
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("pdf_page_info failed: %v", err)), nil
			}
			payload, err := json.Marshal(result)
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("failed to encode response: %v", err)), nil
			}
			return unillm.NewTextResponse(string(payload)), nil
		},
	)
	if err := registry.Register(pageInfoTool); err != nil {
		return err
	}

	metadataGetTool := unillm.NewParallelAgentTool("pdf_metadata_get",
		"Get document metadata from a PDF Info dictionary",
		func(ctx context.Context, input PDFMetadataGetInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			meta, err := readPDFMetadata(input.InputPath)
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("pdf_metadata_get failed: %v", err)), nil
			}
			payload, err := json.Marshal(map[string]any{"metadata": meta})
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("failed to encode response: %v", err)), nil
			}
			return unillm.NewTextResponse(string(payload)), nil
		},
	)
	if err := registry.Register(metadataGetTool); err != nil {
		return err
	}

	metadataSetTool := unillm.NewParallelAgentTool("pdf_metadata_set",
		"Set document metadata fields (Title, Author, Subject, Keywords, Creator, Producer)",
		func(ctx context.Context, input PDFMetadataSetInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			result, err := doPDFMetadataSet(input)
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("pdf_metadata_set failed: %v", err)), nil
			}
			payload, err := json.Marshal(result)
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("failed to encode response: %v", err)), nil
			}
			return unillm.NewTextResponse(string(payload)), nil
		},
	)
	if err := registry.Register(metadataSetTool); err != nil {
		return err
	}

	extractImagesTool := unillm.NewParallelAgentTool("pdf_extract_images",
		"Extract raster images from PDF pages",
		func(ctx context.Context, input PDFExtractImagesInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			result, err := doPDFExtractImages(input)
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("pdf_extract_images failed: %v", err)), nil
			}
			payload, err := json.Marshal(result)
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("failed to encode response: %v", err)), nil
			}
			return unillm.NewTextResponse(string(payload)), nil
		},
	)
	return registry.Register(extractImagesTool)
}

type pdfMergeResult struct {
	OutputPath string `json:"outputPath"`
	FileCount  int    `json:"fileCount"`
	PageCount  int    `json:"pageCount"`
}

type pdfSplitResult struct {
	InputPath   string   `json:"inputPath"`
	OutputPaths []string `json:"outputPaths"`
}

type pdfPageInfo struct {
	Page        int        `json:"page"`
	Width       float64    `json:"width"`
	Height      float64    `json:"height"`
	Rotation    int64      `json:"rotation"`
	MediaBox    [4]float64 `json:"mediaBox"`
	HasMediaBox bool       `json:"hasMediaBox"`
}

type pdfPageInfoResult struct {
	InputPath string        `json:"inputPath"`
	PageCount int           `json:"pageCount"`
	Pages     []pdfPageInfo `json:"pages"`
}

type pdfMetadataSetResult struct {
	InputPath  string            `json:"inputPath"`
	OutputPath string            `json:"outputPath"`
	Metadata   map[string]string `json:"metadata"`
}

type pdfExtractedImageInfo struct {
	Page   int     `json:"page"`
	Index  int     `json:"index"`
	Path   string  `json:"path"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Angle  float64 `json:"angle"`
	Format string  `json:"format"`
}

type pdfExtractImagesResult struct {
	InputPath string                  `json:"inputPath"`
	OutputDir string                  `json:"outputDir"`
	Images    []pdfExtractedImageInfo `json:"images"`
}

func doPDFSearchReplace(input PDFSearchReplaceInput) (map[int]pdfextractor.Match, error) {
	if strings.TrimSpace(input.Pattern) == "" {
		return nil, fmt.Errorf("pattern is required")
	}
	if input.InputPath == "" {
		return nil, fmt.Errorf("inputPath is required")
	}
	if input.OutputPath == "" {
		return nil, fmt.Errorf("outputPath is required")
	}

	editor, pages, closeFn, err := openEditorAndPages(input.InputPath, input.Pages)
	if err != nil {
		return nil, err
	}
	defer closeFn()

	matches, err := editor.Search(input.Pattern, pages)
	if err != nil {
		return nil, err
	}
	if err := editor.Replace(input.Pattern, input.Replacement, pages); err != nil {
		return nil, err
	}
	if err := editor.WriteToFile(input.OutputPath); err != nil {
		return nil, err
	}

	return matches, nil
}

func doPDFSearch(pattern, inputPath, rawPages string) (map[int]pdfextractor.Match, error) {
	if strings.TrimSpace(pattern) == "" {
		return nil, fmt.Errorf("pattern is required")
	}
	if strings.TrimSpace(inputPath) == "" {
		return nil, fmt.Errorf("inputPath is required")
	}

	editor, pages, closeFn, err := openEditorAndPages(inputPath, rawPages)
	if err != nil {
		return nil, err
	}
	defer closeFn()

	return editor.Search(pattern, pages)
}

func doPDFExtract(inputPath, rawPages string) (map[int]string, error) {
	if strings.TrimSpace(inputPath) == "" {
		return nil, fmt.Errorf("inputPath is required")
	}

	inFile, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open input PDF: %w", err)
	}
	defer inFile.Close()

	reader, err := pdfmodel.NewPdfReader(inFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF reader: %w", err)
	}

	pages, err := parsePDFPages(rawPages)
	if err != nil {
		return nil, err
	}
	if pages == nil {
		n, err := reader.GetNumPages()
		if err != nil {
			return nil, err
		}
		pages = make([]int, n)
		for i := 1; i <= n; i++ {
			pages[i-1] = i
		}
	}

	res := map[int]string{}
	for _, pageNum := range pages {
		page, err := reader.GetPage(pageNum)
		if err != nil {
			return nil, err
		}
		ex, err := pdfextractor.New(page)
		if err != nil {
			return nil, err
		}
		text, err := ex.ExtractText()
		if err != nil {
			return nil, err
		}
		res[pageNum] = text
	}
	return res, nil
}

func doPDFMerge(input PDFMergeInput) (*pdfMergeResult, error) {
	if len(input.InputPaths) == 0 {
		return nil, fmt.Errorf("inputPaths is required")
	}
	if strings.TrimSpace(input.OutputPath) == "" {
		return nil, fmt.Errorf("outputPath is required")
	}

	totalPages := 0
	err := withPDFWriterMetadata(map[string]string{}, func() error {
		writer := pdfmodel.NewPdfWriter()
		for _, path := range input.InputPaths {
			if strings.TrimSpace(path) == "" {
				return fmt.Errorf("inputPaths contains empty path")
			}

			f, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open %s: %w", path, err)
			}

			reader, err := pdfmodel.NewPdfReader(f)
			if err != nil {
				_ = f.Close()
				return fmt.Errorf("failed to read %s: %w", path, err)
			}

			n, err := reader.GetNumPages()
			if err != nil {
				_ = f.Close()
				return fmt.Errorf("failed to get page count for %s: %w", path, err)
			}
			for i := 1; i <= n; i++ {
				page, err := reader.GetPage(i)
				if err != nil {
					_ = f.Close()
					return fmt.Errorf("failed to read page %d from %s: %w", i, path, err)
				}
				if err := writer.AddPage(page); err != nil {
					_ = f.Close()
					return fmt.Errorf("failed to add page %d from %s: %w", i, path, err)
				}
				totalPages++
			}
			_ = f.Close()
		}

		out, err := os.Create(input.OutputPath)
		if err != nil {
			return fmt.Errorf("failed to create output PDF: %w", err)
		}
		defer out.Close()

		if err := writer.Write(out); err != nil {
			return fmt.Errorf("failed to write output PDF: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &pdfMergeResult{
		OutputPath: input.OutputPath,
		FileCount:  len(input.InputPaths),
		PageCount:  totalPages,
	}, nil
}

func doPDFSplit(input PDFSplitInput) (*pdfSplitResult, error) {
	if strings.TrimSpace(input.InputPath) == "" {
		return nil, fmt.Errorf("inputPath is required")
	}
	if strings.TrimSpace(input.OutputDir) == "" {
		return nil, fmt.Errorf("outputDir is required")
	}
	if err := os.MkdirAll(input.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	f, err := os.Open(input.InputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open input PDF: %w", err)
	}
	defer f.Close()

	reader, err := pdfmodel.NewPdfReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF reader: %w", err)
	}

	n, err := reader.GetNumPages()
	if err != nil {
		return nil, err
	}
	groups, err := parseSplitRanges(input.Ranges, n)
	if err != nil {
		return nil, err
	}

	outputs := make([]string, 0, len(groups))
	for idx, group := range groups {
		outputPath := filepath.ToSlash(filepath.Join(input.OutputDir, fmt.Sprintf("part_%03d.pdf", idx+1)))

		err := withPDFWriterMetadata(map[string]string{}, func() error {
			writer := pdfmodel.NewPdfWriter()
			for _, pageNum := range group {
				page, err := reader.GetPage(pageNum)
				if err != nil {
					return fmt.Errorf("failed to get page %d: %w", pageNum, err)
				}
				if err := writer.AddPage(page); err != nil {
					return fmt.Errorf("failed to add page %d: %w", pageNum, err)
				}
			}

			out, err := os.Create(outputPath)
			if err != nil {
				return fmt.Errorf("failed to create split output %s: %w", outputPath, err)
			}
			defer out.Close()

			if err := writer.Write(out); err != nil {
				return fmt.Errorf("failed to write split output %s: %w", outputPath, err)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		outputs = append(outputs, outputPath)
	}

	return &pdfSplitResult{
		InputPath:   input.InputPath,
		OutputPaths: outputs,
	}, nil
}

func doPDFPageInfo(inputPath string) (*pdfPageInfoResult, error) {
	if strings.TrimSpace(inputPath) == "" {
		return nil, fmt.Errorf("inputPath is required")
	}

	f, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open input PDF: %w", err)
	}
	defer f.Close()

	reader, err := pdfmodel.NewPdfReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF reader: %w", err)
	}

	n, err := reader.GetNumPages()
	if err != nil {
		return nil, err
	}
	pages := make([]pdfPageInfo, 0, n)
	for i := 1; i <= n; i++ {
		page, err := reader.GetPage(i)
		if err != nil {
			return nil, fmt.Errorf("failed to get page %d: %w", i, err)
		}

		info := pdfPageInfo{Page: i}
		if page.Rotate != nil {
			info.Rotation = *page.Rotate
		}

		if mediaBox, err := page.GetMediaBox(); err == nil && mediaBox != nil {
			info.Width = mediaBox.Width()
			info.Height = mediaBox.Height()
			info.MediaBox = [4]float64{mediaBox.Llx, mediaBox.Lly, mediaBox.Urx, mediaBox.Ury}
			info.HasMediaBox = true
		}

		pages = append(pages, info)
	}

	return &pdfPageInfoResult{
		InputPath: inputPath,
		PageCount: n,
		Pages:     pages,
	}, nil
}

func doPDFMetadataSet(input PDFMetadataSetInput) (*pdfMetadataSetResult, error) {
	if strings.TrimSpace(input.InputPath) == "" {
		return nil, fmt.Errorf("inputPath is required")
	}
	if strings.TrimSpace(input.OutputPath) == "" {
		return nil, fmt.Errorf("outputPath is required")
	}
	if len(input.Metadata) == 0 {
		return nil, fmt.Errorf("metadata is required")
	}

	current, err := readPDFMetadata(input.InputPath)
	if err != nil {
		return nil, err
	}

	final := map[string]string{}
	for _, key := range []string{"Title", "Author", "Subject", "Keywords", "Creator", "Producer"} {
		if val, ok := current[key]; ok {
			final[key] = val
		}
	}

	for key, val := range input.Metadata {
		canonical, ok := canonicalPDFMetadataKey(key)
		if !ok {
			return nil, fmt.Errorf("unsupported metadata key: %s", key)
		}
		final[canonical] = val
	}

	err = withPDFWriterMetadata(final, func() error {
		inFile, err := os.Open(input.InputPath)
		if err != nil {
			return fmt.Errorf("failed to open input PDF: %w", err)
		}
		defer inFile.Close()

		reader, err := pdfmodel.NewPdfReader(inFile)
		if err != nil {
			return fmt.Errorf("failed to create PDF reader: %w", err)
		}

		writer := pdfmodel.NewPdfWriter()
		n, err := reader.GetNumPages()
		if err != nil {
			return err
		}
		for i := 1; i <= n; i++ {
			page, err := reader.GetPage(i)
			if err != nil {
				return fmt.Errorf("failed to read page %d: %w", i, err)
			}
			if err := writer.AddPage(page); err != nil {
				return fmt.Errorf("failed to add page %d: %w", i, err)
			}
		}

		outFile, err := os.Create(input.OutputPath)
		if err != nil {
			return fmt.Errorf("failed to create output PDF: %w", err)
		}
		defer outFile.Close()

		if err := writer.Write(outFile); err != nil {
			return fmt.Errorf("failed to write output PDF: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	updated, err := readPDFMetadata(input.OutputPath)
	if err != nil {
		return nil, err
	}

	return &pdfMetadataSetResult{
		InputPath:  input.InputPath,
		OutputPath: input.OutputPath,
		Metadata:   updated,
	}, nil
}

func doPDFExtractImages(input PDFExtractImagesInput) (*pdfExtractImagesResult, error) {
	if strings.TrimSpace(input.InputPath) == "" {
		return nil, fmt.Errorf("inputPath is required")
	}
	if strings.TrimSpace(input.OutputDir) == "" {
		return nil, fmt.Errorf("outputDir is required")
	}
	if err := os.MkdirAll(input.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	format := strings.ToLower(strings.TrimSpace(input.Format))
	if format == "" {
		format = "png"
	}
	ext := format
	switch format {
	case "png":
		ext = "png"
	case "jpg", "jpeg":
		format = "jpg"
		ext = "jpg"
	default:
		return nil, fmt.Errorf("unsupported format %q, use png or jpg", format)
	}

	f, err := os.Open(input.InputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open input PDF: %w", err)
	}
	defer f.Close()

	reader, err := pdfmodel.NewPdfReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF reader: %w", err)
	}

	n, err := reader.GetNumPages()
	if err != nil {
		return nil, err
	}
	pages, err := parsePDFPageSpec(input.Pages, n)
	if err != nil {
		return nil, err
	}

	result := &pdfExtractImagesResult{
		InputPath: input.InputPath,
		OutputDir: input.OutputDir,
		Images:    make([]pdfExtractedImageInfo, 0),
	}

	for _, pageNum := range pages {
		page, err := reader.GetPage(pageNum)
		if err != nil {
			return nil, fmt.Errorf("failed to get page %d: %w", pageNum, err)
		}

		ex, err := pdfextractor.New(page)
		if err != nil {
			return nil, fmt.Errorf("failed to build extractor for page %d: %w", pageNum, err)
		}
		pageImages, err := ex.ExtractPageImages(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to extract images from page %d: %w", pageNum, err)
		}

		for i, mark := range pageImages.Images {
			if mark.Image == nil {
				continue
			}
			goimg, err := mark.Image.ToGoImage()
			if err != nil {
				return nil, fmt.Errorf("failed to decode image page %d index %d: %w", pageNum, i+1, err)
			}

			outputPath := filepath.ToSlash(filepath.Join(input.OutputDir, fmt.Sprintf("page_%03d_img_%03d.%s", pageNum, i+1, ext)))
			out, err := os.Create(outputPath)
			if err != nil {
				return nil, fmt.Errorf("failed to create image file %s: %w", outputPath, err)
			}

			switch format {
			case "png":
				err = png.Encode(out, goimg)
			case "jpg":
				err = jpeg.Encode(out, goimg, &jpeg.Options{Quality: 90})
			}
			closeErr := out.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to encode image %s: %w", outputPath, err)
			}
			if closeErr != nil {
				return nil, fmt.Errorf("failed to close image file %s: %w", outputPath, closeErr)
			}

			result.Images = append(result.Images, pdfExtractedImageInfo{
				Page:   pageNum,
				Index:  i + 1,
				Path:   outputPath,
				Width:  mark.Width,
				Height: mark.Height,
				X:      mark.X,
				Y:      mark.Y,
				Angle:  mark.Angle,
				Format: format,
			})
		}
	}

	return result, nil
}

func readPDFMetadata(inputPath string) (map[string]string, error) {
	if strings.TrimSpace(inputPath) == "" {
		return nil, fmt.Errorf("inputPath is required")
	}

	f, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open input PDF: %w", err)
	}
	defer f.Close()

	reader, err := pdfmodel.NewPdfReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF reader: %w", err)
	}

	trailer, err := reader.GetTrailer()
	if err != nil {
		return nil, err
	}

	meta := map[string]string{}
	infoObj := trailer.Get("Info")
	if infoObj == nil {
		return meta, nil
	}
	infoDict, ok := core.GetDict(infoObj)
	if !ok {
		return nil, fmt.Errorf("invalid Info dictionary")
	}

	for _, key := range infoDict.Keys() {
		meta[string(key)] = pdfObjectToString(infoDict.Get(key))
	}
	return meta, nil
}

func openEditorAndPages(inputPath, rawPages string) (*pdfextractor.Editor, []int, func(), error) {
	inFile, err := os.Open(inputPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to open input PDF: %w", err)
	}

	reader, err := pdfmodel.NewPdfReader(inFile)
	if err != nil {
		_ = inFile.Close()
		return nil, nil, nil, fmt.Errorf("failed to create PDF reader: %w", err)
	}

	pages, err := parsePDFPages(rawPages)
	if err != nil {
		_ = inFile.Close()
		return nil, nil, nil, err
	}

	return pdfextractor.NewEditor(reader), pages, func() { _ = inFile.Close() }, nil
}

func parsePDFPages(raw string) ([]int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "*" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	seen := map[int]struct{}{}
	pages := make([]int, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, err
		}
		if n < 1 {
			return nil, fmt.Errorf("invalid page number: %d", n)
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		pages = append(pages, n)
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no pages provided")
	}
	sort.Ints(pages)
	return pages, nil
}

func parsePDFPageSpec(raw string, maxPage int) ([]int, error) {
	if maxPage < 1 {
		return nil, fmt.Errorf("pdf has no pages")
	}

	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "*" {
		pages := make([]int, maxPage)
		for i := 1; i <= maxPage; i++ {
			pages[i-1] = i
		}
		return pages, nil
	}

	tokens := strings.Split(raw, ",")
	seen := map[int]struct{}{}
	pages := make([]int, 0, len(tokens))

	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}

		if strings.Contains(token, "-") {
			parts := strings.SplitN(token, "-", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid page range: %s", token)
			}
			start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid page range start %q: %w", parts[0], err)
			}
			end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid page range end %q: %w", parts[1], err)
			}
			if start < 1 || end < 1 || start > end || end > maxPage {
				return nil, fmt.Errorf("invalid page range: %s", token)
			}
			for i := start; i <= end; i++ {
				if _, ok := seen[i]; ok {
					continue
				}
				seen[i] = struct{}{}
				pages = append(pages, i)
			}
			continue
		}

		pageNum, err := strconv.Atoi(token)
		if err != nil {
			return nil, fmt.Errorf("invalid page %q: %w", token, err)
		}
		if pageNum < 1 || pageNum > maxPage {
			return nil, fmt.Errorf("invalid page number: %d", pageNum)
		}
		if _, ok := seen[pageNum]; ok {
			continue
		}
		seen[pageNum] = struct{}{}
		pages = append(pages, pageNum)
	}

	if len(pages) == 0 {
		return nil, fmt.Errorf("no valid pages in spec")
	}
	return pages, nil
}

func parseSplitRanges(raw string, maxPage int) ([][]int, error) {
	if maxPage < 1 {
		return nil, fmt.Errorf("pdf has no pages")
	}

	raw = strings.TrimSpace(raw)
	if raw == "" {
		groups := make([][]int, 0, maxPage)
		for i := 1; i <= maxPage; i++ {
			groups = append(groups, []int{i})
		}
		return groups, nil
	}

	tokens := strings.Split(raw, ",")
	groups := make([][]int, 0, len(tokens))
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}

		if strings.Contains(token, "-") {
			parts := strings.SplitN(token, "-", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid range: %s", token)
			}
			start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid range start %q: %w", parts[0], err)
			}
			end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid range end %q: %w", parts[1], err)
			}
			if start < 1 || end < 1 || start > end || end > maxPage {
				return nil, fmt.Errorf("invalid range: %s", token)
			}

			group := make([]int, 0, end-start+1)
			for i := start; i <= end; i++ {
				group = append(group, i)
			}
			groups = append(groups, group)
			continue
		}

		pageNum, err := strconv.Atoi(token)
		if err != nil {
			return nil, fmt.Errorf("invalid page %q: %w", token, err)
		}
		if pageNum < 1 || pageNum > maxPage {
			return nil, fmt.Errorf("invalid page number: %d", pageNum)
		}
		groups = append(groups, []int{pageNum})
	}

	if len(groups) == 0 {
		return nil, fmt.Errorf("no valid ranges")
	}
	return groups, nil
}

func canonicalPDFMetadataKey(key string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "title":
		return "Title", true
	case "author":
		return "Author", true
	case "subject":
		return "Subject", true
	case "keywords":
		return "Keywords", true
	case "creator":
		return "Creator", true
	case "producer":
		return "Producer", true
	default:
		return "", false
	}
}

func pdfObjectToString(obj core.PdfObject) string {
	if obj == nil {
		return ""
	}

	switch v := core.TraceToDirectObject(obj).(type) {
	case *core.PdfObjectString:
		return v.Decoded()
	case *core.PdfObjectName:
		return string(*v)
	case *core.PdfObjectInteger:
		return strconv.FormatInt(int64(*v), 10)
	case *core.PdfObjectFloat:
		return strconv.FormatFloat(float64(*v), 'f', -1, 64)
	case *core.PdfObjectBool:
		if bool(*v) {
			return "true"
		}
		return "false"
	default:
		return core.TraceToDirectObject(obj).String()
	}
}

func withPDFWriterMetadata(meta map[string]string, fn func() error) error {
	pdfWriterMetadataMu.Lock()
	defer pdfWriterMetadataMu.Unlock()

	setPDFWriterMetadata(meta)
	defer setPDFWriterMetadata(map[string]string{})

	return fn()
}

func setPDFWriterMetadata(meta map[string]string) {
	pdfmodel.SetPdfTitle(meta["Title"])
	pdfmodel.SetPdfAuthor(meta["Author"])
	pdfmodel.SetPdfSubject(meta["Subject"])
	pdfmodel.SetPdfKeywords(meta["Keywords"])
	pdfmodel.SetPdfCreator(meta["Creator"])
	pdfmodel.SetPdfProducer(meta["Producer"])
}
