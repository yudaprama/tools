//go:build !windows

package builtin

import (
	"context"
	"testing"

	"github.com/yudaprama/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresService_Creation(t *testing.T) {
	service, err := NewPostgresService()
	require.NoError(t, err, "should create postgres service")
	require.NotNil(t, service, "service should not be nil")
	defer service.Close()

	assert.NotNil(t, service.db, "db should be initialized")
	assert.NotNil(t, service.connections, "connections map should be initialized")
}

func TestPostgresTools_Registration(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := RegisterPostgres(registry)
	require.NoError(t, err, "should register postgres tools")

	// Check all tools are registered
	expectedTools := []string{
		"postgres_attach",
		"postgres_query",
		"postgres_execute",
		"postgres_list_tables",
		"postgres_describe",
		"postgres_detach",
	}

	for _, toolName := range expectedTools {
		tool, exists := registry.Get(toolName)
		assert.True(t, exists, "tool %s should be registered", toolName)
		assert.NotNil(t, tool, "tool %s should not be nil", toolName)

		info := tool.Info()
		assert.Equal(t, toolName, info.Name, "tool name should match")
		assert.NotEmpty(t, info.Description, "tool should have description")
		assert.NotNil(t, info.Parameters, "tool should have parameters")
	}
}

func TestPostgresAttach_Validation(t *testing.T) {
	service, err := NewPostgresService()
	require.NoError(t, err)
	defer service.Close()

	ctx := context.Background()

	// Test missing required fields
	input := PostgresAttachInput{
		Name: "test",
		// Missing host, database, user
	}

	resp, err := service.attach(ctx, input)
	require.NoError(t, err)
	assert.True(t, resp.IsError, "should return error for missing fields")
	assert.Contains(t, resp.Content, "required", "error should mention required fields")
}

func TestPostgresAttach_SQLInjectionPrevention(t *testing.T) {
	service, err := NewPostgresService()
	require.NoError(t, err)
	defer service.Close()

	ctx := context.Background()

	tests := []struct {
		name          string
		input         PostgresAttachInput
		expectBlocked bool // Should be blocked by identifier validation
	}{
		{
			name: "Single quote in password (escaped)",
			input: PostgresAttachInput{
				Name:     "test2",
				Host:     "localhost",
				Database: "mydb",
				User:     "user",
				Password: "pass'word",
			},
			expectBlocked: false, // Password is escaped, not validated as identifier
		},
		{
			name: "Single quote in schema (blocked by validation)",
			input: PostgresAttachInput{
				Name:     "test3",
				Host:     "localhost",
				Database: "mydb",
				User:     "user",
				Schema:   "public' OR '1'='1",
			},
			expectBlocked: true, // Schema is validated as identifier
		},
		{
			name: "Multiple quotes in password (escaped)",
			input: PostgresAttachInput{
				Name:     "test4",
				Host:     "localhost",
				Database: "mydb",
				User:     "user",
				Password: "pa''ss",
			},
			expectBlocked: false, // Password is escaped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.attach(ctx, tt.input)
			require.NoError(t, err)

			if tt.expectBlocked {
				// Should be blocked by identifier validation
				assert.True(t, resp.IsError)
				assert.Contains(t, resp.Content, "invalid identifier")
			} else {
				// Should fail on execution (no actual DB connection)
				// But quotes should be properly escaped
				if resp.IsError {
					assert.Contains(t, resp.Content, "failed to attach")
					// Should NOT contain SQL syntax errors
					assert.NotContains(t, resp.Content, "syntax error")
				}
			}
		})
	}
}

func TestPostgresQuery_Validation(t *testing.T) {
	service, err := NewPostgresService()
	require.NoError(t, err)
	defer service.Close()

	ctx := context.Background()

	// Test query without connection
	input := PostgresQueryInput{
		Connection: "nonexistent",
		Query:      "SELECT 1",
	}

	resp, err := service.query(ctx, input)
	require.NoError(t, err)
	assert.True(t, resp.IsError, "should return error for nonexistent connection")
	assert.Contains(t, resp.Content, "not found", "error should mention connection not found")
}

func TestPostgresQuery_OnlySelectAllowed(t *testing.T) {
	service, err := NewPostgresService()
	require.NoError(t, err)
	defer service.Close()

	// Add fake connection for testing
	service.connections["test"] = true

	ctx := context.Background()

	// Test non-SELECT query
	input := PostgresQueryInput{
		Connection: "test",
		Query:      "DELETE FROM users",
	}

	resp, err := service.query(ctx, input)
	require.NoError(t, err)
	assert.True(t, resp.IsError, "should reject non-SELECT queries")
	assert.Contains(t, resp.Content, "SELECT", "error should mention SELECT requirement")
}

