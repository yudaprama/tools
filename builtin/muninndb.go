package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/kawai-network/y/paths"
	"github.com/scrypster/muninndb/pkg/embedded"
	"github.com/scrypster/muninndb/pkg/mcp"
)

// MuninnDBService manages embedded MuninnDB connections.
type MuninnDBService struct {
	core *embedded.Service
}

// MuninnAttachInput defines input for opening embedded MuninnDB.
type MuninnAttachInput struct {
	Name         string `json:"name" jsonschema:"required,description=Connection name (e.g. 'memory')"`
	DefaultVault string `json:"default_vault,omitempty" jsonschema:"description=Default vault name when not provided in requests"`
	DataDir      string `json:"data_dir,omitempty" jsonschema:"description=Custom data directory (optional)"`
	CacheSize    int    `json:"cache_size,omitempty" jsonschema:"description=In-memory cache size (default from MuninnDB if omitted)"`
	NoSync       bool   `json:"no_sync,omitempty" jsonschema:"description=Enable faster but less durable writes"`
}

// MuninnDetachInput defines input for closing embedded MuninnDB.
type MuninnDetachInput struct {
	Connection string `json:"connection" jsonschema:"required,description=Connection name to close"`
}

// MuninnRememberInput defines input for muninn_remember.
type MuninnRememberInput struct {
	Connection   string   `json:"connection" jsonschema:"required,description=Connection name from muninn_attach"`
	Vault        string   `json:"vault,omitempty" jsonschema:"description=Target vault name (optional)"`
	Concept      string   `json:"concept" jsonschema:"required,description=Concept/title for the memory"`
	Content      string   `json:"content" jsonschema:"required,description=Memory content"`
	Tags         []string `json:"tags,omitempty" jsonschema:"description=Optional tags for retrieval"`
	Confidence   float32  `json:"confidence,omitempty" jsonschema:"description=Confidence score 0..1"`
	Stability    float32  `json:"stability,omitempty" jsonschema:"description=Stability score in days"`
	CreatedAt    string   `json:"created_at,omitempty" jsonschema:"description=Optional RFC3339 timestamp for historical/future memory"`
	IdempotentID string   `json:"idempotent_id,omitempty" jsonschema:"description=Optional idempotency key"`
	Type         uint8    `json:"type,omitempty" jsonschema:"description=Memory type enum (0..11)"`
	TypeLabel    string   `json:"type_label,omitempty" jsonschema:"description=Optional free-form type label"`
	Summary      string   `json:"summary,omitempty" jsonschema:"description=Optional one-line summary"`
}

// MuninnRememberBatchItem defines one memory item for batch write.
type MuninnRememberBatchItem struct {
	Vault        string   `json:"vault,omitempty" jsonschema:"description=Target vault name (optional)"`
	Concept      string   `json:"concept" jsonschema:"required,description=Concept/title for the memory"`
	Content      string   `json:"content" jsonschema:"required,description=Memory content"`
	Tags         []string `json:"tags,omitempty" jsonschema:"description=Optional tags for retrieval"`
	Confidence   float32  `json:"confidence,omitempty" jsonschema:"description=Confidence score 0..1"`
	Stability    float32  `json:"stability,omitempty" jsonschema:"description=Stability score in days"`
	CreatedAt    string   `json:"created_at,omitempty" jsonschema:"description=Optional RFC3339 timestamp"`
	IdempotentID string   `json:"idempotent_id,omitempty" jsonschema:"description=Optional idempotency key"`
	Type         uint8    `json:"type,omitempty" jsonschema:"description=Memory type enum (0..11)"`
	TypeLabel    string   `json:"type_label,omitempty" jsonschema:"description=Optional free-form type label"`
	Summary      string   `json:"summary,omitempty" jsonschema:"description=Optional one-line summary"`
}

// MuninnRememberBatchInput defines input for muninn_remember_batch.
type MuninnRememberBatchInput struct {
	Connection string                    `json:"connection" jsonschema:"required,description=Connection name from muninn_attach"`
	Vault      string                    `json:"vault,omitempty" jsonschema:"description=Default vault for items that omit vault"`
	Memories   []MuninnRememberBatchItem `json:"memories" jsonschema:"required,description=List of memories (max 50)"`
}

