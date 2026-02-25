# Fantasy Tools Usage Guide

## Arsitektur

```
Frontend (React/TypeScript)
    ↓ (Wails RPC)
Backend (Go)
    ↓
AgentChatService
    ↓
Fantasy Agent (pkg/fantasy)
    ↓
Tool Registry → PostgreSQL/MySQL Tools
```

## Flow Penggunaan Tools

### 1. **Frontend → Backend**
User mengirim pesan melalui chat interface:

```typescript
// frontend/src/store/chat/slices/aiChat/actions/generateAIChat.ts
await ChatRealStream(new ChatRequest({
  session_id: activeId,
  user_id: userId,
  message: "Query database users",
  tools: ["postgres_query", "mysql_query"], // Tools yang diaktifkan
}));
```

### 2. **Backend Agent Service**
Agent service menerima request dan setup Fantasy Agent:

```go
// internal/services/agent_real_stream.go
agentTools := s.toolRegistry.ToAgentTools(session.ToolNames)

agent := fantasy.NewAgent(model,
    fantasy.WithTools(agentTools...),
    fantasy.WithSystemPrompt(systemPrompt),
    fantasy.WithStopConditions(fantasy.StepCountIs(10)),
)

result, err := agent.Stream(ctx, fantasy.AgentStreamCall{
    Prompt:   userPrompt,
    Messages: historyMessages,
    OnTextDelta: func(id, text string) error {
        // Stream text ke frontend
    },
    OnToolCall: func(call fantasy.ToolCallContent) error {
        // Execute tool dan stream result ke frontend
    },
})
```

### 3. **Tool Schema Generation**
Fantasy framework otomatis generate JSON schema dari struct input:

```go
// pkg/fantasy/tools/builtin/postgres.go
type PostgresAttachInput struct {
    Name     string `json:"name" jsonschema:"required,description=Connection name"`
    Host     string `json:"host" jsonschema:"required,description=PostgreSQL host"`
    Database string `json:"database" jsonschema:"required,description=Database name"`
    User     string `json:"user" jsonschema:"required,description=Username"`
    Password string `json:"password,omitempty" jsonschema:"description=Password"`
}

// Tool registration dengan auto schema generation
attachTool := fantasy.NewAgentTool("postgres_attach",
    "Connect to a PostgreSQL database",
    func(ctx context.Context, input PostgresAttachInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
        return service.attach(ctx, input)
    },
)
```

**Generated JSON Schema yang dikirim ke LLM:**
```json
{
  "name": "postgres_attach",
  "description": "Connect to a PostgreSQL database",
  "parameters": {
    "type": "object",
    "properties": {
      "name": {
        "type": "string",
        "description": "Connection name"
      },
      "host": {
        "type": "string",
        "description": "PostgreSQL host"
      },
      "database": {
        "type": "string",
        "description": "Database name"
      },
      "user": {
        "type": "string",
        "description": "Username"
      },
      "password": {
        "type": "string",
        "description": "Password"
      }
    },
    "required": ["name", "host", "database", "user"]
  }
}
```

### 4. **LLM Generates Tool Call**
LLM menerima tool schema dan memutuskan untuk memanggil tool berdasarkan user message:

**User berkata:**
> "Connect to my production database at db.example.com, database name is 'myapp', use user 'readonly' with password 'secret123'"

**LLM menganalisis dan generate tool call:**
```json
{
  "tool_name": "postgres_attach",
  "arguments": {
    "name": "prod_db",
    "host": "db.example.com",
    "database": "myapp",
    "user": "readonly",
    "password": "secret123"
  }
}
```

**Sumber Input:**
- ✅ **Dari user message** - LLM extract informasi dari percakapan
- ✅ **Dari context** - LLM bisa ingat informasi dari pesan sebelumnya
- ✅ **Dari reasoning** - LLM bisa generate nilai default (seperti `name: "prod_db"`)

### 5. **Tool Execution**
Fantasy Agent execute tool dengan arguments dari LLM:

