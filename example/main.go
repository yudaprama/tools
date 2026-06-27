// Example: how an AI agent picks a tool (or some tools) from the registry.
//
// Run:  go run ./example
//
// This demonstrates the two-layer model documented in AGENTS.md:
//   1. Curation  — pick a subset of tools for an agent via GetByNames/GetEnabled.
//   2. Execution — run a tool directly (no LLM) via Execute, and show how the
//                  same curated slice would feed an adk.ChatModelAgent.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/eino/components/tool"
	"github.com/yudaprama/tools"
	"github.com/yudaprama/tools/builtin"
)

func main() {
	ctx := context.Background()

	// --- Layer 1a: build everything + register ---------------------------------
	allTools, err := builtin.All(ctx, nil) // nil DB -> skips image/video describe
	if err != nil {
		log.Fatalf("build tools: %v", err)
	}

	r := tools.NewToolRegistry()
	if err := r.RegisterAll(allTools); err != nil {
		log.Fatalf("register: %v", err)
	}
	fmt.Printf("built %d tools\n\n", len(r.Names()))

	// --- Layer 1b: curate a focused subset for ONE agent -----------------------
	// Keep the set small and topically scoped — better routing, cheaper prompts.
	mathAgentTools := r.GetByNames([]string{"calculator"})
	docsAgentTools := r.GetByNames([]string{"calculator", "pdf_search_text", "pdf_extract_text"})

	fmt.Printf("math agent gets %d tool(s):  %v\n", len(mathAgentTools), names(ctx, mathAgentTools))
	fmt.Printf("docs agent gets %d tool(s):  %v\n\n", len(docsAgentTools), names(ctx, docsAgentTools))

	// --- Layer 2 (model-side) is automatic -------------------------------------
	// The curated slice is handed straight to an Eino agent:
	//
	//   agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
	//       Name: "DocsAgent", Instruction: sys,
	//       Model: chatModel,
	//       ToolsConfig: adk.ToolsConfig{
	//           ToolsNodeConfig: compose.ToolsNodeConfig{Tools: docsAgentTools},
	//       },
	//   })
	//
	// The model then picks among docsAgentTools via native function-calling;
	// you never write selection logic. Its choice depends only on each tool's
	// name + description (schema.ToolInfo).

	// --- Layer 3: force a call without an LLM (scripted / tests) ---------------
	out, err := r.Execute(ctx, "calculator", `{"expression":"2 + 2 * 3"}`)
	if err != nil {
		log.Fatalf("execute calculator: %v", err)
	}
	fmt.Printf("calculator(\"2 + 2 * 3\") = %s\n\n", out)

	// --- Bonus: render the curated subset as OpenAI-style function schemas ------
	prompt, err := r.FormatForPrompt([]string{"calculator"})
	if err != nil {
		log.Fatalf("format: %v", err)
	}
	prettyPrint("calculator schema (what the model actually sees)", prompt)

	// Toggle example: disable a tool, show GetEnabled drops it.
	r.Disable("calculator")
	fmt.Printf("\nafter Disable(\"calculator\"): %d enabled\n", len(r.GetEnabled()))
	r.Enable("calculator")
	fmt.Printf("after Enable(\"calculator\"):  %d enabled\n", len(r.GetEnabled()))

	_ = os.Stdout
}

// names returns the ToolInfo names for a slice, for readable logging.
func names(ctx context.Context, ts []tool.InvokableTool) []string {
	out := make([]string, 0, len(ts))
	for _, t := range ts {
		info, err := t.Info(ctx)
		if err != nil {
			out = append(out, "<error>")
			continue
		}
		out = append(out, info.Name)
	}
	return out
}

func prettyPrint(title, raw string) {
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err == nil {
		b, _ := json.MarshalIndent(v, "", "  ")
		raw = string(b)
	}
	fmt.Printf("--- %s ---\n%s\n", title, raw)
}