func TestPostgresExecute_DangerousOperations(t *testing.T) {
	service, err := NewPostgresService()
	require.NoError(t, err)
	defer service.Close()

	// Add fake connection
	service.connections["test"] = true

	ctx := context.Background()

	dangerousCommands := []string{
		"DROP TABLE users",
		"DELETE FROM users",
		"UPDATE users SET active = false",
		"TRUNCATE TABLE users",
	}

	for _, cmd := range dangerousCommands {
		t.Run(cmd, func(t *testing.T) {
			input := PostgresExecuteInput{
				Connection: "test",
				Command:    cmd,
				Confirm:    false, // Not confirmed
			}

			resp, err := service.execute(ctx, input)
			require.NoError(t, err)
			assert.True(t, resp.IsError, "should reject dangerous operation without confirmation")
			assert.Contains(t, resp.Content, "dangerous", "error should mention dangerous operation")
		})
	}
}

func TestPostgresQuery_CTEAttackPrevention(t *testing.T) {
	service, err := NewPostgresService()
	require.NoError(t, err)
	defer service.Close()

	ctx := context.Background()

	// Mark connection as existing for test
	service.setConnection("test", true)

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "CTE with UPDATE",
			query: "WITH cte AS (SELECT 1) UPDATE users SET password = 'hacked'",
		},
		{
			name:  "CTE with DELETE",
			query: "WITH cte AS (SELECT 1) DELETE FROM users",
		},
		{
			name:  "CTE with INSERT",
			query: "WITH cte AS (SELECT 1) INSERT INTO users VALUES ('evil')",
		},
		{
			name:  "CTE with DROP",
			query: "WITH cte AS (SELECT 1) DROP TABLE users",
		},
		{
			name:  "Multiple CTEs with UPDATE",
			query: "WITH cte1 AS (SELECT 1), cte2 AS (SELECT 2) UPDATE users SET active = false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := PostgresQueryInput{
				Connection: "test",
				Query:      tt.query,
			}

			resp, err := service.query(ctx, input)
			require.NoError(t, err)
			assert.True(t, resp.IsError, "should reject CTE attack")
			assert.Contains(t, resp.Content, "only SELECT queries are allowed", "error should mention SELECT only")
		})
	}

	// Test valid CTE with SELECT (should pass)
	t.Run("Valid CTE with SELECT", func(t *testing.T) {
		input := PostgresQueryInput{
			Connection: "test",
			Query:      "WITH cte AS (SELECT 1 as id) SELECT * FROM cte",
		}

		// This will fail because we don't have actual connection, but validation should pass
		resp, err := service.query(ctx, input)
		require.NoError(t, err)
		// Should fail on execution, not validation
		if resp.IsError {
			assert.NotContains(t, resp.Content, "only SELECT queries are allowed")
		}
	})
}

func TestPostgresDetach_Validation(t *testing.T) {
	service, err := NewPostgresService()
	require.NoError(t, err)
	defer service.Close()

	ctx := context.Background()

	// Test detach nonexistent connection
	input := PostgresDetachInput{
		Connection: "nonexistent",
	}

	resp, err := service.detach(ctx, input)
	require.NoError(t, err)
	assert.True(t, resp.IsError, "should return error for nonexistent connection")
	assert.Contains(t, resp.Content, "not found", "error should mention connection not found")
}

func TestPostgresListTables_Validation(t *testing.T) {
	service, err := NewPostgresService()
	require.NoError(t, err)
	defer service.Close()

	ctx := context.Background()

	// Test without connection
	input := PostgresListTablesInput{
		Connection: "nonexistent",
	}

	resp, err := service.listTables(ctx, input)
	require.NoError(t, err)
	assert.True(t, resp.IsError, "should return error for nonexistent connection")
}

func TestPostgresDescribe_Validation(t *testing.T) {
	service, err := NewPostgresService()
	require.NoError(t, err)
	defer service.Close()

	ctx := context.Background()

	// Test without connection
	input := PostgresDescribeInput{
		Connection: "nonexistent",
		Table:      "users",
	}

	resp, err := service.describe(ctx, input)
	require.NoError(t, err)
	assert.True(t, resp.IsError, "should return error for nonexistent connection")
}

// Integration test (requires actual PostgreSQL instance)
// Uncomment and configure to test against real database
/*
func TestPostgres_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	service, err := NewPostgresService()
	require.NoError(t, err)
	defer service.Close()

	ctx := context.Background()

	// Attach to PostgreSQL
	attachInput := PostgresAttachInput{
		Name:     "testdb",
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
		User:     "postgres",
		Password: "postgres",
		ReadOnly: true,
	}

	attachResp, err := service.attach(ctx, attachInput)
	require.NoError(t, err)
	assert.False(t, attachResp.IsError)

	// List tables
	listInput := PostgresListTablesInput{
		Connection: "testdb",
		Schema:     "public",
	}

	listResp, err := service.listTables(ctx, listInput)
	require.NoError(t, err)
	assert.False(t, listResp.IsError)

	// Query
	queryInput := PostgresQueryInput{
		Connection: "testdb",
		Query:      "SELECT 1 as test",
		Limit:      10,
	}

	queryResp, err := service.query(ctx, queryInput)
	require.NoError(t, err)
	assert.False(t, queryResp.IsError)

	// Detach
	detachInput := PostgresDetachInput{
		Connection: "testdb",
	}

	detachResp, err := service.detach(ctx, detachInput)
	require.NoError(t, err)
	assert.False(t, detachResp.IsError)
}
*/
