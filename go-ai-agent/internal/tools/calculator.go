package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"go-ai-agent/internal/errno"
)

type calculatorTool struct {
	Func ToolFunc
}

func NewCalculatorTool() ITool {
	return &calculatorTool{
		Func: Calculate,
	}
}

// GetToolParameterJSONSchema 获取 Calculate Tool 的参数 json schema
func (calculator *calculatorTool) GetToolParameterJSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"operator": map[string]any{
				"type":        "string",
				"description": "基础运算操作, 包含加, 减, 乘, 除",
				"enum":        []string{"add", "subtract", "multiply", "divide"},
			},
			"left_operand": map[string]any{
				"type":        "number",
				"description": "左操作数",
			},
			"right_operand": map[string]any{
				"type":        "number",
				"description": "右操作数",
			},
		},
		"additionalProperties": false,
		"required":             []string{"operator", "left_operand", "right_operand"},
	}
}

func (calculator *calculatorTool) GetToolHandler() ToolFunc {
	return calculator.Func
}

type CalculateReq struct {
	Operator     string  `json:"operator"`
	LeftOperand  float64 `json:"left_operand"`
	RightOperand float64 `json:"right_operand"`
}

// Calculate 数学运算.
// 支持加减乘除四则基础运算.
func Calculate(ctx context.Context, jsonReq json.RawMessage) (string, error) {
	var req CalculateReq
	err := json.Unmarshal(jsonReq, &req)
	if err != nil {
		return "", errno.RequestInvalid.WithError(err)
	}
	switch req.Operator {
	case "add":
		return fmt.Sprintf("%f", req.LeftOperand+req.RightOperand), nil
	case "subtract":
		return fmt.Sprintf("%f", req.LeftOperand-req.RightOperand), nil
	case "multiply":
		return fmt.Sprintf("%f", req.LeftOperand*req.RightOperand), nil
	case "divide":
		if req.RightOperand == 0 {
			return "", errno.ErrDivideZeroError
		}
		return fmt.Sprintf("%f", req.LeftOperand/req.RightOperand), nil
	default:
		return "", errno.RequestInvalid.WithMsgf("不支持的运算符: %s", req.Operator)
	}
}
