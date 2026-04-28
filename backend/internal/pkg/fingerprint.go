package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

// ComputeFingerprint extracts fields from raw JSON using gjson paths and hashes them
func ComputeFingerprint(raw json.RawMessage, dedupFields []string) (string, error) {
	parts := make([]string, 0, len(dedupFields))
	for _, field := range dedupFields {
		val := gjson.GetBytes(raw, field)
		if !val.Exists() {
			return "", fmt.Errorf("fingerprint field %s not found", field)
		}
		parts = append(parts, val.String())
	}
	joined := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(hash[:]), nil
}
