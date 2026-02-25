package builtin

import (
	"context"
	"fmt"

	"github.com/Knetic/govaluate"
	"github.com/kawai-network/veridium/pkg/fantasy"
	"github.com/getkawai/tools"
)

// CalculatorInput defines input for calculator tool
type CalculatorInput struct {
	Expression string `json:"expression" jsonschema:"description=The mathematical expression to evaluate (e.g. '2 + 2'&#44; 'sqrt(16)'&#44; 'sin(pi/2)')"`
}

// RegisterCalculator registers the calculator tool
func RegisterCalculator(registry *tools.ToolRegistry) error {
	tool := fantasy.NewParallelAgentTool("calculator",
		"Perform mathematical calculations. Supports: +, -, *, /, sqrt(), sin(), cos(), tan(), pow(), pi, e",
		func(ctx context.Context, input CalculatorInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if input.Expression == "" {
				return fantasy.NewTextErrorResponse("expression parameter is required"), nil
			}

			// Create evaluator with math functions
			expr, err := govaluate.NewEvaluableExpression(input.Expression)
			if err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("invalid expression: %v", err)), nil
			}

			// Evaluate expression
			result, err := expr.Evaluate(nil)
			if err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("evaluation failed: %v", err)), nil
			}

			return fantasy.NewTextResponse(fmt.Sprintf("%v", result)), nil
		},
	)

	return registry.Register(tool)
}
