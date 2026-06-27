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
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// MySQLService manages MySQL connections via DuckDB
type MySQLService struct {
	db          *sql.DB
	mu          sync.RWMutex
	connections map[string]bool // Track active connections
}

// NewMySQLService creates a new MySQL service
func NewMySQLService() (*MySQLService, error) {
	// Use in-memory DuckDB for MySQL operations
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open duckdb: %w", err)
	}

	// Install and load mysql extension
	if _, err := db.Exec("INSTALL mysql"); err != nil {
		slog.Warn("Failed to install mysql extension (might be already installed)", "error", err)
	}
	if _, err := db.Exec("LOAD mysql"); err != nil {
		return nil, fmt.Errorf("failed to load mysql extension: %w", err)
	}

	return &MySQLService{
		db:          db,
		connections: make(map[string]bool),
	}, nil
}

// Close closes the service
func (s *MySQLService) Close() error {
	return s.db.Close()
}

// hasConnection checks if a connection exists (thread-safe)
func (s *MySQLService) hasConnection(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connections[name]
}

// setConnection sets or removes a connection (thread-safe)
func (s *MySQLService) setConnection(name string, active bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if active {
		s.connections[name] = true
	} else {
		delete(s.connections, name)
	}
}

// MySQLAttachInput defines input for attaching to MySQL
type MySQLAttachInput struct {
	Name     string `json:"name" jsonschema:"required,description=Connection name (e.g. 'prod_db')"`
	Host     string `json:"host,omitempty" jsonschema:"description=MySQL host (e.g. 'localhost')"`
	Port     int    `json:"port,omitempty" jsonschema:"description=MySQL port (default: 3306)"`
	Database string `json:"database" jsonschema:"required,description=Database name"`
	User     string `json:"user" jsonschema:"required,description=MySQL username"`
	Password string `json:"password,omitempty" jsonschema:"description=MySQL password"`
	Socket   string `json:"socket,omitempty" jsonschema:"description=Unix socket path (alternative to host/port)"`
	SSLMode  string `json:"ssl_mode,omitempty" jsonschema:"description=SSL mode: disabled&#44; required&#44; verify_ca&#44; verify_identity&#44; preferred (default)"`
	ReadOnly *bool  `json:"read_only,omitempty" jsonschema:"description=Attach in read-only mode (default: true for safety)"`
}

// MySQLQueryInput defines input for querying MySQL
type MySQLQueryInput struct {
	Connection string `json:"connection" jsonschema:"required,description=Connection name from attach"`
	Query      string `json:"query" jsonschema:"required,description=SQL SELECT query to execute"`
	Limit      int    `json:"limit,omitempty" jsonschema:"description=Maximum rows to return (default: 100)"`
}

// MySQLExecuteInput defines input for executing MySQL commands
type MySQLExecuteInput struct {
	Connection string `json:"connection" jsonschema:"required,description=Connection name from attach"`
	Command    string `json:"command" jsonschema:"required,description=SQL command to execute (DDL/DML)"`
	Confirm    bool   `json:"confirm,omitempty" jsonschema:"description=Confirm dangerous operations (required for DROP/DELETE)"`
}

// MySQLListTablesInput defines input for listing tables
type MySQLListTablesInput struct {
	Connection string `json:"connection" jsonschema:"required,description=Connection name from attach"`
	Database   string `json:"database,omitempty" jsonschema:"description=Database name (default: attached database)"`
}

// MySQLDescribeInput defines input for describing table schema
type MySQLDescribeInput struct {
	Connection string `json:"connection" jsonschema:"required,description=Connection name from attach"`
	Table      string `json:"table" jsonschema:"required,description=Table name to describe"`
	Database   string `json:"database,omitempty" jsonschema:"description=Database name (default: attached database)"`
}

// MySQLDetachInput defines input for detaching connection
type MySQLDetachInput struct {
	Connection string `json:"connection" jsonschema:"required,description=Connection name to detach"`
}

