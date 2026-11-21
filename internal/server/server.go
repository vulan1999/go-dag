package server

import (
	"context"
	"net/http"
	"time"
)

type Server struct {
	httpServer *http.Server
}

func New(addr string) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)

	serverAddr := ":" + addr
	return &Server{
		&http.Server{
			Addr:              serverAddr,
			Handler:           mux,
			ReadHeaderTimeout: 20 * time.Second,
			WriteTimeout:      20 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
