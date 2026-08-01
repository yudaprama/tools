package builtin

import (
	"context"
	"fmt"

	"github.com/Knetic/govaluate"
	"github.com/getkawai/unillm"
	"github.com/yudaprama/tools"
)

// CalculatorInput defines input for calculator tool
type CalculatorInput struct {
	Expression string `json:"expression" jsonschema:"description=The mathematical expression to evaluate (e.g. '2 + 2'&#44; 'sqrt(16)'&#44; 'sin(pi/2)')"`
}

// RegisterCalculator registers the calculator tool
func RegisterCalculator(registry *tools.ToolRegistry) error {
	tool := unillm.NewParallelAgentTool("calculator",
		"Perform mathematical calculations. Supports: +, -, *, /, sqrt(), sin(), cos(), tan(), pow(), pi, e",
		func(ctx context.Context, input CalculatorInput, call unillm.ToolCall) (unillm.ToolResponse, error) {
			if input.Expression == "" {
				return unillm.NewTextErrorResponse("expression parameter is required"), nil
			}

			// Create evaluator with math functions
			expr, err := govaluate.NewEvaluableExpression(input.Expression)
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("invalid expression: %v", err)), nil
			}

			// Evaluate expression
			result, err := expr.Evaluate(nil)
			if err != nil {
				return unillm.NewTextErrorResponse(fmt.Sprintf("evaluation failed: %v", err)), nil
			}

			return unillm.NewTextResponse(fmt.Sprintf("%v", result)), nil
		},
	)

	return registry.Register(tool)
}
