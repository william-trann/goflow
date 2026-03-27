package queue_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/william-trann/goflow/internal/dlq"
	"github.com/william-trann/goflow/internal/idempotency"
	"github.com/william-trann/goflow/internal/model"
	"github.com/william-trann/goflow/internal/queue"
	"github.com/william-trann/goflow/internal/retry"
	"github.com/william-trann/goflow/internal/storage/memory"
)

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	return c.now
}

func TestServiceNackMovesJobToDLQAfterRetriesExhausted(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 3, 27, 9, 0, 0, 0, time.UTC)}
	store := memory.New()
	service := queue.NewService(store, retry.NewExponentialBackoff(time.Second, 8*time.Second), idempotency.NewService(store), clock)
	dlqService := dlq.NewService(store, clock)

	job, err := service.Enqueue(context.Background(), queue.EnqueueRequest{
		Queue:      "payments",
		Payload:    []byte(`{"order_id":"ord_1"}`),
		MaxRetries: 2,
	})
	if err != nil {
		t.Fatalf("enqueue job: %v", err)
	}

	for attempt := 0; attempt < 2; attempt++ {
		if _, err := service.Dequeue(context.Background(), "payments"); err != nil {
			t.Fatalf("dequeue job: %v", err)
		}
		if _, err := service.Nack(context.Background(), job.ID, errors.New("processor failure")); err != nil {
			t.Fatalf("nack job: %v", err)
		}
		clock.now = clock.now.Add(time.Second << attempt)
		if _, err := service.PromoteScheduled(context.Background()); err != nil {
			t.Fatalf("promote scheduled: %v", err)
		}
	}

	inFlight, err := service.Dequeue(context.Background(), "payments")
	if err != nil {
		t.Fatalf("dequeue final attempt: %v", err)
	}
	if inFlight.Attempts != 3 {
		t.Fatalf("expected third processing attempt, got %d", inFlight.Attempts)
	}

	deadLettered, err := service.Nack(context.Background(), job.ID, errors.New("permanent failure"))
	if err != nil {
		t.Fatalf("move to dlq: %v", err)
	}
	if deadLettered.Status != model.StatusDeadLettered {
		t.Fatalf("expected dead lettered status, got %s", deadLettered.Status)
	}

	items, err := dlqService.List(context.Background())
	if err != nil {
		t.Fatalf("list dlq: %v", err)
	}
	if len(items) != 1 || items[0].ID != job.ID {
		t.Fatalf("expected job in dlq, got %+v", items)
	}
}

func TestServiceEnqueueHonorsIdempotencyKey(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 3, 27, 9, 0, 0, 0, time.UTC)}
	store := memory.New()
	service := queue.NewService(store, retry.NewExponentialBackoff(time.Second, 8*time.Second), idempotency.NewService(store), clock)

	first, err := service.Enqueue(context.Background(), queue.EnqueueRequest{
		Queue:          "emails",
		Payload:        []byte(`{"to":"a@forgeq.dev"}`),
		MaxRetries:     1,
		IdempotencyKey: "welcome-email",
	})
	if err != nil {
		t.Fatalf("enqueue first job: %v", err)
	}

	second, err := service.Enqueue(context.Background(), queue.EnqueueRequest{
		Queue:          "emails",
		Payload:        []byte(`{"to":"b@forgeq.dev"}`),
		MaxRetries:     1,
		IdempotencyKey: "welcome-email",
	})
	if err != nil {
		t.Fatalf("enqueue second job: %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("expected idempotent enqueue to return same job, got %s and %s", first.ID, second.ID)
	}
	if string(second.Payload) != `{"to":"a@forgeq.dev"}` {
		t.Fatalf("expected original payload to be preserved, got %s", string(second.Payload))
	}
}
