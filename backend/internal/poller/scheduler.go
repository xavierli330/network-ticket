package poller

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/repository"
	"github.com/xavierli/network-ticket/internal/service"
)

// Scheduler manages polling workers for all alert sources with poll endpoints.
type Scheduler struct {
	alertSourceRepo *repository.AlertSourceRepo
	alertService    *service.AlertService
	logger          *zap.Logger

	workers map[int64]*Worker
	mu      sync.RWMutex
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewScheduler creates a new polling scheduler.
func NewScheduler(
	alertSourceRepo *repository.AlertSourceRepo,
	alertService *service.AlertService,
	logger *zap.Logger,
) *Scheduler {
	return &Scheduler{
		alertSourceRepo: alertSourceRepo,
		alertService:    alertService,
		logger:          logger,
		workers:         make(map[int64]*Worker),
		stopCh:          make(chan struct{}),
	}
}

// Start initializes and starts all polling workers.
func (s *Scheduler) Start() error {
	sources, err := s.loadPollableSources()
	if err != nil {
		return err
	}

	for i := range sources {
		s.startWorker(&sources[i])
	}

	s.logger.Info("poller scheduler started", zap.Int("workers", len(s.workers)))

	// Start periodic reload goroutine.
	s.wg.Add(1)
	go s.reloadLoop()

	return nil
}

// Stop gracefully stops all workers.
func (s *Scheduler) Stop() {
	close(s.stopCh)

	s.mu.Lock()
	for id, worker := range s.workers {
		worker.Stop()
		delete(s.workers, id)
	}
	s.mu.Unlock()

	s.wg.Wait()
	s.logger.Info("poller scheduler stopped")
}

// Reload immediately refreshes the worker list from database.
func (s *Scheduler) Reload() error {
	sources, err := s.loadPollableSources()
	if err != nil {
		return err
	}

	desired := make(map[int64]*model.AlertSource)
	for i := range sources {
		desired[sources[i].ID] = &sources[i]
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop workers for removed sources.
	for id, worker := range s.workers {
		if _, ok := desired[id]; !ok {
			worker.Stop()
			delete(s.workers, id)
			s.logger.Info("poller worker removed", zap.Int64("source_id", id))
		}
	}

	// Start workers for new sources.
	for id, source := range desired {
		if _, ok := s.workers[id]; !ok {
			s.startWorkerLocked(source)
			s.logger.Info("poller worker added", zap.Int64("source_id", id))
		}
	}

	return nil
}

func (s *Scheduler) loadPollableSources() ([]model.AlertSource, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	all, err := s.alertSourceRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	var pollable []model.AlertSource
	for i := range all {
		if all[i].PollEndpoint != nil && *all[i].PollEndpoint != "" && all[i].Status == "active" {
			pollable = append(pollable, all[i])
		}
	}
	return pollable, nil
}

func (s *Scheduler) startWorker(source *model.AlertSource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.startWorkerLocked(source)
}

func (s *Scheduler) startWorkerLocked(source *model.AlertSource) {
	worker := NewWorker(source, s.alertService, s.logger)
	s.workers[source.ID] = worker
	worker.Start()
}

func (s *Scheduler) reloadLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			if err := s.Reload(); err != nil {
				s.logger.Error("poller reload failed", zap.Error(err))
			}
		}
	}
}
