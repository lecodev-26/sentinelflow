package audit

import (
"encoding/json"
"net/http"
"strconv"
"time"
)

// Handler maneja las peticiones de auditoría
type Handler struct {
logger *Logger
}

// NewHandler crea un nuevo handler de auditoría
func NewHandler(logger *Logger) *Handler {
return &Handler{logger: logger}
}

// GetLogs devuelve logs de auditoría
func (h *Handler) GetLogs(w http.ResponseWriter, r *http.Request) {
// Construir filtro desde query params
filter := Filter{
TenantID:  r.URL.Query().Get("tenant"),
UserID:    r.URL.Query().Get("user"),
RequestID: r.URL.Query().Get("request_id"),
Method:    r.URL.Query().Get("method"),
Path:      r.URL.Query().Get("path"),
Provider:  r.URL.Query().Get("provider"),
Model:     r.URL.Query().Get("model"),
}

if status := r.URL.Query().Get("status"); status != "" {
if s, err := strconv.Atoi(status); err == nil {
filter.Status = s
}
}

if level := r.URL.Query().Get("level"); level != "" {
filter.Level = Level(level)
}

if from := r.URL.Query().Get("from"); from != "" {
if t, err := time.Parse(time.RFC3339, from); err == nil {
filter.From = t
}
}

if to := r.URL.Query().Get("to"); to != "" {
if t, err := time.Parse(time.RFC3339, to); err == nil {
filter.To = t
}
}

if limit := r.URL.Query().Get("limit"); limit != "" {
if l, err := strconv.Atoi(limit); err == nil {
filter.Limit = l
}
}

entries := h.logger.Query(filter)

if filter.Limit > 0 && len(entries) > filter.Limit {
entries = entries[:filter.Limit]
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]interface{}{
"total":   len(entries),
"entries": entries,
})
}
