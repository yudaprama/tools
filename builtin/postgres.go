package builtin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"log/slog"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/getkawai/unillm"
	"github.com/yudaprama/tools"
)

// PostgresService manages PostgreSQL connections via DuckDB
type PostgresService struct {
	db          *sql.DB
	mu          sync.RWMutex
	connections map[string]bool // Track active connections
}

// NewPostgresService creates a new PostgreSQL service
func NewPostgresService() (*PostgresService, error) {
	// Use in-memory DuckDB for PostgreSQL operations
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open duckdb: %w", err)
	}

	// Install and load postgres extension
	if _, err := db.Exec("INSTALL postgres"); err != nil {
		slog.Warn("Failed to install postgres extension (might be already installed)", "error", err)
	}
	if _, err := db.Exec("LOAD postgres"); err != nil {
		return nil, fmt.Errorf("failed to load postgres extension: %w", err)
	}

	return &PostgresService{
		db:          db,
		connections: make(map[string]bool),
	}, nil
}

// Close closes the service
func (s *PostgresService) Close() error {
	return s.db.Close()
}

// hasConnection checks if a connection exists (thread-safe)
func (s *PostgresService) hasConnection(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connections[name]
}

// setConnection sets or removes a connection (thread-safe)
func (s *PostgresService) setConnection(name string, active bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if active {
		s.connections[name] = true
	} else {
		delete(s.connections, name)
	}
}

// PostgresAttachInput defines input for attaching to PostgreSQL
type PostgresAttachInput struct {
	Name     string `json:"name" jsonschema:"required,description=Connection name (e.g. 'prod_db')"`
	Host     string `json:"host" jsonschema:"required,description=PostgreSQL host (e.g. 'localhost')"`
	Port     int    `json:"port,omitempty" jsonschema:"description=PostgreSQL port (default: 5432)"`
	Database string `json:"database" jsonschema:"required,description=Database name"`
	User     string `json:"user" jsonschema:"required,description=PostgreSQL username"`
	Password string `json:"password,omitempty" jsonschema:"description=PostgreSQL password"`
	Schema   string `json:"schema,omitempty" jsonschema:"description=Specific schema to attach (default: all schemas)"`
	ReadOnly *bool  `json:"read_only,omitempty" jsonschema:"description=Attach in read-only mode (default: true for safety)"`
}

// PostgresQueryInput defines input for querying PostgreSQL
type PostgresQueryInput struct {
	Connection string `json:"connection" jsonschema:"required,description=Connection name from attach"`
	Query      string `json:"query" jsonschema:"required,description=SQL SELECT query to execute"`
	Limit      int    `json:"limit,omitempty" jsonschema:"description=Maximum rows to return (default: 100)"`
}

// PostgresExecuteInput defines input for executing PostgreSQL commands
type PostgresExecuteInput struct {
	Connection string `json:"connection" jsonschema:"required,description=Connection name from attach"`
	Command    string `json:"command" jsonschema:"required,description=SQL command to execute (DDL/DML)"`
	Confirm    bool   `json:"confirm,omitempty" jsonschema:"description=Confirm dangerous operations (required for DROP/DELETE)"`
}

// PostgresListTablesInput defines input for listing tables
type PostgresListTablesInput struct {
	Connection string `json:"connection" jsonschema:"required,description=Connection name from attach"`
	Schema     string `json:"schema,omitempty" jsonschema:"description=Schema name (default: public)"`
}

// PostgresDescribeInput defines input for describing table schema
type PostgresDescribeInput struct {
	Connection string `json:"connection" jsonschema:"required,description=Connection name from attach"`
	Table      string `json:"table" jsonschema:"required,description=Table name to describe"`
	Schema     string `json:"schema,omitempty" jsonschema:"description=Schema name (default: public)"`
}

// PostgresDetachInput defines input for detaching connection
type PostgresDetachInput struct {
	Connection string `json:"connection" jsonschema:"required,description=Connection name to detach"`
}

