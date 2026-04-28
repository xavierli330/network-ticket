package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/alert/parser"
	"github.com/xavierli/network-ticket/internal/pkg"
	"github.com/xavierli/network-ticket/internal/repository"
)

// IngestResult holds the result of alert ingestion.
type IngestResult struct {
	TicketID int64
	TicketNo string
	Status   string
	IsNew    bool
}

// AlertService handles alert ingestion, deduplication, and ticket creation.
type AlertService struct {
	alertSourceRepo *repository.AlertSourceRepo
	ticketService   *TicketService
	logger          *zap.Logger
}

// NewAlertService creates a new AlertService.
func NewAlertService(alertSourceRepo *repository.AlertSourceRepo, ticketService *TicketService, logger *zap.Logger) *AlertService {
	return &AlertService{
		alertSourceRepo: alertSourceRepo,
		ticketService:   ticketService,
		logger:          logger,
	}
}

// Ingest processes an incoming alert: parse, dedup, and create or append to a ticket.
func (s *AlertService) Ingest(ctx context.Context, sourceID int64, raw json.RawMessage) (*IngestResult, error) {
	// 1. Get alert source by ID.
	source, err := s.alertSourceRepo.GetByID(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("get alert source: %w", err)
	}

	// 2. Get parser from registry, fallback to "generic".
	sourceType := source.Type
	p, ok := parser.Get(sourceType)
	if !ok {
		p, _ = parser.Get("generic")
		sourceType = "generic"
	}

	// 3. Parse alert.
	parsed, err := p.Parse(ctx, raw)
	if err != nil {
		return nil, fmt.Errorf("parse alert: %w", err)
	}

	// 4. Compute fingerprint if dedup_fields is configured.
	var fingerprint *string
	if len(source.DedupFields) > 0 {
		var dedupFields []string
		if err := json.Unmarshal(source.DedupFields, &dedupFields); err != nil {
			return nil, fmt.Errorf("unmarshal dedup_fields: %w", err)
		}
		if len(dedupFields) > 0 {
			fp, err := pkg.ComputeFingerprint(raw, dedupFields)
			if err != nil {
				s.logger.Warn("failed to compute fingerprint, skipping dedup", zap.Error(err))
			} else {
				fingerprint = &fp
			}
		}
	}

	// 5. Dedup check: if fingerprint exists, try to find existing ticket.
	if fingerprint != nil {
		existing, err := s.ticketService.ticketRepo.GetByFingerprint(ctx, *fingerprint)
		if err == nil && existing != nil {
			// Check if within dedup window.
			windowSec := source.DedupWindow
			if windowSec <= 0 {
				windowSec = 300 // default 5 minutes
			}
			if time.Since(existing.CreatedAt) <= time.Duration(windowSec)*time.Second {
				// Append alert record to existing ticket.
				if err := s.ticketService.AppendAlert(ctx, existing.ID, raw, parsed); err != nil {
					return nil, fmt.Errorf("append alert: %w", err)
				}
				return &IngestResult{
					TicketID: existing.ID,
					TicketNo: existing.TicketNo,
					Status:   existing.Status,
					IsNew:    false,
				}, nil
			}
		}
	}

	// 6. Create new ticket.
	ticket, err := s.ticketService.CreateTicket(ctx, sourceID, sourceType, raw, parsed, nil, fingerprint)
	if err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}

	return &IngestResult{
		TicketID: ticket.ID,
		TicketNo: ticket.TicketNo,
		Status:   ticket.Status,
		IsNew:    true,
	}, nil
}
