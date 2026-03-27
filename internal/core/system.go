package core

import (
	"context"
	stdErrors "errors"

	"github.com/william-trann/goflow/internal/api/handlers"
	apihttp "github.com/william-trann/goflow/internal/api/http"
	"github.com/william-trann/goflow/internal/clock"
	"github.com/william-trann/goflow/internal/config"
	"github.com/william-trann/goflow/internal/retry"
	"github.com/william-trann/goflow/internal/scheduler"
	"github.com/william-trann/goflow/internal/service"
	"github.com/william-trann/goflow/internal/storage"
	"github.com/william-trann/goflow/internal/storage/memory"
	"github.com/william-trann/goflow/internal/worker"
)

type System struct {
	Config    config.Config
	Store     storage.Backend
	App       *service.App
	Registry  *worker.Registry
	Workers   *worker.Pool
	Scheduler *scheduler.Loop
	API       *apihttp.Server
}

type Runtime struct {
	api       *apihttp.Server
	scheduler *scheduler.Loop
	workers   *worker.Pool
}

func NewInMemorySystem(cfg config.Config) *System {
	store := memory.New()
	systemClock := clock.SystemClock{}
	policy := retry.NewExponentialBackoff(cfg.Retry.BaseDelay, cfg.Retry.MaxDelay)
	app := service.NewApp(store, policy, systemClock)
	registry := worker.NewRegistry(worker.NopHandler{})
	workers := worker.NewPool(app.Queue, registry, cfg.Worker.Queues, cfg.Worker.Concurrency, cfg.Worker.PollInterval)
	schedule := scheduler.New(app.Queue, cfg.Scheduler.Interval)
	api := apihttp.New(cfg.API.Addr, handlers.New(app))

	return &System{
		Config:    cfg,
		Store:     store,
		App:       app,
		Registry:  registry,
		Workers:   workers,
		Scheduler: schedule,
		API:       api,
	}
}

func (s *System) Register(queueName string, handler worker.Handler) {
	s.Registry.Register(queueName, handler)
}

func NewCombinedRuntime(system *System) *Runtime {
	return &Runtime{
		api:       system.API,
		scheduler: system.Scheduler,
		workers:   system.Workers,
	}
}

func NewAPIRuntime(system *System) *Runtime {
	return &Runtime{
		api:       system.API,
		scheduler: system.Scheduler,
	}
}

func NewWorkerRuntime(system *System) *Runtime {
	return &Runtime{
		scheduler: system.Scheduler,
		workers:   system.Workers,
	}
}

func (r *Runtime) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if r.scheduler != nil {
		r.scheduler.Start(ctx)
	}
	if r.workers != nil {
		r.workers.Start(ctx)
	}
	if r.api != nil {
		return r.api.Start()
	}

	<-ctx.Done()
	return nil
}

func (r *Runtime) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	var errs []error
	if r.api != nil {
		errs = appendIf(errs, r.api.Shutdown(ctx))
	}
	if r.scheduler != nil {
		errs = appendIf(errs, r.scheduler.Shutdown(ctx))
	}
	if r.workers != nil {
		errs = appendIf(errs, r.workers.Shutdown(ctx))
	}
	return stdErrors.Join(errs...)
}

func appendIf(errs []error, err error) []error {
	if err != nil {
		return append(errs, err)
	}
	return errs
}
