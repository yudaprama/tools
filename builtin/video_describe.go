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

// VideoDescribeInput defines input for video describe tool
type VideoDescribeInput struct {
	FileID string `json:"file_id" jsonschema:"description=The file ID of the uploaded video"`
}

// VideoDescribeService provides video transcription functionality
type VideoDescribeService struct {
	queries *db.Queries
}

// NewVideoDescribeService creates a new video describe service
func NewVideoDescribeService(sqlDB *sql.DB) *VideoDescribeService {
	return &VideoDescribeService{
		queries: db.New(sqlDB),
	}
}

// GetVideoTranscription retrieves the AI-generated transcription for a video
// It polls the database for up to maxWait duration waiting for Whisper processing to complete
func (s *VideoDescribeService) GetVideoTranscription(ctx context.Context, fileID string, maxWait time.Duration) (string, error) {
	const pollInterval = 2 * time.Second
	deadline := time.Now().Add(maxWait)

	for attempt := 1; time.Now().Before(deadline); attempt++ {
		doc, err := s.queries.GetDocumentByFileID(ctx, sql.NullString{String: fileID, Valid: true})
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("⏳ [VideoDescribe] Document not found for file %s, waiting... (attempt %d)", fileID, attempt)
			} else {
				return "", fmt.Errorf("failed to query document: %w", err)
			}
		} else {
			// Check if document has whisper transcription
			if doc.Content.Valid && doc.Content.String != "" {
				content := doc.Content.String
				hasTranscription := strings.Contains(content, "Video Transcription (AI Generated via Whisper)")

				if hasTranscription {
					log.Printf("✅ [VideoDescribe] Found transcription for file %s (%d chars, attempt %d)", fileID, len(content), attempt)
					return content, nil
				}
			}
		}

		// Check if we should continue polling
		if time.Now().Add(pollInterval).After(deadline) {
			break
		}

		log.Printf("⏳ [VideoDescribe] Waiting for transcription for file %s (attempt %d)", fileID, attempt)
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(pollInterval):
			// Continue polling
		}
	}

	return "", fmt.Errorf("timeout waiting for video transcription (file_id: %s)", fileID)
}

// RegisterVideoDescribe registers the video describe tool
func RegisterVideoDescribe(registry *tools.ToolRegistry, sqlDB *sql.DB) error {
	service := NewVideoDescribeService(sqlDB)

	tool := unillm.NewParallelAgentTool("lobe-video-describe__getVideoTranscription",
		"Get AI-generated transcription of an uploaded video's audio. Use this when user asks about what is said in the video, video content, spoken words, dialogue, or audio transcription. The transcription is generated using Whisper STT when the video was uploaded.",
		func(ctx context.Context, input VideoDescribeInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			if input.FileID == "" {
				return unillm.NewTextErrorResponse("file_id parameter is required"), nil
			}

			// Wait up to 3 minutes for transcription (video processing takes longer)
			transcription, err := service.GetVideoTranscription(ctx, input.FileID, 3*time.Minute)
			if err != nil {
				log.Printf("⚠️  [VideoDescribe] Failed to get transcription: %v", err)
				return unillm.NewTextErrorResponse(err.Error()), nil
			}

			result := map[string]interface{}{
				"file_id":       input.FileID,
				"transcription": transcription,
				"status":        "success",
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
