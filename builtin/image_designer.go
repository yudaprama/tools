package builtin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/google/uuid"
	"github.com/yudaprama/tools/image"
)

// Text2ImageInput defines input for text2image tool
type Text2ImageInput struct {
	Prompts []string `json:"prompts" jsonschema:"description=Array of detailed image descriptions. Create diverse prompts if user doesn't specify exact number.,minItems=1,maxItems=4"`
	Quality string   `json:"quality,omitempty" jsonschema:"description=Image quality. 'hd' creates images with finer details.,enum=standard,enum=hd,default=standard"`
	Size    string   `json:"size,omitempty" jsonschema:"description=Image resolution. Use 1024x1024 (square) as default.,enum=1792x1024,enum=1024x1024,enum=1024x1792,default=1024x1024"`
	Style   string   `json:"style,omitempty" jsonschema:"description=Image style. 'vivid' for hyper-real/dramatic&#44; 'natural' for more realistic.,enum=vivid,enum=natural,default=vivid"`
	Seeds   []int    `json:"seeds,omitempty" jsonschema:"description=Optional seeds for reproducible generation when modifying previous images."`
}

// ============================================================================
// Response Types (matching frontend expected format)
// ============================================================================

// DallEImageItem matches frontend DallEImageItem interface
type DallEImageItem struct {
	Prompt     string `json:"prompt"`
	PreviewUrl string `json:"previewUrl,omitempty"`
	ImageId    string `json:"imageId,omitempty"`
	Quality    string `json:"quality"` // "standard" | "hd"
	Size       string `json:"size"`    // "1792x1024" | "1024x1024" | "1024x1792"
	Style      string `json:"style"`   // "vivid" | "natural"
}

// ImageDesignerService provides image generation capabilities using Stable Diffusion
type ImageDesignerService struct {
	sdManager   *image.StableDiffusion
	outputDir   string
	initialized bool
}

// NewImageDesignerService creates a new image designer service
func NewImageDesignerService() *ImageDesignerService {
	outputDir := defaultImageOutputDir()
	os.MkdirAll(outputDir, 0755)

	return &ImageDesignerService{
		sdManager:   image.NewEngine(),
		outputDir:   outputDir,
		initialized: false,
	}
}

// IsAvailable checks if Stable Diffusion is installed and ready
func (s *ImageDesignerService) IsAvailable() bool {
	if !s.sdManager.IsStableDiffusionInstalled() {
		return false
	}

	// Check if at least one model is installed
	models, err := s.sdManager.CheckInstalledModels()
	if err != nil || len(models) == 0 {
		return false
	}

	return true
}

// GetFirstAvailableModel returns the first available SD model
func (s *ImageDesignerService) GetFirstAvailableModel() string {
	modelsPath := s.sdManager.GetModelsPath()
	files, err := os.ReadDir(modelsPath)
	if err != nil {
		return ""
	}

	for _, file := range files {
		if !file.IsDir() {
			name := file.Name()
			// Check for all supported model formats including GGUF
			if strings.HasSuffix(name, ".ckpt") ||
				strings.HasSuffix(name, ".safetensors") ||
				strings.HasSuffix(name, ".pt") ||
				strings.HasSuffix(name, ".bin") ||
				strings.HasSuffix(name, ".gguf") {
				return filepath.Join(modelsPath, name)
			}
		}
	}
	return ""
}

