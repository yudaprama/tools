package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kawai-network/veridium/pkg/fantasy"
)

// ToolRegistry manages fantasy.AgentTool instances
type ToolRegistry struct {
	tools   map[string]fantasy.AgentTool
	enabled map[string]bool
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools:   make(map[string]fantasy.AgentTool),
		enabled: make(map[string]bool),
	}
}

// Register registers a tool
func (r *ToolRegistry) Register(tool fantasy.AgentTool) error {
	info := tool.Info()
	if info.Name == "" {
		return fmt.Errorf("tool name is required")
	}

	r.tools[info.Name] = tool
	r.enabled[info.Name] = true // enabled by default
	return nil
}

// Get retrieves a tool by name
func (r *ToolRegistry) Get(name string) (fantasy.AgentTool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// GetAll returns all tools
func (r *ToolRegistry) GetAll() []fantasy.AgentTool {
	tools := make([]fantasy.AgentTool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// GetEnabled returns all enabled tools
func (r *ToolRegistry) GetEnabled() []fantasy.AgentTool {
	tools := make([]fantasy.AgentTool, 0, len(r.tools))
	for name, t := range r.tools {
		if r.enabled[name] {
			tools = append(tools, t)
		}
	}
	return tools
}

// GetByNames returns tools by names (or all enabled if empty)
func (r *ToolRegistry) GetByNames(names []string) []fantasy.AgentTool {
	if len(names) == 0 {
		return r.GetEnabled()
	}

	tools := make([]fantasy.AgentTool, 0, len(names))
	for _, name := range names {
		if tool, ok := r.tools[name]; ok && r.enabled[name] {
			tools = append(tools, tool)
		}
	}
	return tools
}

// ToAgentTools returns tools as []fantasy.AgentTool (same as GetByNames)
func (r *ToolRegistry) ToAgentTools(names []string) []fantasy.AgentTool {
	return r.GetByNames(names)
}

// Execute executes a tool by name
func (r *ToolRegistry) Execute(ctx context.Context, name string, args map[string]string) (string, error) {
	tool, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}

	if !r.enabled[name] {
		return "", fmt.Errorf("tool is disabled: %s", name)
	}

	// Convert args to JSON for ToolCall
	argsJSON, _ := json.Marshal(args)
	call := fantasy.ToolCall{
		ID:    name,
		Name:  name,
		Input: string(argsJSON),
	}

	resp, err := tool.Run(ctx, call)
	if err != nil {
		return "", err
	}

	if resp.IsError {
		return "", fmt.Errorf("%s", resp.Content)
	}

	return resp.Content, nil
}

// Enable enables a tool
func (r *ToolRegistry) Enable(name string) bool {
	if _, ok := r.tools[name]; ok {
		r.enabled[name] = true
		return true
	}
	return false
}

// Disable disables a tool
func (r *ToolRegistry) Disable(name string) bool {
	if _, ok := r.tools[name]; ok {
		r.enabled[name] = false
		return true
	}
	return false
}

// IsEnabled checks if a tool is enabled
func (r *ToolRegistry) IsEnabled(name string) bool {
	return r.enabled[name]
}

// FormatForPrompt formats tools as JSON for system prompt
func (r *ToolRegistry) FormatForPrompt(toolNames []string) (string, error) {
	tools := r.GetByNames(toolNames)
	if len(tools) == 0 {
		return "", nil
	}

	simplified := make([]map[string]interface{}, len(tools))
	for i, t := range tools {
		info := t.Info()
		simplified[i] = map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        info.Name,
				"description": info.Description,
				"parameters":  info.Parameters,
			},
		}
	}

	toolsJSON, err := json.MarshalIndent(simplified, "", "  ")
	if err != nil {
		return "", err
	}
	return string(toolsJSON), nil
}
