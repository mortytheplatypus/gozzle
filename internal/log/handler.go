package log

import (
	"encoding/json"
	"net/http"
	"strconv"
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

func (h *Handler) Read(w http.ResponseWriter, r *http.Request) {

	serviceID := r.URL.Query().Get("service_id")
	if serviceID == "" {
		http.Error(w, "Missing service_id", http.StatusBadRequest)
		return
	}

	offset := int64(0)
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			http.Error(w, "Bad offset", http.StatusBadRequest)
			return
		}
		offset = n
	}

	entries, err := h.service.Read(serviceID, offset)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)

}
