package model

import "time"

// AlertSource represents an alert source configuration.
type AlertSource struct {
	ID            int64     `db:"id"             json:"id"`
	Name          string    `db:"name"           json:"name"`
	Type          string    `db:"type"           json:"type"`
	Config        JSON      `db:"config"         json:"config"`
	ParserConfig  JSON      `db:"parser_config"  json:"parser_config"`
	WebhookSecret *string   `db:"webhook_secret" json:"webhook_secret"`
	PollEndpoint  *string   `db:"poll_endpoint"  json:"poll_endpoint"`
	PollInterval  int       `db:"poll_interval"  json:"poll_interval"`
	DedupFields   JSON      `db:"dedup_fields"   json:"dedup_fields"`
	DedupWindow   int       `db:"dedup_window_sec" json:"dedup_window_sec"`
	Status        string    `db:"status"         json:"status"`
	CreatedAt     time.Time `db:"created_at"     json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"     json:"updated_at"`
}
