package tests

import (
	"encoding/json"
	"testing"

	"github.com/xavierli/network-ticket/internal/pkg"
)

func TestComputeFingerprint(t *testing.T) {
	dedupFields := []string{"alertname", "instance", "severity"}

	t.Run("same field values produce same fingerprint", func(t *testing.T) {
		raw := json.RawMessage(`{"alertname":"HighCPU","instance":"10.0.0.1","severity":"critical"}`)
		fp1, err := pkg.ComputeFingerprint(raw, dedupFields)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Same JSON with different key order should produce same fingerprint
		raw2 := json.RawMessage(`{"severity":"critical","alertname":"HighCPU","instance":"10.0.0.1"}`)
		fp2, err := pkg.ComputeFingerprint(raw2, dedupFields)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if fp1 != fp2 {
			t.Errorf("expected same fingerprints, got %s and %s", fp1, fp2)
		}
	})

	t.Run("different field values produce different fingerprints", func(t *testing.T) {
		raw1 := json.RawMessage(`{"alertname":"HighCPU","instance":"10.0.0.1","severity":"critical"}`)
		fp1, err := pkg.ComputeFingerprint(raw1, dedupFields)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		raw2 := json.RawMessage(`{"alertname":"HighMemory","instance":"10.0.0.1","severity":"critical"}`)
		fp2, err := pkg.ComputeFingerprint(raw2, dedupFields)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if fp1 == fp2 {
			t.Errorf("expected different fingerprints but both are %s", fp1)
		}
	})

	t.Run("missing field returns error", func(t *testing.T) {
		raw := json.RawMessage(`{"alertname":"HighCPU","instance":"10.0.0.1"}`)
		_, err := pkg.ComputeFingerprint(raw, dedupFields)
		if err == nil {
			t.Error("expected error for missing field but got nil")
		}
	})

	t.Run("fingerprint is hex encoded sha256", func(t *testing.T) {
		raw := json.RawMessage(`{"alertname":"HighCPU","instance":"10.0.0.1","severity":"critical"}`)
		fp, err := pkg.ComputeFingerprint(raw, dedupFields)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// SHA256 hex is 64 characters
		if len(fp) != 64 {
			t.Errorf("expected 64-char hex fingerprint, got %d chars: %s", len(fp), fp)
		}
	})
}
