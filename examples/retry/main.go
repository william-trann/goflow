package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
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
	app := service.NewApp(store, retry.NewExponentialBackoff(50*time.Millisecond, 200*time.Millisecond), clock.SystemClock{})

	var attempts atomic.Int32
	var once sync.Once
	done := make(chan struct{})

	registry := worker.NewRegistry(nil)
	registry.Register("payments", worker.HandlerFunc(func(_ context.Context, job *model.Job) error {
		current := attempts.Add(1)
		if current < 3 {
			return fmt.Errorf("transient payment error on attempt %d", current)
		}
		once.Do(func() {
			close(done)
		})
		return nil
	}))

	schedulerLoop := scheduler.New(app.Queue, 10*time.Millisecond)
	pool := worker.NewPool(app.Queue, registry, []string{"payments"}, 1, 10*time.Millisecond)
	schedulerLoop.Start(context.Background())
	pool.Start(context.Background())
	defer pool.Shutdown(context.Background())
	defer schedulerLoop.Shutdown(context.Background())

	job, _ := app.Queue.Enqueue(context.Background(), queue.EnqueueRequest{
		Queue:      "payments",
		Payload:    json.RawMessage(`{"order_id":"ord_123"}`),
		MaxRetries: 3,
	})

	<-done

	finalJob, _ := app.Queue.GetJob(context.Background(), job.ID)
	fmt.Printf("attempts=%d status=%s\n", finalJob.Attempts, finalJob.Status)
}
