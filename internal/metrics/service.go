package metrics

import (
	"context"

	"github.com/william-trann/goflow/internal/model"
	"github.com/william-trann/goflow/internal/storage"
)

type Service struct {
	store storage.Backend
}

func NewService(store storage.Backend) *Service {
	return &Service{store: store}
}

func (s *Service) Queue(ctx context.Context, queueName string) (model.QueueStats, error) {
	return s.store.Stats(ctx, queueName)
}

func (s *Service) Queues(ctx context.Context) ([]model.QueueStats, error) {
	return s.store.ListQueues(ctx)
}

func (s *Service) System(ctx context.Context) (model.SystemStats, error) {
	queues, err := s.store.ListQueues(ctx)
	if err != nil {
		return model.SystemStats{}, err
	}

	stats := model.SystemStats{
		Queues: len(queues),
	}
	for _, queueStats := range queues {
		stats.Ready += queueStats.Ready
		stats.Scheduled += queueStats.Scheduled
		stats.Retrying += queueStats.Retrying
		stats.Processing += queueStats.Processing
		stats.Succeeded += queueStats.Succeeded
		stats.DeadLettered += queueStats.DeadLettered
		stats.Total += queueStats.Total
	}

	return stats, nil
}