// RegisterPostgres registers all PostgreSQL tools
func RegisterPostgres(registry *tools.ToolRegistry) error {
	service, err := NewPostgresService()
	if err != nil {
		return fmt.Errorf("failed to create postgres service: %w", err)
	}

	// Register postgres_attach tool
	attachTool := unillm.NewAgentTool("postgres_attach",
		"Connect to a PostgreSQL database. Returns connection info. Use read_only=true (default) for safety.",
		func(ctx context.Context, input PostgresAttachInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			return service.attach(ctx, input)
		},
	)
	if err := registry.Register(attachTool); err != nil {
		return err
	}

	// Register postgres_query tool
	queryTool := unillm.NewParallelAgentTool("postgres_query",
		"Execute a SELECT query on attached PostgreSQL database. Returns query results as JSON.",
		func(ctx context.Context, input PostgresQueryInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			return service.query(ctx, input)
		},
	)
	if err := registry.Register(queryTool); err != nil {
		return err
	}

	// Register postgres_execute tool
	executeTool := unillm.NewAgentTool("postgres_execute",
		"Execute DDL/DML commands on PostgreSQL (CREATE, INSERT, UPDATE, DELETE). Requires confirm=true for dangerous operations.",
		func(ctx context.Context, input PostgresExecuteInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			return service.execute(ctx, input)
		},
	)
	if err := registry.Register(executeTool); err != nil {
		return err
	}

	// Register postgres_list_tables tool
	listTool := unillm.NewParallelAgentTool("postgres_list_tables",
		"List all tables in a PostgreSQL schema. Returns table names and row counts.",
		func(ctx context.Context, input PostgresListTablesInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			return service.listTables(ctx, input)
		},
	)
	if err := registry.Register(listTool); err != nil {
		return err
	}

	// Register postgres_describe tool
	describeTool := unillm.NewParallelAgentTool("postgres_describe",
		"Describe table schema (columns, types, constraints). Returns detailed table structure.",
		func(ctx context.Context, input PostgresDescribeInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			return service.describe(ctx, input)
		},
	)
	if err := registry.Register(describeTool); err != nil {
		return err
	}

	// Register postgres_detach tool
	detachTool := unillm.NewAgentTool("postgres_detach",
		"Disconnect from PostgreSQL database. Cleans up connection resources.",
		func(ctx context.Context, input PostgresDetachInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			return service.detach(ctx, input)
		},
	)
	if err := registry.Register(detachTool); err != nil {
		return err
	}

	return nil
}

