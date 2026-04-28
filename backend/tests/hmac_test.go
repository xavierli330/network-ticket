package tests

import (
	"testing"
	"time"

	"github.com/xavierli/network-ticket/internal/pkg"
)

func TestSignAndVerifyHMAC(t *testing.T) {
	secret := "test-secret-key"
	timestamp := time.Now().Unix()
	body := []byte(`{"alert":"cpu high"}`)

	sig := pkg.SignHMAC(secret, timestamp, body)

	// Correct secret should verify
	if !pkg.VerifyHMAC(secret, timestamp, body, sig) {
		t.Error("expected HMAC verification to succeed with correct secret")
	}

	// Wrong secret should fail
	if pkg.VerifyHMAC("wrong-secret", timestamp, body, sig) {
		t.Error("expected HMAC verification to fail with wrong secret")
	}

	// Tampered body should fail
	if pkg.VerifyHMAC(secret, timestamp, []byte(`{"alert":"cpu low"}`), sig) {
		t.Error("expected HMAC verification to fail with tampered body")
	}

	// Wrong timestamp should fail
	if pkg.VerifyHMAC(secret, timestamp+100, body, sig) {
		t.Error("expected HMAC verification to fail with wrong timestamp")
	}
}

func TestVerifyTimestamp(t *testing.T) {
	// Save original and restore after test
	originalNow := pkg.Now
	defer func() { pkg.Now = originalNow }()

	fixedTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	pkg.Now = func() time.Time { return fixedTime }

	maxDrift := int64(300) // 5 minutes

	tests := []struct {
		name      string
		offset    int64
		expectErr bool
	}{
		{"same timestamp", 0, false},
		{"within drift - 1 second", 1, false},
		{"within drift - 299 seconds", 299, false},
		{"at drift boundary - 300 seconds", 300, false},
		{"beyond drift - 301 seconds", 301, true},
		{"beyond drift - 600 seconds", 600, true},
		{"negative offset - within drift", -299, false},
		{"negative offset - beyond drift", -301, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := fixedTime.Unix() + tt.offset
			err := pkg.VerifyTimestamp(ts, maxDrift)
			if tt.expectErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}
