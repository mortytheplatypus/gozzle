package server

import (
	"encoding/json"
	"net/http"

	"github.com/mortytheplatypus/gozzle/internal/config"
	"github.com/mortytheplatypus/gozzle/internal/log"
)

type Server struct {
	http    *http.Server
	store   *log.Store
}

func New(cfg *config.Config) (*Server, error) {

	store, err := log.NewStore(cfg.WALDir, cfg.ServicesFile)
	if err != nil {
		return nil, err
	}

	logService := log.NewLogService(store)
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
		store: store,
	}, nil

}

func (s *Server) ListenAndServe() error {
	return s.http.ListenAndServe()
}

func (s *Server) Close() error {
	return s.store.Close()
}

func health(w http.ResponseWriter, _ *http.Request) {

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	w.WriteHeader(http.StatusOK)
}
