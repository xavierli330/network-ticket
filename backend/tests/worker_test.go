package tests

import (
	"testing"
	"time"

	"github.com/xavierli/network-ticket/internal/client"
)

func TestBackoff(t *testing.T) {
	cfg := client.DefaultRetryConfig()

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{"attempt 0 returns base interval", 0, time.Second},
		{"attempt 1 returns 2s", 1, 2 * time.Second},
		{"attempt 2 returns 4s", 2, 4 * time.Second},
		{"attempt 3 returns 8s", 3, 8 * time.Second},
		{"attempt 4 returns 16s", 4, 16 * time.Second},
		{"attempt 10 capped at max interval", 10, 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.Backoff(tt.attempt, cfg)
			if got != tt.expected {
				t.Errorf("backoff(%d) = %v, want %v", tt.attempt, got, tt.expected)
			}
		})
	}
}
