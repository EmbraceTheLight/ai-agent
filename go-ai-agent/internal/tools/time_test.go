package tools

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestGetCurrentTime(t *testing.T) {
	const layout = "2006-01-02 15:04:05"

	tests := []struct {
		name         string
		req          *GetCurrentTimeReq
		locationName string
		wantErr      bool
	}{
		{
			name:         "指定 Asia/Shanghai 时区",
			req:          &GetCurrentTimeReq{TimeZone: "Asia/Shanghai"},
			locationName: "Asia/Shanghai",
		},
		{
			name:    "非法时区返回错误",
			req:     &GetCurrentTimeReq{TimeZone: "invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawReq, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("marshal request failed: %v", err)
			}

			got, err := GetCurrentTime(context.Background(), rawReq)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}

			location, err := time.LoadLocation(tt.locationName)
			if err != nil {
				t.Fatalf("load test location %q failed: %v", tt.locationName, err)
			}

			parsed, err := time.ParseInLocation(layout, got, location)
			if err != nil {
				t.Fatalf("expected time layout %q, got %q: %v", layout, got, err)
			}

			now := time.Now().In(location)
			if parsed.Before(now.Add(-3*time.Second)) || parsed.After(now.Add(3*time.Second)) {
				t.Fatalf("expected time close to now, got %q, now is %q", got, now.Format(layout))
			}
		})
	}
}
