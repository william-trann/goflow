package queue

import (
	"encoding/json"
	"time"
)

type EnqueueRequest struct {
	Queue          string
	Payload        json.RawMessage
	Priority       int
	MaxRetries     int
	RunAt          *time.Time
	IdempotencyKey string
}
