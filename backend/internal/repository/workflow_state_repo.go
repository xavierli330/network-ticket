package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/xavierli/network-ticket/internal/model"
)

type WorkflowStateRepo struct {
	db *sqlx.DB
}

func NewWorkflowStateRepo(db *sqlx.DB) *WorkflowStateRepo {
	return &WorkflowStateRepo{db: db}
}

// Create inserts a new workflow state record.
func (r *WorkflowStateRepo) Create(ctx context.Context, ws *model.WorkflowState) (int64, error) {
	query := `INSERT INTO workflow_states
		(ticket_id, node_name, status, operator, input_data, output_data, error_message, started_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		ws.TicketID, ws.NodeName, ws.Status, ws.Operator,
		ws.InputData, ws.OutputData, ws.ErrorMessage, ws.StartedAt, ws.CompletedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("insert workflow_state: %w", err)
	}
	id, _ := result.LastInsertId()
	return id, nil
}

// ListByTicketID returns all workflow states for a given ticket.
func (r *WorkflowStateRepo) ListByTicketID(ctx context.Context, ticketID int64) ([]model.WorkflowState, error) {
	var states []model.WorkflowState
	query := `SELECT * FROM workflow_states WHERE ticket_id = ? ORDER BY id`
	if err := r.db.SelectContext(ctx, &states, query, ticketID); err != nil {
		return nil, fmt.Errorf("list workflow_states by ticket_id: %w", err)
	}
	return states, nil
}

// UpdateStatus updates the status of a workflow node.
func (r *WorkflowStateRepo) UpdateStatus(ctx context.Context, ticketID int64, nodeName string, status string) error {
	query := `UPDATE workflow_states SET status = ? WHERE ticket_id = ? AND node_name = ?`
	if _, err := r.db.ExecContext(ctx, query, status, ticketID, nodeName); err != nil {
		return fmt.Errorf("update workflow_state status: %w", err)
	}
	return nil
}

// UpdateNodeData updates output_data and error_message for a workflow node.
func (r *WorkflowStateRepo) UpdateNodeData(ctx context.Context, ticketID int64, nodeName string, outputData model.JSON, errorMessage *string) error {
	query := `UPDATE workflow_states SET output_data = ?, error_message = ? WHERE ticket_id = ? AND node_name = ?`
	if _, err := r.db.ExecContext(ctx, query, outputData, errorMessage, ticketID, nodeName); err != nil {
		return fmt.Errorf("update workflow_state node data: %w", err)
	}
	return nil
}
