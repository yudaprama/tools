package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/getkawai/tools"
	"github.com/getkawai/unillm"
	"github.com/kawai-network/y/paths"
	"github.com/scrypster/muninndb/pkg/engine"
	"github.com/scrypster/muninndb/pkg/engine/activation"
	"github.com/scrypster/muninndb/pkg/engine/trigger"
	"github.com/scrypster/muninndb/pkg/index/fts"
	hnswpkg "github.com/scrypster/muninndb/pkg/index/hnsw"
	"github.com/scrypster/muninndb/pkg/storage"
	"github.com/scrypster/muninndb/pkg/transport/mbp"
)

// MuninnDBService manages embedded MuninnDB connections.
type MuninnDBService struct {
	mu          sync.RWMutex
	connections map[string]*connectionEntry
	opening     map[string]struct{}
}

type muninnDB struct {
	store *storage.PebbleStore
	eng   *engine.Engine
}

type connectionEntry struct {
	db           *muninnDB
	defaultVault string
	mu           sync.RWMutex
	closing      bool
}

type muninnOpenOptions struct {
	DataDir      string
	CacheSize    int
	NoSyncEngram bool
}

func openMuninnDB(opts muninnOpenOptions) (*muninnDB, error) {
	if strings.TrimSpace(opts.DataDir) == "" {
		return nil, fmt.Errorf("data dir is required")
	}
	pebbleDir := filepath.Join(opts.DataDir, "pebble")
	if err := os.MkdirAll(pebbleDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	pdb, err := storage.OpenPebble(pebbleDir, storage.DefaultOptions())
	if err != nil {
		return nil, err
	}

	store := storage.NewPebbleStore(pdb, storage.PebbleStoreConfig{
		CacheSize:     opts.CacheSize,
		NoSyncEngrams: opts.NoSyncEngram,
	})

	ftsIndex := fts.New(pdb)
	hnswRegistry := hnswpkg.NewRegistry(pdb)
	embedder := activation.NewNoopEmbedder()
	actEngine := activation.New(store, activation.NewFTSAdapter(ftsIndex), activation.NewHNSWAdapter(hnswRegistry), embedder)
	trigSystem := trigger.New(store, trigger.NewFTSAdapter(ftsIndex), trigger.NewHNSWAdapter(hnswRegistry), embedder)
	eng := engine.NewEngine(store, nil, ftsIndex, actEngine, trigSystem, nil, nil, nil, embedder, hnswRegistry)

	return &muninnDB{store: store, eng: eng}, nil
}

func (m *muninnDB) Close() error {
	if m.eng != nil {
		m.eng.Stop()
	}
	if m.store != nil {
		return m.store.Close()
	}
	return nil
}

// NewMuninnDBService creates a new MuninnDB service.
func NewMuninnDBService() *MuninnDBService {
	return &MuninnDBService{
		connections: make(map[string]*connectionEntry),
		opening:     make(map[string]struct{}),
	}
}

func (s *MuninnDBService) getConnection(name string) (*connectionEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	conn, ok := s.connections[name]
	return conn, ok
}

func (s *MuninnDBService) completeConnection(name string, db *muninnDB, defaultVault string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.opening, name)
	s.connections[name] = &connectionEntry{db: db, defaultVault: strings.TrimSpace(defaultVault)}
}

func (s *MuninnDBService) removeConnection(name string, conn *connectionEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.connections[name]
	if !ok {
		return
	}
	if current != conn {
		return
	}
	delete(s.connections, name)
}

func (s *MuninnDBService) reserveConnection(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.connections[name]; ok {
		return fmt.Errorf("connection '%s' already exists", name)
	}
	if _, ok := s.opening[name]; ok {
		return fmt.Errorf("connection '%s' is being initialized", name)
	}

	s.opening[name] = struct{}{}
	return nil
}

func (s *MuninnDBService) releaseReservation(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.opening, name)
}

