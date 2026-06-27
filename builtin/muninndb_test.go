package builtin

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/yudaprama/tools"
	"github.com/kawai-network/y/paths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMuninnTools_Registration(t *testing.T) {
	muninnTools, err := NewMuninnDB(context.Background())
	require.NoError(t, err)

	registry := tools.NewToolRegistry()
	require.NoError(t, registry.RegisterAll(muninnTools))

	expectedTools := []string{
		"muninn_remember",
		"muninn_remember_batch",
		"muninn_recall",
		"muninn_read",
		"muninn_link",
		"muninn_forget",
		"muninn_status",
	}

	for _, toolName := range expectedTools {
		invTool, exists := registry.Get(toolName)
		assert.True(t, exists, "tool %s should be registered", toolName)
		assert.NotNil(t, invTool, "tool %s should not be nil", toolName)
	}
}

func TestMuninnService_BasicFlow(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{
		Name: "mem",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem"})
	})

	rememberContent, err := service.remember(ctx, MuninnRememberInput{
		Connection: "mem",
		Concept:    "project",
		Content:    "kawai contributor uses muninn",
		Tags:       []string{"kawai", "memory"},
	})
	require.NoError(t, err)

	var writeOut struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(rememberContent), &writeOut))
	require.NotEmpty(t, writeOut.ID)

	_, err = service.read(ctx, MuninnReadInput{Connection: "mem", ID: writeOut.ID})
	require.NoError(t, err)

	_, err = service.status(ctx, MuninnStatusInput{Connection: "mem"})
	require.NoError(t, err)

	_, err = service.detach(ctx, MuninnDetachInput{Connection: "mem"})
	require.NoError(t, err)
}

func TestMuninnService_AttachUsesYPathsBaseWhenDataDirEmpty(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})

	paths.SetDataDir(t.TempDir())

	attachContent, err := service.attach(ctx, MuninnAttachInput{
		Name: "mem2",
	})
	require.NoError(t, err)
	assert.Contains(t, attachContent, "muninndb")
	assert.Contains(t, attachContent, "mem2")

	_, err = service.detach(ctx, MuninnDetachInput{Connection: "mem2"})
	require.NoError(t, err)
}

func TestMuninnService_AttachSameNameConcurrent(t *testing.T) {
	service := NewMuninnDBService()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	var wg sync.WaitGroup
	wg.Add(2)

	results := make([]bool, 2)
	for i := 0; i < 2; i++ {
		i := i
		go func() {
			defer wg.Done()
			_, err := service.attach(context.Background(), MuninnAttachInput{Name: "same"})
			results[i] = err == nil
		}()
	}
	wg.Wait()

	successes := 0
	for _, ok := range results {
		if ok {
			successes++
		}
	}
	require.Equal(t, 1, successes, "only one concurrent attach should succeed")

	_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "same"})
}

