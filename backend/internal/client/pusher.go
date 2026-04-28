package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/xavierli/network-ticket/internal/pkg"
)

// PushRequest is the payload sent to the client callback endpoint.
type PushRequest struct {
	TicketNo    string      `json:"ticket_no"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Severity    string      `json:"severity"`
	AlertParsed interface{} `json:"alert_parsed"`
	CallbackURL string      `json:"callback_url"`
}

// PushResponse captures the result of a single HTTP push attempt.
type PushResponse struct {
	StatusCode int
	Body       string
	Success    bool
}

// Push sends an authenticated, HMAC-signed POST to the client endpoint.
func Push(ctx context.Context, endpoint, apiKey, hmacSecret string, req *PushRequest) (*PushResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal push request: %w", err)
	}

	timestamp := time.Now().Unix()
	signature := pkg.SignHMAC(hmacSecret, timestamp, body)
	nonce := uuid.New().String()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Api-Key", apiKey)
	httpReq.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	httpReq.Header.Set("X-Signature", signature)
	httpReq.Header.Set("X-Nonce", nonce)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return &PushResponse{Success: false}, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	return &PushResponse{
		StatusCode: resp.StatusCode,
		Body:       string(respBody),
		Success:    success,
	}, nil
}
