package storage

import (
	"context"
	"time"

	"github.com/william-trann/goflow/internal/model"
)

type Backend interface {
	Enqueue(context.Context, *model.Job) (*model.Job, error)
	GetJob(context.Context, string) (*model.Job, error)
	FindByIdempotencyKey(context.Context, string, string) (*model.Job, error)
	Dequeue(context.Context, string, time.Time) (*model.Job, error)
	Ack(context.Context, string, time.Time) (*model.Job, error)
	Retry(context.Context, string, time.Time, string, time.Time) (*model.Job, error)
	MoveToDLQ(context.Context, string, string, time.Time) (*model.Job, error)
	RequeueFromDLQ(context.Context, string, time.Time, time.Time) (*model.Job, error)
	PromoteScheduled(context.Context, time.Time) (int, error)
	Stats(context.Context, string) (model.QueueStats, error)
	ListQueues(context.Context) ([]model.QueueStats, error)
	ListDLQ(context.Context) ([]*model.Job, error)
}
