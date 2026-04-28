package model

import "time"

// Workflow node name constants.
const (
	NodeNameAlertReceived = "alert_received"
	NodeNameParsed        = "parsed"
	NodeNamePushed        = "pushed"
	NodeNameAwaitingAuth  = "awaiting_auth"
	NodeNameAuthorized    = "authorized"
	NodeNameExecuting     = "executing"
	NodeNameCompleted     = "completed"
)

// Workflow node status constants.
const (
	NodeStatusPending = "pending"
	NodeStatusActive  = "active"
	NodeStatusDone    = "done"
	NodeStatusFailed  = "failed"
	NodeStatusSkipped = "skipped"
	NodeStatusTimeout = "timeout"
)

// WorkflowState represents a single workflow node state for a ticket.
type WorkflowState struct {
	ID           int64      `db:"id"            json:"id"`
	TicketID     int64      `db:"ticket_id"     json:"ticket_id"`
	NodeName     string     `db:"node_name"     json:"node_name"`
	Status       string     `db:"status"        json:"status"`
	Operator     *string    `db:"operator"      json:"operator"`
	InputData    JSON       `db:"input_data"    json:"input_data"`
	OutputData   JSON       `db:"output_data"   json:"output_data"`
	ErrorMessage *string    `db:"error_message" json:"error_message"`
	StartedAt    *time.Time `db:"started_at"    json:"started_at"`
	CompletedAt  *time.Time `db:"completed_at"  json:"completed_at"`
	CreatedAt    time.Time  `db:"created_at"    json:"created_at"`
}
