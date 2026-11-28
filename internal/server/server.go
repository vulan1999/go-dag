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

func New(addr string, pipelineRepo pipeline.Repository) *Server {
	readHeaderTimeout, _ := strconv.Atoi(GetEnvValue("READ_HEADER_TIMEOUT", "20"))
	writeTimeout, _ := strconv.Atoi(GetEnvValue("WRITE_TIMEOUT", "20"))

	if readHeaderTimeout <= 0 || writeTimeout <= 0 {
		log.Fatal("Invalid Read Header Timeout or Write Timeout, it must be a positive integer value and greater than 0")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /{$}", homeHandler)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Pipeline API & UI group
	pipelineHandler := pipeline.NewHandler(pipelineRepo)
	mux.HandleFunc("GET /pipelines", pipelineHandler.List)
	mux.HandleFunc("POST /pipelines", pipelineHandler.Create)
	mux.HandleFunc("GET /pipelines/{id}", pipelineHandler.GetByID)
	mux.HandleFunc("PUT /pipelines/{id}", pipelineHandler.Update)
	mux.HandleFunc("DELETE /pipelines/{id}", pipelineHandler.Delete)
	mux.HandleFunc("GET /pipelines/{id}/dag", pipelineHandler.GetDAG)
	mux.HandleFunc("GET /stages/sql-columns", pipelineHandler.GetSQLColumns)
	mux.HandleFunc("GET /pipelines/{id}/stages/config-fields", pipelineHandler.GetStageConfigFields)
	mux.HandleFunc("POST /pipelines/{id}/nodes", pipelineHandler.AddNode)
	mux.HandleFunc("GET /pipelines/{id}/nodes/{nodeId}", pipelineHandler.GetNode)
	mux.HandleFunc("PUT /pipelines/{id}/nodes/{nodeId}", pipelineHandler.UpdateNode)
	mux.HandleFunc("DELETE /pipelines/{id}/nodes/{nodeId}", pipelineHandler.DeleteNode)
	mux.HandleFunc("POST /pipelines/{id}/edges", pipelineHandler.AddEdge)
	mux.HandleFunc("DELETE /pipelines/{id}/edges", pipelineHandler.DeleteEdge)

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
