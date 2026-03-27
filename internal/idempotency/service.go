package idempotency

import (
	"context"
	stdErrors "errors"
	"strings"

	"github.com/william-trann/goflow/internal/errors"
	"github.com/william-trann/goflow/internal/model"
	"github.com/william-trann/goflow/internal/storage"
)

type Service struct {
	store storage.Backend
}

func NewService(store storage.Backend) *Service {
	return &Service{store: store}
}

func (s *Service) Lookup(ctx context.Context, queueName, key string) (*model.Job, error) {
	queueName = strings.TrimSpace(queueName)
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, nil
	}

	job, err := s.store.FindByIdempotencyKey(ctx, queueName, key)
	if err != nil {
		if stdErrors.Is(err, errors.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return job, nil
}
