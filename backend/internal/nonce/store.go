package nonce

import (
	"context"
	"time"
)

// Store is the interface for nonce deduplication storage.
// Implementations must be safe for concurrent use.
type Store interface {
	// CheckAndSet atomically checks whether the nonce already exists.
	// Returns true if the nonce was new (inserted), false if it was a duplicate.
	CheckAndSet(ctx context.Context, nonce string, ttl time.Duration) (bool, error)

	// Clean removes expired entries. The TTL used for expiry is determined
	// by the implementation or the most recent CheckAndSet call.
	Clean(ctx context.Context) error
}