```go
// LLM generates tool call dengan arguments
toolCall := fantasy.ToolCallContent{
    ToolName: "postgres_attach",
    Input: `{
        "name": "prod_db",
        "host": "db.example.com",
        "database": "myapp",
        "user": "readonly",
        "password": "secret123"
    }`,
}

// Fantasy Agent executes tool
tool, ok := registry.Get("postgres_attach")
response, err := tool.Execute(ctx, toolCall)

// Result streamed back to frontend
{
  "status": "connected",
  "connection": "prod_db",
  "host": "db.example.com",
  "database": "myapp",
  "read_only": true
}
```

### 6. **Frontend Receives Stream**
Frontend menerima streaming events:

```typescript
// Events handled in App.tsx
internal_handleStreamEvent(event: ChatStreamEvent) {
  switch (event.type) {
    case 'text_delta':
      // Update assistant message dengan text baru
      break;
    case 'tool_call':
      // Show tool execution in UI
      break;
    case 'tool_result':
      // Show tool result in UI
      break;
  }
}
```

## Contoh Penggunaan PostgreSQL/MySQL Tools

### Dari Mana Input Tool Arguments Berasal?

**PENTING:** Input arguments untuk tools **TIDAK** datang dari frontend atau hardcoded. Input datang dari **LLM (Language Model)** yang menganalisis percakapan dan memutuskan parameter apa yang dibutuhkan.

#### Skenario 1: User Memberikan Semua Info

**User berkata:**
> "Connect to database at localhost port 5432, database name 'production', user 'admin', password 'secret123'"

**LLM extract info dan generate:**
```json
{
  "tool": "postgres_attach",
  "args": {
    "name": "prod",           // LLM generate nama connection
    "host": "localhost",      // Dari user message
    "port": 5432,             // Dari user message
    "database": "production", // Dari user message
    "user": "admin",          // Dari user message
    "password": "secret123"   // Dari user message
  }
}
```

#### Skenario 2: User Memberikan Info Partial

**User berkata:**
> "Connect to my local database"

**LLM bisa:**
1. **Bertanya ke user** untuk info yang kurang
2. **Gunakan default values** (localhost, port 5432, dll)
3. **Ingat dari context** jika user pernah mention sebelumnya

**LLM response:**
> "I need more information to connect. What's the database name, username, and password?"

#### Skenario 3: User Sudah Pernah Connect Sebelumnya

**Previous conversation:**
> User: "My database is at db.example.com, name is 'myapp', user 'readonly', password 'pass123'"

**Current message:**
> User: "Connect to that database again"

**LLM ingat dari history dan generate:**
```json
{
  "tool": "postgres_attach",
  "args": {
    "name": "myapp_conn",
    "host": "db.example.com",
    "database": "myapp",
    "user": "readonly",
    "password": "pass123"
  }
}
```

#### Skenario 4: LLM Reasoning

**User berkata:**
> "Show me users from production database"

**LLM reasoning:**
1. User minta query users
2. Perlu connect dulu ke database
3. User bilang "production" tapi tidak ada detail
4. Harus tanya user dulu

**LLM response:**
> "I need connection details for your production database. Please provide:
> - Host address
> - Database name
> - Username
> - Password"

### Scenario 1: Connect dan Query Database

**User:** "Connect to my production database and show me all users"

**LLM Decision Flow:**
1. Call `postgres_attach` untuk connect
2. Call `postgres_list_tables` untuk lihat tables
3. Call `postgres_query` untuk query users
4. Format hasil dan tampilkan ke user

**Tool Calls:**
```json
// Step 1: Attach
{
  "tool": "postgres_attach",
  "args": {
    "name": "prod",
    "host": "db.example.com",
    "database": "production",
    "user": "readonly",
    "read_only": true
  }
}

// Step 2: Query
{
  "tool": "postgres_query",
  "args": {
    "connection": "prod",
    "query": "SELECT id, name, email FROM users LIMIT 10"
  }
}

// Step 3: Detach
{
  "tool": "postgres_detach",
  "args": {
    "connection": "prod"
  }
}
```

### Scenario 2: MySQL Socket Connection

