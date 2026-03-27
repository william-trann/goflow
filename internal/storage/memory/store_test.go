package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/william-trann/goflow/internal/model"
	"github.com/william-trann/goflow/internal/storage/memory"
)

func TestStoreDequeueRespectsPriority(t *testing.T) {
	store := memory.New()
	now := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)

	_, err := store.Enqueue(context.Background(), &model.Job{
		ID:         "low",
		Queue:      "critical",
		Payload:    []byte(`{"name":"low"}`),
		Priority:   1,
		RunAt:      now,
		Status:     model.StatusQueued,
		CreatedAt:  now,
		UpdatedAt:  now,
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("enqueue low: %v", err)
	}

	_, err = store.Enqueue(context.Background(), &model.Job{
		ID:         "high",
		Queue:      "critical",
		Payload:    []byte(`{"name":"high"}`),
		Priority:   9,
		RunAt:      now,
		Status:     model.StatusQueued,
		CreatedAt:  now,
		UpdatedAt:  now,
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("enqueue high: %v", err)
	}

	first, err := store.Dequeue(context.Background(), "critical", now)
	if err != nil {
		t.Fatalf("dequeue first: %v", err)
	}
	second, err := store.Dequeue(context.Background(), "critical", now)
	if err != nil {
		t.Fatalf("dequeue second: %v", err)
	}

	if first.ID != "high" {
		t.Fatalf("expected first job to be high, got %s", first.ID)
	}
	if second.ID != "low" {
		t.Fatalf("expected second job to be low, got %s", second.ID)
	}
}

func TestStorePromotesDelayedJobs(t *testing.T) {
	store := memory.New()
	now := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)

	_, err := store.Enqueue(context.Background(), &model.Job{
		ID:         "immediate",
		Queue:      "default",
		Payload:    []byte(`{}`),
		Priority:   1,
		RunAt:      now,
		Status:     model.StatusQueued,
		CreatedAt:  now,
		UpdatedAt:  now,
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("enqueue immediate: %v", err)
	}

	_, err = store.Enqueue(context.Background(), &model.Job{
		ID:         "delayed",
		Queue:      "default",
		Payload:    []byte(`{}`),
		Priority:   1,
		RunAt:      now.Add(2 * time.Second),
		Status:     model.StatusScheduled,
		CreatedAt:  now,
		UpdatedAt:  now,
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("enqueue delayed: %v", err)
	}

	first, err := store.Dequeue(context.Background(), "default", now)
	if err != nil {
		t.Fatalf("dequeue immediate: %v", err)
	}
	if first.ID != "immediate" {
		t.Fatalf("expected immediate job, got %s", first.ID)
	}

	if _, err := store.PromoteScheduled(context.Background(), now.Add(2*time.Second)); err != nil {
		t.Fatalf("promote scheduled: %v", err)
	}

	second, err := store.Dequeue(context.Background(), "default", now.Add(2*time.Second))
	if err != nil {
		t.Fatalf("dequeue delayed: %v", err)
	}
	if second.ID != "delayed" {
		t.Fatalf("expected delayed job, got %s", second.ID)
	}
}
