package handlers

import (
	"net/http"

	"github.com/william-trann/goflow/internal/service"
)

type API struct {
	app *service.App
}

func New(app *service.App) *API {
	return &API{app: app}
}

func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", a.handleHealth)
	mux.HandleFunc("/jobs", a.handleJobs)
	mux.HandleFunc("/jobs/", a.handleJobByID)
	mux.HandleFunc("/queues", a.handleQueues)
	mux.HandleFunc("/queues/", a.handleQueueByName)
	mux.HandleFunc("/dlq", a.handleDLQ)
	mux.HandleFunc("/dlq/", a.handleDLQByID)
}

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
