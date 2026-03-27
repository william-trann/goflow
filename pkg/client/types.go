package client

import (
	"encoding/json"
	"time"
)

type Job struct {
	ID             string          `json:"id"`
	Queue          string          `json:"queue"`
	Payload        json.RawMessage `json:"payload"`
	Priority       int             `json:"priority"`
	Attempts       int             `json:"attempts"`
	MaxRetries     int             `json:"max_retries"`
	RunAt          time.Time       `json:"run_at"`
	Status         string          `json:"status"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	LastError      string          `json:"last_error,omitempty"`
}

type EnqueueRequest struct {
	Queue          string          `json:"queue"`
	Payload        json.RawMessage `json:"payload"`
	Priority       int             `json:"priority"`
	MaxRetries     int             `json:"max_retries"`
	RunAt          *time.Time      `json:"run_at,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
}

type QueueStats struct {
	Name         string `json:"name"`
	Ready        int    `json:"ready"`
	Scheduled    int    `json:"scheduled"`
	Retrying     int    `json:"retrying"`
	Processing   int    `json:"processing"`
	Succeeded    int    `json:"succeeded"`
	DeadLettered int    `json:"dead_lettered"`
	Total        int    `json:"total"`
}

type SystemStats struct {
	Queues       int `json:"queues"`
	Ready        int `json:"ready"`
	Scheduled    int `json:"scheduled"`
	Retrying     int `json:"retrying"`
	Processing   int `json:"processing"`
	Succeeded    int `json:"succeeded"`
	DeadLettered int `json:"dead_lettered"`
	Total        int `json:"total"`
}

type QueuesResponse struct {
	Queues []QueueStats `json:"queues"`
	System SystemStats  `json:"system"`
}

type RequeueRequest struct {
	RunAt *time.Time `json:"run_at,omitempty"`
}
