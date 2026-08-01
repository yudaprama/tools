package image

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GenerationOptions defines image generation options used by tools/builtin.
type GenerationOptions struct {
	Prompt         string
	NegativePrompt string
	ModelPath      string
	OutputPath     string
	ImageUrl       *string
	ImageUrls      []string
	Width          int
	Height         int
	Size           string
	AspectRatio    string
	Steps          int
	Cfg            float64
	Strength       float64
	Seed           *int64
	Quality        string
	SamplerName    string
	Scheduler      string
	OutputFormat   string
	Model          string
}

// StableDiffusion is a minimal backend facade for image generation.
// It keeps tools decoupled from veridium/internal packages.
type StableDiffusion struct {
	modelsPath string
}

func NewEngine() *StableDiffusion {
	return &StableDiffusion{
		modelsPath: defaultModelsPath(),
	}
}

func defaultModelsPath() string {
	if p := os.Getenv("KAWAI_SD_MODELS_DIR"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "kawai-tools", "models")
	}
	return filepath.Join(home, ".kawai", "models", "stable-diffusion")
}

// IsStableDiffusionInstalled checks if the backend is explicitly enabled.
func (s *StableDiffusion) IsStableDiffusionInstalled() bool {
	return os.Getenv("KAWAI_SD_ENABLE") == "1"
}

func (s *StableDiffusion) GetModelsPath() string {
	return s.modelsPath
}

func (s *StableDiffusion) CheckInstalledModels() ([]string, error) {
	entries, err := os.ReadDir(s.modelsPath)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.ToLower(e.Name())
		if strings.HasSuffix(name, ".ckpt") ||
			strings.HasSuffix(name, ".safetensors") ||
			strings.HasSuffix(name, ".pt") ||
			strings.HasSuffix(name, ".bin") ||
			strings.HasSuffix(name, ".gguf") {
			out = append(out, filepath.Join(s.modelsPath, e.Name()))
		}
	}
	return out, nil
}

// CreateImageWithOptions currently requires external backend integration.
func (s *StableDiffusion) CreateImageWithOptions(opts GenerationOptions) error {
	return fmt.Errorf("stable diffusion backend not configured in github.com/yudaprama/tools/image")
}
