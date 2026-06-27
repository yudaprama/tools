# MySQL Tool for Fantasy Framework

## Overview

The MySQL tool enables AI agents to interact with MySQL databases through DuckDB's mysql extension. This provides a safe, efficient way to query and manage MySQL data.

## Features

### 6 Core Tools

1. **mysql_attach** - Connect to MySQL database
2. **mysql_query** - Execute SELECT queries (read-only)
3. **mysql_execute** - Execute DDL/DML commands (with safety checks)
4. **mysql_list_tables** - List tables in database
5. **mysql_describe** - Describe table structure
6. **mysql_detach** - Disconnect from database

## Key Differences from PostgreSQL Tool

### MySQL-Specific Features
- **SHOW queries** - Supports SHOW TABLES, SHOW DATABASES, etc.
- **Unix socket** - Can connect via socket instead of TCP
- **SSL modes** - Supports disabled, required, verify_ca, verify_identity, preferred
- **Type conversions** - BIT(1) and TINYINT(1) auto-convert to BOOLEAN

### Connection Options
- TCP connection (host + port)
- Unix socket connection
- SSL/TLS support
- Environment variable support

## Safety Features

### Read-Only by Default
- Connections default to `read_only=true` for safety
- Prevents accidental data modifications

### Query Validation
- `mysql_query` accepts SELECT, WITH, and SHOW statements
- Non-SELECT queries are rejected with clear error messages

### Dangerous Operation Protection
- DROP, DELETE without WHERE, UPDATE without WHERE, TRUNCATE require `confirm=true`
- Prevents accidental data loss

### Timeouts
- Connection timeout: 10 seconds
- Query timeout: 30 seconds
- Prevents hanging operations

### Row Limits
- Default limit: 100 rows
- Prevents overwhelming responses
- Configurable per query

## Usage Examples

### 1. Connect to MySQL (TCP)

```json
{
  "name": "mysql_attach",
  "input": {
    "name": "prod_db",
    "host": "localhost",
    "port": 3306,
    "database": "myapp",
    "user": "readonly_user",
    "password": "secret",
    "read_only": true
  }
}
```

### 2. Connect via Unix Socket

```json
{
  "name": "mysql_attach",
  "input": {
    "name": "local_db",
    "database": "myapp",
    "user": "root",
    "socket": "/tmp/mysql.sock",
    "read_only": true
  }
}
```

### 3. Connect with SSL

```json
{
  "name": "mysql_attach",
  "input": {
    "name": "secure_db",
    "host": "prod.example.com",
    "port": 3306,
    "database": "myapp",
    "user": "app_user",
    "password": "secret",
    "ssl_mode": "verify_identity",
    "read_only": true
  }
}
```

**Response:**
```json
{
  "status": "connected",
  "connection": "prod_db",
  "host": "localhost",
  "database": "myapp",
  "read_only": true
}
```

### 4. Query Data

```json
{
  "name": "mysql_query",
  "input": {
    "connection": "prod_db",
    "query": "SELECT id, name, email FROM users WHERE active = 1",
    "limit": 50
  }
}
```

**Response:**
```json
{
  "connection": "prod_db",
  "rows": 42,
  "columns": ["id", "name", "email"],
  "data": [
    {"id": 1, "name": "Alice", "email": "alice@example.com"},
    {"id": 2, "name": "Bob", "email": "bob@example.com"}
  ]
}
```

### 5. SHOW Queries

```json
{
  "name": "mysql_query",
  "input": {
    "connection": "prod_db",
    "query": "SHOW TABLES"
  }
}
```

```json
{
  "name": "mysql_query",
  "input": {
    "connection": "prod_db",
    "query": "SHOW CREATE TABLE users"
  }
}
```

### 6. List Tables

```json
{
  "name": "mysql_list_tables",
  "input": {
    "connection": "prod_db"
  }
}
```

**Response:**
```json
{
  "connection": "prod_db",
  "tables": ["users", "orders", "products"],
  "count": 3
}
```

### 7. Describe Table Schema

```json
{
  "name": "mysql_describe",
  "input": {
    "connection": "prod_db",
    "table": "users"
  }
}
```

**Response:**
```json
{
  "connection": "prod_db",
  "table": "users",
  "columns": [
    {
      "name": "id",
      "type": "int",
      "nullable": false,
      "key": "PRI",
      "extra": "auto_increment"
    },
    {
      "name": "email",
      "type": "varchar(255)",
      "nullable": false,
      "key": "UNI"
    }
  ]
}
```

### 8. Execute Commands (Write Operations)

```json
{
  "name": "mysql_execute",
  "input": {
    "connection": "prod_db",
    "command": "INSERT INTO logs (message, created_at) VALUES ('Test', NOW())",
    "confirm": false
  }
}
```

**Note:** For dangerous operations (DROP, DELETE without WHERE), set `confirm: true`

### 9. Disconnect

```json
{
  "name": "mysql_detach",
  "input": {
    "connection": "prod_db"
  }
}
```

## Use Cases

### 1. Data Analysis
```
Agent: "Show me the top 10 products by sales"
Tool: mysql_query with aggregation query
```

### 2. Schema Exploration
```
Agent: "What tables are in the database?"
Tool: mysql_list_tables
Agent: "Describe the orders table"
Tool: mysql_describe
```

### 3. Database Monitoring
```
Agent: "Show me all tables"
Tool: mysql_query with SHOW TABLES
Agent: "Check table status"
Tool: mysql_query with SHOW TABLE STATUS
```

