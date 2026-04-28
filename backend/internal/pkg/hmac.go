package pkg

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

// Now is package-level variable for testing injection
var Now = time.Now

// SignHMAC computes HMAC-SHA256(secret, timestamp || body)
func SignHMAC(secret string, timestamp int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyHMAC checks that the signature matches the computed HMAC
func VerifyHMAC(secret string, timestamp int64, body []byte, signature string) bool {
	expected := SignHMAC(secret, timestamp, body)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// VerifyTimestamp checks that the timestamp is within maxDriftSec seconds of current time
func VerifyTimestamp(timestamp int64, maxDriftSec int64) error {
	drift := Now().Unix() - timestamp
	if drift < 0 {
		drift = -drift
	}
	if drift > maxDriftSec {
		return fmt.Errorf("timestamp drift %d seconds exceeds max %d", drift, maxDriftSec)
	}
	return nil
}
