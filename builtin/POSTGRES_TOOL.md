# PostgreSQL Tool for Fantasy Framework

## Overview

The PostgreSQL tool enables AI agents to interact with PostgreSQL databases through DuckDB's postgres extension. This provides a safe, efficient way to query and manage PostgreSQL data.

## Features

### 6 Core Tools

1. **postgres_attach** - Connect to PostgreSQL database
2. **postgres_query** - Execute SELECT queries (read-only)
3. **postgres_execute** - Execute DDL/DML commands (with safety checks)
4. **postgres_list_tables** - List tables in schema
5. **postgres_describe** - Describe table structure
6. **postgres_detach** - Disconnect from database

## Safety Features

### Read-Only by Default
- Connections default to `read_only=true` for safety
- Prevents accidental data modifications

### Query Validation
- `postgres_query` only accepts SELECT statements
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

### 1. Connect to PostgreSQL

```json
{
  "name": "postgres_attach",
  "input": {
    "name": "prod_db",
    "host": "localhost",
    "port": 5432,
    "database": "myapp",
    "user": "readonly_user",
    "password": "secret",
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

### 2. Query Data

```json
{
  "name": "postgres_query",
  "input": {
    "connection": "prod_db",
    "query": "SELECT id, name, email FROM users WHERE active = true",
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

### 3. List Tables

```json
{
  "name": "postgres_list_tables",
  "input": {
    "connection": "prod_db",
    "schema": "public"
  }
}
```

**Response:**
```json
{
  "connection": "prod_db",
  "schema": "public",
  "tables": ["users", "orders", "products"],
  "count": 3
}
```

### 4. Describe Table Schema

```json
{
  "name": "postgres_describe",
  "input": {
    "connection": "prod_db",
    "table": "users",
    "schema": "public"
  }
}
```

**Response:**
```json
{
  "connection": "prod_db",
  "schema": "public",
  "table": "users",
  "columns": [
    {
      "name": "id",
      "type": "integer",
      "nullable": false,
      "default": "nextval('users_id_seq'::regclass)"
    },
    {
      "name": "email",
      "type": "character varying",
      "nullable": false
    }
  ]
}
```

### 5. Execute Commands (Write Operations)

```json
{
  "name": "postgres_execute",
  "input": {
    "connection": "prod_db",
    "command": "INSERT INTO logs (message, created_at) VALUES ('Test', NOW())",
    "confirm": false
  }
}
```

**Note:** For dangerous operations (DROP, DELETE without WHERE), set `confirm: true`

### 6. Disconnect

```json
{
  "name": "postgres_detach",
  "input": {
    "connection": "prod_db"
  }
}
```

## Use Cases

### 1. Data Analysis
```
Agent: "Show me the top 10 customers by order count"
Tool: postgres_query with aggregation query
```

### 2. Schema Exploration
```
Agent: "What tables are in the database?"
Tool: postgres_list_tables
Agent: "Describe the users table"
Tool: postgres_describe
```

### 3. Data Export
```
Agent: "Export user data to analyze in DuckDB"
Tool: postgres_query → DuckDB local table
```

### 4. Monitoring
```
Agent: "Check if there are any failed jobs"
Tool: postgres_query on jobs table
```

### 5. ETL Operations
```
Agent: "Copy data from Postgres to Parquet"
Tool: postgres_query → DuckDB → COPY TO parquet
```

## Connection String Formats

### Standard Format
```
host=localhost port=5432 dbname=mydb user=postgres password=secret
```

### URI Format
```
postgresql://username:password@hostname:5432/database
```

### With SSL
```
host=prod.example.com port=5432 dbname=mydb user=app sslmode=require
```

## Security Best Practices

### 1. Use Read-Only Connections
```json
{
  "read_only": true  // Always use for production queries
}
```

### 2. Use Dedicated Users
- Create PostgreSQL users with minimal permissions
- Use `GRANT SELECT` only for read-only access

### 3. Use Secrets Management
- Store credentials in environment variables
- Use DuckDB secrets feature (future enhancement)

### 4. Limit Query Scope
- Always specify schema when possible
- Use WHERE clauses to limit data

### 5. Monitor Query Performance
- Set appropriate timeouts
- Use EXPLAIN for complex queries

## Error Handling

### Connection Errors
```json
{
  "type": "text",
  "content": "failed to attach: connection refused",
  "is_error": true
}
```

### Query Errors
```json
{
  "type": "text",
  "content": "query failed: column 'xyz' does not exist",
  "is_error": true
}
```

### Validation Errors
```json
{
  "type": "text",
  "content": "only SELECT queries are allowed. Use postgres_execute for other commands.",
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
- Default limit: 64 concurrent connections
- Configurable via `pg_connection_limit` setting

### Parallel Queries
- `postgres_query`, `postgres_list_tables`, `postgres_describe` are marked as parallel-safe
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
  SELECT * FROM users WHERE active = true
)
SELECT * FROM active_users WHERE created_at > '2024-01-01'
```

### Schema-Specific Queries
```json
{
  "connection": "prod_db",
  "query": "SELECT * FROM analytics.daily_stats LIMIT 10"
}
```

## Limitations

1. **Binary Data**: Large binary columns may not display well in JSON
2. **Array Types**: PostgreSQL arrays converted to strings by default
3. **Custom Types**: Some PostgreSQL custom types may need special handling
4. **Transactions**: Each query runs in its own transaction
5. **Streaming**: Large result sets loaded into memory (use LIMIT)

## Future Enhancements

- [ ] Support for DuckDB secrets management
- [ ] Export to Parquet/CSV directly
- [ ] Streaming large result sets
- [ ] Query result caching
- [ ] Connection pooling configuration
- [ ] Support for prepared statements
- [ ] PostgreSQL COPY protocol for bulk operations

## Testing

Run tests:
```bash
go test -v ./pkg/fantasy/tools/builtin -run TestPostgres
```

Integration tests (requires PostgreSQL):
```bash
# Uncomment integration test in postgres_test.go
# Configure connection details
go test -v ./pkg/fantasy/tools/builtin -run TestPostgres_Integration
```

## Dependencies

- `github.com/duckdb/duckdb-go/v2` - DuckDB Go driver
- DuckDB postgres extension (auto-installed)

## References

- [DuckDB Postgres Extension](https://duckdb.org/docs/stable/core_extensions/postgres)
- [PostgreSQL Connection Strings](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING)
- [DuckDB VSS Extension](https://duckdb.org/docs/stable/core_extensions/vss)
