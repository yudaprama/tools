package builtin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/getkawai/unillm"
	"github.com/yudaprama/tools"
	"github.com/getkawai/database/db"
)

// ImageDescribeInput defines input for image describe tool
type ImageDescribeInput struct {
	FileID string `json:"file_id" jsonschema:"description=The file ID of the uploaded image or video"`
}

// ImageDescribeService provides image description functionality
type ImageDescribeService struct {
	queries *db.Queries
}

// NewImageDescribeService creates a new image describe service
func NewImageDescribeService(sqlDB *sql.DB) *ImageDescribeService {
	return &ImageDescribeService{
		queries: db.New(sqlDB),
	}
}

// GetImageDescription retrieves the AI-generated description or OCR text for an image
// It polls the database for up to maxWait duration waiting for processing to complete
// Supports both fast OCR path (Tesseract) and slow VL path
func (s *ImageDescribeService) GetImageDescription(ctx context.Context, fileID string, maxWait time.Duration) (string, error) {
	const pollInterval = 2 * time.Second
	deadline := time.Now().Add(maxWait)

	for attempt := 1; time.Now().Before(deadline); attempt++ {
		doc, err := s.queries.GetDocumentByFileID(ctx, sql.NullString{String: fileID, Valid: true})
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("⏳ [ImageDescribe] Document not found for file %s, waiting... (attempt %d)", fileID, attempt)
			} else {
				return "", fmt.Errorf("failed to query document: %w", err)
			}
		} else {
			// Check if document has image content (OCR or VL description)
			if doc.Content.Valid && doc.Content.String != "" {
				content := doc.Content.String

				// Check for any of the possible content markers
				hasContent := strings.Contains(content, "OCR Text (Tesseract") ||
					strings.Contains(content, "Image Description (VL Model)") ||
					strings.Contains(content, "Image Description (AI Generated)") ||
					strings.Contains(content, "Video Description (AI Generated)")

				if hasContent {
					log.Printf("✅ [ImageDescribe] Found content for file %s (%d chars, attempt %d)", fileID, len(content), attempt)
					return content, nil
				}
			}
		}

		// Check if we should continue polling
		if time.Now().Add(pollInterval).After(deadline) {
			break
		}

		log.Printf("⏳ [ImageDescribe] Waiting for image content for file %s (attempt %d)", fileID, attempt)
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(pollInterval):
			// Continue polling
		}
	}

	return "", fmt.Errorf("timeout waiting for image description (file_id: %s)", fileID)
}

// RegisterImageDescribe registers the image describe tool
func RegisterImageDescribe(registry *tools.ToolRegistry, sqlDB *sql.DB) error {
	service := NewImageDescribeService(sqlDB)

	tool := unillm.NewParallelAgentTool("lobe-image-describe__getImageDescription",
		"Get AI-generated description of an uploaded image or video. Use this when user asks about image content, text extraction, OCR, or visual analysis. The description is pre-generated when the file was uploaded.",
		func(ctx context.Context, input ImageDescribeInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			if input.FileID == "" {
				return unillm.NewTextErrorResponse("file_id parameter is required"), nil
			}

			// Wait up to 2 minutes for VL description
			description, err := service.GetImageDescription(ctx, input.FileID, 2*time.Minute)
			if err != nil {
				log.Printf("⚠️  [ImageDescribe] Failed to get description: %v", err)
				return unillm.NewTextErrorResponse(err.Error()), nil
			}

			result := map[string]interface{}{
				"file_id":     input.FileID,
				"description": description,
				"status":      "success",
			}

			resultJSON, err := json.Marshal(result)
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("failed to marshal result: %v", err)), nil
			}

			return unillm.NewTextResponse(string(resultJSON)), nil
		},
	)

	return registry.Register(tool)
}
