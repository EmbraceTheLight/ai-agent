package tools

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"testing"
)

func TestCalculatorTime(t *testing.T) {
	tests := []struct {
		name    string
		req     *CalculateReq
		rawReq  json.RawMessage
		want    float64
		wantErr bool
	}{
		{
			name: "加法",
			req: &CalculateReq{
				Operator:     "add",
				LeftOperand:  3,
				RightOperand: 5,
			},
			want: 8,
		},
		{
			name: "减法",
			req: &CalculateReq{
				Operator:     "subtract",
				LeftOperand:  10,
				RightOperand: 4,
			},
			want: 6,
		},
		{
			name: "乘法",
			req: &CalculateReq{
				Operator:     "multiply",
				LeftOperand:  6,
				RightOperand: 7,
			},
			want: 42,
		},
		{
			name: "除法",
			req: &CalculateReq{
				Operator:     "divide",
				LeftOperand:  8,
				RightOperand: 2,
			},
			want: 4,
		},
		{
			name: "除以 0 返回错误",
			req: &CalculateReq{
				Operator:     "divide",
				LeftOperand:  8,
				RightOperand: 0,
			},
			wantErr: true,
		},
		{
			name: "不支持的运算符返回错误",
			req: &CalculateReq{
				Operator:     "mod",
				LeftOperand:  8,
				RightOperand: 3,
			},
			wantErr: true,
		},
		{
			name:    "非法 JSON 返回错误",
			rawReq:  json.RawMessage(`{`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawReq := tt.rawReq
			if rawReq == nil {
				var err error
				rawReq, err = json.Marshal(tt.req)
				if err != nil {
					t.Fatalf("marshal request failed: %v", err)
				}
			}

			got, err := Calculate(context.Background(), rawReq)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}

			gotFloat, err := strconv.ParseFloat(got, 64)
			if err != nil {
				t.Fatalf("parse result %q failed: %v", got, err)
			}

			if math.Abs(gotFloat-tt.want) > 1e-9 {
				t.Fatalf("expected %v, got %v", tt.want, gotFloat)
			}
		})
	}
}
