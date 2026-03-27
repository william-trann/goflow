package retry

import (
	"time"

	"github.com/william-trann/goflow/internal/model"
)

type Policy interface {
	Next(*model.Job, time.Time) (time.Time, bool)
}

type ExponentialBackoff struct {
	BaseDelay time.Duration
	MaxDelay  time.Duration
}

func NewExponentialBackoff(baseDelay, maxDelay time.Duration) *ExponentialBackoff {
	if baseDelay <= 0 {
		baseDelay = time.Second
	}
	if maxDelay < baseDelay {
		maxDelay = baseDelay
	}
	return &ExponentialBackoff{
		BaseDelay: baseDelay,
		MaxDelay:  maxDelay,
	}
}

func (p *ExponentialBackoff) Next(job *model.Job, now time.Time) (time.Time, bool) {
	if job == nil || job.MaxRetries <= 0 {
		return time.Time{}, false
	}

	retriesUsed := job.Attempts - 1
	if retriesUsed < 0 {
		retriesUsed = 0
	}
	if retriesUsed >= job.MaxRetries {
		return time.Time{}, false
	}

	delay := p.BaseDelay
	for i := 0; i < retriesUsed; i++ {
		delay *= 2
		if delay >= p.MaxDelay {
			delay = p.MaxDelay
			break
		}
	}

	return now.UTC().Add(delay), true
}
