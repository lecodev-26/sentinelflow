package approvals

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/lecodev-26/sentinelflow/internal/gateway/v3/middleware"
	"github.com/lecodev-26/sentinelflow/internal/rbac"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

type createRequest struct {
	Action     string          `json:"action"`
	TargetType string          `json:"target_type"`
	TargetID   string          `json:"target_id"`
	Payload    json.RawMessage `json:"payload"`
	Reason     string          `json:"reason"`
	ExpiresIn  int             `json:"expires_in_seconds"`
}
type decisionRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) Register(r *mux.Router, a *middleware.Auth) {
	api := r.PathPrefix("/v1/approvals").Subrouter()
	api.Use(a.Handler)
	read := api.PathPrefix("").Subrouter()
	read.Use(middleware.RequireScope(rbac.ScopeReadApprovals))
	read.HandleFunc("", h.list).Methods("GET")
	read.HandleFunc("/{id}", h.get).Methods("GET")
	write := api.PathPrefix("").Subrouter()
	write.Use(middleware.RequireScope(rbac.ScopeWriteApprovals))
	write.HandleFunc("", h.create).Methods("POST")
	write.HandleFunc("/{id}/approve", h.approve).Methods("POST")
	write.HandleFunc("/{id}/reject", h.reject).Methods("POST")
	write.HandleFunc("/{id}/cancel", h.cancel).Methods("POST")
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit := 50
	if x := r.URL.Query().Get("limit"); x != "" {
		if n, e := strconv.Atoi(x); e == nil {
			limit = n
		}
	}
	rows, e := h.service.List(r.Context(), middleware.GetTenantID(r.Context()), middleware.GetEnvironment(r.Context()), status, limit)
	if e != nil {
		errJSON(w, http.StatusBadRequest, e.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": rows, "total": len(rows)})
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	row, e := h.service.Get(r.Context(), middleware.GetTenantID(r.Context()), middleware.GetEnvironment(r.Context()), mux.Vars(r)["id"])
	if e != nil {
		errJSON(w, http.StatusNotFound, "approval not found")
		return
	}
	writeJSON(w, http.StatusOK, row)
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var q createRequest
	if e := json.NewDecoder(r.Body).Decode(&q); e != nil {
		errJSON(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if strings.TrimSpace(q.Action) == "" {
		errJSON(w, http.StatusBadRequest, "action is required")
		return
	}
	var payload any
	if len(q.Payload) > 0 {
		if e := json.Unmarshal(q.Payload, &payload); e != nil {
			errJSON(w, http.StatusBadRequest, "payload must be valid JSON")
			return
		}
	}
	exp := 30 * time.Minute
	if q.ExpiresIn > 0 {
		exp = time.Duration(q.ExpiresIn) * time.Second
	}
	row, e := h.service.Create(r.Context(), CreateInput{TenantID: middleware.GetTenantID(r.Context()), ProjectID: middleware.GetProjectID(r.Context()), Environment: middleware.GetEnvironment(r.Context()), RequesterID: middleware.GetUserID(r.Context()), Action: q.Action, TargetType: q.TargetType, TargetID: q.TargetID, Payload: payload, Reason: q.Reason, ExpiresIn: exp})
	if e != nil {
		errJSON(w, http.StatusConflict, e.Error())
		return
	}
	writeJSON(w, http.StatusCreated, row)
}
func (h *Handler) approve(w http.ResponseWriter, r *http.Request) { h.decide(w, r, StatusApproved) }
func (h *Handler) reject(w http.ResponseWriter, r *http.Request)  { h.decide(w, r, StatusRejected) }
func (h *Handler) decide(w http.ResponseWriter, r *http.Request, status string) {
	var q decisionRequest
	_ = json.NewDecoder(r.Body).Decode(&q)
	row, e := h.service.Decide(r.Context(), middleware.GetTenantID(r.Context()), middleware.GetEnvironment(r.Context()), mux.Vars(r)["id"], middleware.GetUserID(r.Context()), status, q.Reason)
	if e != nil {
		errJSON(w, http.StatusConflict, e.Error())
		return
	}
	writeJSON(w, http.StatusOK, row)
}
func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	e := h.service.Cancel(r.Context(), middleware.GetTenantID(r.Context()), middleware.GetEnvironment(r.Context()), mux.Vars(r)["id"], middleware.GetUserID(r.Context()))
	if e != nil {
		errJSON(w, http.StatusConflict, e.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": StatusCancelled})
}
func writeJSON(w http.ResponseWriter, s int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s)
	_ = json.NewEncoder(w).Encode(v)
}
func errJSON(w http.ResponseWriter, s int, m string) {
	writeJSON(w, s, map[string]any{"error": map[string]string{"type": "approval_error", "message": m}})
}
