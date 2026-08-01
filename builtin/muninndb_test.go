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
	registry := tools.NewToolRegistry()
	err := RegisterMuninnDB(registry)
	require.NoError(t, err)

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
		tool, exists := registry.Get(toolName)
		assert.True(t, exists, "tool %s should be registered", toolName)
		assert.NotNil(t, tool, "tool %s should not be nil", toolName)
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

	attachResp, err := service.attach(ctx, MuninnAttachInput{
		Name: "mem",
	})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem"})
	})

	rememberResp, err := service.remember(ctx, MuninnRememberInput{
		Connection: "mem",
		Concept:    "project",
		Content:    "kawai contributor uses muninn",
		Tags:       []string{"kawai", "memory"},
	})
	require.NoError(t, err)
	require.False(t, rememberResp.IsError)

	var writeOut struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(rememberResp.Content), &writeOut))
	require.NotEmpty(t, writeOut.ID)

	readResp, err := service.read(ctx, MuninnReadInput{Connection: "mem", ID: writeOut.ID})
	require.NoError(t, err)
	require.False(t, readResp.IsError)

	statusResp, err := service.status(ctx, MuninnStatusInput{Connection: "mem"})
	require.NoError(t, err)
	require.False(t, statusResp.IsError)

	detachResp, err := service.detach(ctx, MuninnDetachInput{Connection: "mem"})
	require.NoError(t, err)
	require.False(t, detachResp.IsError)
}

func TestMuninnService_AttachUsesYPathsBaseWhenDataDirEmpty(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})

	paths.SetDataDir(t.TempDir())

	attachResp, err := service.attach(ctx, MuninnAttachInput{
		Name: "mem2",
	})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	assert.Contains(t, attachResp.Content, "muninndb")
	assert.Contains(t, attachResp.Content, "mem2")

	detachResp, err := service.detach(ctx, MuninnDetachInput{Connection: "mem2"})
	require.NoError(t, err)
	require.False(t, detachResp.IsError)
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
			resp, err := service.attach(context.Background(), MuninnAttachInput{Name: "same"})
			results[i] = err == nil && !resp.IsError
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

	attachResp, err := service.attach(ctx, MuninnAttachInput{Name: "mem3"})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem3"})
	})

	recallResp, err := service.recall(ctx, MuninnRecallInput{
		Connection: "mem3",
		Context:    []string{"   ", ""},
	})
	require.NoError(t, err)
	require.True(t, recallResp.IsError)
	assert.Contains(t, recallResp.Content, "context is required")
}

// ==================== Validation Error Tests ====================

func TestMuninnService_AttachEmptyName(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()

	resp, err := service.attach(ctx, MuninnAttachInput{Name: ""})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	assert.Contains(t, resp.Content, "name is required")
}

func TestMuninnService_AttachInvalidName(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()

	resp, err := service.attach(ctx, MuninnAttachInput{Name: "invalid;name"})
	require.NoError(t, err)
	require.True(t, resp.IsError)
}

func TestMuninnService_DetachEmptyConnection(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()

	resp, err := service.detach(ctx, MuninnDetachInput{Connection: ""})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	assert.Contains(t, resp.Content, "connection is required")
}

func TestMuninnService_DetachNotFound(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()

	resp, err := service.detach(ctx, MuninnDetachInput{Connection: "nonexistent"})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	assert.Contains(t, resp.Content, "not found")
}

func TestMuninnService_RememberEmptyConcept(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	attachResp, err := service.attach(ctx, MuninnAttachInput{Name: "mem_val"})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_val"})
	})

	resp, err := service.remember(ctx, MuninnRememberInput{
		Connection: "mem_val",
		Concept:    "",
		Content:    "some content",
	})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	assert.Contains(t, resp.Content, "concept and content are required")
}

func TestMuninnService_RememberEmptyContent(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	attachResp, err := service.attach(ctx, MuninnAttachInput{Name: "mem_val2"})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_val2"})
	})

	resp, err := service.remember(ctx, MuninnRememberInput{
		Connection: "mem_val2",
		Concept:    "test",
		Content:    "",
	})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	assert.Contains(t, resp.Content, "concept and content are required")
}