// MuninnAttachInput defines input for opening embedded MuninnDB.
type MuninnAttachInput struct {
	Name         string `json:"name" jsonschema:"required,description=Connection name (e.g. 'memory')"`
	DefaultVault string `json:"default_vault,omitempty" jsonschema:"description=Default vault name when not provided in requests"`
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

// RegisterMuninnDB registers embedded MuninnDB tools.
func RegisterMuninnDB(registry *tools.ToolRegistry) error {
	service := NewMuninnDBService()

	attachTool := unillm.NewAgentTool("muninn_attach",
		"Open an embedded MuninnDB connection for local memory operations.",
		func(ctx context.Context, input MuninnAttachInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			return service.attach(ctx, input)
		},
	)
	if err := registry.Register(attachTool); err != nil {
		return err
	}

	rememberTool := unillm.NewAgentTool("muninn_remember",
		"Store a single memory engram.",
		func(ctx context.Context, input MuninnRememberInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			return service.remember(ctx, input)
		},
	)
	if err := registry.Register(rememberTool); err != nil {
		return err
	}

	rememberBatchTool := unillm.NewAgentTool("muninn_remember_batch",
		"Store multiple memories in one request (max 50).",
		func(ctx context.Context, input MuninnRememberBatchInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			return service.rememberBatch(ctx, input)
		},
	)
	if err := registry.Register(rememberBatchTool); err != nil {
		return err
	}

	recallTool := unillm.NewParallelAgentTool("muninn_recall",
		"Recall relevant memories using context cues.",
		func(ctx context.Context, input MuninnRecallInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			return service.recall(ctx, input)
		},
	)
	if err := registry.Register(recallTool); err != nil {
		return err
	}

	readTool := unillm.NewParallelAgentTool("muninn_read",
		"Read one memory by ID.",
		func(ctx context.Context, input MuninnReadInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			return service.read(ctx, input)
		},
	)
	if err := registry.Register(readTool); err != nil {
		return err
	}

	linkTool := unillm.NewAgentTool("muninn_link",
		"Create or update a relationship between two memories.",
		func(ctx context.Context, input MuninnLinkInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			return service.link(ctx, input)
		},
	)
	if err := registry.Register(linkTool); err != nil {
		return err
	}

	forgetTool := unillm.NewAgentTool("muninn_forget",
		"Forget one memory (soft delete by default).",
		func(ctx context.Context, input MuninnForgetInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			return service.forget(ctx, input)
		},
	)
	if err := registry.Register(forgetTool); err != nil {
		return err
	}

	statusTool := unillm.NewParallelAgentTool("muninn_status",
		"Get memory and coherence stats for a vault.",
		func(ctx context.Context, input MuninnStatusInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			return service.status(ctx, input)
		},
	)
	if err := registry.Register(statusTool); err != nil {
		return err
	}

	detachTool := unillm.NewAgentTool("muninn_detach",
		"Close an embedded MuninnDB connection and release resources.",
		func(ctx context.Context, input MuninnDetachInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			return service.detach(ctx, input)
		},
	)
	if err := registry.Register(detachTool); err != nil {
		return err
	}

	return nil
}

func (s *MuninnDBService) attach(_ context.Context, input MuninnAttachInput) (unillm.ToolResponse, error) {
	if strings.TrimSpace(input.Name) == "" {
		return unillm.NewTextErrorResponse("name is required"), nil
	}
	if err := validateSQLIdent(input.Name); err != nil {
		return unillm.NewTextErrorResponse(err.Error()), nil
	}
	if err := s.reserveConnection(input.Name); err != nil {
		return unillm.NewTextErrorResponse(err.Error()), nil
	}

	dataDir := filepath.Join(paths.Base(), "muninndb")
	safeConnName := sanitizeDirName(input.Name)
	if safeConnName != "" {
		dataDir = filepath.Join(dataDir, safeConnName)
	}

	db, err := openMuninnDB(muninnOpenOptions{
		DataDir:      dataDir,
		CacheSize:    input.CacheSize,
		NoSyncEngram: input.NoSync,
	})
	if err != nil {
		s.releaseReservation(input.Name)
		return unillm.NewTextErrorResponse(fmt.Sprintf("failed to open muninndb: %v", err)), nil
	}

	s.completeConnection(input.Name, db, input.DefaultVault)

	result := map[string]any{
		"status":        "connected",
		"connection":    input.Name,
		"data_dir":      dataDir,
		"default_vault": input.DefaultVault,
	}
	return marshalToolResponse(result)
}

func (s *MuninnDBService) detach(_ context.Context, input MuninnDetachInput) (unillm.ToolResponse, error) {
	if strings.TrimSpace(input.Connection) == "" {
		return unillm.NewTextErrorResponse("connection is required"), nil
	}
	if err := validateSQLIdent(input.Connection); err != nil {
		return unillm.NewTextErrorResponse(err.Error()), nil
	}

	conn, ok := s.getConnection(input.Connection)
	if !ok {
		return unillm.NewTextErrorResponse(fmt.Sprintf("connection '%s' not found", input.Connection)), nil
	}

	conn.mu.Lock()
	conn.closing = true
	if err := conn.db.Close(); err != nil {
		conn.closing = false
		conn.mu.Unlock()
		return unillm.NewTextErrorResponse(fmt.Sprintf("failed to close connection: %v", err)), nil
	}
	conn.mu.Unlock()

	s.removeConnection(input.Connection, conn)

	result := map[string]any{"status": "disconnected", "connection": input.Connection}
	return marshalToolResponse(result)
}

func (s *MuninnDBService) remember(ctx context.Context, input MuninnRememberInput) (unillm.ToolResponse, error) {
	if strings.TrimSpace(input.Concept) == "" || strings.TrimSpace(input.Content) == "" {
		return unillm.NewTextErrorResponse("concept and content are required"), nil
	}

	createdAt, err := parseCreatedAt(input.CreatedAt)
	if err != nil {
		return unillm.NewTextErrorResponse(err.Error()), nil
	}

	return s.withConnection(input.Connection, func(conn *connectionEntry) (unillm.ToolResponse, error) {
		resp, err := conn.db.eng.Write(ctx, &mbp.WriteRequest{
			Vault:        pickVault(input.Vault, conn.defaultVault),
			Concept:      input.Concept,
			Content:      input.Content,
			Tags:         input.Tags,
			Confidence:   input.Confidence,
			Stability:    input.Stability,
			CreatedAt:    createdAt,
			IdempotentID: input.IdempotentID,
			MemoryType:   input.Type,
			TypeLabel:    input.TypeLabel,
			Summary:      input.Summary,
		})
		if err != nil {
			return unillm.NewTextErrorResponse(fmt.Sprintf("failed to remember: %v", err)), nil
		}

		return marshalToolResponse(resp)
	})
}

func (s *MuninnDBService) rememberBatch(ctx context.Context, input MuninnRememberBatchInput) (unillm.ToolResponse, error) {
	if len(input.Memories) == 0 {
		return unillm.NewTextErrorResponse("memories is required"), nil
	}
	if len(input.Memories) > 50 {
		return unillm.NewTextErrorResponse("maximum 50 memories per batch"), nil
	}

	return s.withConnection(input.Connection, func(conn *connectionEntry) (unillm.ToolResponse, error) {
		written := make([]*mbp.WriteResponse, 0, len(input.Memories))
		for i := range input.Memories {
			item := input.Memories[i]
			if strings.TrimSpace(item.Concept) == "" || strings.TrimSpace(item.Content) == "" {
				return unillm.NewTextErrorResponse(fmt.Sprintf("memory index %d requires concept and content", i)), nil
			}
			createdAt, err := parseCreatedAt(item.CreatedAt)
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("memory index %d: %v", i, err)), nil
			}

			vault := item.Vault
			if strings.TrimSpace(vault) == "" {
				vault = input.Vault
			}
			vault = pickVault(vault, conn.defaultVault)

			resp, err := conn.db.eng.Write(ctx, &mbp.WriteRequest{
				Vault:        vault,
				Concept:      item.Concept,
				Content:      item.Content,
				Tags:         item.Tags,
				Confidence:   item.Confidence,
				Stability:    item.Stability,
				CreatedAt:    createdAt,
				IdempotentID: item.IdempotentID,
				MemoryType:   item.Type,
				TypeLabel:    item.TypeLabel,
				Summary:      item.Summary,
			})
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("failed to remember batch at index %d: %v", i, err)), nil
			}
			written = append(written, resp)
		}

		return marshalToolResponse(map[string]any{
			"count":    len(written),
			"memories": written,
		})
	})
}