**User:** "Connect to local MySQL via socket and describe users table"

```json
// Step 1: Attach via socket
{
  "tool": "mysql_attach",
  "args": {
    "name": "local",
    "socket": "/tmp/mysql.sock",
    "database": "myapp",
    "user": "root"
  }
}

// Step 2: Describe table
{
  "tool": "mysql_describe",
  "args": {
    "connection": "local",
    "table": "users"
  }
}
```

## Tool Registration

### **Kapan Tools Dibuat?**

Tools dan services dibuat **SAAT APLIKASI STARTUP**, bukan saat user enable tool!

```go
// Application Startup Flow:
main()
  ↓
Initialize Services
  ↓
builtin.RegisterAll(toolRegistry)  // ← Tools created HERE
  ↓
  RegisterPostgres(registry)
    ↓
    service := NewPostgresService()  // ← Service created at startup
    ↓
    Register 6 tools (attach, query, execute, etc.)
  ↓
  RegisterMySQL(registry)
    ↓
    service := NewMySQLService()     // ← Service created at startup
    ↓
    Register 6 tools (attach, query, execute, etc.)
  ↓
Application Ready (all tools registered)
```

### **Lifecycle:**

1. **Startup (Once):**
   - `NewPostgresService()` creates DuckDB connection
   - `NewMySQLService()` creates DuckDB connection
   - All 12 tools registered in ToolRegistry
   - Services stay alive for app lifetime

2. **User Enable Tool (Many times):**
   - User clicks "Enable postgres_query" in UI
   - Frontend saves to agent config
   - Backend filters tools from registry
   - **NO new service created** - reuses existing service

3. **Tool Execution (Many times):**
   - LLM calls `postgres_attach`
   - Uses existing PostgresService instance
   - Service manages connections in memory
   - Multiple connections can coexist

### **Why Create at Startup?**

✅ **Performance** - No overhead saat user enable tool
✅ **Resource Management** - Single DuckDB instance per service
✅ **Connection Pooling** - Service manages multiple DB connections
✅ **Stateful** - Service maintains connection state across tool calls

```go
// pkg/fantasy/tools/builtin/builtin.go
func RegisterAll(registry *tools.ToolRegistry) error {
    // Called ONCE at startup
    if err := RegisterPostgres(registry); err != nil {
        return err
    }
    if err := RegisterMySQL(registry); err != nil {
        return err
    }
    // ... register other tools
    return nil
}
```

### **Service Lifecycle:**

```go
// PostgresService lifecycle
type PostgresService struct {
    db          *sql.DB              // Created at startup
    mu          sync.RWMutex         // Thread-safe
    connections map[string]bool      // Tracks active connections
}

// Created ONCE at startup
service := NewPostgresService()

// Used MANY times during runtime
service.attach(ctx, input)   // User connects to DB1
service.attach(ctx, input)   // User connects to DB2
service.query(ctx, input)    // Query DB1
service.query(ctx, input)    // Query DB2
service.detach(ctx, input)   // Disconnect DB1
```

## Tool Enable/Disable dari UI

### **TIDAK Semua Tools Otomatis Aktif!**

Tools harus **diaktifkan secara manual** oleh user dari UI. Berikut cara kerjanya:

### 1. **UI Plugin/Tool Settings**

User bisa enable/disable tools dari:
- **Chat Input → Tools Button** - Quick toggle tools untuk session saat ini
- **Plugin Store** - Install dan enable/disable plugins
- **Agent Settings** - Configure tools untuk specific agent

```typescript
// frontend/src/store/agent/slices/chat/action.ts
togglePlugin: async (id, open) => {
  const config = produce(originConfig, (draft) => {
    draft.plugins = produce(draft.plugins || [], (plugins) => {
      if (shouldOpen) {
        plugins.push(id);  // Enable tool
      } else {
        plugins.splice(index, 1);  // Disable tool
      }
    });
  });
  await updateAgentConfig(config);  // Save to database
}
```

### 2. **Agent Config Storage**

Enabled tools disimpan di **agent config** (database):