func TestMuninnService_RememberInvalidCreatedAt(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	attachResp, err := service.attach(ctx, MuninnAttachInput{Name: "mem_val3"})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_val3"})
	})

	resp, err := service.remember(ctx, MuninnRememberInput{
		Connection: "mem_val3",
		Concept:    "test",
		Content:    "content",
		CreatedAt:  "invalid-date",
	})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	assert.Contains(t, resp.Content, "RFC3339")
}

func TestMuninnService_RememberBatchEmpty(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	attachResp, err := service.attach(ctx, MuninnAttachInput{Name: "mem_batch"})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_batch"})
	})

	resp, err := service.rememberBatch(ctx, MuninnRememberBatchInput{
		Connection: "mem_batch",
		Memories:   []MuninnRememberBatchItem{},
	})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	assert.Contains(t, resp.Content, "memories is required")
}

func TestMuninnService_RememberBatchExceedsLimit(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	attachResp, err := service.attach(ctx, MuninnAttachInput{Name: "mem_batch2"})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
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

	resp, err := service.rememberBatch(ctx, MuninnRememberBatchInput{
		Connection: "mem_batch2",
		Memories:   memories,
	})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	assert.Contains(t, resp.Content, "maximum 50")
}

func TestMuninnService_RememberBatchInvalidItem(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	attachResp, err := service.attach(ctx, MuninnAttachInput{Name: "mem_batch3"})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_batch3"})
	})

	resp, err := service.rememberBatch(ctx, MuninnRememberBatchInput{
		Connection: "mem_batch3",
		Memories: []MuninnRememberBatchItem{
			{Concept: "valid", Content: "content"},
			{Concept: "", Content: "content"},
		},
	})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	assert.Contains(t, resp.Content, "concept and content")
}

func TestMuninnService_ReadEmptyID(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	attachResp, err := service.attach(ctx, MuninnAttachInput{Name: "mem_read"})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_read"})
	})

	resp, err := service.read(ctx, MuninnReadInput{
		Connection: "mem_read",
		ID:         "",
	})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	assert.Contains(t, resp.Content, "id is required")
}

func TestMuninnService_LinkEmptyIDs(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	attachResp, err := service.attach(ctx, MuninnAttachInput{Name: "mem_link"})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_link"})
	})

	resp, err := service.link(ctx, MuninnLinkInput{
		Connection: "mem_link",
		SourceID:   "",
		TargetID:   "target",
	})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	assert.Contains(t, resp.Content, "source_id and target_id are required")
}

func TestMuninnService_ForgetEmptyID(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	attachResp, err := service.attach(ctx, MuninnAttachInput{Name: "mem_forget"})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_forget"})
	})

	resp, err := service.forget(ctx, MuninnForgetInput{
		Connection: "mem_forget",
		ID:         "",
	})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	assert.Contains(t, resp.Content, "id is required")
}

// ==================== Connection Error Tests ====================

func TestMuninnService_OperationOnNotFoundConnection(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()

	resp, err := service.remember(ctx, MuninnRememberInput{
		Connection: "nonexistent",
		Concept:    "test",
		Content:    "content",
	})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	assert.Contains(t, resp.Content, "not found")
}

func TestMuninnService_OperationOnEmptyConnection(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()

	resp, err := service.remember(ctx, MuninnRememberInput{
		Connection: "",
		Concept:    "test",
		Content:    "content",
	})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	assert.Contains(t, resp.Content, "connection is required")
}

