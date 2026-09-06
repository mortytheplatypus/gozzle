package log

import (
	"encoding/json"
	"errors"
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

type RegisterServiceRequest struct {
	ServiceID string `json:"service_id"`
}

func NewLogHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidServiceID):
		http.Error(w, "Invalid service_id", http.StatusBadRequest)
	case errors.Is(err, ErrNotRegistered):
		http.Error(w, "Service not registered", http.StatusNotFound)
	case errors.Is(err, ErrBusy):
		http.Error(w, "Ingestion queue full", http.StatusServiceUnavailable)
	default:
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {

	var req RegisterServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request body", http.StatusBadRequest)
		return
	}

	if err := h.service.Register(req.ServiceID); err != nil {
		writeStoreError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Append(w http.ResponseWriter, r *http.Request) {

	var req AppendLogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request body", http.StatusBadRequest)
		return
	}

	if err := h.service.Append(req); err != nil {
		writeStoreError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
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
		writeStoreError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)

}