```typescript
// Agent Config Structure
{
  "sessionId": "abc123",
  "plugins": [
    "postgres_query",    // ✅ Enabled
    "mysql_query",       // ✅ Enabled
    "web_search"         // ✅ Enabled
  ],
  // postgres_attach, mysql_attach NOT in list = ❌ Disabled
}
```

### 3. **Backend Receives Enabled Tools**

Ketika user mengirim message, frontend kirim list enabled tools:

```typescript
// frontend/src/store/chat/slices/aiChat/actions/generateAIChat.ts
const enabledPlugins = agentSelectors.currentAgentPlugins(agentState);

await ChatRealStream(new ChatRequest({
  session_id: activeId,
  message: "Query database",
  tools: enabledPlugins,  // ["postgres_query", "mysql_query"]
}));
```

### 4. **Backend Filters Tools**

Backend hanya load tools yang enabled:

```go
// internal/services/agent_real_stream.go
agentTools := s.toolRegistry.ToAgentTools(session.ToolNames)
// Only returns tools that are in session.ToolNames

agent := fantasy.NewAgent(model,
    fantasy.WithTools(agentTools...),  // Only enabled tools
    // ...
)
```

### 5. **Tool Registry GetByNames**

Tool registry filter berdasarkan names:

```go
// pkg/fantasy/tools/registry.go
func (r *ToolRegistry) GetByNames(names []string) []fantasy.AgentTool {
    if len(names) == 0 {
        return r.GetEnabled()  // All enabled tools
    }
    
    tools := make([]fantasy.AgentTool, 0, len(names))
    for _, name := range names {
        if tool, ok := r.tools[name]; ok && r.enabled[name] {
            tools = append(tools, tool)
        }
    }
    return tools
}
```

### **Flow Lengkap:**

```
1. User clicks "Enable postgres_query" in UI
   ↓
2. Frontend updates agent config: plugins: ["postgres_query"]
   ↓
3. Config saved to database
   ↓
4. User sends message
   ↓
5. Frontend sends: tools: ["postgres_query"]
   ↓
6. Backend loads only postgres_query tool
   ↓
7. LLM only sees postgres_query in available tools
   ↓
8. LLM can only call postgres_query (not other tools)
```

### **Default Behavior:**

- ❌ **Tidak ada tools yang aktif by default**
- ✅ User harus **manually enable** tools yang dibutuhkan
- ✅ Setiap session/agent bisa punya **different enabled tools**
- ✅ Tools bisa di-toggle **on/off kapan saja**

### **Contoh Skenario:**

**Session A (Data Analysis):**
- ✅ postgres_query
- ✅ mysql_query
- ✅ postgres_describe
- ❌ web_search (disabled)

**Session B (Research):**
- ✅ web_search
- ✅ web_fetch
- ❌ postgres_query (disabled)
- ❌ mysql_query (disabled)

## Security Features

### 1. **Read-Only by Default**
```go
// Default ReadOnly = true untuk safety
readOnly := true
if input.ReadOnly != nil {
    readOnly = *input.ReadOnly
}
```

### 2. **SQL Injection Protection**
```go
// Validate all identifiers
if err := validateSQLIdent(input.Name); err != nil {
    return fantasy.NewTextErrorResponse(err.Error()), nil
}
```

### 3. **Dangerous Operation Confirmation**
```go
// Require explicit confirmation for DROP/DELETE/TRUNCATE
if isDangerous && !input.Confirm {
    return fantasy.NewTextErrorResponse(
        "dangerous operation detected. Set confirm=true to proceed."
    ), nil
}
```

### 4. **Query Limits**
```go
// Default limit 100 rows
limit := input.Limit
if limit == 0 {
    limit = 100
}
```

### 5. **Timeouts**
```go
// Connection timeout: 10s
execCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
defer cancel()

// Query timeout: 30s
execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
```

## Concurrency Safety

Tools menggunakan mutex untuk thread-safe connection management:

```go
type PostgresService struct {
    db          *sql.DB
    mu          sync.RWMutex
    connections map[string]bool
}

func (s *PostgresService) hasConnection(name string) bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.connections[name]
}
```