func TestMuninnService_OperationOnInvalidConnection(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()

	resp, err := service.remember(ctx, MuninnRememberInput{
		Connection: "invalid;conn",
		Concept:    "test",
		Content:    "content",
	})
	require.NoError(t, err)
	require.True(t, resp.IsError)
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

	attachResp, err := service.attach(ctx, MuninnAttachInput{Name: "mem_batch_ok"})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_batch_ok"})
	})

	resp, err := service.rememberBatch(ctx, MuninnRememberBatchInput{
		Connection: "mem_batch_ok",
		Vault:      "default_vault",
		Memories: []MuninnRememberBatchItem{
			{Concept: "concept1", Content: "content1", Tags: []string{"tag1"}},
			{Concept: "concept2", Content: "content2"},
		},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError)

	var out struct {
		Count    int   `json:"count"`
		Memories []any `json:"memories"`
	}
	require.NoError(t, json.Unmarshal([]byte(resp.Content), &out))
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

	attachResp, err := service.attach(ctx, MuninnAttachInput{
		Name:         "mem_batch_vault",
		DefaultVault: "default_vault",
	})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_batch_vault"})
	})

	resp, err := service.rememberBatch(ctx, MuninnRememberBatchInput{
		Connection: "mem_batch_vault",
		Memories: []MuninnRememberBatchItem{
			{Concept: "c1", Content: "content1"},
			{Concept: "c2", Content: "content2", Vault: "override_vault"},
		},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError)
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

	attachResp, err := service.attach(ctx, MuninnAttachInput{Name: "mem_link_ok"})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_link_ok"})
	})

	rememberResp1, err := service.remember(ctx, MuninnRememberInput{
		Connection: "mem_link_ok",
		Concept:    "concept1",
		Content:    "content1",
	})
	require.NoError(t, err)
	require.False(t, rememberResp1.IsError)

	var out1 struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(rememberResp1.Content), &out1))

	rememberResp2, err := service.remember(ctx, MuninnRememberInput{
		Connection: "mem_link_ok",
		Concept:    "concept2",
		Content:    "content2",
	})
	require.NoError(t, err)
	require.False(t, rememberResp2.IsError)

	var out2 struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(rememberResp2.Content), &out2))

	linkResp, err := service.link(ctx, MuninnLinkInput{
		Connection: "mem_link_ok",
		SourceID:   out1.ID,
		TargetID:   out2.ID,
		RelType:    1,
		Weight:     0.8,
	})
	require.NoError(t, err)
	require.False(t, linkResp.IsError)
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

	attachResp, err := service.attach(ctx, MuninnAttachInput{Name: "mem_forget_soft"})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_forget_soft"})
	})

	rememberResp, err := service.remember(ctx, MuninnRememberInput{
		Connection: "mem_forget_soft",
		Concept:    "to_forget",
		Content:    "will be forgotten",
	})
	require.NoError(t, err)
	require.False(t, rememberResp.IsError)

	var out struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(rememberResp.Content), &out))

	forgetResp, err := service.forget(ctx, MuninnForgetInput{
		Connection: "mem_forget_soft",
		ID:         out.ID,
		Hard:       false,
	})
	require.NoError(t, err)
	require.False(t, forgetResp.IsError)
}

func TestMuninnService_ForgetHardDelete(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	attachResp, err := service.attach(ctx, MuninnAttachInput{Name: "mem_forget_hard"})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_forget_hard"})
	})

	rememberResp, err := service.remember(ctx, MuninnRememberInput{
		Connection: "mem_forget_hard",
		Concept:    "to_forget_hard",
		Content:    "will be hard deleted",
	})
	require.NoError(t, err)
	require.False(t, rememberResp.IsError)

	var out struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(rememberResp.Content), &out))

	forgetResp, err := service.forget(ctx, MuninnForgetInput{
		Connection: "mem_forget_hard",
		ID:         out.ID,
		Hard:       true,
	})
	require.NoError(t, err)
	require.False(t, forgetResp.IsError)
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

	attachResp, err := service.attach(ctx, MuninnAttachInput{Name: "mem_recall"})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_recall"})
	})

	rememberResp, err := service.remember(ctx, MuninnRememberInput{
		Connection: "mem_recall",
		Concept:    "test concept",
		Content:    "test content for recall",
		Tags:       []string{"recall", "test"},
	})
	require.NoError(t, err)
	require.False(t, rememberResp.IsError)

	recallResp, err := service.recall(ctx, MuninnRecallInput{
		Connection: "mem_recall",
		Context:    []string{"test"},
		Threshold:  0.1,
		MaxResults: 10,
		MaxHops:    2,
	})
	require.NoError(t, err)
	require.False(t, recallResp.IsError)
}

