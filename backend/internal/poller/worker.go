package poller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/service"
)

// Worker polls a single alert source at regular intervals.
type Worker struct {
	source       *model.AlertSource
	alertService *service.AlertService
	logger       *zap.Logger
	client       *http.Client
	stopCh       chan struct{}
}

// NewWorker creates a new polling worker for an alert source.
func NewWorker(source *model.AlertSource, alertService *service.AlertService, logger *zap.Logger) *Worker {
	return &Worker{
		source:       source,
		alertService: alertService,
		logger:       logger,
		client:       &http.Client{Timeout: 30 * time.Second},
		stopCh:       make(chan struct{}),
	}
}

// Start begins polling in a background goroutine.
func (w *Worker) Start() {
	go w.run()
}

// Stop signals the worker to stop.
func (w *Worker) Stop() {
	close(w.stopCh)
}

func (w *Worker) run() {
	interval := time.Duration(w.source.PollInterval) * time.Second
	if interval <= 0 {
		interval = 60 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Poll immediately on start.
	w.pollWithRetry()

	for {
		select {
		case <-w.stopCh:
			w.logger.Info("poller worker stopped", zap.Int64("source_id", w.source.ID))
			return
		case <-ticker.C:
			w.pollWithRetry()
		}
	}
}

// pollWithRetry performs polling with exponential backoff on failure.
func (w *Worker) pollWithRetry() {
	baseInterval := time.Second
	maxInterval := 30 * time.Second
	maxRetries := 5

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := baseInterval * time.Duration(1<<attempt)
			if backoff > maxInterval {
				backoff = maxInterval
			}
			w.logger.Warn("poll retry scheduled",
				zap.Int64("source_id", w.source.ID),
				zap.Int("attempt", attempt),
				zap.Duration("backoff", backoff),
			)
			time.Sleep(backoff)
		}

		if err := w.poll(); err != nil {
			w.logger.Warn("poll failed",
				zap.Int64("source_id", w.source.ID),
				zap.Int("attempt", attempt),
				zap.Error(err),
			)
			continue
		}
		return
	}

	w.logger.Error("poll exhausted all retries",
		zap.Int64("source_id", w.source.ID),
		zap.Int("max_retries", maxRetries),
	)
}

func (w *Worker) poll() error {
	w.logger.Debug("polling alert source",
		zap.Int64("source_id", w.source.ID),
		zap.String("endpoint", *w.source.PollEndpoint),
	)

	req, err := http.NewRequest(http.MethodGet, *w.source.PollEndpoint, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	// Try to parse as array first.
	var alerts []json.RawMessage
	if err := json.Unmarshal(body, &alerts); err != nil {
		// Try single alert.
		var single json.RawMessage
		if err := json.Unmarshal(body, &single); err != nil {
			return fmt.Errorf("parse JSON: %w", err)
		}
		alerts = []json.RawMessage{single}
	}

	w.logger.Info("poll received alerts",
		zap.Int64("source_id", w.source.ID),
		zap.Int("count", len(alerts)),
	)

	for i, alert := range alerts {
		if _, err := w.alertService.Ingest(context.Background(), w.source.ID, alert); err != nil {
			w.logger.Error("failed to ingest polled alert",
				zap.Int64("source_id", w.source.ID),
				zap.Int("index", i),
				zap.Error(err),
			)
		}
	}

	return nil
}
