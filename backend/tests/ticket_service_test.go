package tests

import (
	"testing"

	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/service"
)

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{"pending -> in_progress", model.TicketStatusPending, model.TicketStatusInProgress, true},
		{"pending -> failed", model.TicketStatusPending, model.TicketStatusFailed, true},
		{"pending -> completed", model.TicketStatusPending, model.TicketStatusCompleted, false},
		{"in_progress -> completed", model.TicketStatusInProgress, model.TicketStatusCompleted, true},
		{"in_progress -> rejected", model.TicketStatusInProgress, model.TicketStatusRejected, true},
		{"completed -> pending", model.TicketStatusCompleted, model.TicketStatusPending, false},
		{"failed -> pending", model.TicketStatusFailed, model.TicketStatusPending, true},
		{"failed -> cancelled", model.TicketStatusFailed, model.TicketStatusCancelled, true},
		{"cancelled -> pending", model.TicketStatusCancelled, model.TicketStatusPending, false},
		{"same status (pending -> pending)", model.TicketStatusPending, model.TicketStatusPending, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.CanTransition(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("CanTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}