## Error Handling

Tools mengembalikan user-friendly error messages:

```go
// Invalid identifier
return fantasy.NewTextErrorResponse(
    "invalid identifier 'my-db': must start with letter/underscore"
), nil

// Connection not found
return fantasy.NewTextErrorResponse(
    "connection 'prod' not found. Use postgres_attach first."
), nil

// Query failed
return fantasy.NewTextErrorResponse(
    fmt.Sprintf("query failed: %v", err)
), nil
```

## Best Practices

1. **Always detach connections** setelah selesai
2. **Use read_only=true** untuk production databases
3. **Validate user input** sebelum pass ke tools
4. **Set appropriate limits** untuk query results
5. **Handle timeouts** gracefully
6. **Log tool executions** untuk debugging
7. **Test with race detector** untuk concurrency issues

## Testing

```bash
# Run all tests
go test ./pkg/fantasy/tools/builtin -v

# Run with race detector
go test -race ./pkg/fantasy/tools/builtin -v

# Run specific test
go test ./pkg/fantasy/tools/builtin -run TestPostgresAttach -v
```

## Monitoring

Tool executions di-log dengan structured logging:

```go
xlog.Info("Attaching to PostgreSQL", 
    "name", input.Name, 
    "host", input.Host, 
    "database", input.Database,
)

xlog.Info("Executing PostgreSQL query", 
    "connection", input.Connection, 
    "query", query,
)
```

## Performance Impact: Unused Tools

### **Pertanyaan: Jika tool tidak pernah digunakan, apakah ada performance issue?**

**Jawaban Singkat:** Ada **minimal overhead** tapi **tidak signifikan** untuk production use.

### **Resource Usage Analysis:**

#### 1. **Startup Cost (One-time)**

```go
// Per service at startup:
NewPostgresService() {
    db = sql.Open("duckdb", ":memory:")  // ~1-5ms, minimal memory
    db.Exec("INSTALL postgres")          // ~10-50ms, download extension
    db.Exec("LOAD postgres")             // ~5-10ms, load into memory
    connections = make(map[string]bool)  // ~1KB
}

NewMySQLService() {
    db = sql.Open("duckdb", ":memory:")  // ~1-5ms
    db.Exec("INSTALL mysql")             // ~10-50ms
    db.Exec("LOAD mysql")                // ~5-10ms
    connections = make(map[string]bool)  // ~1KB
}
```

**Total Startup Overhead:**
- ⏱️ **Time:** ~50-150ms (one-time, during app startup)
- 💾 **Memory:** ~5-10MB per service (DuckDB + extensions)
- 🔌 **Connections:** 0 (no actual DB connections until used)

#### 2. **Runtime Cost (If Never Used)**

```go
// Memory footprint if NEVER used:
PostgresService {
    db:          *sql.DB           // ~2-3MB (idle DuckDB)
    mu:          sync.RWMutex      // ~24 bytes
    connections: map[string]bool   // ~1KB (empty map)
}

MySQLService {
    db:          *sql.DB           // ~2-3MB (idle DuckDB)
    mu:          sync.RWMutex      // ~24 bytes
    connections: map[string]bool   // ~1KB (empty map)
}
```

**Total Runtime Overhead (Unused):**
- 💾 **Memory:** ~10-15MB total (both services idle)
- 🔄 **CPU:** 0% (no operations)
- 🌐 **Network:** 0 (no connections)
- 📊 **I/O:** 0 (no queries)

#### 3. **Comparison with Other Approaches:**

| Approach | Startup Time | Memory (Unused) | Memory (Used) | Complexity |
|----------|--------------|-----------------|---------------|------------|
| **Current: Create at Startup** | 50-150ms | 10-15MB | 10-15MB + connections | Low |
| Lazy Init (on first use) | 0ms | 0MB | 10-15MB + connections | Medium |
| Per-session services | 0ms | 0MB | 10-15MB × sessions | High |

### **Performance Impact Assessment:**

