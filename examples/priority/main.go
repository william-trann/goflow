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
	"github.com/william-trann/goflow/internal/service"
	"github.com/william-trann/goflow/internal/storage/memory"
	"github.com/william-trann/goflow/internal/worker"
)

func main() {
	store := memory.New()
	app := service.NewApp(store, retry.NewExponentialBackoff(time.Second, 8*time.Second), clock.SystemClock{})

	_, _ = app.Queue.Enqueue(context.Background(), queue.EnqueueRequest{
		Queue:      "critical",
		Payload:    json.RawMessage(`{"name":"low"}`),
		Priority:   1,
		MaxRetries: 1,
	})
	_, _ = app.Queue.Enqueue(context.Background(), queue.EnqueueRequest{
		Queue:      "critical",
		Payload:    json.RawMessage(`{"name":"high"}`),
		Priority:   10,
		MaxRetries: 1,
	})

	order := make(chan string, 2)
	registry := worker.NewRegistry(nil)
	registry.Register("critical", worker.HandlerFunc(func(_ context.Context, job *model.Job) error {
		var payload map[string]string
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return err
		}
		order <- payload["name"]
		return nil
	}))

	pool := worker.NewPool(app.Queue, registry, []string{"critical"}, 1, 10*time.Millisecond)
	pool.Start(context.Background())
	defer pool.Shutdown(context.Background())

	fmt.Println(<-order)
	fmt.Println(<-order)
}