// MuninnRecallInput defines input for muninn_recall.
type MuninnRecallInput struct {
	Connection string   `json:"connection" jsonschema:"required,description=Connection name from muninn_attach"`
	Vault      string   `json:"vault,omitempty" jsonschema:"description=Vault name (optional)"`
	Context    []string `json:"context" jsonschema:"required,description=List of context cues for recall"`
	Threshold  float32  `json:"threshold,omitempty" jsonschema:"description=Minimum score threshold"`
	MaxResults int      `json:"max_results,omitempty" jsonschema:"description=Maximum activation results"`
	MaxHops    int      `json:"max_hops,omitempty" jsonschema:"description=Graph traversal hops"`
	IncludeWhy bool     `json:"include_why,omitempty" jsonschema:"description=Include explanation fields"`
}

// MuninnReadInput defines input for muninn_read.
type MuninnReadInput struct {
	Connection string `json:"connection" jsonschema:"required,description=Connection name from muninn_attach"`
	Vault      string `json:"vault,omitempty" jsonschema:"description=Vault name (optional)"`
	ID         string `json:"id" jsonschema:"required,description=Memory ID to fetch"`
}

// MuninnLinkInput defines input for muninn_link.
type MuninnLinkInput struct {
	Connection string  `json:"connection" jsonschema:"required,description=Connection name from muninn_attach"`
	Vault      string  `json:"vault,omitempty" jsonschema:"description=Vault name (optional)"`
	SourceID   string  `json:"source_id" jsonschema:"required,description=Source memory ID"`
	TargetID   string  `json:"target_id" jsonschema:"required,description=Target memory ID"`
	RelType    uint16  `json:"rel_type,omitempty" jsonschema:"description=Relationship type code"`
	Weight     float32 `json:"weight,omitempty" jsonschema:"description=Relationship weight 0..1"`
}

// MuninnForgetInput defines input for muninn_forget.
type MuninnForgetInput struct {
	Connection string `json:"connection" jsonschema:"required,description=Connection name from muninn_attach"`
	Vault      string `json:"vault,omitempty" jsonschema:"description=Vault name (optional)"`
	ID         string `json:"id" jsonschema:"required,description=Memory ID to remove"`
	Hard       bool   `json:"hard,omitempty" jsonschema:"description=Hard delete if true"`
}

// MuninnStatusInput defines input for muninn_status.
type MuninnStatusInput struct {
	Connection string `json:"connection" jsonschema:"required,description=Connection name from muninn_attach"`
	Vault      string `json:"vault,omitempty" jsonschema:"description=Vault name (optional)"`
}

// NewMuninnDBService creates a new MuninnDB service.
func NewMuninnDBService() *MuninnDBService {
	return &MuninnDBService{core: embedded.NewService()}
}

