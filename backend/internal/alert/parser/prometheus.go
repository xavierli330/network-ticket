package parser

import (
	"context"
	"encoding/json"
	"time"

	"github.com/tidwall/gjson"
)

type PrometheusParser struct{}

func (p *PrometheusParser) SourceType() string {
	return "prometheus"
}

func (p *PrometheusParser) Parse(ctx context.Context, raw json.RawMessage) (*ParsedAlert, error) {
	s := string(raw)

	title := gjson.Get(s, "alerts.0.labels.alertname").String()

	description := gjson.Get(s, "alerts.0.annotations.summary").String()
	if description == "" {
		description = gjson.Get(s, "alerts.0.annotations.description").String()
	}

	severity := normalizeSeverity(gjson.Get(s, "alerts.0.labels.severity").String())
	if severity == "info" {
		status := gjson.Get(s, "alerts.0.status").String()
		if status == "firing" {
			severity = "warning"
		}
	}

	sourceIP := gjson.Get(s, "alerts.0.labels.instance").String()
	deviceName := gjson.Get(s, "alerts.0.labels.device").String()

	alertTime := time.Now()
	startsAt := gjson.Get(s, "alerts.0.startsAt").String()
	if startsAt != "" {
		if t, err := time.Parse(time.RFC3339, startsAt); err == nil {
			alertTime = t
		}
	}

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
		AlertTime:   alertTime,
		Fields:      fields,
	}, nil
}
