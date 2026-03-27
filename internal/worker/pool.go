package worker

import (
	"context"
	stdErrors "errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/william-trann/goflow/internal/errors"
	"github.com/william-trann/goflow/internal/model"
	"github.com/william-trann/goflow/internal/queue"
)

type Pool struct {
	service      *queue.Service
	registry     *Registry
	queues       []string
	concurrency  int
	pollInterval time.Duration
	onError      func(error)
	baseCtx      context.Context
	stopCh       chan struct{}
	wg           sync.WaitGroup
	rr           atomic.Uint64
	startOnce    sync.Once
	stopOnce     sync.Once
}

func NewPool(service *queue.Service, registry *Registry, queues []string, concurrency int, pollInterval time.Duration) *Pool {
	if concurrency <= 0 {
		concurrency = 1
	}
	if pollInterval <= 0 {
		pollInterval = 100 * time.Millisecond
	}
	if registry == nil {
		registry = NewRegistry(NopHandler{})
	}

	return &Pool{
		service:      service,
		registry:     registry,
		queues:       normalizeQueues(queues),
		concurrency:  concurrency,
		pollInterval: pollInterval,
		onError:      func(error) {},
		stopCh:       make(chan struct{}),
	}
}

func (p *Pool) SetErrorHandler(fn func(error)) {
	if fn == nil {
		return
	}
	p.onError = fn
}

func (p *Pool) Start(ctx context.Context) {
	p.startOnce.Do(func() {
		if ctx == nil {
			ctx = context.Background()
		}
		p.baseCtx = ctx
		for i := 0; i < p.concurrency; i++ {
			p.wg.Add(1)
			go p.run()
		}
	})
}

func (p *Pool) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	p.stopOnce.Do(func() {
		close(p.stopCh)
	})

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Pool) run() {
	defer p.wg.Done()

	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-p.baseCtx.Done():
			return
		case <-timer.C:
		}

		job, err := p.dequeue()
		if err != nil {
			p.report(err)
			resetTimer(timer, p.pollInterval)
			continue
		}
		if job == nil {
			resetTimer(timer, p.pollInterval)
			continue
		}

		p.process(job)
		resetTimer(timer, 0)
	}
}

func (p *Pool) dequeue() (*model.Job, error) {
	start := 0
	if len(p.queues) > 1 {
		start = int(p.rr.Add(1)-1) % len(p.queues)
	}

	for offset := 0; offset < len(p.queues); offset++ {
		queueName := p.queues[(start+offset)%len(p.queues)]
		job, err := p.service.Dequeue(context.Background(), queueName)
		if err == nil {
			return job, nil
		}
		if stdErrors.Is(err, errors.ErrQueueEmpty) {
			continue
		}
		return nil, err
	}

	return nil, nil
}

func (p *Pool) process(job *model.Job) {
	handler := p.registry.Resolve(job.Queue)
	if err := handler.Handle(p.baseCtx, job); err != nil {
		if _, nackErr := p.service.Nack(context.Background(), job.ID, err); nackErr != nil {
			p.report(nackErr)
		}
		return
	}

	if _, err := p.service.Ack(context.Background(), job.ID); err != nil {
		p.report(err)
	}
}

func (p *Pool) report(err error) {
	if err != nil {
		p.onError(err)
	}
}

func normalizeQueues(queues []string) []string {
	seen := make(map[string]struct{})
	normalized := make([]string, 0, len(queues))
	for _, queueName := range queues {
		queueName = strings.TrimSpace(queueName)
		if queueName == "" {
			continue
		}
		if _, ok := seen[queueName]; ok {
			continue
		}
		seen[queueName] = struct{}{}
		normalized = append(normalized, queueName)
	}
	if len(normalized) == 0 {
		return []string{"default"}
	}
	return normalized
}

func resetTimer(timer *time.Timer, d time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(d)
}
