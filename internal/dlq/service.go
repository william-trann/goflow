package dlq

import (
	"context"
	"time"

	"github.com/william-trann/goflow/internal/clock"
	"github.com/william-trann/goflow/internal/model"
	"github.com/william-trann/goflow/internal/storage"
)

type Service struct {
	store storage.Backend
	clock clock.Clock
}

func NewService(store storage.Backend, clock clock.Clock) *Service {
	return &Service{
		store: store,
		clock: clock,
	}
}

func (s *Service) List(ctx context.Context) ([]*model.Job, error) {
	return s.store.ListDLQ(ctx)
}

func (s *Service) Requeue(ctx context.Context, jobID string, runAt *time.Time) (*model.Job, error) {
	now := s.clock.Now().UTC()
	target := now
	if runAt != nil {
		target = runAt.UTC()
	}
	return s.store.RequeueFromDLQ(ctx, jobID, target, now)
}
