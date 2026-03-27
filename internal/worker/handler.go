package worker

import (
	"context"

	"github.com/william-trann/goflow/internal/model"
)

type Handler interface {
	Handle(context.Context, *model.Job) error
}

type HandlerFunc func(context.Context, *model.Job) error

func (f HandlerFunc) Handle(ctx context.Context, job *model.Job) error {
	return f(ctx, job)
}

type NopHandler struct{}

func (NopHandler) Handle(context.Context, *model.Job) error {
	return nil
}