// NewMuninnDB creates the embedded MuninnDB tools.
func NewMuninnDB(ctx context.Context) ([]tool.InvokableTool, error) {
	service := NewMuninnDBService()
	descriptions := mcpToolDescriptions()

	rememberTool, err := utils.InferTool("muninn_remember",
		mcpToolDescription(descriptions, "muninn_remember", "Store a single memory engram."),
		func(ctx context.Context, input *MuninnRememberInput) (string, error) {
			return service.remember(ctx, *input)
		},
	)
	if err != nil {
		return nil, err
	}

	rememberBatchTool, err := utils.InferTool("muninn_remember_batch",
		mcpToolDescription(descriptions, "muninn_remember_batch", "Store multiple memories in one request (max 50)."),
		func(ctx context.Context, input *MuninnRememberBatchInput) (string, error) {
			return service.rememberBatch(ctx, *input)
		},
	)
	if err != nil {
		return nil, err
	}

	recallTool, err := utils.InferTool("muninn_recall",
		mcpToolDescription(descriptions, "muninn_recall", "Recall relevant memories using context cues."),
		func(ctx context.Context, input *MuninnRecallInput) (string, error) {
			return service.recall(ctx, *input)
		},
	)
	if err != nil {
		return nil, err
	}

	readTool, err := utils.InferTool("muninn_read",
		mcpToolDescription(descriptions, "muninn_read", "Read one memory by ID."),
		func(ctx context.Context, input *MuninnReadInput) (string, error) {
			return service.read(ctx, *input)
		},
	)
	if err != nil {
		return nil, err
	}

	linkTool, err := utils.InferTool("muninn_link",
		mcpToolDescription(descriptions, "muninn_link", "Create or update a relationship between two memories."),
		func(ctx context.Context, input *MuninnLinkInput) (string, error) {
			return service.link(ctx, *input)
		},
	)
	if err != nil {
		return nil, err
	}

	forgetTool, err := utils.InferTool("muninn_forget",
		mcpToolDescription(descriptions, "muninn_forget", "Forget one memory (soft delete by default)."),
		func(ctx context.Context, input *MuninnForgetInput) (string, error) {
			return service.forget(ctx, *input)
		},
	)
	if err != nil {
		return nil, err
	}

	statusTool, err := utils.InferTool("muninn_status",
		mcpToolDescription(descriptions, "muninn_status", "Get memory and coherence stats for a vault."),
		func(ctx context.Context, input *MuninnStatusInput) (string, error) {
			return service.status(ctx, *input)
		},
	)
	if err != nil {
		return nil, err
	}

	return []tool.InvokableTool{rememberTool, rememberBatchTool, recallTool, readTool, linkTool, forgetTool, statusTool}, nil
}

func (s *MuninnDBService) attach(_ context.Context, input MuninnAttachInput) (string, error) {
	if strings.TrimSpace(input.Name) == "" {
		return "", fmt.Errorf("name is required")
	}
	if err := validateSQLIdent(input.Name); err != nil {
		return "", err
	}

	dataDir := strings.TrimSpace(input.DataDir)
	if dataDir == "" {
		dataDir = filepath.Join(paths.Base(), "muninndb")
		safeConnName := sanitizeDirName(input.Name)
		if safeConnName != "" {
			dataDir = filepath.Join(dataDir, safeConnName)
		}
	}

	_, err := s.core.Attach(embedded.AttachOptions{
		Name:         input.Name,
		DataDir:      dataDir,
		DefaultVault: input.DefaultVault,
		CacheSize:    input.CacheSize,
		NoSync:       input.NoSync,
	})
	if err != nil {
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "is being initialized") {
			return "", err
		}
		return "", fmt.Errorf("failed to open muninndb: %v", err)
	}

	result := map[string]any{
		"status":        "connected",
		"connection":    input.Name,
		"data_dir":      dataDir,
		"default_vault": input.DefaultVault,
	}
	return marshalToolResult(result)
}

func (s *MuninnDBService) detach(_ context.Context, input MuninnDetachInput) (string, error) {
	if strings.TrimSpace(input.Connection) == "" {
		return "", fmt.Errorf("connection is required")
	}
	if err := validateSQLIdent(input.Connection); err != nil {
		return "", err
	}
	if err := s.core.Detach(input.Connection); err != nil {
		return "", err
	}

	result := map[string]any{"status": "disconnected", "connection": input.Connection}
	return marshalToolResult(result)
}

func (s *MuninnDBService) remember(ctx context.Context, input MuninnRememberInput) (string, error) {
	createdAt, err := embedded.ParseCreatedAt(input.CreatedAt)
	if err != nil {
		return "", err
	}

	resp, err := s.core.Remember(ctx, embedded.RememberInput{
		Connection:   input.Connection,
		Vault:        input.Vault,
		Concept:      input.Concept,
		Content:      input.Content,
		Tags:         input.Tags,
		Confidence:   input.Confidence,
		Stability:    input.Stability,
		CreatedAt:    createdAt,
		IdempotentID: input.IdempotentID,
		Type:         input.Type,
		TypeLabel:    input.TypeLabel,
		Summary:      input.Summary,
	})
	if err != nil {
		return "", fmt.Errorf("failed to remember: %v", err)
	}
	return marshalToolResult(resp)
}

