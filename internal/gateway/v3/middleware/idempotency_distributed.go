package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"time"

	"github.com/lecodev-26/sentinelflow/internal/logger"
	"github.com/lecodev-26/sentinelflow/internal/storage/postgres"
)

// IdempotencyDistributedMiddleware provee idempotencia compartida entre
// múltiples gateways usando PostgreSQL como store.
//
// Flujo:
//  1. Cliente envía Idempotency-Key: abc123
//  2. TryAcquire en Postgres (INSERT ... ON CONFLICT DO NOTHING)
//  3. Si somos el primero:
//     - Ejecutar handler
//     - Complete() con la respuesta
//  4. Si ya existe:
//     - status=completed → REPLAY response (X-Idempotency-Replayed: true)
//     - status=pending   → 409 Conflict (otro gateway procesando)
//     - status=failed    → reintentar (borrar y readquirir)
type IdempotencyDistributedMiddleware struct {
	repo    *postgres.IdempotencyRepo
	enabled bool
	ttl     time.Duration
}

// NewIdempotencyDistributedMiddleware crea un nuevo middleware
func NewIdempotencyDistributedMiddleware(repo *postgres.IdempotencyRepo, enabled bool, ttl time.Duration) *IdempotencyDistributedMiddleware {
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	return &IdempotencyDistributedMiddleware{
		repo:    repo,
		enabled: enabled,
		ttl:     ttl,
	}
}

// Handler envuelve un handler con idempotencia distribuida
func (m *IdempotencyDistributedMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled || m.repo == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Solo POST/PUT/DELETE
		if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodDelete {
			next.ServeHTTP(w, r)
			return
		}

		idemKey := r.Header.Get("Idempotency-Key")
		if idemKey == "" {
			next.ServeHTTP(w, r)
			return
		}

		tenantID := GetTenantID(r.Context())

		// Leer body para calcular request_hash
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "invalid_request", "error reading body")
			return
		}
		// Restaurar body con bytes.Reader (nativo, no hace falta helper)
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		// Calcular request_hash (SHA-256 del body)
		reqHash := sha256.Sum256(bodyBytes)
		requestHash := hex.EncodeToString(reqHash[:])

		// Intentar adquirir lock
		entry, isNew, err := m.repo.TryAcquire(r.Context(), idemKey, tenantID, requestHash, m.ttl)
		if err != nil {
			logger.Errorf("❌ Idempotency TryAcquire error: %v", err)
			// Fail-open: si el store falla, dejamos pasar la request
			next.ServeHTTP(w, r)
			return
		}

		// === CASO 1: Somos los primeros ===
		if isNew {
			logger.Infof("🔑 Idempotency: acquired lock key=%s tenant=%s", idemKey, tenantID)

			recorder := NewResponseWriterWrapper(w, true)
			next.ServeHTTP(recorder, r)

			if err := m.repo.Complete(r.Context(), idemKey, recorder.Buffer(), recorder.StatusCode()); err != nil {
				logger.Errorf("❌ Idempotency Complete error: %v", err)
			} else {
				logger.Infof("✅ Idempotency: completed key=%s status=%d", idemKey, recorder.StatusCode())
			}
			return
		}

		// === CASO 2: Ya existe una entrada ===

		// Verificar que el request_hash coincide
		if entry.RequestHash != requestHash {
			logger.Warnf("🚫 Idempotency: key=%s reused with different body", idemKey)
			writeErrorJSON(w, http.StatusConflict, "idempotency_conflict",
				"Idempotency-Key reused with a different request body")
			return
		}

		switch entry.Status {
		case postgres.IdempotencyCompleted:
			logger.Infof("🔄 Idempotency: replay key=%s status=%d", idemKey, entry.StatusCode)
			w.Header().Set("X-Idempotency-Replayed", "true")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(entry.StatusCode)
			w.Write(entry.Response)
			return

		case postgres.IdempotencyPending:
			logger.Warnf("⏳ Idempotency: conflict key=%s (in progress)", idemKey)
			writeErrorJSON(w, http.StatusConflict, "idempotency_in_progress",
				"Request with this Idempotency-Key is already in progress")
			return

		case postgres.IdempotencyFailed:
			logger.Infof("♻️  Idempotency: retry failed key=%s", idemKey)
			_ = m.repo.Delete(r.Context(), idemKey)

			_, isNew, err := m.repo.TryAcquire(r.Context(), idemKey, tenantID, requestHash, m.ttl)
			if err != nil || !isNew {
				writeErrorJSON(w, http.StatusConflict, "idempotency_conflict", "could not reacquire")
				return
			}

			recorder := NewResponseWriterWrapper(w, true)
			next.ServeHTTP(recorder, r)

			if err := m.repo.Complete(r.Context(), idemKey, recorder.Buffer(), recorder.StatusCode()); err != nil {
				logger.Errorf("❌ Idempotency Complete error: %v", err)
			}
			return
		}

		// Estado desconocido, dejar pasar
		next.ServeHTTP(w, r)
	})
}
