package service

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/alert/parser"
	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/pkg"
	"github.com/xavierli/network-ticket/internal/repository"
)

// allowedTransitions defines the valid state transitions for a ticket.
var allowedTransitions = map[string][]string{
	model.TicketStatusPending:    {model.TicketStatusInProgress, model.TicketStatusFailed, model.TicketStatusCancelled},
	model.TicketStatusInProgress: {model.TicketStatusCompleted, model.TicketStatusFailed, model.TicketStatusRejected, model.TicketStatusCancelled},
	model.TicketStatusFailed:     {model.TicketStatusPending, model.TicketStatusCancelled},
}

// CanTransition returns true if transitioning from one ticket status to another is allowed.
func CanTransition(from, to string) bool {
	allowed, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// allWorkflowNodes is the ordered list of workflow nodes initialized for each ticket.
var allWorkflowNodes = []string{
	model.NodeNameAlertReceived,
	model.NodeNameParsed,
	model.NodeNamePushed,
	model.NodeNameAwaitingAuth,
	model.NodeNameAuthorized,
	model.NodeNameExecuting,
	model.NodeNameCompleted,
}

// TicketService provides business logic for ticket operations.
type TicketService struct {
	ticketRepo      *repository.TicketRepo
	workflowRepo    *repository.WorkflowStateRepo
	ticketLogRepo   *repository.TicketLogRepo
	auditLogRepo    *repository.AuditLogRepo
	alertRecordRepo *repository.AlertRecordRepo
	logger          *zap.Logger
}

// NewTicketService creates a new TicketService.
func NewTicketService(
	ticketRepo *repository.TicketRepo,
	workflowRepo *repository.WorkflowStateRepo,
	ticketLogRepo *repository.TicketLogRepo,
	auditLogRepo *repository.AuditLogRepo,
	alertRecordRepo *repository.AlertRecordRepo,
	logger *zap.Logger,
) *TicketService {
	return &TicketService{
		ticketRepo:      ticketRepo,
		workflowRepo:    workflowRepo,
		ticketLogRepo:   ticketLogRepo,
		auditLogRepo:    auditLogRepo,
		alertRecordRepo: alertRecordRepo,
		logger:          logger,
	}
}

// CreateTicket creates a new ticket from an alert, initializes workflow states, and logs the creation.
func (s *TicketService) CreateTicket(ctx context.Context, alertSourceID int64, sourceType string, alertRaw json.RawMessage, parsedAlert interface{}, clientID *int64, fingerprint *string) (*model.Ticket, error) {
	ticketNo := pkg.GenerateTicketNo()

	alertParsedJSON, err := json.Marshal(parsedAlert)
	if err != nil {
		return nil, fmt.Errorf("marshal parsed alert: %w", err)
	}

	ticket := &model.Ticket{
		TicketNo:      ticketNo,
		AlertSourceID: &alertSourceID,
		SourceType:    sourceType,
		AlertRaw:      model.JSON(alertRaw),
		AlertParsed:   model.JSON(alertParsedJSON),
		Status:        model.TicketStatusPending,
		ClientID:      clientID,
		Fingerprint:   fingerprint,
	}

	// Extract title and severity from parsed alert if it's a *parser.ParsedAlert.
	if pa, ok := parsedAlert.(*parser.ParsedAlert); ok {
		ticket.Title = pa.Title
		ticket.Severity = pa.Severity
		if pa.Description != "" {
			ticket.Description = &pa.Description
		}
	}

	ticketID, err := s.ticketRepo.Create(ctx, ticket)
	if err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	ticket.ID = ticketID

	// Initialize all 7 workflow state nodes.
	for i, nodeName := range allWorkflowNodes {
		status := model.NodeStatusPending
		if i < 2 { // alert_received and parsed are done
			status = model.NodeStatusDone
		}
		ws := &model.WorkflowState{
			TicketID: ticketID,
			NodeName: nodeName,
			Status:   status,
		}
		if _, err := s.workflowRepo.Create(ctx, ws); err != nil {
			s.logger.Error("failed to create workflow state", zap.Int64("ticket_id", ticketID), zap.String("node", nodeName), zap.Error(err))
			return nil, fmt.Errorf("create workflow state %s: %w", nodeName, err)
		}
	}

	// Log the ticket creation.
	if err := s.logTransition(ctx, ticketID, nil, strPtr(model.TicketStatusPending), "system"); err != nil {
		s.logger.Warn("failed to log ticket creation", zap.Int64("ticket_id", ticketID), zap.Error(err))
	}

	s.logger.Info("ticket created", zap.Int64("id", ticketID), zap.String("ticket_no", ticketNo))
	return ticket, nil
}

// AppendAlert appends a duplicate alert record to an existing ticket.
func (s *TicketService) AppendAlert(ctx context.Context, ticketID int64, raw json.RawMessage, parsed interface{}) error {
	parsedJSON, err := json.Marshal(parsed)
	if err != nil {
		return fmt.Errorf("marshal parsed alert: %w", err)
	}
	ar := &model.AlertRecord{
		TicketID:    ticketID,
		AlertRaw:    model.JSON(raw),
		AlertParsed: model.JSON(parsedJSON),
	}
	if _, err := s.alertRecordRepo.Create(ctx, ar); err != nil {
		return fmt.Errorf("create alert record: %w", err)
	}
	s.logger.Info("alert appended to ticket", zap.Int64("ticket_id", ticketID))
	return nil
}

// TransitionStatus validates and performs a ticket status transition.
func (s *TicketService) TransitionStatus(ctx context.Context, ticketID int64, to string, operator string) error {
	ticket, err := s.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("get ticket: %w", err)
	}

	if !CanTransition(ticket.Status, to) {
		return fmt.Errorf("invalid transition from %s to %s", ticket.Status, to)
	}

	from := ticket.Status

	if err := s.ticketRepo.UpdateStatus(ctx, ticketID, to); err != nil {
		return fmt.Errorf("update ticket status: %w", err)
	}

	if err := s.logTransition(ctx, ticketID, strPtr(from), strPtr(to), operator); err != nil {
		s.logger.Warn("failed to log transition", zap.Int64("ticket_id", ticketID), zap.Error(err))
	}

	s.logger.Info("ticket status transitioned",
		zap.Int64("ticket_id", ticketID),
		zap.String("from", from),
		zap.String("to", to),
		zap.String("operator", operator),
	)
	return nil
}

