package tests

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/xavierli/network-ticket/internal/alert/parser"
)

func TestZabbixParser(t *testing.T) {
	p, ok := parser.Get("zabbix")
	if !ok {
		t.Fatal("zabbix parser not registered")
	}

	data, err := os.ReadFile("testdata/zabbix_alert.json")
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	alert, err := p.Parse(context.Background(), json.RawMessage(data))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if alert.Title != "Interface Gi0/1 is down on switch-01" {
		t.Errorf("expected title 'Interface Gi0/1 is down on switch-01', got '%s'", alert.Title)
	}
	if alert.SourceIP != "10.0.0.1" {
		t.Errorf("expected source_ip '10.0.0.1', got '%s'", alert.SourceIP)
	}
	if alert.Severity != "warning" {
		t.Errorf("expected severity 'warning', got '%s'", alert.Severity)
	}
	if alert.DeviceName != "switch-01" {
		t.Errorf("expected device_name 'switch-01', got '%s'", alert.DeviceName)
	}
}

func TestPrometheusParser(t *testing.T) {
	p, ok := parser.Get("prometheus")
	if !ok {
		t.Fatal("prometheus parser not registered")
	}

	data, err := os.ReadFile("testdata/prometheus_alert.json")
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	alert, err := p.Parse(context.Background(), json.RawMessage(data))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if alert.Title != "HighCPU" {
		t.Errorf("expected title 'HighCPU', got '%s'", alert.Title)
	}
	if alert.Severity != "critical" {
		t.Errorf("expected severity 'critical', got '%s'", alert.Severity)
	}
	if alert.SourceIP != "10.0.1.5" {
		t.Errorf("expected source_ip '10.0.1.5', got '%s'", alert.SourceIP)
	}
	if alert.DeviceName != "server-03" {
		t.Errorf("expected device_name 'server-03', got '%s'", alert.DeviceName)
	}
	if alert.Description != "CPU usage above 90%" {
		t.Errorf("expected description \"CPU usage above 90%%\", got %q", alert.Description)
	}
}

func TestGenericParser(t *testing.T) {
	p, ok := parser.Get("generic")
	if !ok {
		t.Fatal("generic parser not registered")
	}

	data, err := os.ReadFile("testdata/generic_alert.json")
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	alert, err := p.Parse(context.Background(), json.RawMessage(data))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if alert.Title != "Disk usage high" {
		t.Errorf("expected title 'Disk usage high', got '%s'", alert.Title)
	}
	if alert.Description != "Disk / is 92% full" {
		t.Errorf("expected description \"Disk / is 92%% full\", got %q", alert.Description)
	}
	if alert.Severity != "warning" {
		t.Errorf("expected severity 'warning', got '%s'", alert.Severity)
	}
	if alert.SourceIP != "192.168.1.100" {
		t.Errorf("expected source_ip '192.168.1.100', got '%s'", alert.SourceIP)
	}
	if alert.DeviceName != "web-server-01" {
		t.Errorf("expected device_name 'web-server-01', got '%s'", alert.DeviceName)
	}
}

func TestGenericParserFallback(t *testing.T) {
	p, ok := parser.Get("generic")
	if !ok {
		t.Fatal("generic parser not registered")
	}

	raw := json.RawMessage(`{"alertname": "TestAlert", "message": "something happened"}`)
	alert, err := p.Parse(context.Background(), raw)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// title should fall back to alertname
	if alert.Title != "TestAlert" {
		t.Errorf("expected fallback title 'TestAlert', got '%s'", alert.Title)
	}
	// description should fall back to message
	if alert.Description != "something happened" {
		t.Errorf("expected fallback description 'something happened', got '%s'", alert.Description)
	}
	// severity defaults to info when empty
	if alert.Severity != "info" {
		t.Errorf("expected default severity 'info', got '%s'", alert.Severity)
	}
}
