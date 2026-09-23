package v3

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/lecodev-26/sentinelflow/internal/gateway/v3/middleware"
	"github.com/lecodev-26/sentinelflow/internal/idgen"
	"github.com/lecodev-26/sentinelflow/internal/rbac"
)

// === Tipos ===

type Webhook struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	URL         string    `json:"url"`
	Secret      string    `json:"secret,omitempty"`
	EventTypes  []string  `json:"event_types"`
	Description string    `json:"description,omitempty"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type createWebhookReq struct {
	URL         string   `json:"url"`
	EventTypes  []string `json:"event_types"`
	Description string   `json:"description,omitempty"`
}

// RegisterWebhooks añade las rutas CRUD al router existente.
func (h *Handlers) RegisterWebhooks(api *mux.Router) {
	read := api.PathPrefix("").Subrouter()
	if h.authMw != nil {
		read.Use(middleware.RequireScope(rbac.ScopeReadAdmin))
	}
	read.HandleFunc("/webhooks", h.ListWebhooks).Methods("GET")
	read.HandleFunc("/webhooks/{id}", h.GetWebhook).Methods("GET")
	read.HandleFunc("/webhooks/{id}/deliveries", h.ListWebhookDeliveries).Methods("GET")

	write := api.PathPrefix("").Subrouter()
	if h.authMw != nil {
		write.Use(middleware.RequireScope(rbac.ScopeWriteAdmin))
	}
	write.HandleFunc("/webhooks", h.CreateWebhook).Methods("POST")
	write.HandleFunc("/webhooks/{id}", h.DeleteWebhook).Methods("DELETE")
}

// === Handlers ===

func (h *Handlers) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	rows, err := h.svc.Pool().Query(r.Context(),
		`SELECT id, tenant_id, url, event_types, COALESCE(description,''), active, created_at, updated_at
 FROM webhooks WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	defer rows.Close()

	out := []Webhook{}
	for rows.Next() {
		var wh Webhook
		var evRaw []byte
		if err := rows.Scan(&wh.ID, &wh.TenantID, &wh.URL, &evRaw, &wh.Description, &wh.Active, &wh.CreatedAt, &wh.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "scan_error", err.Error())
			return
		}
		_ = json.Unmarshal(evRaw, &wh.EventTypes)
		wh.Secret = "" // nunca devolver secret en list
		out = append(out, wh)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"webhooks": out, "total": len(out)})
}

func (h *Handlers) GetWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	id := mux.Vars(r)["id"]

	var wh Webhook
	var evRaw []byte
	err := h.svc.Pool().QueryRow(r.Context(),
		`SELECT id, tenant_id, url, event_types, COALESCE(description,''), active, created_at, updated_at
 FROM webhooks WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	).Scan(&wh.ID, &wh.TenantID, &wh.URL, &evRaw, &wh.Description, &wh.Active, &wh.CreatedAt, &wh.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "webhook not found")
		return
	}
	_ = json.Unmarshal(evRaw, &wh.EventTypes)
	wh.Secret = ""
	writeJSON(w, http.StatusOK, wh)
}

func (h *Handlers) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())

	var req createWebhookReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "url is required")
		return
	}
	if len(req.EventTypes) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "event_types must not be empty")
		return
	}

	secret := randomHex(32)
	evJSON, _ := json.Marshal(req.EventTypes)

	wh := Webhook{
		ID:          "wh_" + idgen.RandomHex(8),
		TenantID:    tenantID,
		URL:         req.URL,
		EventTypes:  req.EventTypes,
		Description: req.Description,
		Active:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	_, err := h.svc.Pool().Exec(r.Context(),
		`INSERT INTO webhooks (id, tenant_id, url, secret, event_types, description, active, created_at, updated_at)
 VALUES ($1, $2, $3, $4, $5, $6, true, NOW(), NOW())`,
		wh.ID, wh.TenantID, wh.URL, secret, evJSON, wh.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	// Solo devolvemos el secret UNA VEZ en la creación.
	wh.Secret = secret
	writeJSON(w, http.StatusCreated, wh)
}

func (h *Handlers) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	id := mux.Vars(r)["id"]

	tag, err := h.svc.Pool().Exec(r.Context(),
		`DELETE FROM webhooks WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "not_found", "webhook not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

type deliveryView struct {
	ID           string     `json:"id"`
	EventID      string     `json:"event_id"`
	EventType    string     `json:"event_type"`
	Status       string     `json:"status"`
	Attempts     int        `json:"attempts"`
	ResponseCode *int       `json:"response_code,omitempty"`
	LastError    string     `json:"last_error,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	DeliveredAt  *time.Time `json:"delivered_at,omitempty"`
}

func (h *Handlers) ListWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	webhookID := mux.Vars(r)["id"]

	// Verificar ownership
	var owned string
	err := h.svc.Pool().QueryRow(r.Context(),
		`SELECT tenant_id FROM webhooks WHERE id = $1`, webhookID).Scan(&owned)
	if err != nil || owned != tenantID {
		writeError(w, http.StatusNotFound, "not_found", "webhook not found")
		return
	}

	rows, err := h.svc.Pool().Query(r.Context(),
		`SELECT id, event_id, event_type, status, attempts, response_code, COALESCE(last_error,''), created_at, delivered_at
 FROM webhook_deliveries WHERE webhook_id = $1 ORDER BY created_at DESC LIMIT 100`, webhookID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	defer rows.Close()

	out := []deliveryView{}
	for rows.Next() {
		var d deliveryView
		var code *int
		var delivered *time.Time
		if err := rows.Scan(&d.ID, &d.EventID, &d.EventType, &d.Status, &d.Attempts, &code, &d.LastError, &d.CreatedAt, &delivered); err != nil {
			writeError(w, http.StatusInternalServerError, "scan_error", err.Error())
			return
		}
		d.ResponseCode = code
		d.DeliveredAt = delivered
		out = append(out, d)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"deliveries": out, "total": len(out)})
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
