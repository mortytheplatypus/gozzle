package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/mortytheplatypus/gozzle/internal/config"
	"github.com/mortytheplatypus/gozzle/internal/log"
)

type Server struct {
	http    *http.Server
	service *log.Service
}

func New(cfg *config.Config) (*Server, error) {

	store, err := log.NewStore(cfg.WALDir, cfg.ServicesFile)
	if err != nil {
		return nil, err
	}

	logService := log.NewLogService(store, cfg.IngestQueue)
	logHandler := log.NewLogHandler(logService)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", health)
	mux.HandleFunc("POST /api/services", logHandler.Register)
	mux.HandleFunc("POST /api/logs", logHandler.Append)
	mux.HandleFunc("GET /api/logs", logHandler.Read)

	return &Server{
		http: &http.Server{
			Addr:    cfg.ListenAddr,
			Handler: mux,
		},
		service: logService,
	}, nil

}

func (s *Server) ListenAndServe() error {
	return s.http.ListenAndServe()
}

// Shutdown stops accepting connections and waits for in-flight requests to finish.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

// Close drains ingest workers and closes WAL file descriptors.
func (s *Server) Close() error {
	return s.service.Close()
}

func health(w http.ResponseWriter, _ *http.Request) {

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	w.WriteHeader(http.StatusOK)
}