// NewMySQL registers all MySQL tools
func NewMySQL(_ context.Context) ([]tool.InvokableTool, error) {
	service, err := NewMySQLService()
	if err != nil {
		return nil, fmt.Errorf("failed to create mysql service: %w", err)
	}

	attachTool, err := utils.InferTool("mysql_attach",
		"Connect to a MySQL database. Returns connection info. Use read_only=true (default) for safety.",
		func(ctx context.Context, input *MySQLAttachInput) (string, error) {
			return service.attach(ctx, *input)
		},
	)
	if err != nil {
		return nil, err
	}

	queryTool, err := utils.InferTool("mysql_query",
		"Execute a SELECT query on attached MySQL database. Returns query results as JSON.",
		func(ctx context.Context, input *MySQLQueryInput) (string, error) {
			return service.query(ctx, *input)
		},
	)
	if err != nil {
		return nil, err
	}

	executeTool, err := utils.InferTool("mysql_execute",
		"Execute DDL/DML commands on MySQL (CREATE, INSERT, UPDATE, DELETE). Requires confirm=true for dangerous operations.",
		func(ctx context.Context, input *MySQLExecuteInput) (string, error) {
			return service.execute(ctx, *input)
		},
	)
	if err != nil {
		return nil, err
	}

	listTool, err := utils.InferTool("mysql_list_tables",
		"List all tables in a MySQL database. Returns table names and row counts.",
		func(ctx context.Context, input *MySQLListTablesInput) (string, error) {
			return service.listTables(ctx, *input)
		},
	)
	if err != nil {
		return nil, err
	}

	describeTool, err := utils.InferTool("mysql_describe",
		"Describe table schema (columns, types, constraints). Returns detailed table structure.",
		func(ctx context.Context, input *MySQLDescribeInput) (string, error) {
			return service.describe(ctx, *input)
		},
	)
	if err != nil {
		return nil, err
	}

	detachTool, err := utils.InferTool("mysql_detach",
		"Disconnect from MySQL database. Cleans up connection resources.",
		func(ctx context.Context, input *MySQLDetachInput) (string, error) {
			return service.detach(ctx, *input)
		},
	)
	if err != nil {
		return nil, err
	}

	return []tool.InvokableTool{attachTool, queryTool, executeTool, listTool, describeTool, detachTool}, nil
}

// attach connects to MySQL database
func (s *MySQLService) attach(ctx context.Context, input MySQLAttachInput) (string, error) {
	// Validate input
	if input.Name == "" || input.Database == "" || input.User == "" {
		return "", fmt.Errorf("name, database, and user are required")
	}

	// Validate that either host or socket is provided
	if input.Socket == "" && input.Host == "" {
		return "", fmt.Errorf("either host or socket must be provided")
	}

	// Validate identifiers for SQL injection protection
	if err := validateSQLIdent(input.Name); err != nil {
		return "", err
	}

	// Check if already connected
	if s.hasConnection(input.Name) {
		return "", fmt.Errorf("connection '%s' already exists", input.Name)
	}

	// Default values
	if input.Port == 0 && input.Socket == "" {
		input.Port = 3306
	}
	// Default ReadOnly to true for safety
	readOnly := true
	if input.ReadOnly != nil {
		readOnly = *input.ReadOnly
	}

	// Build connection string
	var connStr string
	if input.Socket != "" {
		// Unix socket connection
		connStr = fmt.Sprintf("database=%s user=%s socket=%s",
			input.Database, input.User, input.Socket)
	} else {
		// TCP connection
		connStr = fmt.Sprintf("database=%s host=%s port=%d user=%s",
			input.Database, input.Host, input.Port, input.User)
	}

	if input.Password != "" {
		connStr += fmt.Sprintf(" password=%s", input.Password)
	}

	if input.SSLMode != "" {
		connStr += fmt.Sprintf(" ssl_mode=%s", input.SSLMode)
	}

	// Build ATTACH command
	safeConnStr := strings.ReplaceAll(connStr, "'", "''")
	attachCmd := fmt.Sprintf("ATTACH '%s' AS %s (TYPE MYSQL", safeConnStr, input.Name)
	if readOnly {
		attachCmd += ", READ_ONLY"
	}
	attachCmd += ")"

	slog.InfoContext(ctx, "Attaching to MySQL", "name", input.Name, "host", input.Host, "database", input.Database)

	// Execute ATTACH with timeout
	execCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if _, err := s.db.ExecContext(execCtx, attachCmd); err != nil {
		return "", fmt.Errorf("failed to attach: %v", err)
	}

	// Mark as connected
	s.setConnection(input.Name, true)

	result := map[string]interface{}{
		"status":     "connected",
		"connection": input.Name,
		"host":       input.Host,
		"database":   input.Database,
		"read_only":  readOnly,
	}

	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}

