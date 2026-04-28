package parser

import (
	"context"
	"encoding/json"
	"time"
)

type ParsedAlert struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"`
	SourceIP    string                 `json:"source_ip"`
	DeviceName  string                 `json:"device_name"`
	AlertTime   time.Time              `json:"alert_time"`
	Fields      map[string]interface{} `json:"fields"`
}

type AlertParser interface {
	Parse(ctx context.Context, raw json.RawMessage) (*ParsedAlert, error)
	SourceType() string
}

var registry = map[string]AlertParser{}

func Register(p AlertParser) {
	registry[p.SourceType()] = p
}

func Get(sourceType string) (AlertParser, bool) {
	p, ok := registry[sourceType]
	return p, ok
}

func init() {
	Register(&GenericJSONParser{})
	Register(&ZabbixParser{})
	Register(&PrometheusParser{})
}
