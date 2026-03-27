package service

import (
	"github.com/william-trann/goflow/internal/clock"
	"github.com/william-trann/goflow/internal/dlq"
	"github.com/william-trann/goflow/internal/idempotency"
	"github.com/william-trann/goflow/internal/metrics"
	"github.com/william-trann/goflow/internal/queue"
	"github.com/william-trann/goflow/internal/retry"
	"github.com/william-trann/goflow/internal/storage"
)

type App struct {
	Queue   *queue.Service
	DLQ     *dlq.Service
	Metrics *metrics.Service
}

func NewApp(store storage.Backend, policy retry.Policy, clock clock.Clock) *App {
	idem := idempotency.NewService(store)
	return &App{
		Queue:   queue.NewService(store, policy, idem, clock),
		DLQ:     dlq.NewService(store, clock),
		Metrics: metrics.NewService(store),
	}
}