// GetByTicketNo retrieves a ticket by its ticket number.
func (s *TicketService) GetByTicketNo(ctx context.Context, ticketNo string) (*model.Ticket, error) {
	return s.ticketRepo.GetByTicketNo(ctx, ticketNo)
}

// GetByID retrieves a ticket by its primary key.
func (s *TicketService) GetByID(ctx context.Context, id int64) (*model.Ticket, error) {
	return s.ticketRepo.GetByID(ctx, id)
}

// List returns a paginated list of tickets matching the filter.
func (s *TicketService) List(ctx context.Context, filter model.TicketFilter) ([]model.Ticket, int64, error) {
	tickets, total, err := s.ticketRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	if tickets == nil {
		tickets = []model.Ticket{}
	}
	return tickets, int64(total), nil
}

// GetWorkflowStates returns all workflow states for a ticket.
func (s *TicketService) GetWorkflowStates(ctx context.Context, ticketID int64) ([]model.WorkflowState, error) {
	return s.workflowRepo.ListByTicketID(ctx, ticketID)
}

// logTransition creates a ticket log entry for a status change.
func (s *TicketService) logTransition(ctx context.Context, ticketID int64, fromState, toState *string, operator string) error {
	tl := &model.TicketLog{
		TicketID:  ticketID,
		Action:    "status_change",
		FromState: fromState,
		ToState:   toState,
		Operator:  &operator,
		Detail:    model.JSON([]byte("{}")),
	}
	_, err := s.ticketLogRepo.Create(ctx, tl)
	return err
}

// strPtr is a helper to get a pointer to a string.
func strPtr(s string) *string {
	return &s
}