func (s *MuninnDBService) recall(ctx context.Context, input MuninnRecallInput) (unillm.ToolResponse, error) {
	hasContext := false
	for i := range input.Context {
		if strings.TrimSpace(input.Context[i]) != "" {
			hasContext = true
			break
		}
	}
	if !hasContext {
		return unillm.NewTextErrorResponse("context is required and must contain at least one non-empty item"), nil
	}

	return s.withConnection(input.Connection, func(conn *connectionEntry) (unillm.ToolResponse, error) {
		resp, err := conn.db.eng.Activate(ctx, &mbp.ActivateRequest{
			Vault:      pickVault(input.Vault, conn.defaultVault),
			Context:    input.Context,
			Threshold:  input.Threshold,
			MaxResults: input.MaxResults,
			MaxHops:    input.MaxHops,
			IncludeWhy: input.IncludeWhy,
		})
		if err != nil {
			return unillm.NewTextErrorResponse(fmt.Sprintf("failed to recall: %v", err)), nil
		}

		return marshalToolResponse(resp)
	})
}

func (s *MuninnDBService) read(ctx context.Context, input MuninnReadInput) (unillm.ToolResponse, error) {
	if strings.TrimSpace(input.ID) == "" {
		return unillm.NewTextErrorResponse("id is required"), nil
	}

	return s.withConnection(input.Connection, func(conn *connectionEntry) (unillm.ToolResponse, error) {
		resp, err := conn.db.eng.Read(ctx, &mbp.ReadRequest{Vault: pickVault(input.Vault, conn.defaultVault), ID: input.ID})
		if err != nil {
			return unillm.NewTextErrorResponse(fmt.Sprintf("failed to read: %v", err)), nil
		}

		return marshalToolResponse(resp)
	})
}

