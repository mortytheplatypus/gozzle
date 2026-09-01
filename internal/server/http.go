package server

import (
	"net/http"

	"github.com/mortytheplatypus/gozzle/internal/config"
)

type Server struct {
	http *http.Server
}

func New(cfg *config.Config) *Server {

	handler := http.NewServeMux()
	handler.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return &Server{
		http: &http.Server{
			Addr:    cfg.ListenAddr,
			Handler: handler,
		},
	}
}

func (s *Server) ListenAndServe() error {
	return s.http.ListenAndServe()
}
