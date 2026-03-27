package retry_test

import (
	"testing"
	"time"

	"github.com/william-trann/goflow/internal/model"
	"github.com/william-trann/goflow/internal/retry"
)

func TestExponentialBackoffNext(t *testing.T) {
	policy := retry.NewExponentialBackoff(time.Second, 8*time.Second)
	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		attempts   int
		maxRetries int
		delay      time.Duration
		ok         bool
	}{
		{name: "first retry", attempts: 1, maxRetries: 3, delay: time.Second, ok: true},
		{name: "second retry", attempts: 2, maxRetries: 3, delay: 2 * time.Second, ok: true},
		{name: "third retry", attempts: 3, maxRetries: 4, delay: 4 * time.Second, ok: true},
		{name: "capped retry", attempts: 5, maxRetries: 8, delay: 8 * time.Second, ok: true},
		{name: "budget exhausted", attempts: 3, maxRetries: 2, delay: 0, ok: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			job := &model.Job{
				Attempts:   tc.attempts,
				MaxRetries: tc.maxRetries,
			}

			nextRunAt, ok := policy.Next(job, now)
			if ok != tc.ok {
				t.Fatalf("expected ok=%v, got %v", tc.ok, ok)
			}
			if !tc.ok {
				return
			}
			if got := nextRunAt.Sub(now); got != tc.delay {
				t.Fatalf("expected delay %s, got %s", tc.delay, got)
			}
		})
	}
}
