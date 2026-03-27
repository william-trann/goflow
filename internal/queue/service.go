package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/william-trann/goflow/internal/clock"
	"github.com/william-trann/goflow/internal/errors"
	"github.com/william-trann/goflow/internal/idempotency"
	"github.com/william-trann/goflow/internal/model"
	"github.com/william-trann/goflow/internal/retry"
	"github.com/william-trann/goflow/internal/storage"
)

type Service struct {
	store  storage.Backend
	policy retry.Policy
	idem   *idempotency.Service
	clock  clock.Clock
}

func NewService(store storage.Backend, policy retry.Policy, idem *idempotency.Service, clock clock.Clock) *Service {
	return &Service{
		store:  store,
		policy: policy,
		idem:   idem,
		clock:  clock,
	}
}

func (s *Service) Enqueue(ctx context.Context, req EnqueueRequest) (*model.Job, error) {
	queueName := strings.TrimSpace(req.Queue)
	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if queueName == "" {
		return nil, fmt.Errorf("%w: queue is required", errors.ErrValidation)
	}
	if req.MaxRetries < 0 {
		return nil, fmt.Errorf("%w: max_retries must be >= 0", errors.ErrValidation)
	}

	payload := append(json.RawMessage(nil), req.Payload...)
	if len(payload) == 0 {
		payload = json.RawMessage("{}")
	}
	if !json.Valid(payload) {
		return nil, fmt.Errorf("%w: payload must be valid JSON", errors.ErrValidation)
	}

	if existing, err := s.idem.Lookup(ctx, queueName, idempotencyKey); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}

	now := s.clock.Now().UTC()
	runAt := now
	status := model.StatusQueued
	if req.RunAt != nil {
		runAt = req.RunAt.UTC()
		if runAt.After(now) {
			status = model.StatusScheduled
		}
	}

	jobID, err := model.NewJobID()
	if err != nil {
		return nil, err
	}

	job := &model.Job{
		ID:             jobID,
		Queue:          queueName,
		Payload:        payload,
		Priority:       req.Priority,
		MaxRetries:     req.MaxRetries,
		RunAt:          runAt,
		Status:         status,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	return s.store.Enqueue(ctx, job)
}

func (s *Service) GetJob(ctx context.Context, jobID string) (*model.Job, error) {
	return s.store.GetJob(ctx, strings.TrimSpace(jobID))
}

func (s *Service) Dequeue(ctx context.Context, queueName string) (*model.Job, error) {
	queueName = strings.TrimSpace(queueName)
	if queueName == "" {
		return nil, fmt.Errorf("%w: queue is required", errors.ErrValidation)
	}
	return s.store.Dequeue(ctx, queueName, s.clock.Now().UTC())
}

func (s *Service) Ack(ctx context.Context, jobID string) (*model.Job, error) {
	return s.store.Ack(ctx, strings.TrimSpace(jobID), s.clock.Now().UTC())
}

func (s *Service) Retry(ctx context.Context, jobID string, cause error) (*model.Job, error) {
	job, err := s.store.GetJob(ctx, strings.TrimSpace(jobID))
	if err != nil {
		return nil, err
	}

	now := s.clock.Now().UTC()
	nextRunAt, ok := s.policy.Next(job, now)
	if !ok {
		return nil, fmt.Errorf("%w: retry budget exhausted for job %s", errors.ErrConflict, jobID)
	}

	return s.store.Retry(ctx, job.ID, nextRunAt, errorText(cause), now)
}

func (s *Service) Nack(ctx context.Context, jobID string, cause error) (*model.Job, error) {
	job, err := s.store.GetJob(ctx, strings.TrimSpace(jobID))
	if err != nil {
		return nil, err
	}

	now := s.clock.Now().UTC()
	if nextRunAt, ok := s.policy.Next(job, now); ok {
		return s.store.Retry(ctx, job.ID, nextRunAt, errorText(cause), now)
	}

	return s.store.MoveToDLQ(ctx, job.ID, errorText(cause), now)
}

func (s *Service) MoveToDLQ(ctx context.Context, jobID, reason string) (*model.Job, error) {
	return s.store.MoveToDLQ(ctx, strings.TrimSpace(jobID), strings.TrimSpace(reason), s.clock.Now().UTC())
}

func (s *Service) RequeueFromDLQ(ctx context.Context, jobID string, runAt *time.Time) (*model.Job, error) {
	now := s.clock.Now().UTC()
	target := now
	if runAt != nil {
		target = runAt.UTC()
	}
	return s.store.RequeueFromDLQ(ctx, strings.TrimSpace(jobID), target, now)
}

func (s *Service) PromoteScheduled(ctx context.Context) (int, error) {
	return s.store.PromoteScheduled(ctx, s.clock.Now().UTC())
}

func (s *Service) Stats(ctx context.Context, queueName string) (model.QueueStats, error) {
	return s.store.Stats(ctx, strings.TrimSpace(queueName))
}

func (s *Service) ListQueues(ctx context.Context) ([]model.QueueStats, error) {
	return s.store.ListQueues(ctx)
}

func errorText(err error) string {
	if err == nil {
		return "job failed"
	}
	return err.Error()
}