// attach connects to PostgreSQL database
func (s *PostgresService) attach(ctx context.Context, input PostgresAttachInput) (unillm.ToolResponse, error) {
	// Validate input
	if input.Name == "" || input.Host == "" || input.Database == "" || input.User == "" {
		return unillm.NewTextErrorResponse("name, host, database, and user are required"), nil
	}

	// Validate identifiers for SQL injection protection
	if err := validateSQLIdent(input.Name); err != nil {
		return unillm.NewTextErrorResponse(err.Error()), nil
	}
	if err := validateSQLIdent(input.Schema); err != nil {
		return unillm.NewTextErrorResponse(err.Error()), nil
	}

	// Check if already connected
	if s.hasConnection(input.Name) {
		return unillm.NewTextErrorResponse(fmt.Sprintf("connection '%s' already exists", input.Name)), nil
	}

	// Default values
	if input.Port == 0 {
		input.Port = 5432
	}
	// Default ReadOnly to true for safety
	readOnly := true
	if input.ReadOnly != nil {
		readOnly = *input.ReadOnly
	}

	// Build connection string
	connStr := fmt.Sprintf("dbname=%s host=%s port=%d user=%s",
		input.Database, input.Host, input.Port, input.User)
	if input.Password != "" {
		connStr += fmt.Sprintf(" password=%s", input.Password)
	}

	// Escape single quotes in connection string to prevent SQL injection
	safeConnStr := strings.ReplaceAll(connStr, "'", "''")

	// Build ATTACH command
	attachCmd := fmt.Sprintf("ATTACH '%s' AS %s (TYPE POSTGRES", safeConnStr, input.Name)
	if input.Schema != "" {
		// Escape schema name as well
		safeSchema := strings.ReplaceAll(input.Schema, "'", "''")
		attachCmd += fmt.Sprintf(", SCHEMA '%s'", safeSchema)
	}
	if readOnly {
		attachCmd += ", READ_ONLY"
	}
	attachCmd += ")"

	slog.InfoContext(ctx, "Attaching to PostgreSQL", "name", input.Name, "host", input.Host, "database", input.Database)

	// Execute ATTACH with timeout
	execCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if _, err := s.db.ExecContext(execCtx, attachCmd); err != nil {
		return unillm.NewTextErrorResponse(fmt.Sprintf("failed to attach: %v", err)), nil
	}

	// Mark as connected
	s.setConnection(input.Name, true)

	result := map[string]interface{}{
		"status":     "connected",
		"connection": input.Name,
		"host":       input.Host,
		"database":   input.Database,
		"read_only":  readOnly,
		"schema":     input.Schema,
	}

	resultJSON, _ := json.Marshal(result)
	return unillm.NewTextResponse(string(resultJSON)), nil
}

// query executes a SELECT query
func (s *PostgresService) query(ctx context.Context, input PostgresQueryInput) (unillm.ToolResponse, error) {
	// Validate input
	if input.Connection == "" || input.Query == "" {
		return unillm.NewTextErrorResponse("connection and query are required"), nil
	}

	// Validate identifier
	if err := validateSQLIdent(input.Connection); err != nil {
		return unillm.NewTextErrorResponse(err.Error()), nil
	}

	// Check connection exists
	if !s.hasConnection(input.Connection) {
		return unillm.NewTextErrorResponse(fmt.Sprintf("connection '%s' not found. Use postgres_attach first.", input.Connection)), nil
	}

	// Validate query is SELECT
	rawQuery := strings.TrimSpace(input.Query)
	rawQuery = strings.TrimSuffix(rawQuery, ";")
	if strings.Contains(rawQuery, ";") {
		return unillm.NewTextErrorResponse("multiple statements are not allowed in postgres_query"), nil
	}
	queryUpper := strings.ToUpper(rawQuery)

	// Strip leading CTEs and validate the actual query command
	actualCommand := stripLeadingCTEs(queryUpper)
	if !strings.HasPrefix(actualCommand, "SELECT") {
		return unillm.NewTextErrorResponse("only SELECT queries are allowed. Use postgres_execute for other commands."), nil
	}

	// Additional check: ensure no DML keywords appear in the query
	// This catches cases like: WITH cte AS (...) UPDATE/DELETE/INSERT
	dangerousKeywords := []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE ", "DROP ", "ALTER ", "CREATE ", "TRUNCATE "}
	for _, keyword := range dangerousKeywords {
		if strings.Contains(queryUpper, keyword) {
			return unillm.NewTextErrorResponse("only SELECT queries are allowed. Use postgres_execute for other commands."), nil
		}
	}

	// Default limit
	limit := input.Limit
	if limit == 0 {
		limit = 100
	}

	// Add LIMIT if not present
	query := rawQuery
	if !strings.Contains(queryUpper, "LIMIT") {
		query = fmt.Sprintf("%s LIMIT %d", query, limit)
	}

	slog.InfoContext(ctx, "Executing PostgreSQL query", "connection", input.Connection, "query", query)

	// Execute query with timeout
	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(execCtx, query)
	if err != nil {
		return unillm.NewTextErrorResponse(fmt.Sprintf("query failed: %v", err)), nil
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return unillm.NewTextErrorResponse(fmt.Sprintf("failed to get columns: %v", err)), nil
	}

	// Fetch results
	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		// Create slice of interface{} for scanning
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return unillm.NewTextErrorResponse(fmt.Sprintf("scan failed: %v", err)), nil
		}

		// Build result map
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Convert []byte to string for better JSON representation
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return unillm.NewTextErrorResponse(fmt.Sprintf("rows iteration error: %v", err)), nil
	}

	response := map[string]interface{}{
		"connection": input.Connection,
		"rows":       len(results),
		"columns":    columns,
		"data":       results,
	}

	resultJSON, _ := json.MarshalIndent(response, "", "  ")
	return unillm.NewTextResponse(string(resultJSON)), nil
}