func TestMuninnService_RecallWithVault(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	attachResp, err := service.attach(ctx, MuninnAttachInput{
		Name:         "mem_recall_vault",
		DefaultVault: "my_vault",
	})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_recall_vault"})
	})

	rememberResp, err := service.remember(ctx, MuninnRememberInput{
		Connection: "mem_recall_vault",
		Concept:    "vault concept",
		Content:    "vault content",
	})
	require.NoError(t, err)
	require.False(t, rememberResp.IsError)

	recallResp, err := service.recall(ctx, MuninnRecallInput{
		Connection: "mem_recall_vault",
		Vault:      "my_vault",
		Context:    []string{"vault"},
	})
	require.NoError(t, err)
	require.False(t, recallResp.IsError)
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

	attachResp, err := service.attach(ctx, MuninnAttachInput{Name: "mem_read_ok"})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_read_ok"})
	})

	rememberResp, err := service.remember(ctx, MuninnRememberInput{
		Connection: "mem_read_ok",
		Concept:    "readable concept",
		Content:    "readable content",
		Tags:       []string{"read"},
	})
	require.NoError(t, err)
	require.False(t, rememberResp.IsError)

	var out struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(rememberResp.Content), &out))

	readResp, err := service.read(ctx, MuninnReadInput{
		Connection: "mem_read_ok",
		ID:         out.ID,
	})
	require.NoError(t, err)
	require.False(t, readResp.IsError)

	var readOut struct {
		ID      string `json:"id"`
		Concept string `json:"concept"`
		Content string `json:"content"`
	}
	require.NoError(t, json.Unmarshal([]byte(readResp.Content), &readOut))
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

	attachResp, err := service.attach(ctx, MuninnAttachInput{
		Name:         "mem_read_vault",
		DefaultVault: "read_vault",
	})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_read_vault"})
	})

	rememberResp, err := service.remember(ctx, MuninnRememberInput{
		Connection: "mem_read_vault",
		Concept:    "vault read concept",
		Content:    "vault read content",
	})
	require.NoError(t, err)
	require.False(t, rememberResp.IsError)

	var out struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(rememberResp.Content), &out))

	readResp, err := service.read(ctx, MuninnReadInput{
		Connection: "mem_read_vault",
		ID:         out.ID,
		Vault:      "read_vault",
	})
	require.NoError(t, err)
	require.False(t, readResp.IsError)
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

	attachResp, err := service.attach(ctx, MuninnAttachInput{Name: "mem_time"})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_time"})
	})

	futureTime := "2030-01-01T00:00:00Z"
	resp, err := service.remember(ctx, MuninnRememberInput{
		Connection: "mem_time",
		Concept:    "time concept",
		Content:    "time content",
		CreatedAt:  futureTime,
	})
	require.NoError(t, err)
	require.False(t, resp.IsError)
}

func TestMuninnService_RememberWithAllOptionalFields(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	attachResp, err := service.attach(ctx, MuninnAttachInput{Name: "mem_full"})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_full"})
	})

	resp, err := service.remember(ctx, MuninnRememberInput{
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
	require.False(t, resp.IsError)
}

func TestMuninnService_StatusWithVault(t *testing.T) {
	service := NewMuninnDBService()
	ctx := context.Background()
	prevDataDir := paths.Base()
	t.Cleanup(func() {
		paths.SetDataDir(prevDataDir)
	})
	paths.SetDataDir(t.TempDir())

	attachResp, err := service.attach(ctx, MuninnAttachInput{
		Name:         "mem_status_vault",
		DefaultVault: "status_vault",
	})
	require.NoError(t, err)
	require.False(t, attachResp.IsError)
	t.Cleanup(func() {
		_, _ = service.detach(context.Background(), MuninnDetachInput{Connection: "mem_status_vault"})
	})

	statusResp, err := service.status(ctx, MuninnStatusInput{
		Connection: "mem_status_vault",
		Vault:      "status_vault",
	})
	require.NoError(t, err)
	require.False(t, statusResp.IsError)
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
