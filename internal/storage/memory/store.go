package memory

import (
	"container/heap"
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/william-trann/goflow/internal/errors"
	"github.com/william-trann/goflow/internal/model"
)

type Store struct {
	mu          sync.RWMutex
	jobs        map[string]*model.Job
	ready       map[string]*readyHeap
	delayed     delayedHeap
	sequence    int64
	idempotency map[string]string
}

func New() *Store {
	delayed := delayedHeap{}
	heap.Init(&delayed)

	return &Store{
		jobs:        make(map[string]*model.Job),
		ready:       make(map[string]*readyHeap),
		delayed:     delayed,
		idempotency: make(map[string]string),
	}
}

func (s *Store) Enqueue(_ context.Context, job *model.Job) (*model.Job, error) {
	if job == nil {
		return nil, fmt.Errorf("%w: job is required", errors.ErrValidation)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if job.IdempotencyKey != "" {
		key := idempotencyKey(job.Queue, job.IdempotencyKey)
		if existingID, ok := s.idempotency[key]; ok {
			if existingJob := s.jobs[existingID]; existingJob != nil {
				return existingJob.Clone(), nil
			}
			delete(s.idempotency, key)
		}
	}

	if _, exists := s.jobs[job.ID]; exists {
		return nil, fmt.Errorf("%w: job %s already exists", errors.ErrConflict, job.ID)
	}

	stored := job.Clone()
	s.jobs[stored.ID] = stored
	if stored.IdempotencyKey != "" {
		s.idempotency[idempotencyKey(stored.Queue, stored.IdempotencyKey)] = stored.ID
	}

	switch stored.Status {
	case model.StatusScheduled, model.StatusRetrying:
		s.pushDelayedLocked(stored)
	default:
		stored.Status = model.StatusQueued
		s.pushReadyLocked(stored)
	}

	return stored.Clone(), nil
}

func (s *Store) GetJob(_ context.Context, jobID string) (*model.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("%w: job %s", errors.ErrNotFound, jobID)
	}

	return job.Clone(), nil
}

func (s *Store) FindByIdempotencyKey(_ context.Context, queueName, key string) (*model.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobID, ok := s.idempotency[idempotencyKey(queueName, key)]
	if !ok {
		return nil, fmt.Errorf("%w: idempotency key %s", errors.ErrNotFound, key)
	}

	job, ok := s.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("%w: job %s", errors.ErrNotFound, jobID)
	}

	return job.Clone(), nil
}

func (s *Store) Dequeue(_ context.Context, queueName string, now time.Time) (*model.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.promoteLocked(now)

	readyQueue := s.ready[queueName]
	if readyQueue == nil || readyQueue.Len() == 0 {
		return nil, fmt.Errorf("%w: %s", errors.ErrQueueEmpty, queueName)
	}

	for readyQueue.Len() > 0 {
		item := heap.Pop(readyQueue).(readyItem)
		job := s.jobs[item.jobID]
		if job == nil || job.Status != model.StatusQueued {
			continue
		}

		job.Status = model.StatusProcessing
		job.Attempts++
		job.UpdatedAt = now.UTC()
		return job.Clone(), nil
	}

	return nil, fmt.Errorf("%w: %s", errors.ErrQueueEmpty, queueName)
}

func (s *Store) Ack(_ context.Context, jobID string, now time.Time) (*model.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("%w: job %s", errors.ErrNotFound, jobID)
	}
	if job.Status != model.StatusProcessing {
		return nil, fmt.Errorf("%w: job %s is %s", errors.ErrInvalidState, jobID, job.Status)
	}

	job.Status = model.StatusSucceeded
	job.LastError = ""
	job.UpdatedAt = now.UTC()

	return job.Clone(), nil
}

func (s *Store) Retry(_ context.Context, jobID string, nextRunAt time.Time, lastError string, now time.Time) (*model.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("%w: job %s", errors.ErrNotFound, jobID)
	}
	if job.Status != model.StatusProcessing {
		return nil, fmt.Errorf("%w: job %s is %s", errors.ErrInvalidState, jobID, job.Status)
	}

	job.LastError = strings.TrimSpace(lastError)
	job.RunAt = nextRunAt.UTC()
	job.UpdatedAt = now.UTC()
	if job.RunAt.After(now) {
		job.Status = model.StatusRetrying
		s.pushDelayedLocked(job)
	} else {
		job.Status = model.StatusQueued
		s.pushReadyLocked(job)
	}

	return job.Clone(), nil
}

func (s *Store) MoveToDLQ(_ context.Context, jobID string, reason string, now time.Time) (*model.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("%w: job %s", errors.ErrNotFound, jobID)
	}
	if job.IsTerminal() {
		return nil, fmt.Errorf("%w: job %s is %s", errors.ErrInvalidState, jobID, job.Status)
	}

	job.Status = model.StatusDeadLettered
	job.LastError = strings.TrimSpace(reason)
	job.UpdatedAt = now.UTC()

	return job.Clone(), nil
}

