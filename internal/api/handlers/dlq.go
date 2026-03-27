package handlers

import (
	"net/http"
	"strings"
	"time"
)

type requeueRequest struct {
	RunAt *time.Time `json:"run_at,omitempty"`
}

func (a *API) handleDLQ(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	jobs, err := a.app.DLQ.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, jobs)
}

func (a *API) handleDLQByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "dlq" || parts[2] != "requeue" {
		http.NotFound(w, r)
		return
	}

	var req requeueRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	job, err := a.app.DLQ.Requeue(r.Context(), parts[1], req.RunAt)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, job)
}