// query executes a SELECT query
func (s *MySQLService) query(ctx context.Context, input MySQLQueryInput) (string, error) {
	// Validate input
	if input.Connection == "" || input.Query == "" {
		return "", fmt.Errorf("connection and query are required")
	}

	// Validate identifier
	if err := validateSQLIdent(input.Connection); err != nil {
		return "", err
	}

	// Check connection exists
	if !s.hasConnection(input.Connection) {
		return "", fmt.Errorf("connection '%s' not found. Use mysql_attach first.", input.Connection)
	}

	// Validate query is SELECT
	rawQuery := strings.TrimSpace(input.Query)
	rawQuery = strings.TrimSuffix(rawQuery, ";")
	queryUpper := strings.ToUpper(rawQuery)

	// Strip leading CTEs and validate the actual query command
	actualCommand := stripLeadingCTEs(queryUpper)
	if !strings.HasPrefix(actualCommand, "SELECT") && !strings.HasPrefix(actualCommand, "SHOW") {
		return "", fmt.Errorf("only SELECT/WITH/SHOW queries are allowed. Use mysql_execute for other commands.")
	}

	// Additional check: ensure no DML keywords appear in the query
	// This catches cases like: WITH cte AS (...) UPDATE/DELETE/INSERT
	dangerousKeywords := []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE ", "DROP ", "ALTER ", "CREATE ", "TRUNCATE "}
	for _, keyword := range dangerousKeywords {
		if strings.Contains(queryUpper, keyword) {
			return "", fmt.Errorf("only SELECT/WITH/SHOW queries are allowed. Use mysql_execute for other commands.")
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

	slog.InfoContext(ctx, "Executing MySQL query", "connection", input.Connection, "query", query)

	// Execute query with timeout
	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(execCtx, query)
	if err != nil {
		return "", fmt.Errorf("query failed: %v", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("failed to get columns: %v", err)
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
			return "", fmt.Errorf("scan failed: %v", err)
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
		return "", fmt.Errorf("rows iteration error: %v", err)
	}

	response := map[string]interface{}{
		"connection": input.Connection,
		"rows":       len(results),
		"columns":    columns,
		"data":       results,
	}

	resultJSON, _ := json.MarshalIndent(response, "", "  ")
	return string(resultJSON), nil
}

// execute runs DDL/DML commands
func (s *MySQLService) execute(ctx context.Context, input MySQLExecuteInput) (string, error) {
	// Validate input
	if input.Connection == "" || input.Command == "" {
		return "", fmt.Errorf("connection and command are required")
	}

	// Validate identifier
	if err := validateSQLIdent(input.Connection); err != nil {
		return "", err
	}

	// Check connection exists
	if !s.hasConnection(input.Connection) {
		return "", fmt.Errorf("connection '%s' not found", input.Connection)
	}

	// Check for dangerous operations
	cmdUpper := strings.ToUpper(strings.TrimSpace(input.Command))
	isDangerous := strings.Contains(cmdUpper, "DROP") ||
		(strings.Contains(cmdUpper, "DELETE") && !strings.Contains(cmdUpper, "WHERE")) ||
		(strings.Contains(cmdUpper, "UPDATE") && !strings.Contains(cmdUpper, "WHERE")) ||
		strings.Contains(cmdUpper, "TRUNCATE")

	if isDangerous && !input.Confirm {
		return "", fmt.Errorf("dangerous operation detected. Set confirm=true to proceed.")
	}

	slog.InfoContext(ctx, "Executing MySQL command", "connection", input.Connection, "command", input.Command)

	// Use mysql_execute function for proper execution
	execQuery := fmt.Sprintf("CALL mysql_execute('%s', '%s')",
		input.Connection,
		strings.ReplaceAll(input.Command, "'", "''")) // Escape single quotes

	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := s.db.ExecContext(execCtx, execQuery)
	if err != nil {
		return "", fmt.Errorf("execution failed: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()

	response := map[string]interface{}{
		"status":        "success",
		"connection":    input.Connection,
		"rows_affected": rowsAffected,
	}

	resultJSON, _ := json.Marshal(response)
	return string(resultJSON), nil
}

// listTables lists all tables in database
func (s *MySQLService) listTables(ctx context.Context, input MySQLListTablesInput) (string, error) {
	if input.Connection == "" {
		return "", fmt.Errorf("connection is required")
	}

	// Validate identifiers
	if err := validateSQLIdent(input.Connection); err != nil {
		return "", err
	}
	if input.Database != "" {
		if err := validateSQLIdent(input.Database); err != nil {
			return "", err
		}
	}

	if !s.hasConnection(input.Connection) {
		return "", fmt.Errorf("connection '%s' not found", input.Connection)
	}

	// Use SHOW TABLES query
	schemaExpr := "DATABASE()"
	if input.Database != "" {
		schemaExpr = fmt.Sprintf("'%s'", input.Database)
	}
	query := fmt.Sprintf(
		"SELECT table_name FROM %s.information_schema.tables WHERE table_schema = %s ORDER BY table_name",
		input.Connection,
		schemaExpr,
	)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to list tables: %v", err)
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
		"tables":     tables,
		"count":      len(tables),
	}

	resultJSON, _ := json.MarshalIndent(response, "", "  ")
	return string(resultJSON), nil
}

// describe describes table schema
func (s *MySQLService) describe(ctx context.Context, input MySQLDescribeInput) (string, error) {
	if input.Connection == "" || input.Table == "" {
		return "", fmt.Errorf("connection and table are required")
	}

	// Validate identifiers
	if err := validateSQLIdent(input.Connection); err != nil {
		return "", err
	}
	if err := validateSQLIdent(input.Table); err != nil {
		return "", err
	}
	if err := validateSQLIdent(input.Database); err != nil {
		return "", err
	}

	if !s.hasConnection(input.Connection) {
		return "", fmt.Errorf("connection '%s' not found", input.Connection)
	}

	query := fmt.Sprintf(`
		SELECT
			column_name,
			data_type,
			is_nullable,
			column_default,
			column_key,
			extra
		FROM %s.information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = '%s'
		ORDER BY ordinal_position
	`, input.Connection, input.Table)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to describe table: %v", err)
	}
	defer rows.Close()

	columns := make([]map[string]interface{}, 0)
	for rows.Next() {
		var colName, dataType, isNullable string
		var colDefault, colKey, extra sql.NullString
		if err := rows.Scan(&colName, &dataType, &isNullable, &colDefault, &colKey, &extra); err != nil {
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
		if colKey.Valid && colKey.String != "" {
			col["key"] = colKey.String
		}
		if extra.Valid && extra.String != "" {
			col["extra"] = extra.String
		}
		columns = append(columns, col)
	}

	response := map[string]interface{}{
		"connection": input.Connection,
		"table":      input.Table,
		"columns":    columns,
	}

	resultJSON, _ := json.MarshalIndent(response, "", "  ")
	return string(resultJSON), nil
}

// detach disconnects from MySQL
func (s *MySQLService) detach(ctx context.Context, input MySQLDetachInput) (string, error) {
	if input.Connection == "" {
		return "", fmt.Errorf("connection is required")
	}

	// Validate identifier
	if err := validateSQLIdent(input.Connection); err != nil {
		return "", err
	}

	if !s.hasConnection(input.Connection) {
		return "", fmt.Errorf("connection '%s' not found", input.Connection)
	}

	detachCmd := fmt.Sprintf("DETACH %s", input.Connection)
	if _, err := s.db.ExecContext(ctx, detachCmd); err != nil {
		return "", fmt.Errorf("failed to detach: %v", err)
	}

	s.setConnection(input.Connection, false)

	result := map[string]interface{}{
		"status":     "disconnected",
		"connection": input.Connection,
	}

	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}
