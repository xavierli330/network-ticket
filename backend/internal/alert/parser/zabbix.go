package parser

import (
	"context"
	"encoding/json"
	"time"

	"github.com/tidwall/gjson"
)

type ZabbixParser struct{}

func (z *ZabbixParser) SourceType() string {
	return "zabbix"
}

func (z *ZabbixParser) Parse(ctx context.Context, raw json.RawMessage) (*ParsedAlert, error) {
	s := string(raw)

	title := gjson.Get(s, "subject").String()
	if title == "" {
		title = gjson.Get(s, `event\.name`).String()
	}

	description := gjson.Get(s, "message").String()
	severity := normalizeSeverity(gjson.Get(s, `event\.severity`).String())
	sourceIP := gjson.Get(s, "host.ip").String()
	deviceName := gjson.Get(s, "host.name").String()

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
