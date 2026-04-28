package tests

import (
	"context"
	"testing"
	"time"

	"github.com/xavierli/network-ticket/internal/nonce"
)

func TestFileNonceStore(t *testing.T) {
	path := t.TempDir() + "/nonces.log"
	store, err := nonce.NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	// First nonce should succeed
	ok, err := store.CheckAndSet(ctx, "nonce-1", 5*time.Minute)
	if err != nil || !ok {
		t.Fatalf("first check should succeed: ok=%v err=%v", ok, err)
	}

	// Duplicate should fail
	ok, _ = store.CheckAndSet(ctx, "nonce-1", 5*time.Minute)
	if ok {
		t.Error("duplicate nonce should return false")
	}

	// Different nonce should succeed
	ok, _ = store.CheckAndSet(ctx, "nonce-2", 5*time.Minute)
	if !ok {
		t.Error("different nonce should succeed")
	}
}
