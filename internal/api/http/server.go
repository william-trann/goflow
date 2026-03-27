package http

import (
	"context"
	stdErrors "errors"
	"net/http"
	"time"

	"github.com/william-trann/goflow/internal/api/handlers"
)

type Server struct {
	server *http.Server
}

func New(addr string, api *handlers.API) *Server {
	mux := http.NewServeMux()
	api.Register(mux)

	return &Server{
		server: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	err := s.server.ListenAndServe()
	if stdErrors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
