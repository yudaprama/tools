package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
)

// ToolRegistry manages eino tool.InvokableTool instances.
type ToolRegistry struct {
	tools   map[string]tool.InvokableTool
	order   []string
	enabled map[string]bool
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools:   make(map[string]tool.InvokableTool),
		enabled: make(map[string]bool),
	}
}

// Register registers a tool.
func (r *ToolRegistry) Register(t tool.InvokableTool) error {
	info, err := t.Info(context.Background())
	if err != nil {
		return fmt.Errorf("failed to read tool info: %w", err)
	}
	if info.Name == "" {
		return fmt.Errorf("tool name is required")
	}

	if _, exists := r.tools[info.Name]; !exists {
		r.order = append(r.order, info.Name)
	}
	r.tools[info.Name] = t
	r.enabled[info.Name] = true // enabled by default
	return nil
}

// RegisterAll registers a slice of tools.
func (r *ToolRegistry) RegisterAll(ts []tool.InvokableTool) error {
	for _, t := range ts {
		if err := r.Register(t); err != nil {
			return err
		}
	}
	return nil
}

// Get retrieves a tool by name.
func (r *ToolRegistry) Get(name string) (tool.InvokableTool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// Names returns the names of all registered tools in registration order.
func (r *ToolRegistry) Names() []string {
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}

// GetAll returns all tools in registration order.
func (r *ToolRegistry) GetAll() []tool.InvokableTool {
	tools := make([]tool.InvokableTool, 0, len(r.order))
	for _, name := range r.order {
		tools = append(tools, r.tools[name])
	}
	return tools
}

// GetEnabled returns all enabled tools in registration order.
func (r *ToolRegistry) GetEnabled() []tool.InvokableTool {
	tools := make([]tool.InvokableTool, 0, len(r.order))
	for _, name := range r.order {
		if r.enabled[name] {
			tools = append(tools, r.tools[name])
		}
	}
	return tools
}

// GetByNames returns tools by names (or all enabled if empty).
func (r *ToolRegistry) GetByNames(names []string) []tool.InvokableTool {
	if len(names) == 0 {
		return r.GetEnabled()
	}

	tools := make([]tool.InvokableTool, 0, len(names))
	for _, name := range names {
		if t, ok := r.tools[name]; ok && r.enabled[name] {
			tools = append(tools, t)
		}
	}
	return tools
}

// ToAgentTools returns tools as []tool.InvokableTool (same as GetByNames),
// ready to pass directly to an eino agent.
func (r *ToolRegistry) ToAgentTools(names []string) []tool.InvokableTool {
	return r.GetByNames(names)
}

// Execute executes a tool by name with a JSON-encoded arguments string.
func (r *ToolRegistry) Execute(ctx context.Context, name string, argsJSON string) (string, error) {
	t, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}

	if !r.enabled[name] {
		return "", fmt.Errorf("tool is disabled: %s", name)
	}

	return t.InvokableRun(ctx, argsJSON)
}

// Enable enables a tool.
func (r *ToolRegistry) Enable(name string) bool {
	if _, ok := r.tools[name]; ok {
		r.enabled[name] = true
		return true
	}
	return false
}

// Disable disables a tool.
func (r *ToolRegistry) Disable(name string) bool {
	if _, ok := r.tools[name]; ok {
		r.enabled[name] = false
		return true
	}
	return false
}

// IsEnabled checks if a tool is enabled.
func (r *ToolRegistry) IsEnabled(name string) bool {
	return r.enabled[name]
}

// FormatForPrompt formats tools as JSON (OpenAI function-calling style) for
// injection into a system prompt. Returns "" when no tools are selected.
func (r *ToolRegistry) FormatForPrompt(toolNames []string) (string, error) {
	tools := r.GetByNames(toolNames)
	if len(tools) == 0 {
		return "", nil
	}

	simplified := make([]map[string]interface{}, 0, len(tools))
	for _, t := range tools {
		info, err := t.Info(context.Background())
		if err != nil {
			return "", err
		}

		var params any
		if info.ParamsOneOf != nil {
			js, err := info.ParamsOneOf.ToJSONSchema()
			if err != nil {
				return "", fmt.Errorf("failed to render schema for %s: %w", info.Name, err)
			}
			params = js
		} else {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}

		simplified = append(simplified, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        info.Name,
				"description": info.Desc,
				"parameters":  params,
			},
		})
	}

	toolsJSON, err := json.MarshalIndent(simplified, "", "  ")
	if err != nil {
		return "", err
	}
	return string(toolsJSON), nil
}
