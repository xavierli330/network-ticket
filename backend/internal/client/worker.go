package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/repository"
)

// PushJob represents a single push-to-client task.
type PushJob struct {
	Ticket  *model.Ticket
	Client  *model.Client
	Attempt int
}

// WorkerPool manages a pool of goroutines that process PushJobs.
type WorkerPool struct {
	jobs         chan *PushJob
	wg           sync.WaitGroup
	poolSize     int
	retryCfg     RetryConfig
	ticketRepo   *repository.TicketRepo
	clientRepo   *repository.ClientRepo
	workflowRepo *repository.WorkflowStateRepo
	logger       *zap.Logger
}

// NewWorkerPool creates a WorkerPool with the given configuration.
func NewWorkerPool(
	poolSize int,
	retryCfg RetryConfig,
	ticketRepo *repository.TicketRepo,
	clientRepo *repository.ClientRepo,
	workflowRepo *repository.WorkflowStateRepo,
	logger *zap.Logger,
) *WorkerPool {
	return &WorkerPool{
		jobs:         make(chan *PushJob, poolSize*10),
		poolSize:     poolSize,
		retryCfg:     retryCfg,
		ticketRepo:   ticketRepo,
		clientRepo:   clientRepo,
		workflowRepo: workflowRepo,
		logger:       logger,
	}
}

// Start launches poolSize worker goroutines.
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.poolSize; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// Stop closes the job channel and waits for all workers to finish.
func (wp *WorkerPool) Stop() {
	close(wp.jobs)
	wp.wg.Wait()
}

// Submit enqueues a push job for processing.
func (wp *WorkerPool) Submit(job *PushJob) {
	wp.jobs <- job
}

// worker is the main loop for each goroutine in the pool.
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()
	for job := range wp.jobs {
		wp.process(job, id)
	}
}

// process handles a single push job: build the request, call Push,
// update ticket/workflow state, and schedule retries on failure.
func (wp *WorkerPool) process(job *PushJob, workerID int) {
	ctx := context.Background()

	wp.logger.Info("processing push job",
		zap.Int("worker", workerID),
		zap.String("ticket_no", job.Ticket.TicketNo),
		zap.Int64("client_id", job.Client.ID),
		zap.Int("attempt", job.Attempt),
	)

	// Build the push request from the ticket and client data.
	var alertParsed interface{}
	if len(job.Ticket.AlertParsed) > 0 {
		_ = json.Unmarshal(job.Ticket.AlertParsed, &alertParsed)
	}

	description := ""
	if job.Ticket.Description != nil {
		description = *job.Ticket.Description
	}

	callbackURL := ""
	if job.Client.CallbackURL != nil {
		callbackURL = *job.Client.CallbackURL
	}

	pushReq := &PushRequest{
		TicketNo:    job.Ticket.TicketNo,
		Title:       job.Ticket.Title,
		Description: description,
		Severity:    job.Ticket.Severity,
		AlertParsed: alertParsed,
		CallbackURL: callbackURL,
	}

	// Perform the HTTP push.
	resp, err := Push(ctx, job.Client.APIEndpoint, job.Client.APIKey, job.Client.HMACSecret, pushReq)
	if err != nil {
		wp.logger.Error("push failed",
			zap.String("ticket_no", job.Ticket.TicketNo),
			zap.Int("attempt", job.Attempt),
			zap.Error(err),
		)
		wp.handleFailure(job, err.Error(), workerID)
		return
	}

	if resp.Success {
		wp.logger.Info("push succeeded",
			zap.String("ticket_no", job.Ticket.TicketNo),
			zap.Int("status_code", resp.StatusCode),
		)
		wp.handleSuccess(job, resp)
	} else {
		wp.logger.Warn("push returned non-success status",
			zap.String("ticket_no", job.Ticket.TicketNo),
			zap.Int("status_code", resp.StatusCode),
			zap.String("body", resp.Body),
		)
		wp.handleFailure(job, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Body), workerID)
	}
}

// handleSuccess updates the ticket to in_progress and marks the pushed workflow node as done.
func (wp *WorkerPool) handleSuccess(job *PushJob, resp *PushResponse) {
	ctx := context.Background()

	// Update ticket status to in_progress.
	if err := wp.ticketRepo.UpdateStatus(ctx, job.Ticket.ID, model.TicketStatusInProgress); err != nil {
		wp.logger.Error("failed to update ticket status after successful push",
			zap.String("ticket_no", job.Ticket.TicketNo),
			zap.Error(err),
		)
	}

	// Update the "pushed" workflow node to done.
	outputData := model.JSON(fmt.Sprintf(`{"status_code":%d,"body":%q}`, resp.StatusCode, resp.Body))
	if err := wp.workflowRepo.UpdateStatus(ctx, job.Ticket.ID, model.NodeNamePushed, model.NodeStatusDone); err != nil {
		wp.logger.Error("failed to update workflow state after successful push",
			zap.String("ticket_no", job.Ticket.TicketNo),
			zap.Error(err),
		)
	}
	if err := wp.workflowRepo.UpdateNodeData(ctx, job.Ticket.ID, model.NodeNamePushed, outputData, nil); err != nil {
		wp.logger.Error("failed to update workflow node data after successful push",
			zap.String("ticket_no", job.Ticket.TicketNo),
			zap.Error(err),
		)
	}
}

// handleFailure either schedules a retry with backoff or marks the ticket as failed
// when retries are exhausted.
func (wp *WorkerPool) handleFailure(job *PushJob, errMsg string, workerID int) {
	ctx := context.Background()

	if job.Attempt+1 < wp.retryCfg.MaxAttempts {
		// Schedule a retry.
		delay := Backoff(job.Attempt, wp.retryCfg)
		wp.logger.Info("scheduling retry",
			zap.String("ticket_no", job.Ticket.TicketNo),
			zap.Int("next_attempt", job.Attempt+1),
			zap.Duration("delay", delay),
		)

		nextJob := &PushJob{
			Ticket:  job.Ticket,
			Client:  job.Client,
			Attempt: job.Attempt + 1,
		}
		time.AfterFunc(delay, func() {
			wp.Submit(nextJob)
		})
	} else {
		// Retries exhausted: mark ticket as failed.
		wp.logger.Error("push retries exhausted, marking ticket as failed",
			zap.String("ticket_no", job.Ticket.TicketNo),
			zap.Int("attempts", job.Attempt+1),
		)

		if err := wp.ticketRepo.UpdateStatus(ctx, job.Ticket.ID, model.TicketStatusFailed); err != nil {
			wp.logger.Error("failed to update ticket status to failed",
				zap.String("ticket_no", job.Ticket.TicketNo),
				zap.Error(err),
			)
		}

		// Update the "pushed" workflow node to failed.
		errMsgPtr := &errMsg
		if err := wp.workflowRepo.UpdateStatus(ctx, job.Ticket.ID, model.NodeNamePushed, model.NodeStatusFailed); err != nil {
			wp.logger.Error("failed to update workflow state to failed",
				zap.String("ticket_no", job.Ticket.TicketNo),
				zap.Error(err),
			)
		}
		if err := wp.workflowRepo.UpdateNodeData(ctx, job.Ticket.ID, model.NodeNamePushed, nil, errMsgPtr); err != nil {
			wp.logger.Error("failed to update workflow node data on failure",
				zap.String("ticket_no", job.Ticket.TicketNo),
				zap.Error(err),
			)
		}
	}
}