func (s *MuninnDBService) rememberBatch(ctx context.Context, input MuninnRememberBatchInput) (string, error) {
	items := make([]embedded.RememberBatchItem, 0, len(input.Memories))
	for i := range input.Memories {
		createdAt, err := embedded.ParseCreatedAt(input.Memories[i].CreatedAt)
		if err != nil {
			return "", fmt.Errorf("memory index %d: %v", i, err)
		}
		item := input.Memories[i]
		items = append(items, embedded.RememberBatchItem{
			Vault:        item.Vault,
			Concept:      item.Concept,
			Content:      item.Content,
			Tags:         item.Tags,
			Confidence:   item.Confidence,
			Stability:    item.Stability,
			CreatedAt:    createdAt,
			IdempotentID: item.IdempotentID,
			Type:         item.Type,
			TypeLabel:    item.TypeLabel,
			Summary:      item.Summary,
		})
	}

	written, err := s.core.RememberBatch(ctx, embedded.RememberBatchInput{
		Connection: input.Connection,
		Vault:      input.Vault,
		Memories:   items,
	})
	if err != nil {
		return "", err
	}

	return marshalToolResult(map[string]any{"count": len(written), "memories": written})
}

func (s *MuninnDBService) recall(ctx context.Context, input MuninnRecallInput) (string, error) {
	resp, err := s.core.Recall(ctx, embedded.RecallInput{
		Connection: input.Connection,
		Vault:      input.Vault,
		Context:    input.Context,
		Threshold:  input.Threshold,
		MaxResults: input.MaxResults,
		MaxHops:    input.MaxHops,
		IncludeWhy: input.IncludeWhy,
	})
	if err != nil {
		return "", fmt.Errorf("failed to recall: %v", err)
	}

	return marshalToolResult(resp)
}

func (s *MuninnDBService) read(ctx context.Context, input MuninnReadInput) (string, error) {
	resp, err := s.core.Read(ctx, embedded.ReadInput{
		Connection: input.Connection,
		Vault:      input.Vault,
		ID:         input.ID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to read: %v", err)
	}
	return marshalToolResult(resp)
}

func (s *MuninnDBService) link(ctx context.Context, input MuninnLinkInput) (string, error) {
	resp, err := s.core.Link(ctx, embedded.LinkInput{
		Connection: input.Connection,
		Vault:      input.Vault,
		SourceID:   input.SourceID,
		TargetID:   input.TargetID,
		RelType:    input.RelType,
		Weight:     input.Weight,
	})
	if err != nil {
		return "", fmt.Errorf("failed to link: %v", err)
	}
	return marshalToolResult(resp)
}

func (s *MuninnDBService) forget(ctx context.Context, input MuninnForgetInput) (string, error) {
	resp, err := s.core.Forget(ctx, embedded.ForgetInput{
		Connection: input.Connection,
		Vault:      input.Vault,
		ID:         input.ID,
		Hard:       input.Hard,
	})
	if err != nil {
		return "", fmt.Errorf("failed to forget: %v", err)
	}
	return marshalToolResult(resp)
}

func (s *MuninnDBService) status(ctx context.Context, input MuninnStatusInput) (string, error) {
	resp, err := s.core.Status(ctx, embedded.StatusInput{Connection: input.Connection, Vault: input.Vault})
	if err != nil {
		return "", fmt.Errorf("failed to get status: %v", err)
	}
	return marshalToolResult(resp)
}

func marshalToolResult(v any) (string, error) {
	resultJSON, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %v", err)
	}
	return string(resultJSON), nil
}

func sanitizeDirName(value string) string {
	return embedded.SanitizeDirName(value)
}

func pickVault(vault, defaultVault string) string {
	return embedded.PickVault(vault, defaultVault)
}

func mcpToolDescriptions() map[string]string {
	descriptions := make(map[string]string)
	for _, def := range mcp.ToolDefinitions() {
		name := strings.TrimSpace(def.Name)
		if name == "" {
			continue
		}
		if desc := strings.TrimSpace(def.Description); desc != "" {
			descriptions[name] = desc
		}
	}
	return descriptions
}

func mcpToolDescription(descriptions map[string]string, name, fallback string) string {
	if desc, ok := descriptions[name]; ok && strings.TrimSpace(desc) != "" {
		return desc
	}
	return fallback
}