### 4. Data Export
```
Agent: "Export user data to analyze in DuckDB"
Tool: mysql_query → DuckDB local table
```

### 5. ETL Operations
```
Agent: "Copy data from MySQL to Parquet"
Tool: mysql_query → DuckDB → COPY TO parquet
```

## Connection String Formats

### Standard Format
```
host=localhost port=3306 database=mydb user=root password=secret
```

### Unix Socket
```
database=mydb user=root socket=/tmp/mysql.sock
```

### With SSL
```
host=prod.example.com port=3306 database=mydb user=app ssl_mode=verify_identity
```

## Environment Variables

MySQL connection can use environment variables:

- `MYSQL_HOST` - Default host
- `MYSQL_TCP_PORT` - Default port
- `MYSQL_DATABASE` - Default database
- `MYSQL_USER` - Default user
- `MYSQL_PWD` - Default password
- `MYSQL_UNIX_PORT` - Unix socket path

## SSL Modes

- **disabled** - No SSL
- **preferred** - Use SSL if available (default)
- **required** - Require SSL
- **verify_ca** - Verify CA certificate
- **verify_identity** - Verify server identity

## Security Best Practices

### 1. Use Read-Only Connections
```json
{
  "read_only": true  // Always use for production queries
}
```

### 2. Use Dedicated Users
- Create MySQL users with minimal permissions
- Use `GRANT SELECT` only for read-only access

### 3. Use SSL for Remote Connections
```json
{
  "ssl_mode": "verify_identity"
}
```

### 4. Limit Query Scope
- Always use WHERE clauses to limit data
- Set appropriate LIMIT values

### 5. Monitor Query Performance
- Set appropriate timeouts
- Use EXPLAIN for complex queries

## Error Handling

### Connection Errors
```json
{
  "type": "text",
  "content": "failed to attach: access denied for user",
  "is_error": true
}
```

### Query Errors
```json
{
  "type": "text",
  "content": "query failed: unknown column 'xyz'",
  "is_error": true
}
```

### Validation Errors
```json
{
  "type": "text",
  "content": "only SELECT/WITH/SHOW queries are allowed",
  "is_error": true
}
```

## Performance Considerations

### Query Optimization
- Use LIMIT to prevent large result sets
- Create indexes on frequently queried columns
- Use WHERE clauses to filter data

### Connection Pooling
- DuckDB manages connection pooling automatically
- Reuse connections when possible

### Parallel Queries
- `mysql_query`, `mysql_list_tables`, `mysql_describe` are marked as parallel-safe
- Can run concurrently with other tools

## Advanced Features

### Complex Queries
```sql
-- Aggregations
SELECT category, COUNT(*), AVG(price) 
FROM products 
GROUP BY category

-- Joins
SELECT u.name, COUNT(o.id) as order_count
FROM users u
LEFT JOIN orders o ON u.id = o.user_id
GROUP BY u.name

-- CTEs
WITH active_users AS (
  SELECT * FROM users WHERE active = 1
)
SELECT * FROM active_users WHERE created_at > '2024-01-01'
```

### SHOW Commands
```sql
SHOW TABLES
SHOW DATABASES
SHOW CREATE TABLE users
SHOW TABLE STATUS
SHOW COLUMNS FROM users
SHOW INDEX FROM users
```

## MySQL-Specific Settings

DuckDB provides MySQL-specific settings:

- `mysql_bit1_as_boolean` - Convert BIT(1) to BOOLEAN (default: true)
- `mysql_tinyint1_as_boolean` - Convert TINYINT(1) to BOOLEAN (default: true)
- `mysql_debug_show_queries` - Debug: print all queries (default: false)
- `mysql_experimental_filter_pushdown` - Filter pushdown (default: false)

## Limitations

1. **Binary Data**: Large binary columns may not display well in JSON
2. **Custom Types**: Some MySQL custom types may need special handling
3. **Transactions**: Each query runs in its own transaction
4. **Streaming**: Large result sets loaded into memory (use LIMIT)
5. **DDL Transactions**: DDL statements are not transactional in MySQL

## Comparison: MySQL vs PostgreSQL Tools

| Feature | MySQL | PostgreSQL |
| --- | --- | --- |
| SHOW queries | ✅ Supported | ❌ Not supported |
| Unix socket | ✅ Supported | ❌ Not supported |
| SSL modes | ✅ 5 modes | ✅ Basic support |
| Schema support | Database-level | Schema-level |
| Default port | 3306 | 5432 |
| Type conversion | BIT(1), TINYINT(1) | Standard types |
## Testing

Run tests:
```bash
go test -v ./pkg/fantasy/tools/builtin -run TestMySQL
```

Integration tests (requires MySQL):
```bash
# Uncomment integration test in mysql_test.go
# Configure connection details
go test -v ./pkg/fantasy/tools/builtin -run TestMySQL_Integration
```

## Dependencies

- `github.com/duckdb/duckdb-go/v2` - DuckDB Go driver
- DuckDB mysql extension (auto-installed)

## References

- [DuckDB MySQL Extension](https://duckdb.org/docs/stable/core_extensions/mysql)
- [MySQL Connection Options](https://dev.mysql.com/doc/refman/8.0/en/connecting.html)
- [MySQL SSL Configuration](https://dev.mysql.com/doc/refman/8.0/en/using-encrypted-connections.html)
