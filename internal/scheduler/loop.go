package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/william-trann/goflow/internal/queue"
)

type Loop struct {
	service   *queue.Service
	interval  time.Duration
	onError   func(error)
	baseCtx   context.Context
	stopCh    chan struct{}
	wg        sync.WaitGroup
	startOnce sync.Once
	stopOnce  sync.Once
}

func New(service *queue.Service, interval time.Duration) *Loop {
	if interval <= 0 {
		interval = 250 * time.Millisecond
	}

	return &Loop{
		service:  service,
		interval: interval,
		onError:  func(error) {},
		stopCh:   make(chan struct{}),
	}
}

func (l *Loop) SetErrorHandler(fn func(error)) {
	if fn == nil {
		return
	}
	l.onError = fn
}

func (l *Loop) Start(ctx context.Context) {
	l.startOnce.Do(func() {
		if ctx == nil {
			ctx = context.Background()
		}
		l.baseCtx = ctx
		l.wg.Add(1)
		go l.run()
	})
}

func (l *Loop) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	l.stopOnce.Do(func() {
		close(l.stopCh)
	})

	done := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *Loop) run() {
	defer l.wg.Done()

	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()

	for {
		select {
		case <-l.stopCh:
			return
		case <-l.baseCtx.Done():
			return
		case <-ticker.C:
			if _, err := l.service.PromoteScheduled(context.Background()); err != nil {
				l.onError(err)
			}
		}
	}
}
