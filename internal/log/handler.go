package log

import (
	"encoding/json"
	"net/http"
)

type Handler struct {
	service *Service
}

type AppendLogRequest struct {
	ServiceID string `json:"service_id"`
	Message   string `json:"message"`
	Level     string `json:"level"`
}

func NewLogHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Append(w http.ResponseWriter, r *http.Request) {

	var req AppendLogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request body", http.StatusBadRequest)
	}

	if err := h.service.Append(req); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}

}