func (s *Store) RequeueFromDLQ(_ context.Context, jobID string, runAt time.Time, now time.Time) (*model.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("%w: job %s", errors.ErrNotFound, jobID)
	}
	if job.Status != model.StatusDeadLettered {
		return nil, fmt.Errorf("%w: job %s is %s", errors.ErrInvalidState, jobID, job.Status)
	}

	job.Attempts = 0
	job.LastError = ""
	job.RunAt = runAt.UTC()
	job.UpdatedAt = now.UTC()
	if job.RunAt.After(now) {
		job.Status = model.StatusScheduled
		s.pushDelayedLocked(job)
	} else {
		job.Status = model.StatusQueued
		s.pushReadyLocked(job)
	}

	return job.Clone(), nil
}

func (s *Store) PromoteScheduled(_ context.Context, now time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.promoteLocked(now.UTC()), nil
}

func (s *Store) Stats(_ context.Context, queueName string) (model.QueueStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := model.QueueStats{Name: queueName}
	for _, job := range s.jobs {
		if job.Queue != queueName {
			continue
		}

		stats.Total++
		switch job.Status {
		case model.StatusQueued:
			stats.Ready++
		case model.StatusScheduled:
			stats.Scheduled++
		case model.StatusRetrying:
			stats.Retrying++
		case model.StatusProcessing:
			stats.Processing++
		case model.StatusSucceeded:
			stats.Succeeded++
		case model.StatusDeadLettered:
			stats.DeadLettered++
		}
	}

	if stats.Total == 0 {
		return model.QueueStats{}, fmt.Errorf("%w: queue %s", errors.ErrNotFound, queueName)
	}

	return stats, nil
}

func (s *Store) ListQueues(_ context.Context) ([]model.QueueStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	byQueue := make(map[string]*model.QueueStats)
	for _, job := range s.jobs {
		queueStats, ok := byQueue[job.Queue]
		if !ok {
			queueStats = &model.QueueStats{Name: job.Queue}
			byQueue[job.Queue] = queueStats
		}

		queueStats.Total++
		switch job.Status {
		case model.StatusQueued:
			queueStats.Ready++
		case model.StatusScheduled:
			queueStats.Scheduled++
		case model.StatusRetrying:
			queueStats.Retrying++
		case model.StatusProcessing:
			queueStats.Processing++
		case model.StatusSucceeded:
			queueStats.Succeeded++
		case model.StatusDeadLettered:
			queueStats.DeadLettered++
		}
	}

	names := make([]string, 0, len(byQueue))
	for name := range byQueue {
		names = append(names, name)
	}
	sort.Strings(names)

	queues := make([]model.QueueStats, 0, len(names))
	for _, name := range names {
		queues = append(queues, *byQueue[name])
	}

	return queues, nil
}

func (s *Store) ListDLQ(_ context.Context) ([]*model.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*model.Job, 0)
	for _, job := range s.jobs {
		if job.Status == model.StatusDeadLettered {
			jobs = append(jobs, job.Clone())
		}
	}

	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].UpdatedAt.Equal(jobs[j].UpdatedAt) {
			return jobs[i].ID < jobs[j].ID
		}
		return jobs[i].UpdatedAt.After(jobs[j].UpdatedAt)
	})

	return jobs, nil
}

func (s *Store) promoteLocked(now time.Time) int {
	count := 0
	for s.delayed.Len() > 0 {
		next := s.delayed[0]
		if next.runAt.After(now) {
			break
		}

		item := heap.Pop(&s.delayed).(delayedItem)
		job := s.jobs[item.jobID]
		if job == nil {
			continue
		}
		if job.Status != model.StatusScheduled && job.Status != model.StatusRetrying {
			continue
		}
		if job.RunAt.After(now) {
			continue
		}

		job.Status = model.StatusQueued
		job.UpdatedAt = now.UTC()
		s.pushReadyLocked(job)
		count++
	}

	return count
}

func (s *Store) pushReadyLocked(job *model.Job) {
	queue := s.readyQueueLocked(job.Queue)
	heap.Push(queue, readyItem{
		jobID:    job.ID,
		priority: job.Priority,
		sequence: s.nextSequenceLocked(),
	})
}

func (s *Store) pushDelayedLocked(job *model.Job) {
	heap.Push(&s.delayed, delayedItem{
		jobID:    job.ID,
		runAt:    job.RunAt.UTC(),
		priority: job.Priority,
		sequence: s.nextSequenceLocked(),
	})
}

func (s *Store) readyQueueLocked(queueName string) *readyHeap {
	queue, ok := s.ready[queueName]
	if ok {
		return queue
	}

	queue = &readyHeap{}
	heap.Init(queue)
	s.ready[queueName] = queue
	return queue
}

func (s *Store) nextSequenceLocked() int64 {
	s.sequence++
	return s.sequence
}

func idempotencyKey(queueName, key string) string {
	return queueName + "\x00" + key
}