// execute runs DDL/DML commands
func (s *PostgresService) execute(ctx context.Context, input PostgresExecuteInput) (unillm.ToolResponse, error) {
	// Validate input
	if input.Connection == "" || input.Command == "" {
		return unillm.NewTextErrorResponse("connection and command are required"), nil
	}

	// Validate identifier
	if err := validateSQLIdent(input.Connection); err != nil {
		return unillm.NewTextErrorResponse(err.Error()), nil
	}

	// Check connection exists
	if !s.hasConnection(input.Connection) {
		return unillm.NewTextErrorResponse(fmt.Sprintf("connection '%s' not found", input.Connection)), nil
	}

	// Check for dangerous operations
	cmdUpper := strings.ToUpper(strings.TrimSpace(input.Command))
	isDangerous := strings.Contains(cmdUpper, "DROP") ||
		(strings.Contains(cmdUpper, "DELETE") && !strings.Contains(cmdUpper, "WHERE")) ||
		(strings.Contains(cmdUpper, "UPDATE") && !strings.Contains(cmdUpper, "WHERE")) ||
		strings.Contains(cmdUpper, "TRUNCATE")

	if isDangerous && !input.Confirm {
		return unillm.NewTextErrorResponse("dangerous operation detected. Set confirm=true to proceed."), nil
	}

	slog.InfoContext(ctx, "Executing PostgreSQL command", "connection", input.Connection, "command", input.Command)

	// Use postgres_execute function for proper execution
	execQuery := fmt.Sprintf("CALL postgres_execute('%s', '%s')",
		input.Connection,
		strings.ReplaceAll(input.Command, "'", "''")) // Escape single quotes

	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := s.db.ExecContext(execCtx, execQuery)
	if err != nil {
		return unillm.NewTextErrorResponse(fmt.Sprintf("execution failed: %v", err)), nil
	}

	rowsAffected, _ := result.RowsAffected()

	response := map[string]interface{}{
		"status":        "success",
		"connection":    input.Connection,
		"rows_affected": rowsAffected,
	}

	resultJSON, _ := json.Marshal(response)
	return unillm.NewTextResponse(string(resultJSON)), nil
}

// listTables lists all tables in schema
func (s *PostgresService) listTables(ctx context.Context, input PostgresListTablesInput) (unillm.ToolResponse, error) {
	if input.Connection == "" {
		return unillm.NewTextErrorResponse("connection is required"), nil
	}

	// Validate identifiers
	if err := validateSQLIdent(input.Connection); err != nil {
		return unillm.NewTextErrorResponse(err.Error()), nil
	}
	if err := validateSQLIdent(input.Schema); err != nil {
		return unillm.NewTextErrorResponse(err.Error()), nil
	}

	if !s.hasConnection(input.Connection) {
		return unillm.NewTextErrorResponse(fmt.Sprintf("connection '%s' not found", input.Connection)), nil
	}

	schema := input.Schema
	if schema == "" {
		schema = "public"
	}

	query := fmt.Sprintf(`
		SELECT table_name 
		FROM %s.information_schema.tables 
		WHERE table_schema = '%s' 
		ORDER BY table_name
	`, input.Connection, schema)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return unillm.NewTextErrorResponse(fmt.Sprintf("failed to list tables: %v", err)), nil
	}
	defer rows.Close()

	tables := make([]string, 0)
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}
		tables = append(tables, tableName)
	}

	response := map[string]interface{}{
		"connection": input.Connection,
		"schema":     schema,
		"tables":     tables,
		"count":      len(tables),
	}

	resultJSON, _ := json.MarshalIndent(response, "", "  ")
	return unillm.NewTextResponse(string(resultJSON)), nil
}