#### ✅ **Negligible Impact:**
1. **Startup:** 50-150ms adalah **<1%** dari total app startup time
2. **Memory:** 10-15MB adalah **<0.5%** dari typical app memory (2-4GB)
3. **Idle Cost:** Services consume **0 CPU** when not used
4. **No Connections:** No actual database connections until `attach()` called

#### ⚠️ **Potential Issues (Edge Cases):**

1. **Memory-Constrained Environments:**
   - Embedded devices dengan <512MB RAM
   - **Solution:** Conditional compilation, disable unused tools

2. **Many Unused Tools:**
   - 50+ tools × 10MB each = 500MB overhead
   - **Solution:** Lazy initialization for rarely-used tools

3. **Extension Download:**
   - First `INSTALL` downloads extension from network
   - **Solution:** Pre-install extensions in Docker image

### **Optimization Strategies:**

#### **Option 1: Lazy Initialization (If Needed)**

```go
type PostgresService struct {
    db          *sql.DB
    mu          sync.RWMutex
    connections map[string]bool
    initialized bool
}

func (s *PostgresService) ensureInitialized() error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if s.initialized {
        return nil
    }
    
    // Initialize only when first used
    db, err := sql.Open("duckdb", ":memory:")
    if err != nil {
        return err
    }
    
    s.db = db
    s.initialized = true
    return nil
}
```

**Pros:**
- ✅ Zero overhead if never used
- ✅ Lower startup time

**Cons:**
- ❌ First use has latency
- ❌ More complex code
- ❌ Thread-safety complexity

#### **Option 2: Conditional Registration**

```go
// Only register tools if enabled in config
func RegisterAll(registry *tools.ToolRegistry, config Config) error {
    if config.EnablePostgres {
        RegisterPostgres(registry)
    }
    if config.EnableMySQL {
        RegisterMySQL(registry)
    }
}
```

**Pros:**
- ✅ No overhead for disabled tools
- ✅ User control

**Cons:**
- ❌ Requires configuration
- ❌ Less flexible (need restart to enable)

#### **Option 3: Extension Pre-installation**

```dockerfile
# Dockerfile
RUN duckdb -c "INSTALL postgres; INSTALL mysql;"
```

**Pros:**
- ✅ Faster startup (no download)
- ✅ Offline support

**Cons:**
- ❌ Larger Docker image
- ❌ Version management

### **Recommendation:**

**Current approach is OPTIMAL for most use cases:**

1. ✅ **Simple** - Easy to understand and maintain
2. ✅ **Fast** - No latency on first use
3. ✅ **Predictable** - Consistent performance
4. ✅ **Minimal Overhead** - 10-15MB is negligible in modern systems

**Only optimize if:**
- Running on memory-constrained devices (<512MB RAM)
- Have 50+ unused tools
- Startup time is critical (<100ms requirement)

### **Monitoring:**

```go
// Add metrics to track usage
type PostgresService struct {
    db          *sql.DB
    mu          sync.RWMutex
    connections map[string]bool
    
    // Metrics
    attachCount   int64
    queryCount    int64
    lastUsedAt    time.Time
}

// Log unused services
func (s *PostgresService) LogMetrics() {
    if s.attachCount == 0 {
        log.Info("PostgresService never used", 
            "memory", "~5MB", 
            "recommendation", "consider lazy init")
    }
}
```

### **Conclusion:**

**Performance impact of unused tools is MINIMAL:**
- 💾 Memory: ~10-15MB (0.5% of typical app)
- ⏱️ Startup: ~50-150ms (1% of app startup)
- 🔄 Runtime: 0% CPU when idle

**No optimization needed unless:**
- Memory < 512MB
- Startup time < 100ms requirement
- 50+ unused tools

Current design prioritizes **simplicity** and **predictability** over minimal resource savings.

## Future Enhancements

- [ ] Connection pooling untuk reuse connections
- [ ] Query result caching
- [ ] Transaction support
- [ ] Prepared statement support
- [ ] Query explain/analyze
- [ ] Schema migration tools
- [ ] Backup/restore tools
- [ ] Performance monitoring
