package tools

import (
	"context"
	"encoding/json"
	"fmt"

	jsonschema "github.com/eino-contrib/jsonschema"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ToolExecutor executes a tool with string-keyed arguments.
type ToolExecutor func(ctx context.Context, args map[string]string) (string, error)

// SimpleTool is a simple implementation of tool.InvokableTool backed by an
// explicit ToolExecutor and a JSON-Schema parameter map.
type SimpleTool struct {
	name        string
	description string
	paramsOneOf *schema.ParamsOneOf
	executor    ToolExecutor
}

// SimpleToolConfig configures a SimpleTool.
type SimpleToolConfig struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON Schema "properties"-like object
	Required    []string
	Executor    ToolExecutor
}

// NewSimpleTool creates a new SimpleTool from config.
func NewSimpleTool(cfg SimpleToolConfig) (*SimpleTool, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("tool name is required")
	}
	if cfg.Executor == nil {
		return nil, fmt.Errorf("executor is required")
	}

	t := &SimpleTool{
		name:        cfg.Name,
		description: cfg.Description,
		executor:    cfg.Executor,
	}

	if cfg.Parameters != nil {
		js := &jsonschema.Schema{
			Type:       "object",
			Properties: jsonschema.NewProperties(),
			Required:   cfg.Required,
		}
		for propName, propVal := range cfg.Parameters {
			raw, err := json.Marshal(propVal)
			if err != nil {
				return nil, fmt.Errorf("failed to encode parameter %q: %w", propName, err)
			}
			var ps jsonschema.Schema
			if err := json.Unmarshal(raw, &ps); err != nil {
				return nil, fmt.Errorf("failed to decode parameter %q: %w", propName, err)
			}
			js.Properties.Set(propName, &ps)
		}
		t.paramsOneOf = schema.NewParamsOneOfByJSONSchema(js)
	}

	return t, nil
}

// Info returns tool metadata.
func (t *SimpleTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        t.name,
		Desc:        t.description,
		ParamsOneOf: t.paramsOneOf,
	}, nil
}

// InvokableRun executes the tool with a JSON-encoded arguments string.
func (t *SimpleTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	args := make(map[string]string)
	if argumentsInJSON != "" {
		var argsAny map[string]interface{}
		if err := json.Unmarshal([]byte(argumentsInJSON), &argsAny); err == nil {
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
					if jsonBytes, err := json.Marshal(v); err == nil {
						args[k] = string(jsonBytes)
					}
				}
			}
		}
	}

	result, err := t.executor(ctx, args)
	if err != nil {
		return "", err
	}
	return result, nil
}

// formatNumber converts float64 to string without scientific notation for integers.
func formatNumber(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%v", f)
}
