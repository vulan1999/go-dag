package server

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/vulan1999/go-dag/internal/pipeline"
)

type Server struct {
	httpServer *http.Server
}

func New(addr string) *Server {
	readHeaderTimeout, _ := strconv.Atoi(GetEnvValue("READ_HEADER_TIMEOUT", "20"))
	writeTimeout, _ := strconv.Atoi(GetEnvValue("WRITE_TIMEOUT", "20"))

	if readHeaderTimeout <= 0 || writeTimeout <= 0 {
		log.Fatal("Invalid Read Header Timeout or Write Timeout, it must be a positive integer value and greater than 0")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /{$}", homeHandler)

	// Pipeline API group
	mux.HandleFunc("POST /pipelines", pipeline.CreateHandler)

	return &Server{
		&http.Server{
			Addr:              addr,
			Handler:           RequestLogger(mux),
			ReadHeaderTimeout: time.Duration(readHeaderTimeout) * time.Second,
			WriteTimeout:      time.Duration(writeTimeout) * time.Second,
		},
	}
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
