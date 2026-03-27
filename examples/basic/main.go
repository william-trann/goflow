package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/william-trann/goflow/internal/clock"
	"github.com/william-trann/goflow/internal/model"
	"github.com/william-trann/goflow/internal/queue"
	"github.com/william-trann/goflow/internal/retry"
	"github.com/william-trann/goflow/internal/scheduler"
	"github.com/william-trann/goflow/internal/service"
	"github.com/william-trann/goflow/internal/storage/memory"
	"github.com/william-trann/goflow/internal/worker"
)

func main() {
	store := memory.New()
	app := service.NewApp(store, retry.NewExponentialBackoff(time.Second, 8*time.Second), clock.SystemClock{})

	processed := make(chan string, 2)
	registry := worker.NewRegistry(nil)
	registry.Register("emails", worker.HandlerFunc(func(_ context.Context, job *model.Job) error {
		var payload map[string]string
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return err
		}
		processed <- payload["to"]
		return nil
	}))

	schedulerLoop := scheduler.New(app.Queue, 25*time.Millisecond)
	pool := worker.NewPool(app.Queue, registry, []string{"emails"}, 2, 10*time.Millisecond)

	baseCtx := context.Background()
	schedulerLoop.Start(baseCtx)
	pool.Start(baseCtx)
	defer pool.Shutdown(context.Background())
	defer schedulerLoop.Shutdown(context.Background())

	_, _ = app.Queue.Enqueue(baseCtx, queueRequest("emails", `{"to":"ops@forgeq.dev"}`))
	_, _ = app.Queue.Enqueue(baseCtx, queueRequest("emails", `{"to":"infra@forgeq.dev"}`))

	fmt.Println(<-processed)
	fmt.Println(<-processed)
}

func queueRequest(queueName string, payload string) queue.EnqueueRequest {
	return queue.EnqueueRequest{
		Queue:      queueName,
		Payload:    json.RawMessage(payload),
		MaxRetries: 3,
	}
}