func (s *MuninnDBService) link(ctx context.Context, input MuninnLinkInput) (unillm.ToolResponse, error) {
	if strings.TrimSpace(input.SourceID) == "" || strings.TrimSpace(input.TargetID) == "" {
		return unillm.NewTextErrorResponse("source_id and target_id are required"), nil
	}

	return s.withConnection(input.Connection, func(conn *connectionEntry) (unillm.ToolResponse, error) {
		resp, err := conn.db.eng.Link(ctx, &mbp.LinkRequest{
			Vault:    pickVault(input.Vault, conn.defaultVault),
			SourceID: input.SourceID,
			TargetID: input.TargetID,
			RelType:  input.RelType,
			Weight:   input.Weight,
		})
		if err != nil {
			return unillm.NewTextErrorResponse(fmt.Sprintf("failed to link: %v", err)), nil
		}

		return marshalToolResponse(resp)
	})
}

func (s *MuninnDBService) forget(ctx context.Context, input MuninnForgetInput) (unillm.ToolResponse, error) {
	if strings.TrimSpace(input.ID) == "" {
		return unillm.NewTextErrorResponse("id is required"), nil
	}

	return s.withConnection(input.Connection, func(conn *connectionEntry) (unillm.ToolResponse, error) {
		resp, err := conn.db.eng.Forget(ctx, &mbp.ForgetRequest{Vault: pickVault(input.Vault, conn.defaultVault), ID: input.ID, Hard: input.Hard})
		if err != nil {
			return unillm.NewTextErrorResponse(fmt.Sprintf("failed to forget: %v", err)), nil
		}

		return marshalToolResponse(resp)
	})
}

func (s *MuninnDBService) status(ctx context.Context, input MuninnStatusInput) (unillm.ToolResponse, error) {
	return s.withConnection(input.Connection, func(conn *connectionEntry) (unillm.ToolResponse, error) {
		resp, err := conn.db.eng.Stat(ctx, &mbp.StatRequest{Vault: pickVault(input.Vault, conn.defaultVault)})
		if err != nil {
			return unillm.NewTextErrorResponse(fmt.Sprintf("failed to get status: %v", err)), nil
		}

		return marshalToolResponse(resp)
	})
}

func (s *MuninnDBService) withConnection(name string, fn func(conn *connectionEntry) (unillm.ToolResponse, error)) (unillm.ToolResponse, error) {
	if strings.TrimSpace(name) == "" {
		return unillm.NewTextErrorResponse("connection is required"), nil
	}
	if err := validateSQLIdent(name); err != nil {
		return unillm.NewTextErrorResponse(err.Error()), nil
	}

	conn, ok := s.getConnection(name)
	if !ok {
		return unillm.NewTextErrorResponse(fmt.Sprintf("connection '%s' not found", name)), nil
	}

	conn.mu.RLock()
	if conn.closing {
		conn.mu.RUnlock()
		return unillm.NewTextErrorResponse(fmt.Sprintf("connection '%s' is closing", name)), nil
	}
	defer conn.mu.RUnlock()

	return fn(conn)
}

func marshalToolResponse(v any) (unillm.ToolResponse, error) {
	resultJSON, err := json.Marshal(v)
	if err != nil {
		return unillm.NewTextErrorResponse(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}
	return unillm.NewTextResponse(string(resultJSON)), nil
}

func parseCreatedAt(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("created_at must be RFC3339 format")
	}
	return &t, nil
}

func sanitizeDirName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
			continue
		}
		b.WriteRune('_')
	}
	return b.String()
}

func pickVault(vault, defaultVault string) string {
	vault = strings.TrimSpace(vault)
	if vault != "" {
		return vault
	}
	return strings.TrimSpace(defaultVault)
}
