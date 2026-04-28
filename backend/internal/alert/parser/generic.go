package parser

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

type GenericJSONParser struct{}

func (g *GenericJSONParser) SourceType() string {
	return "generic"
}

func (g *GenericJSONParser) Parse(ctx context.Context, raw json.RawMessage) (*ParsedAlert, error) {
	s := string(raw)

	title := gjson.Get(s, "title").String()
	if title == "" {
		title = gjson.Get(s, "alertname").String()
	}

	description := gjson.Get(s, "description").String()
	if description == "" {
		description = gjson.Get(s, "message").String()
	}

	severity := normalizeSeverity(gjson.Get(s, "severity").String())
	sourceIP := gjson.Get(s, "source_ip").String()
	deviceName := gjson.Get(s, "device_name").String()

	var fields map[string]interface{}
	if err := json.Unmarshal(raw, &fields); err != nil {
		fields = make(map[string]interface{})
	}

	return &ParsedAlert{
		Title:       title,
		Description: description,
		Severity:    severity,
		SourceIP:    sourceIP,
		DeviceName:  deviceName,
		AlertTime:   time.Now(),
		Fields:      fields,
	}, nil
}

func normalizeSeverity(s string) string {
	s = strings.ToLower(s)
	switch {
	case strings.Contains(s, "crit") || strings.Contains(s, "p1") || strings.Contains(s, "emerg"):
		return "critical"
	case strings.Contains(s, "warn") || strings.Contains(s, "p2") || strings.Contains(s, "high"):
		return "warning"
	default:
		return "info"
	}
}
