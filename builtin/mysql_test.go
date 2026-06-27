//go:build !windows

package builtin

import (
	"context"
	"testing"

	"github.com/yudaprama/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMySQLService_Creation(t *testing.T) {
	service, err := NewMySQLService()
	require.NoError(t, err, "should create mysql service")
	require.NotNil(t, service, "service should not be nil")
	defer service.Close()

	assert.NotNil(t, service.db, "db should be initialized")
	assert.NotNil(t, service.connections, "connections map should be initialized")
}

func TestMySQLTools_Registration(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := RegisterMySQL(registry)
	require.NoError(t, err, "should register mysql tools")

	// Check all tools are registered
	expectedTools := []string{
		"mysql_attach",
		"mysql_query",
		"mysql_execute",
		"mysql_list_tables",
		"mysql_describe",
		"mysql_detach",
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

func TestMySQLAttach_Validation(t *testing.T) {
	service, err := NewMySQLService()
	require.NoError(t, err)
	defer service.Close()

	ctx := context.Background()

	// Test missing required fields
	input := MySQLAttachInput{
		Name: "test",
		// Missing database, user
	}

	resp, err := service.attach(ctx, input)
	require.NoError(t, err)
	assert.True(t, resp.IsError, "should return error for missing fields")
	assert.Contains(t, resp.Content, "required", "error should mention required fields")
}

func TestMySQLAttach_SQLInjectionPrevention(t *testing.T) {
	service, err := NewMySQLService()
	require.NoError(t, err)
	defer service.Close()

	ctx := context.Background()

	tests := []struct {
		name  string
		input MySQLAttachInput
	}{
		{
			name: "Single quote in database name",
			input: MySQLAttachInput{
				Name:     "test1",
				Host:     "localhost",
				Database: "mydb' OR '1'='1",
				User:     "user",
			},
		},
		{
			name: "Single quote in password",
			input: MySQLAttachInput{
				Name:     "test2",
				Host:     "localhost",
				Database: "mydb",
				User:     "user",
				Password: "pass'word",
			},
		},
		{
			name: "Single quote in socket path",
			input: MySQLAttachInput{
				Name:     "test3",
				Socket:   "/tmp/mysql'.sock",
				Database: "mydb",
				User:     "user",
			},
		},
		{
			name: "Multiple single quotes",
			input: MySQLAttachInput{
				Name:     "test4",
				Host:     "localhost",
				Database: "my'db'test",
				User:     "user",
				Password: "pa''ss",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.attach(ctx, tt.input)
			require.NoError(t, err)

			// Should fail on execution (no actual DB connection)
			// But the important thing is it doesn't cause SQL injection
			// The error should be about connection failure, not SQL syntax
			if resp.IsError {
				assert.Contains(t, resp.Content, "failed to attach")
				// Should NOT contain unescaped quotes that would break SQL
				assert.NotContains(t, resp.Content, "syntax error")
			}
		})
	}
}

func TestMySQLQuery_Validation(t *testing.T) {
	service, err := NewMySQLService()
	require.NoError(t, err)
	defer service.Close()

	ctx := context.Background()

	// Test query without connection
	input := MySQLQueryInput{
		Connection: "nonexistent",
		Query:      "SELECT 1",
	}

	resp, err := service.query(ctx, input)
	require.NoError(t, err)
	assert.True(t, resp.IsError, "should return error for nonexistent connection")
	assert.Contains(t, resp.Content, "not found", "error should mention connection not found")
}

func TestMySQLQuery_OnlySelectAllowed(t *testing.T) {
	service, err := NewMySQLService()
	require.NoError(t, err)
	defer service.Close()

	// Add fake connection for testing (use mutex-protected method)
	service.setConnection("test", true)
	ctx := context.Background()

	// Test non-SELECT query
	input := MySQLQueryInput{
		Connection: "test",
		Query:      "DELETE FROM users",
	}

	resp, err := service.query(ctx, input)
	require.NoError(t, err)
	assert.True(t, resp.IsError, "should reject non-SELECT queries")
	assert.Contains(t, resp.Content, "SELECT", "error should mention SELECT requirement")
}

func TestMySQLQuery_ShowAllowed(t *testing.T) {
	service, err := NewMySQLService()
	require.NoError(t, err)
	defer service.Close()

	// Add fake connection
	service.connections["test"] = true

	ctx := context.Background()

	// Test SHOW query (should be allowed)
	input := MySQLQueryInput{
		Connection: "test",
		Query:      "SHOW TABLES",
	}

	// This will fail because connection doesn't exist, but validation should pass
	resp, err := service.query(ctx, input)
	require.NoError(t, err)
	// Error will be about execution, not validation
	if resp.IsError {
		assert.NotContains(t, resp.Content, "only SELECT", "SHOW queries should be allowed")
	}
}

func TestMySQLExecute_DangerousOperations(t *testing.T) {
	service, err := NewMySQLService()
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
			input := MySQLExecuteInput{
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

func TestMySQLQuery_CTEAttackPrevention(t *testing.T) {
	service, err := NewMySQLService()
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
			input := MySQLQueryInput{
				Connection: "test",
				Query:      tt.query,
			}

			resp, err := service.query(ctx, input)
			require.NoError(t, err)
			assert.True(t, resp.IsError, "should reject CTE attack")
			assert.Contains(t, resp.Content, "only SELECT", "error should mention SELECT only")
		})
	}

	// Test valid CTE with SELECT (should pass validation)
	t.Run("Valid CTE with SELECT", func(t *testing.T) {
		input := MySQLQueryInput{
			Connection: "test",
			Query:      "WITH cte AS (SELECT 1 as id) SELECT * FROM cte",
		}

		// This will fail because we don't have actual connection, but validation should pass
		resp, err := service.query(ctx, input)
		require.NoError(t, err)
		// Should fail on execution, not validation
		if resp.IsError {
			assert.NotContains(t, resp.Content, "only SELECT")
		}
	})
}

func TestMySQLDetach_Validation(t *testing.T) {
	service, err := NewMySQLService()
	require.NoError(t, err)
	defer service.Close()

	ctx := context.Background()

	// Test detach nonexistent connection
	input := MySQLDetachInput{
		Connection: "nonexistent",
	}

	resp, err := service.detach(ctx, input)
	require.NoError(t, err)
	assert.True(t, resp.IsError, "should return error for nonexistent connection")
	assert.Contains(t, resp.Content, "not found", "error should mention connection not found")
}

func TestMySQLListTables_Validation(t *testing.T) {
	service, err := NewMySQLService()
	require.NoError(t, err)
	defer service.Close()

	ctx := context.Background()

	// Test without connection
	input := MySQLListTablesInput{
		Connection: "nonexistent",
	}

	resp, err := service.listTables(ctx, input)
	require.NoError(t, err)
	assert.True(t, resp.IsError, "should return error for nonexistent connection")
}

func TestMySQLDescribe_Validation(t *testing.T) {
	service, err := NewMySQLService()
	require.NoError(t, err)
	defer service.Close()

	ctx := context.Background()

	// Test without connection
	input := MySQLDescribeInput{
		Connection: "nonexistent",
		Table:      "users",
	}

	resp, err := service.describe(ctx, input)
	require.NoError(t, err)
	assert.True(t, resp.IsError, "should return error for nonexistent connection")
}

// Integration test (requires actual MySQL instance)
// Uncomment and configure to test against real database
/*
func TestMySQL_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	service, err := NewMySQLService()
	require.NoError(t, err)
	defer service.Close()

	ctx := context.Background()

	// Attach to MySQL
	attachInput := MySQLAttachInput{
		Name:     "testdb",
		Host:     "localhost",
		Port:     3306,
		Database: "testdb",
		User:     "root",
		Password: "password",
		ReadOnly: true,
	}

	attachResp, err := service.attach(ctx, attachInput)
	require.NoError(t, err)
	assert.False(t, attachResp.IsError)

	// List tables
	listInput := MySQLListTablesInput{
		Connection: "testdb",
	}

	listResp, err := service.listTables(ctx, listInput)
	require.NoError(t, err)
	assert.False(t, listResp.IsError)

	// Query
	queryInput := MySQLQueryInput{
		Connection: "testdb",
		Query:      "SELECT 1 as test",
		Limit:      10,
	}

	queryResp, err := service.query(ctx, queryInput)
	require.NoError(t, err)
	assert.False(t, queryResp.IsError)

	// Detach
	detachInput := MySQLDetachInput{
		Connection: "testdb",
	}

	detachResp, err := service.detach(ctx, detachInput)
	require.NoError(t, err)
	assert.False(t, detachResp.IsError)
}
*/