// describe describes table schema
func (s *PostgresService) describe(ctx context.Context, input PostgresDescribeInput) (unillm.ToolResponse, error) {
	if input.Connection == "" || input.Table == "" {
		return unillm.NewTextErrorResponse("connection and table are required"), nil
	}

	// Validate identifiers
	if err := validateSQLIdent(input.Connection); err != nil {
		return unillm.NewTextErrorResponse(err.Error()), nil
	}
	if err := validateSQLIdent(input.Table); err != nil {
		return unillm.NewTextErrorResponse(err.Error()), nil
	}
	if err := validateSQLIdent(input.Schema); err != nil {
		return unillm.NewTextErrorResponse(err.Error()), nil
	}

	if !s.hasConnection(input.Connection) {
		return unillm.NewTextErrorResponse(fmt.Sprintf("connection '%s' not found", input.Connection)), nil
	}

	schema := input.Schema
	if schema == "" {
		schema = "public"
	}

	query := fmt.Sprintf(`
		SELECT 
			column_name,
			data_type,
			is_nullable,
			column_default
		FROM %s.information_schema.columns
		WHERE table_schema = '%s' AND table_name = '%s'
		ORDER BY ordinal_position
	`, input.Connection, schema, input.Table)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return unillm.NewTextErrorResponse(fmt.Sprintf("failed to describe table: %v", err)), nil
	}
	defer rows.Close()

	columns := make([]map[string]interface{}, 0)
	for rows.Next() {
		var colName, dataType, isNullable string
		var colDefault sql.NullString
		if err := rows.Scan(&colName, &dataType, &isNullable, &colDefault); err != nil {
			continue
		}

		col := map[string]interface{}{
			"name":     colName,
			"type":     dataType,
			"nullable": isNullable == "YES",
		}
		if colDefault.Valid {
			col["default"] = colDefault.String
		}
		columns = append(columns, col)
	}

	response := map[string]interface{}{
		"connection": input.Connection,
		"schema":     schema,
		"table":      input.Table,
		"columns":    columns,
	}

	resultJSON, _ := json.MarshalIndent(response, "", "  ")
	return unillm.NewTextResponse(string(resultJSON)), nil
}

// detach disconnects from PostgreSQL
func (s *PostgresService) detach(ctx context.Context, input PostgresDetachInput) (unillm.ToolResponse, error) {
	if input.Connection == "" {
		return unillm.NewTextErrorResponse("connection is required"), nil
	}

	// Validate identifier
	if err := validateSQLIdent(input.Connection); err != nil {
		return unillm.NewTextErrorResponse(err.Error()), nil
	}

	if !s.hasConnection(input.Connection) {
		return unillm.NewTextErrorResponse(fmt.Sprintf("connection '%s' not found", input.Connection)), nil
	}

	detachCmd := fmt.Sprintf("DETACH %s", input.Connection)
	if _, err := s.db.ExecContext(ctx, detachCmd); err != nil {
		return unillm.NewTextErrorResponse(fmt.Sprintf("failed to detach: %v", err)), nil
	}

	s.setConnection(input.Connection, false)

	result := map[string]interface{}{
		"status":     "disconnected",
		"connection": input.Connection,
	}

	resultJSON, _ := json.Marshal(result)
	return unillm.NewTextResponse(string(resultJSON)), nil
}
