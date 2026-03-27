package handlers

import (
	"net/http"
	"strings"

	"github.com/william-trann/goflow/internal/model"
)

type queuesResponse struct {
	Queues []model.QueueStats `json:"queues"`
	System model.SystemStats  `json:"system"`
}

func (a *API) handleQueues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	queues, err := a.app.Metrics.Queues(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}

	systemStats, err := a.app.Metrics.System(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, queuesResponse{
		Queues: queues,
		System: systemStats,
	})
}

func (a *API) handleQueueByName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "queues" || parts[2] != "stats" {
		http.NotFound(w, r)
		return
	}

	stats, err := a.app.Metrics.Queue(r.Context(), parts[1])
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, stats)
}
