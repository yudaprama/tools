package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kawai-network/veridium/pkg/fantasy"
)

// ToolExecutor is a function that executes a tool with string arguments
type ToolExecutor func(ctx context.Context, args map[string]string) (string, error)

// SimpleTool is a simple implementation of fantasy.AgentTool
type SimpleTool struct {
	name            string
	description     string
	parameters      map[string]any
	required        []string
	parallel        bool
	executor        ToolExecutor
	providerOptions fantasy.ProviderOptions
}

// SimpleToolConfig configures a SimpleTool
type SimpleToolConfig struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON Schema properties
	Required    []string
	Parallel    bool
	Executor    ToolExecutor
}

// NewSimpleTool creates a new SimpleTool from config
func NewSimpleTool(cfg SimpleToolConfig) *SimpleTool {
	return &SimpleTool{
		name:        cfg.Name,
		description: cfg.Description,
		parameters:  cfg.Parameters,
		required:    cfg.Required,
		parallel:    cfg.Parallel,
		executor:    cfg.Executor,
	}
}

// Info returns tool metadata
func (t *SimpleTool) Info() fantasy.ToolInfo {
	return fantasy.ToolInfo{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.parameters,
		Required:    t.required,
		Parallel:    t.parallel,
	}
}

// Run executes the tool
func (t *SimpleTool) Run(ctx context.Context, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
	// Parse input JSON to map[string]string
	args := make(map[string]string)
	if call.Input != "" {
		var argsAny map[string]interface{}
		if err := json.Unmarshal([]byte(call.Input), &argsAny); err == nil {
			for k, v := range argsAny {
				switch val := v.(type) {
				case string:
					args[k] = val
				case float64:
					args[k] = formatNumber(val)
				case bool:
					if val {
						args[k] = "true"
					} else {
						args[k] = "false"
					}
				default:
					// For complex types, marshal back to JSON
					if jsonBytes, err := json.Marshal(v); err == nil {
						args[k] = string(jsonBytes)
					}
				}
			}
		}
	}

	// Execute the tool
	result, err := t.executor(ctx, args)
	if err != nil {
		return fantasy.NewTextErrorResponse(err.Error()), nil
	}

	return fantasy.NewTextResponse(result), nil
}

// ProviderOptions returns provider-specific options
func (t *SimpleTool) ProviderOptions() fantasy.ProviderOptions {
	return t.providerOptions
}

// SetProviderOptions sets provider-specific options
func (t *SimpleTool) SetProviderOptions(opts fantasy.ProviderOptions) {
	t.providerOptions = opts
}

// formatNumber converts float64 to string without scientific notation for integers
func formatNumber(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%v", f)
}
