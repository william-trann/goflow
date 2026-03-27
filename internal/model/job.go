package model

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"
)

type Status string

const (
	StatusQueued       Status = "queued"
	StatusScheduled    Status = "scheduled"
	StatusProcessing   Status = "processing"
	StatusRetrying     Status = "retrying"
	StatusSucceeded    Status = "succeeded"
	StatusDeadLettered Status = "dead_lettered"
)

type Job struct {
	ID             string          `json:"id"`
	Queue          string          `json:"queue"`
	Payload        json.RawMessage `json:"payload"`
	Priority       int             `json:"priority"`
	Attempts       int             `json:"attempts"`
	MaxRetries     int             `json:"max_retries"`
	RunAt          time.Time       `json:"run_at"`
	Status         Status          `json:"status"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	LastError      string          `json:"last_error,omitempty"`
}

func (j Job) Clone() *Job {
	clone := j
	if j.Payload != nil {
		clone.Payload = append(json.RawMessage(nil), j.Payload...)
	}
	return &clone
}

func (j Job) IsTerminal() bool {
	return j.Status == StatusSucceeded || j.Status == StatusDeadLettered
}

func NewJobID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}