func TestMuninnService_RecallRequiresNonEmptyContext(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{Name: "mem3"})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem3"})
	})

	_, err = service.recall(ctx, MuninnRecallInput{
		Connection: "mem3",
		Context:    []string{"   ", ""},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context is required")
}

// ==================== Validation Error Tests ====================

func TestMuninnService_AttachEmptyName(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()

	_, err := service.attach(ctx, MuninnAttachInput{Name: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestMuninnService_AttachInvalidName(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()

	_, err := service.attach(ctx, MuninnAttachInput{Name: "invalid;name"})
	require.Error(t, err)
}

func TestMuninnService_DetachEmptyConnection(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()

	_, err := service.detach(ctx, MuninnDetachInput{Connection: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection is required")
}

func TestMuninnService_DetachNotFound(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()

	_, err := service.detach(ctx, MuninnDetachInput{Connection: "nonexistent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMuninnService_RememberEmptyConcept(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{Name: "mem_val"})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_val"})
	})

	_, err = service.remember(ctx, MuninnRememberInput{
		Connection: "mem_val",
		Concept:    "",
		Content:    "some content",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "concept and content are required")
}

func TestMuninnService_RememberEmptyContent(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{Name: "mem_val2"})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_val2"})
	})

	_, err = service.remember(ctx, MuninnRememberInput{
		Connection: "mem_val2",
		Concept:    "test",
		Content:    "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "concept and content are required")
}

func TestMuninnService_RememberInvalidCreatedAt(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{Name: "mem_val3"})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_val3"})
	})

	_, err = service.remember(ctx, MuninnRememberInput{
		Connection: "mem_val3",
		Concept:    "test",
		Content:    "content",
		CreatedAt:  "invalid-date",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RFC3339")
}

func TestMuninnService_RememberBatchEmpty(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{Name: "mem_batch"})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_batch"})
	})

	_, err = service.rememberBatch(ctx, MuninnRememberBatchInput{
		Connection: "mem_batch",
		Memories:   []MuninnRememberBatchItem{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "memories is required")
}

func TestMuninnService_RememberBatchExceedsLimit(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{Name: "mem_batch2"})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_batch2"})
	})

	memories := make([]MuninnRememberBatchItem, 51)
	for i := range memories {
		memories[i] = MuninnRememberBatchItem{
			Concept: "test",
			Content: "content",
		}
	}

	_, err = service.rememberBatch(ctx, MuninnRememberBatchInput{
		Connection: "mem_batch2",
		Memories:   memories,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum 50")
}

func TestMuninnService_RememberBatchInvalidItem(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{Name: "mem_batch3"})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_batch3"})
	})

	_, err = service.rememberBatch(ctx, MuninnRememberBatchInput{
		Connection: "mem_batch3",
		Memories: []MuninnRememberBatchItem{
			{Concept: "valid", Content: "content"},
			{Concept: "", Content: "content"},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "concept and content")
}

func TestMuninnService_ReadEmptyID(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{Name: "mem_read"})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_read"})
	})

	_, err = service.read(ctx, MuninnReadInput{
		Connection: "mem_read",
		ID:         "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "id is required")
}

func TestMuninnService_LinkEmptyIDs(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{Name: "mem_link"})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_link"})
	})

	_, err = service.link(ctx, MuninnLinkInput{
		Connection: "mem_link",
		SourceID:   "",
		TargetID:   "target",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source_id and target_id are required")
}

func TestMuninnService_ForgetEmptyID(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{Name: "mem_forget"})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_forget"})
	})

	_, err = service.forget(ctx, MuninnForgetInput{
		Connection: "mem_forget",
		ID:         "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "id is required")
}

// ==================== Connection Error Tests ====================

func TestMuninnService_OperationOnNotFoundConnection(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()

	_, err := service.remember(ctx, MuninnRememberInput{
		Connection: "nonexistent",
		Concept:    "test",
		Content:    "content",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMuninnService_OperationOnEmptyConnection(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()

	_, err := service.remember(ctx, MuninnRememberInput{
		Connection: "",
		Concept:    "test",
		Content:    "content",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection is required")
}

func TestMuninnService_OperationOnInvalidConnection(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()

	_, err := service.remember(ctx, MuninnRememberInput{
		Connection: "invalid;conn",
		Concept:    "test",
		Content:    "content",
	})
	require.Error(t, err)
}

// ==================== MuninnRememberBatch Tests ====================

func TestMuninnService_RememberBatchSuccess(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{Name: "mem_batch_ok"})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_batch_ok"})
	})

	content, err := service.rememberBatch(ctx, MuninnRememberBatchInput{
		Connection: "mem_batch_ok",
		Vault:      "default_vault",
		Memories: []MuninnRememberBatchItem{
			{Concept: "concept1", Content: "content1", Tags: []string{"tag1"}},
			{Concept: "concept2", Content: "content2"},
		},
	})
	require.NoError(t, err)

	var out struct {
		Count    int   `json:"count"`
		Memories []any `json:"memories"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &out))
	assert.Equal(t, 2, out.Count)
	assert.Len(t, out.Memories, 2)
}

func TestMuninnService_RememberBatchWithVaultFallback(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{
		Name:         "mem_batch_vault",
		DefaultVault: "default_vault",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_batch_vault"})
	})

	_, err = service.rememberBatch(ctx, MuninnRememberBatchInput{
		Connection: "mem_batch_vault",
		Memories: []MuninnRememberBatchItem{
			{Concept: "c1", Content: "content1"},
			{Concept: "c2", Content: "content2", Vault: "override_vault"},
		},
	})
	require.NoError(t, err)
}

// ==================== MuninnLink Tests ====================

func TestMuninnService_LinkSuccess(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{Name: "mem_link_ok"})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_link_ok"})
	})

	rememberContent1, err := service.remember(ctx, MuninnRememberInput{
		Connection: "mem_link_ok",
		Concept:    "concept1",
		Content:    "content1",
	})
	require.NoError(t, err)

	var out1 struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(rememberContent1), &out1))

	rememberContent2, err := service.remember(ctx, MuninnRememberInput{
		Connection: "mem_link_ok",
		Concept:    "concept2",
		Content:    "content2",
	})
	require.NoError(t, err)

	var out2 struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(rememberContent2), &out2))

	_, err = service.link(ctx, MuninnLinkInput{
		Connection: "mem_link_ok",
		SourceID:   out1.ID,
		TargetID:   out2.ID,
		RelType:    1,
		Weight:     0.8,
	})
	require.NoError(t, err)
}

// ==================== MuninnForget Tests ====================

func TestMuninnService_ForgetSoftDelete(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{Name: "mem_forget_soft"})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_forget_soft"})
	})

	rememberContent, err := service.remember(ctx, MuninnRememberInput{
		Connection: "mem_forget_soft",
		Concept:    "to_forget",
		Content:    "will be forgotten",
	})
	require.NoError(t, err)

	var out struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(rememberContent), &out))

	_, err = service.forget(ctx, MuninnForgetInput{
		Connection: "mem_forget_soft",
		ID:         out.ID,
		Hard:       false,
	})
	require.NoError(t, err)
}

func TestMuninnService_ForgetHardDelete(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{Name: "mem_forget_hard"})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_forget_hard"})
	})

	rememberContent, err := service.remember(ctx, MuninnRememberInput{
		Connection: "mem_forget_hard",
		Concept:    "to_forget_hard",
		Content:    "will be hard deleted",
	})
	require.NoError(t, err)

	var out struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(rememberContent), &out))

	_, err = service.forget(ctx, MuninnForgetInput{
		Connection: "mem_forget_hard",
		ID:         out.ID,
		Hard:       true,
	})
	require.NoError(t, err)
}

// ==================== MuninnRecall Tests ====================

func TestMuninnService_RecallSuccess(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{Name: "mem_recall"})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_recall"})
	})

	_, err = service.remember(ctx, MuninnRememberInput{
		Connection: "mem_recall",
		Concept:    "test concept",
		Content:    "test content for recall",
		Tags:       []string{"recall", "test"},
	})
	require.NoError(t, err)

	_, err = service.recall(ctx, MuninnRecallInput{
		Connection: "mem_recall",
		Context:    []string{"test"},
		Threshold:  0.1,
		MaxResults: 10,
		MaxHops:    2,
	})
	require.NoError(t, err)
}

func TestMuninnService_RecallWithVault(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{
		Name:         "mem_recall_vault",
		DefaultVault: "my_vault",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_recall_vault"})
	})

	_, err = service.remember(ctx, MuninnRememberInput{
		Connection: "mem_recall_vault",
		Concept:    "vault concept",
		Content:    "vault content",
	})
	require.NoError(t, err)

	_, err = service.recall(ctx, MuninnRecallInput{
		Connection: "mem_recall_vault",
		Vault:      "my_vault",
		Context:    []string{"vault"},
	})
	require.NoError(t, err)
}

// ==================== MuninnRead Tests ====================

func TestMuninnService_ReadSuccess(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{Name: "mem_read_ok"})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_read_ok"})
	})

	rememberContent, err := service.remember(ctx, MuninnRememberInput{
		Connection: "mem_read_ok",
		Concept:    "readable concept",
		Content:    "readable content",
		Tags:       []string{"read"},
	})
	require.NoError(t, err)

	var out struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(rememberContent), &out))

	readContent, err := service.read(ctx, MuninnReadInput{
		Connection: "mem_read_ok",
		ID:         out.ID,
	})
	require.NoError(t, err)

	var readOut struct {
		ID      string `json:"id"`
		Concept string `json:"concept"`
		Content string `json:"content"`
	}
	require.NoError(t, json.Unmarshal([]byte(readContent), &readOut))
	assert.Equal(t, "readable concept", readOut.Concept)
	assert.Equal(t, "readable content", readOut.Content)
}

func TestMuninnService_ReadWithVault(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{
		Name:         "mem_read_vault",
		DefaultVault: "read_vault",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_read_vault"})
	})

	rememberContent, err := service.remember(ctx, MuninnRememberInput{
		Connection: "mem_read_vault",
		Concept:    "vault read concept",
		Content:    "vault read content",
	})
	require.NoError(t, err)

	var out struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(rememberContent), &out))

	_, err = service.read(ctx, MuninnReadInput{
		Connection: "mem_read_vault",
		ID:         out.ID,
		Vault:      "read_vault",
	})
	require.NoError(t, err)
}

// ==================== Edge Case Tests ====================

func TestMuninnService_RememberWithCreatedAt(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{Name: "mem_time"})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_time"})
	})

	futureTime := "2030-01-01T00:00:00Z"
	_, err = service.remember(ctx, MuninnRememberInput{
		Connection: "mem_time",
		Concept:    "time concept",
		Content:    "time content",
		CreatedAt:  futureTime,
	})
	require.NoError(t, err)
}

func TestMuninnService_RememberWithAllOptionalFields(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{Name: "mem_full"})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_full"})
	})

	_, err = service.remember(ctx, MuninnRememberInput{
		Connection:   "mem_full",
		Vault:        "full_vault",
		Concept:      "full concept",
		Content:      "full content",
		Tags:         []string{"tag1", "tag2"},
		Confidence:   0.9,
		Stability:    7.0,
		IdempotentID: "idem-123",
		Type:         1,
		TypeLabel:    "custom_type",
		Summary:      "one line summary",
	})
	require.NoError(t, err)
}

func TestMuninnService_StatusWithVault(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	_, err := service.attach(ctx, MuninnAttachInput{
		Name:         "mem_status_vault",
		DefaultVault: "status_vault",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_status_vault"})
	})

	_, err = service.status(ctx, MuninnStatusInput{
		Connection: "mem_status_vault",
		Vault:      "status_vault",
	})
	require.NoError(t, err)
}

func TestMuninnService_PickVaultLogic(t *testing.T) {
	tests := []struct {
		name         string
		vault        string
		defaultVault string
		expected     string
	}{
		{"vault takes precedence", "my_vault", "default", "my_vault"},
		{"empty vault uses default", "", "default", "default"},
		{"whitespace vault uses default", "   ", "default", "default"},
		{"both empty returns empty", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pickVault(tt.vault, tt.defaultVault)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMuninnService_SanitizeDirName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"valid-name", "valid-name"},
		{"valid_name", "valid_name"},
		{"valid.name", "valid.name"},
		{"valid123", "valid123"},
		{"invalid;name", "invalid_name"},
		{"invalid name", "invalid_name"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeDirName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
