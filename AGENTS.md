# tools

A tool library for AI agents, built on [CloudWeGo Eino](https://github.com/cloudwego/eino). Every tool implements `tool.InvokableTool` (`Info(ctx)` + `InvokableRun(ctx, argsJSON)`), so they drop straight into any Eino agent (`adk.ChatModelAgent` / `compose.ToolsNode`).

> Migrated from `github.com/getkawai/unillm` to Eino. `unillm` is no longer a dependency.

## Mental model: who picks the tool?

Tool selection is **two layers**. Don't conflate them.

| Layer | Who decides | Mechanism |
|---|---|---|
| **1. Curation (build-time)** | **You** — the agent author | Hand a *subset* of tools to an agent via the registry/catalog |
| **2. Selection (runtime)** | **The LLM** | Native function-calling over the `schema.ToolInfo` you gave it |

You almost never write tool-selection logic. Your job is (a) curate a minimal, topically-scoped set per agent, and (b) write good `name` + `description` so the model can disambiguate. The model does the rest.

## Quick reference

```bash
go build ./...          # build
go test ./...           # test (note: search/ hits live DuckDuckGo — network)
go vet ./...
go run ./example        # runnable curation + execution demo
```

## Layer 1 — curate which tools an agent sees

### Build + register

```go
ctx := context.Background()
ts, err := builtin.All(ctx, nil)        // []tool.InvokableTool (skip DB tools with nil)
// builtin.All(ctx, sqlDB)               // include lobe-image-describe / lobe-video-describe

r := tools.NewToolRegistry()
if err := r.RegisterAll(ts); err != nil { log.Fatal(err) }
```

### Pick a subset for one agent

```go
// a focused "math + docs" agent — keep the set small
agentTools := r.GetByNames([]string{"calculator", "pdf_extract_text", "pdf_search_text"})
// or: r.GetEnabled(), r.GetAll(), r.ToAgentTools(nil)
```

| Method | Returns | Use when |
|---|---|---|
| `GetByNames(names)` | `[]tool.InvokableTool` | you know exactly which tools this agent needs |
| `GetEnabled()` | `[]tool.InvokableTool` | respect `Enable/Disable` toggles |
| `GetAll()` | `[]tool.InvokableTool` | registration-order, ignore enable state |
| `ToAgentTools(names)` | `[]tool.InvokableTool` | alias for `GetByNames`; pass straight to an agent |
| `Execute(ctx, name, argsJSON)` | `(string, error)` | force a call bypassing the model |
| `Enable(name)` / `Disable(name)` | `bool` | toggle at runtime |

**Selection rule:** give an agent the *smallest* set that covers its job. Fewer tools → fewer hallucinated calls, cheaper prompts, better routing accuracy. This is exactly how `egent-public-apis` splits 100 tools across 12 category agents (one curated YAML per agent).

### Register a custom tool

Use `utils.InferTool` — the JSON schema is inferred from struct tags:

```go
import (
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/components/tool/utils"
)

type Expr struct {
    Expression string `json:"expression" jsonschema:"required,description=math to evaluate"`
}

calc, err := utils.InferTool("calculator",
    "Evaluate a math expression.",
    func(ctx context.Context, in *Expr) (string, error) {
        // ... compute ...
        return result, nil
    })
```

Struct tags the reflector honors (verified against `eino-contrib/jsonschema`):
- `json:"name,omitempty"` — field name / optionality
- `jsonschema:"required,description=…,enum=a,enum=b,default=x,minItems=1"` — schema keywords
- `jsonschema_description:"…"` — description-only shorthand

`SimpleTool` (in `simple_tool.go`) is the non-inferred alternative: build a tool from an explicit JSON-schema `map[string]any` + a `func(ctx, map[string]string) (string, error)`.

## Layer 2 — the model picks at runtime

You pass the curated slice; Eino ships each tool's `schema.ToolInfo` to the model and the model returns a `schema.ToolCall`. Pattern (matches `egent-public-apis/agent/agent.go:107`):

```go
agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:        "DocsAgent",
    Description: "Math + PDF operations",
    Instruction: systemPrompt,
    Model:       chatModel,
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{Tools: agentTools}, // <-- curated subset
    },
})
```

The model's pick is only as good as each tool's **name + description**. Make them self-disambiguating: one clear verb in the name, a description that says *when to use it vs. siblings*, required params marked `required`.

### Non-function-calling (text) models

If the model can't do native function-calling, `tools.ParseToolCalls(text)` parses `<tool_call>{…}</tool_call>`, inline `<name {json}>`, XML, and pure-JSON formats into `[]schema.ToolCall`:

```go
calls := tools.ParseToolCalls(llmOutput)       // []schema.ToolCall
for _, c := range calls {
    out, err := r.Execute(ctx, c.Function.Name, c.Function.Arguments)
}
```

## Layer 3 — force a call (no model)

```go
out, err := r.Execute(ctx, "calculator", `{"expression":"2+2*3"}`)
```

Useful for tests, scripted pipelines, and fan-out where you don't want an LLM in the loop.

## Tool inventory

Built by `builtin.All` (see `builtin/builtin.go`):

| Group | Tools |
|---|---|
| `lobe-web-browsing` | search, crawlSinglePage, crawlMultiPages |
| `lobe-local-system` | listLocalFiles, readLocalFile, searchLocalFiles, writeLocalFile, renameLocalFile, moveLocalFiles |
| `pdf` | search_replace, search_text, extract_text, merge, split, page_info, metadata_get, metadata_set, extract_images |
| `lobe-image-designer` | text2image |
| `lobe-code-interpreter` | python |
| `calculator` | calculator |
| `muninndb` | remember, remember_batch, recall, read, link, forget, status |
| `office-word` / `office-excel` / `office-powerpoint` | create, update, read each |
| `postgres` / `mysql` | attach, query, execute, list_tables, describe, detach each (skipped if the DuckDB extension is missing) |
| `lobe-image-describe` / `lobe-video-describe` | getImageDescription / getVideoTranscription (DB-only) |

Each group is also exposed as its own constructor: `builtin.NewCalculator(ctx)`, `builtin.NewPostgres(ctx)`, etc. — returns `([]tool.InvokableTool, error)`, so you can build a single group without the rest.

## Gotchas

1. **`builtin.All` needs `*sql.DB` for image/video describe** — pass `nil` to skip them.
2. **Postgres/MySQL tools may no-op in CI** — they load the DuckDB `postgres`/`mysql` extension; `All` skips the group gracefully on failure.
3. **`Execute` takes a JSON string, not a map** — `Execute(ctx, name, argsJSON string)`. The old `map[string]string` signature is gone.
4. **`search/` tests hit live DuckDuckGo** — they fail offline / behind a TLS-intercepting proxy; not a code issue.
5. **Struct tags drive the schema** — there is no manual params map anymore. Bad/missing tags = a broken tool schema the model can't call.
6. **`USAGE_GUIDE.md` is stale** — it still shows the old `unillm` API; treat `AGENTS.md` (this file) and `./example` as current.

## Runnable demo

`./example` builds the registry, curates two subsets, runs `calculator` end-to-end, and prints the rendered tool schemas:

```bash
go run ./example
```