// Text2Image generates images from text prompts using Stable Diffusion
func (s *ImageDesignerService) Text2Image(prompts []string, quality, size, style string, seeds []int) ([]DallEImageItem, error) {
	// Default values
	if quality == "" {
		quality = "standard"
	}
	if size == "" {
		size = "1024x1024"
	}
	if style == "" {
		style = "vivid"
	}

	// Check if SD is available
	if !s.IsAvailable() {
		log.Printf("⚠️  Stable Diffusion not available, using placeholder images")
		return s.generatePlaceholders(prompts, quality, size, style, seeds)
	}

	// Get SD binary and model paths
	// Get model path
	modelPath := s.GetFirstAvailableModel()

	if modelPath == "" {
		log.Printf("⚠️  No SD model found, using placeholder images")
		return s.generatePlaceholders(prompts, quality, size, style, seeds)
	}

	log.Printf("🎨 Using Stable Diffusion: %s", filepath.Base(modelPath))

	results := make([]DallEImageItem, 0, len(prompts))

	for i, prompt := range prompts {
		// Generate unique output filename
		imageId := uuid.New().String()
		outputPath := filepath.Join(s.outputDir, fmt.Sprintf("%s.png", imageId))

		// Determine seed
		seed := time.Now().UnixNano() + int64(i)
		if len(seeds) > i && seeds[i] > 0 {
			seed = int64(seeds[i])
		}

		// Parse size
		width, height := parseDallESize(size)

		// Determine steps based on quality
		steps := 20
		if quality == "hd" {
			steps = 30
		}

		// Add negative prompt for better quality
		negativePrompt := "ugly, blurry, low quality, distorted"
		if style == "natural" {
			negativePrompt = "cartoon, anime, illustration, " + negativePrompt
		}

		// Prepare generation options
		options := image.GenerationOptions{
			Prompt:         prompt,
			NegativePrompt: negativePrompt,
			ModelPath:      modelPath,
			OutputPath:     outputPath,
			Width:          width,
			Height:         height,
			Steps:          steps,
			Seed:           &seed,
		}

		log.Printf("🖼️  Generating image %d/%d: %s", i+1, len(prompts), truncateString(prompt, 50))

		// Execute SD via the manager
		if err := s.sdManager.CreateImageWithOptions(options); err != nil {
			log.Printf("⚠️  SD generation failed: %v", err)
			// Fallback to placeholder
			result := s.generateSinglePlaceholder(prompt, quality, size, style, i)
			results = append(results, result)
			continue
		}

		// Check if output was created
		if _, err := os.Stat(outputPath); err != nil {
			log.Printf("⚠️  Output image not found: %s", outputPath)
			result := s.generateSinglePlaceholder(prompt, quality, size, style, i)
			results = append(results, result)
			continue
		}

		// Read image and convert to data URL for preview
		imageData, err := os.ReadFile(outputPath)
		if err != nil {
			log.Printf("⚠️  Failed to read output image: %v", err)
			result := s.generateSinglePlaceholder(prompt, quality, size, style, i)
			results = append(results, result)
			continue
		}

		// Create data URL
		previewUrl := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(imageData))

		results = append(results, DallEImageItem{
			Prompt:     prompt,
			PreviewUrl: previewUrl,
			ImageId:    imageId,
			Quality:    quality,
			Size:       size,
			Style:      style,
		})

		log.Printf("✅ Generated image: %s", imageId)
	}

	return results, nil
}

// generatePlaceholders generates placeholder images when SD is not available
func (s *ImageDesignerService) generatePlaceholders(prompts []string, quality, size, style string, seeds []int) ([]DallEImageItem, error) {
	results := make([]DallEImageItem, 0, len(prompts))

	for i, prompt := range prompts {
		result := s.generateSinglePlaceholder(prompt, quality, size, style, i)
		if len(seeds) > i {
			// Use seed in placeholder URL for consistency
			width, height := parseDallESize(size)
			result.PreviewUrl = fmt.Sprintf("https://picsum.photos/seed/%d/%d/%d", seeds[i], width, height)
		}
		results = append(results, result)
	}

	return results, nil
}

// generateSinglePlaceholder generates a single placeholder image
func (s *ImageDesignerService) generateSinglePlaceholder(prompt, quality, size, style string, index int) DallEImageItem {
	width, height := parseDallESize(size)
	placeholderUrl := fmt.Sprintf("https://picsum.photos/seed/%d/%d/%d", index, width, height)

	log.Printf("🎨 Generated placeholder for: %s", truncateString(prompt, 50))

	return DallEImageItem{
		Prompt:     prompt,
		PreviewUrl: placeholderUrl,
		Quality:    quality,
		Size:       size,
		Style:      style,
	}
}

// parseDallESize parses DALL-E size string to width and height
func parseDallESize(size string) (int, int) {
	switch size {
	case "1792x1024":
		return 1792, 1024
	case "1024x1792":
		return 1024, 1792
	default:
		return 1024, 1024
	}
}

func defaultImageOutputDir() string {
	base := os.Getenv("KAWAI_TOOLS_OUTPUT_DIR")
	if base == "" {
		base = filepath.Join(os.TempDir(), "kawai-tools", "images")
	}
	return base
}

// truncateString truncates a string to max length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ============================================================================
// Tool Registration
// ============================================================================

// NewImageDesigner creates the lobe-image-designer tool (DALL-E compatible).
func NewImageDesigner(_ context.Context) ([]tool.InvokableTool, error) {
	service := NewImageDesignerService()

	text2imageTool, err := utils.InferTool("lobe-image-designer__text2image",
		"Create images from text prompts using AI image generation. Generate up to 4 diverse images based on the description.",
		func(ctx context.Context, input *Text2ImageInput) (string, error) {
			if len(input.Prompts) == 0 {
				return "", fmt.Errorf("at least one prompt is required")
			}

			prompts := input.Prompts
			if len(prompts) > 4 {
				prompts = prompts[:4]
			}

			results, err := service.Text2Image(prompts, input.Quality, input.Size, input.Style, input.Seeds)
			if err != nil {
				return "", err
			}

			resultJSON, _ := json.Marshal(results)
			log.Printf("🖼️  Generated %d images", len(results))
			return string(resultJSON), nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to infer text2image tool: %w", err)
	}

	return []tool.InvokableTool{text2imageTool}, nil
}
