package builtin

import (
	"context"
	"fmt"

	"github.com/Knetic/govaluate"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// CalculatorInput defines input for calculator tool
type CalculatorInput struct {
	Expression string `json:"expression" jsonschema:"description=The mathematical expression to evaluate (e.g. '2 + 2'&#44; 'sqrt(16)'&#44; 'sin(pi/2)')"`
}

// NewCalculator creates the calculator tool.
func NewCalculator(_ context.Context) ([]tool.InvokableTool, error) {
	calcTool, err := utils.InferTool("calculator",
		"Perform mathematical calculations. Supports: +, -, *, /, sqrt(), sin(), cos(), tan(), pow(), pi, e",
		func(ctx context.Context, input *CalculatorInput) (string, error) {
			if input.Expression == "" {
				return "", fmt.Errorf("expression parameter is required")
			}

			expr, err := govaluate.NewEvaluableExpression(input.Expression)
			if err != nil {
				return "", fmt.Errorf("invalid expression: %v", err)
			}

			result, err := expr.Evaluate(nil)
			if err != nil {
				return "", fmt.Errorf("evaluation failed: %v", err)
			}

			return fmt.Sprintf("%v", result), nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to infer calculator tool: %w", err)
	}

	return []tool.InvokableTool{calcTool}, nil
}
