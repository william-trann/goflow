package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/william-trann/goflow/internal/queue"
)

type enqueueRequest struct {
	Queue          string          `json:"queue"`
	Payload        json.RawMessage `json:"payload"`
	Priority       int             `json:"priority"`
	MaxRetries     int             `json:"max_retries"`
	RunAt          *time.Time      `json:"run_at,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
}

func (a *API) handleJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	var req enqueueRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	job, err := a.app.Queue.Enqueue(r.Context(), queue.EnqueueRequest{
		Queue:          req.Queue,
		Payload:        req.Payload,
		Priority:       req.Priority,
		MaxRetries:     req.MaxRetries,
		RunAt:          req.RunAt,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, job)
}

func (a *API) handleJobByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	jobID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/jobs/"), "/")
	if jobID == "" || strings.Contains(jobID, "/") {
		http.NotFound(w, r)
		return
	}

	job, err := a.app.Queue.GetJob(r.Context(), jobID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, job)
}
