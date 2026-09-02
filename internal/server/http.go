package server

import (
	"encoding/json"
	"net/http"

	"github.com/mortytheplatypus/gozzle/internal/config"
)

type Server struct {
	http *http.Server
}

func New(cfg *config.Config) *Server {

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", health)

	return &Server{
		http: &http.Server{
			Addr:    cfg.ListenAddr,
			Handler: mux,
		},
	}

}

func (s *Server) ListenAndServe() error {
	return s.http.ListenAndServe()
}

func health(w http.ResponseWriter, _ *http.Request) {

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	w.WriteHeader(http.StatusOK)
}
